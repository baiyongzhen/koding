package index_test

import (
	"fmt"
	"testing"

	"koding/klient/machine/index"
)

func TestEntryPromiseString(t *testing.T) {
	tests := []struct {
		EP     index.EntryPromise
		Result string
	}{
		{
			// 0 //
			EP:     index.EntryPromiseSync,
			Result: "S----",
		},
		{
			// 1 //
			EP:     index.EntryPromiseSync | index.EntryPromiseDel | index.EntryPromiseUnlink,
			Result: "S--DN",
		},
		{
			// 2 //
			EP:     index.EntryPromiseAdd,
			Result: "-A---",
		},
		{
			// 3 //
			EP:     0,
			Result: "-----",
		},
	}

	for i, test := range tests {
		test := test // Capture range variable.
		t.Run(fmt.Sprintf("test_no_%d", i), func(t *testing.T) {
			t.Parallel()

			if got := test.EP.String(); got != test.Result {
				t.Errorf("want ep string = %q; got %q", test.Result, got)
			}
		})
	}
}
