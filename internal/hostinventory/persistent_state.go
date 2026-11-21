package hostinventory

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the cache envelope and both payload types. The
// payloads are JSON embedded in RawMessage and are not reachable by reflection
// from the envelope itself. Changes to the interpretation of an unchanged
// payload must also increment hostFactsSchemaVersion so old readers re-probe.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "host-facts-cache", WholeSource: true,
		Roots: []reflect.Type{
			reflect.TypeFor[hostFactsFile](),
			reflect.TypeFor[hostFactsEntry](), // legacy single-entry format
			reflect.TypeFor[Snapshot](),
			reflect.TypeFor[WorkloadSnapshot](),
		},
		Facts: map[string]any{"hostFactsSchemaVersion": hostFactsSchemaVersion},
	}
}
