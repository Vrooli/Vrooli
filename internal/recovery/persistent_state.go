package recovery

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// PersistentState declares the recovery broker and coding-agent admission
// records and the versioned coding-agent execution sidecar. Its outcome and
// version literals sit inline behind a strict codec, so its whole source
// participates.
func PersistentState() statecontract.Owner {
	return statecontract.Owner{
		Name:        "coding-agent-recovery",
		WholeSource: true,
		Roots: []reflect.Type{
			reflect.TypeFor[Record](),
			reflect.TypeFor[CodingAgentRecord](),
			reflect.TypeFor[CodingAgentExecution](),
		},
	}
}
