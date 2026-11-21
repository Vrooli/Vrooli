package setpoint

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the sustain-window first-observation files.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "sustain-observations",
		Roots: []reflect.Type{reflect.TypeFor[fileObservation]()},
		Facts: map[string]any{"fallbackDocumentSchemaVersion": fallbackDocumentSchemaVersion},
	}
}
