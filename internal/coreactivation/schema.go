package coreactivation

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

const (
	recordSchemaVersion     = 1
	enrollmentSchemaVersion = 1
)

// PersistentState declares the activation store: the operation record, the
// enrollment policy, and the canonical inputs of the digests the record binds
// (frozen layout and native drop-in inventory). This is the rollback
// machinery itself, with inline phase and failure literals behind exact
// readers, so its whole source participates.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:        "core-activation",
		WholeSource: true,
		Roots: []reflect.Type{
			reflect.TypeFor[Record](),
			reflect.TypeFor[Enrollment](),
			reflect.TypeFor[Layout](),
			reflect.TypeFor[nativeOverrideItem](),
		},
		Facts: map[string]any{
			"recordSchemaVersion":     recordSchemaVersion,
			"enrollmentSchemaVersion": enrollmentSchemaVersion,
			"store_witness":           initializedContent,
			"enrollment_witness":      enrollmentWitness,
		},
	}
}
