package flow

import (
	"testing"

	"github.com/ecosystem-manager/api/internal/tasks/flow/generated"
)

func TestLifecycleFormalReplay(t *testing.T) { // [REQ:EM-FLOW-001]
	generated.RunReplay(t, func(status generated.Status, event generated.Event) (generated.Status, error) {
		return TransitionLifecycle(status, event)
	})
}
