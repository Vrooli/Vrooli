package flow

import (
	"testing"

	"plan-manager/internal/execution/flow/generated"
)

func TestPhaseLifecycleFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(s generated.PhaseLifecycleStatus, e generated.PhaseLifecycleEvent) (generated.PhaseLifecycleStatus, error) {
		next, err := TransitionPhaseLifecycle(PhaseLifecycleState{Status: s}, e)
		return next.Status, err
	})
}
