package capacity

import (
	"reflect"
	"time"

	"github.com/vrooli/vrooli/internal/statecontract"
	"github.com/vrooli/vrooli/internal/storagetime"
)

// PersistentState declares the capacity ledger: its SQLite schema and
// version and the claim mirror including the degrade-profile JSON column. Its
// stored statuses, kinds and policy keys are untyped constants, so every
// package-level constant participates (statecontract.ConstantSources).
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:           "capacity",
		StampConstants: true,
		Roots:          []reflect.Type{reflect.TypeFor[CapacityClaim]()},
		Facts: map[string]any{
			"SchemaVersion":           SchemaVersion,
			"timestamp_format_sample": storagetime.FormatUTC(time.Unix(1, 2)),
			"ddl":                     schemaSQL,
		},
	}
}
