package flow

import (
	"testing"

	"{{SCENARIO_ID}}/scenarios/channel-manager/api/internal/channelmanager/flow/generated"
)

func TestActionLifecycleFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(status generated.Status, event generated.Event) (generated.Status, error) {
		return TransitionActionLifecycle(status, event)
	})
}
