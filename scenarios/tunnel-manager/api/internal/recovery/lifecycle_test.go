package recovery_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	recovery "tunnel-manager/internal/recovery"
)

func TestRecoveryStatusTransitionMatrix(t *testing.T) {
	statuses := []recovery.RecoveryStatus{
		recovery.StatusIdle,
		recovery.StatusMonitoring,
		recovery.StatusRecovering,
		recovery.StatusCircuitOpen,
	}
	allowed := map[recovery.RecoveryStatus]map[recovery.RecoveryStatus]bool{
		recovery.StatusIdle: {
			recovery.StatusMonitoring: true,
			recovery.StatusRecovering: true,
		},
		recovery.StatusMonitoring: {
			recovery.StatusIdle:       true,
			recovery.StatusRecovering: true,
		},
		recovery.StatusRecovering: {
			recovery.StatusIdle:        true,
			recovery.StatusMonitoring:  true,
			recovery.StatusCircuitOpen: true,
		},
		recovery.StatusCircuitOpen: {
			recovery.StatusIdle:       true,
			recovery.StatusMonitoring: true,
			recovery.StatusRecovering: true,
		},
	}

	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				state := recovery.RecoveryState{Status: from}
				err := state.TransitionStatus(to)
				want := from == to || allowed[from][to]
				if want {
					require.NoError(t, err)
					require.Equal(t, to, state.Status)
					return
				}
				require.Error(t, err)
				require.Equal(t, from, state.Status, "rejected transitions leave the source state unchanged")
			})
		}
	}
}

func TestRecoveryStatusTransitionRejectsInvalidState(t *testing.T) {
	state := recovery.RecoveryState{Status: recovery.RecoveryStatus("corrupt")}
	before := state
	require.Error(t, state.TransitionStatus(recovery.StatusIdle))
	require.Equal(t, before, state, "unknown source state is preserved")

	state = recovery.RecoveryState{Status: recovery.StatusIdle}
	before = state
	require.Error(t, state.TransitionStatus(recovery.RecoveryStatus("corrupt")))
	require.Equal(t, before, state, "unknown target state is rejected without mutation")

	var nilState *recovery.RecoveryState
	require.Error(t, nilState.TransitionStatus(recovery.StatusIdle))
}

func TestCircuitOpenIsDerivedFromStatus(t *testing.T) {
	require.False(t, (recovery.RecoveryState{Status: recovery.StatusMonitoring}).CircuitIsOpen())
	require.True(t, (recovery.RecoveryState{Status: recovery.StatusCircuitOpen}).CircuitIsOpen())
}
