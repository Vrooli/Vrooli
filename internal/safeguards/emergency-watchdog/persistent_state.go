package emergencywatchdog

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the watchdog publication receipt and the retained
// baseline restoration journal. The watchdog performs rollback and its
// readers are strict about inline phase and version literals, so its whole
// source participates.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:        "watchdog-publication",
		WholeSource: true,
		Roots: []reflect.Type{
			reflect.TypeFor[watchdogUpdateReceipt](),
			reflect.TypeFor[RetainedWatchdogBaseline](),
			reflect.TypeFor[baselineRestoreJournal](),
		},
	}
}
