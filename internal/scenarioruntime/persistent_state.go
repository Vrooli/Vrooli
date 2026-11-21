package scenarioruntime

import (
	"reflect"
	"time"

	"github.com/vrooli/vrooli/internal/statecontract"
	"github.com/vrooli/vrooli/internal/storagetime"
)

// PersistentState declares the runtime registry. The SQLite schema is the
// stored shape of its tables (a newer user_version is refused by an older
// binary); the row mirrors and the JSON documents stored in TEXT columns are
// the compiled envelopes. Its stored statuses, kinds and reasons are untyped
// constants an older binary queries by value, so every package-level constant
// participates (statecontract.ConstantSources).
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:           "runtime-registry",
		StampConstants: true,
		Roots: []reflect.Type{
			reflect.TypeFor[Instance](),
			reflect.TypeFor[SupervisorSession](),
			reflect.TypeFor[DemandTransition](),
			reflect.TypeFor[StartOperationStep](),
			reflect.TypeFor[ProcessRef](),
			reflect.TypeFor[Event](),
			reflect.TypeFor[EditorLease](),
			reflect.TypeFor[EditorLeaseProvenance](),
		},
		Facts: map[string]any{
			"SchemaVersion":                 SchemaVersion,
			"timestamp_format_sample":       storagetime.FormatUTC(time.Unix(1, 2)),
			"ddl":                           schemaSQL,
			"demandAuditEventSchemaVersion": demandAuditEventSchemaVersion,
		},
	}
}
