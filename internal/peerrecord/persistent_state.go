package peerrecord

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState covers the peer envelope and the address-derived path used
// by readers and publishers. identity.go is retained as a whole-source fact.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:        "peer-records",
		WholeSource: true,
		Roots:       []reflect.Type{reflect.TypeFor[Record]()},
		Facts:       map[string]any{"SchemaVersion": SchemaVersion},
	}
}
