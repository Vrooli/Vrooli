package flow

import core "channel-manager/internal/channelmanager"

type (
	ActionStatus = core.ActionStatus
	ActionEvent  = core.ActionEvent
)

// TransitionAction delegates formal replay to the production transition table.
func TransitionAction(status ActionStatus, event ActionEvent) (ActionStatus, error) {
	return core.TransitionAction(status, event)
}
