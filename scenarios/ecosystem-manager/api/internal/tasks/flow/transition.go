package flow

import (
	"github.com/ecosystem-manager/api/internal/tasks/flow/generated"
)

// TransitionLifecycle is the hand-authored wrapper around the
// generated state machine for the tasks.lifecycle.api flow.
func TransitionLifecycle(status generated.Status, event generated.Event) (generated.Status, error) {
	return generated.TransitionLifecycleStatus(status, event)
}
