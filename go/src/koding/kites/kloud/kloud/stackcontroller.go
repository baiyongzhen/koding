package kloud

import (
	"koding/kites/kloud/contexthelper/publickeys"
	"koding/kites/kloud/contexthelper/request"
	"koding/kites/kloud/eventer"

	"github.com/koding/kite"
	"golang.org/x/net/context"
)

// GroupNameKey is used to pass group name to stack handler.
var GroupNameKey struct {
	byte `key:"groupName"`
}

func GroupFromContext(ctx context.Context) (string, bool) {
	groupName, ok := ctx.Value(GroupNameKey).(string)

	if !ok || groupName == "" {
		return "", false
	}

	return groupName, true
}

// Stacker is a provider-specific handler that implements team methods.
type Stacker interface {
	Apply(context.Context) (interface{}, error)
	Authenticate(context.Context) (interface{}, error)
	Bootstrap(context.Context) (interface{}, error)
	Plan(context.Context) (interface{}, error)
}

// StackProvider is responsible for creating stack providers.
type StackProvider interface {
	Stack(ctx context.Context) (Stacker, error)
}

// StackFunc handles execution of a single team method.
type StackFunc func(Stacker, context.Context) (interface{}, error)

// stackMethod routes the team method call to a requested provider.
func (k *Kloud) stackMethod(r *kite.Request, fn StackFunc) (interface{}, error) {
	if r.Args == nil {
		return nil, NewError(ErrNoArguments)
	}

	var args struct {
		Provider    string `json:"provider"`
		StackID     string `json:"stackId,omitempty"`
		GroupName   string `json:"groupName,omitempty"`
		Debug       bool   `json:"debug,omitempty"`
		Impersonate string `json:"impersonate,omitempty"` // only for kloudctl
	}

	// Unamrshal common arguments.
	if err := r.Args.One().Unmarshal(&args); err != nil {
		return nil, err
	}

	// TODO(rjeczalik): compatibility code, remove
	if args.Provider == "" {
		args.Provider = "aws"
	}

	if r.Auth != nil && r.Auth.Type == "kloudctl" && r.Auth.Key == KloudSecretKey {
		// kloudctl is not authenticated with username, let it overwrite it
		r.Username = args.Impersonate
	}

	k.Log.Debug("Called %q by %q with %q", r.Method, r.Username, r.Args.Raw)

	groupName := args.GroupName
	if groupName == "" {
		groupName = "koding"
	}

	p, ok := k.providers[args.Provider].(StackProvider)
	if !ok {
		return nil, NewError(ErrProviderNotFound)
	}

	// Build context value.
	ctx := request.NewContext(context.Background(), r)
	ctx = context.WithValue(ctx, GroupNameKey, groupName)
	if k.PublicKeys != nil {
		ctx = publickeys.NewContext(ctx, k.PublicKeys)
	}

	if k.ContextCreator != nil {
		ctx = k.ContextCreator(ctx)
	}

	if args.StackID != "" {
		evID := r.Method + "-" + args.StackID
		ctx = eventer.NewContext(ctx, k.NewEventer(evID))
	}

	if args.Debug {
		ctx = k.setTraceID(r.Username, r.Method, ctx)
	}

	// Create stack handler.
	s, err := p.Stack(ctx)
	if err != nil {
		return nil, err
	}

	// Currently only apply method is asynchronous, rest
	// of the is sync. That's why the fn execution is synchronous here,
	// and the fn itself emits events if needed.
	//
	// This differs from k.coreMethods.
	resp, err := fn(s, ctx)

	// Do not log error in production as most of them are expected:
	//
	//  - authenticate errors due to invalid credentials
	//  - plan errors due to invalud user input
	//
	// TODO(rjeczalik): Refactor errors so the user-originated have different
	// type and log unexpected errors with k.Log.Error().
	if err != nil {
		k.Log.Debug("method %q for user %q failed: %s", r.Method, r.Username, err)
	}

	return resp, err
}
