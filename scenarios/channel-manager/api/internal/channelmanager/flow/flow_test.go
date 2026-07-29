package flow

import (
	"testing"

	core "channel-manager/internal/channelmanager"
	"channel-manager/internal/channelmanager/flow/generated"
	"github.com/stretchr/testify/require"
)

func TestActionLifecycleFormalReplay(t *testing.T) {
	generated.RunReplay(t, TransitionAction)
	next, err := TransitionAction(core.Scheduled, core.ActionMakeDue)
	require.NoError(t, err)
	require.Equal(t, core.Due, next)
}
