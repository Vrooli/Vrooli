package domain

import "testing"

func TestToolCallStatusTransitionGraph(t *testing.T) {
	statuses := []ToolCallStatus{
		StatusPending, StatusPendingApproval, StatusApproved, StatusRejected,
		StatusRunning, StatusCompleted, StatusFailed, StatusCancelled,
	}
	for _, from := range statuses {
		for _, to := range statuses {
			want := from == to
			switch from {
			case StatusPending:
				want = want || to == StatusPendingApproval || to == StatusApproved || to == StatusRunning || to == StatusCompleted || to == StatusFailed || to == StatusCancelled
			case StatusPendingApproval:
				want = want || to == StatusApproved || to == StatusRejected || to == StatusCancelled
			case StatusApproved:
				want = want || to == StatusRunning || to == StatusCompleted || to == StatusFailed || to == StatusCancelled
			case StatusRunning:
				want = want || to == StatusCompleted || to == StatusFailed || to == StatusCancelled
			}
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("%s -> %s allowed=%v, want %v", from, to, got, want)
			}
		}
	}
}

func TestToolCallStatusRejectsUnknownAndTerminalResurrection(t *testing.T) {
	for _, test := range []struct {
		from, to ToolCallStatus
	}{
		{StatusCompleted, StatusRunning},
		{StatusFailed, StatusPending},
		{StatusRejected, StatusApproved},
		{StatusCancelled, StatusCompleted},
		{"mystery", StatusCompleted},
		{StatusPending, "mystery"},
	} {
		if test.from.CanTransitionTo(test.to) {
			t.Errorf("unexpectedly allowed %s -> %s", test.from, test.to)
		}
	}
	var nilRecord *ToolCallRecord
	if err := nilRecord.TransitionTo(StatusFailed); err == nil {
		t.Fatal("nil record transition succeeded")
	}
	record := &ToolCallRecord{Status: StatusCompleted}
	if err := record.TransitionTo(StatusRunning); err == nil {
		t.Fatal("terminal resurrection succeeded")
	}
	if record.Status != StatusCompleted {
		t.Fatalf("rejected transition mutated record to %q", record.Status)
	}
}
