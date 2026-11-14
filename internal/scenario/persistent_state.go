package scenario

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the scenario CLI contract (install, build and invocation declarations a resource manifest carries). The root control plane stores these
// declarations inside resource state and reads them back after a rollback, so
// their stored vocabulary is part of the persistent-state contract.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "scenario-contract",
		Roots: []reflect.Type{reflect.TypeFor[CLIConfig]()},
		Facts: map[string]any{"authenticationProfileVersion": authenticationProfileVersion},
	}
}
