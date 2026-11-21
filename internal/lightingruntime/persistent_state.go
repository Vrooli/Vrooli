package lightingruntime

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the durable native reconciliation intent.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "host-lighting-intent",
		Roots: []reflect.Type{reflect.TypeFor[Intent]()},
		Facts: map[string]any{
			"lightingStateVersion": lightingStateVersion,
		},
	}
}
