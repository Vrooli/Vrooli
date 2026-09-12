package flow

import (
	"{{SCENARIO_ID}}/scenarios/channel-manager/api/internal/channelmanager/flow/generated"
)

// TransitionActionLifecycle is the hand-authored wrapper around the
// generated state machine for the channelmanager.action-lifecycle.api flow.
func TransitionActionLifecycle(status generated.Status, event generated.Event) (generated.Status, error) {
	return generated.TransitionActionLifecycleStatus(status, event)
}
