package domain

import "testing"

func TestRunPhaseOrdinalUsesDeclaredExecutionOrder(t *testing.T) {
	cases := []struct {
		phase      RunPhase
		ordinal    int
		canResume  bool
		isTerminal bool
	}{
		{RunPhaseQueued, 0, true, false},
		{RunPhaseInitializing, 1, true, false},
		{RunPhaseSandboxCreating, 2, true, false},
		{RunPhaseRunnerAcquiring, 3, true, false},
		{RunPhaseExecuting, 4, true, false},
		{RunPhaseCollectingResults, 5, false, false},
		{RunPhaseAwaitingReview, 6, false, false},
		{RunPhaseApplying, 7, false, false},
		{RunPhaseCleaningUp, 8, false, false},
		{RunPhaseCompleted, 9, false, true},
		{RunPhase("unknown"), 0, false, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.phase), func(t *testing.T) {
			if got := tc.phase.Ordinal(); got != tc.ordinal {
				t.Errorf("Ordinal() = %d, want %d", got, tc.ordinal)
			}
			if got := tc.phase.CanResumeFromPhase(); got != tc.canResume {
				t.Errorf("CanResumeFromPhase() = %v, want %v", got, tc.canResume)
			}
			if got := tc.phase.IsTerminal(); got != tc.isTerminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tc.isTerminal)
			}
		})
	}
}
