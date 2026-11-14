package flow_test

import (
	"testing"

	generated "plan-manager/internal/plans/phase/flow/generated"
)

func TestGeneratedPhaseLifecycleReplaysFormalMatrixAndTraces(t *testing.T) {
	generated.RunReplay(t, generated.TransitionPhaseStatus)
}
