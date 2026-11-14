package process

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the per-step process records under the scenario
// state directory and the incarnation format pinned records are compared
// with (also stored in supervisor metadata).
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:  "process-records",
		Roots: []reflect.Type{reflect.TypeFor[Record]()},
		Facts: map[string]any{"incarnation_format_sample": incarnationFormatSample()},
	}
}
