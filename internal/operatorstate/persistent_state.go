package operatorstate

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// documentVersion is the operator-state document format.
const documentVersion = "1.0.0"

// PersistentState declares the operator-state document, which the resource
// control plane updates when it adopts declared enablement.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "operator-state",
		Roots: []reflect.Type{reflect.TypeFor[Document]()},
		Facts: map[string]any{
			"documentVersion": documentVersion,
			"SchemaPath":      SchemaPath,
		},
	}
}
