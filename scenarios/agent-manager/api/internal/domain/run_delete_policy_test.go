package domain

import "testing"

func TestCanDeleteRunRequiresResolvedTerminalExecution(t *testing.T) {
	for status, want := range map[RunStatus]bool{
		RunStatusComplete: true, RunStatusFailed: true, RunStatusCancelled: true,
		RunStatusPending: false, RunStatusStarting: false, RunStatusRunning: false,
		RunStatusParked: false, RunStatusNeedsReview: false, RunStatusUnknown: false,
	} {
		got, _ := CanDeleteRun(&Run{Status: status})
		if got != want {
			t.Errorf("CanDeleteRun(%q)=%v want %v", status, got, want)
		}
	}
}
