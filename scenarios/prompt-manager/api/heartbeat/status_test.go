package heartbeat

import "testing"

func TestIsTerminalStatus(t *testing.T) {
	terminal := []string{
		"RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED",
		"complete", "failed", "cancelled",
	}
	for _, s := range terminal {
		if !IsTerminalStatus(s) {
			t.Errorf("expected %q to be terminal", s)
		}
	}

	nonTerminal := []string{
		"RUN_STATUS_RUNNING", "running", "pending", "queued", "", "UNKNOWN",
	}
	for _, s := range nonTerminal {
		if IsTerminalStatus(s) {
			t.Errorf("expected %q to NOT be terminal", s)
		}
	}
}

func TestIsFailedStatus(t *testing.T) {
	failed := []string{"RUN_STATUS_FAILED", "failed"}
	for _, s := range failed {
		if !IsFailedStatus(s) {
			t.Errorf("expected %q to be failed", s)
		}
	}

	notFailed := []string{
		"RUN_STATUS_COMPLETE", "complete", "cancelled",
		"RUN_STATUS_CANCELLED", "running", "",
	}
	for _, s := range notFailed {
		if IsFailedStatus(s) {
			t.Errorf("expected %q to NOT be failed", s)
		}
	}
}

func TestIsCancelledStatus(t *testing.T) {
	cancelled := []string{"RUN_STATUS_CANCELLED", "cancelled"}
	for _, s := range cancelled {
		if !IsCancelledStatus(s) {
			t.Errorf("expected %q to be cancelled", s)
		}
	}

	notCancelled := []string{
		"RUN_STATUS_COMPLETE", "complete", "failed",
		"RUN_STATUS_FAILED", "running", "",
	}
	for _, s := range notCancelled {
		if IsCancelledStatus(s) {
			t.Errorf("expected %q to NOT be cancelled", s)
		}
	}
}
