package agentsession

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the owner evidence stored as JSON in the runtime
// registry's editor-lease rows.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "agent-sessions",
		Roots: []reflect.Type{reflect.TypeFor[Evidence]()},
	}
}
