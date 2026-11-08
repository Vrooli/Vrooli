package manifest

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the resource manifest (acquisition, lifecycle, health, storage and acceleration declarations). The root control plane stores these
// declarations inside resource state and reads them back after a rollback, so
// their stored vocabulary is part of the persistent-state contract.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "resource-manifest",
		Roots: []reflect.Type{reflect.TypeFor[ResourceManifest]()},
	}
}
