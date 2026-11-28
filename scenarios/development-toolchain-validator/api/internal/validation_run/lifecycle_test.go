package validation_run

import (
	"errors"
	"testing"
)

func TestRunLifecycleOwnsLegalEdgesAndRejectsReversal(t *testing.T) {
	for _, edge := range [][2]Status{
		{StatusQueued, StatusRunning},
		{StatusRunning, StatusEvaluating},
		{StatusRunning, StatusTerminal},
		{StatusEvaluating, StatusTerminal},
		{StatusRunning, StatusRunning},
	} {
		if err := TransitionStatus(edge[0], edge[1]); err != nil {
			t.Errorf("allowed edge %d -> %d: %v", edge[0], edge[1], err)
		}
	}
	for _, edge := range [][2]Status{
		{StatusQueued, StatusEvaluating},
		{StatusTerminal, StatusRunning},
		{StatusEvaluating, StatusQueued},
		{StatusUnspecified, StatusQueued},
		{StatusQueued, Status(99)},
	} {
		if err := TransitionStatus(edge[0], edge[1]); !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("invalid edge %d -> %d error=%v", edge[0], edge[1], err)
		}
	}
	run := Run{Status: StatusTerminal}
	if err := run.TransitionStatus(StatusRunning); !errors.Is(err, ErrInvalidTransition) || run.Status != StatusTerminal {
		t.Fatalf("terminal reversal error=%v changed state=%d", err, run.Status)
	}
}
