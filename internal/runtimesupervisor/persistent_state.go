package runtimesupervisor

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the owner identity the running supervisor writes
// into its registry session metadata.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "supervisor-owner",
		Roots: []reflect.Type{reflect.TypeFor[supervisorOwnerIdentity]()},
	}
}
