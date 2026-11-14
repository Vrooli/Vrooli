package operatorcapability

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// pendingQueueVersion is the operator input queue format.
const pendingQueueVersion = 1

// PersistentState declares the operator input queue, the capability
// inventory and the evidence references written beside them.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name: "operator-capability",
		Roots: []reflect.Type{
			reflect.TypeFor[Pending](),
			reflect.TypeFor[Inventory](),
			reflect.TypeFor[EvidenceReference](),
		},
		Facts: map[string]any{
			"pendingQueueVersion":   pendingQueueVersion,
			"ContractVersion":       ContractVersion,
			"EvidenceSchemaVersion": EvidenceSchemaVersion,
		},
	}
}
