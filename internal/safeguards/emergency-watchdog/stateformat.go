package emergencywatchdog

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// watchdogSchemaDigest pins the stored shapes of watchdog publication state:
// the update receipt, the retained baseline and the restore journal. A
// deliberate shape change must bump the version and re-pin.
const watchdogSchemaDigest = "1f9efc8c75c2cc93890e81e9b47daed01349e2b3a27dcabb0dac51acf2a15183"

// The watchdog writes its durable records into the runtime-home state
// directory. The disposal proposal log keeps a rotated predecessor, so both
// members of the family are declared.
const (
	watchdogStatePattern = "state/watchdog-*"
	watchdogReportPath   = "state/emergency-watchdog/last-report.json"
)

// StateFormat describes watchdog publication state as a versioned store.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[watchdogUpdateReceipt](),
		reflect.TypeFor[RetainedWatchdogBaseline](),
		reflect.TypeFor[baselineRestoreJournal](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:       "watchdog-publication",
		Write:      1,
		Reads:      []stateformat.Version{1},
		Schemas:    map[stateformat.Version]string{1: digest},
		StatePaths: []string{watchdogStatePattern, watchdogReportPath},
	}, nil
}
