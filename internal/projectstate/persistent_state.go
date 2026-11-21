package projectstate

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares migration and configuration marker formats. The
// project key and path algorithms are pinned separately as whole source.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "project-setup", WholeSource: true,
		Roots: []reflect.Type{reflect.TypeFor[MigrationLedger](), reflect.TypeFor[ConfigurationCompletion]()},
		Facts: map[string]any{"migrationLedgerSchemaVersion": migrationLedgerSchemaVersion},
	}
}
