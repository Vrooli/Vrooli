package cleanup

import "testing"

func TestCleanupStatusTransitionGraph(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		{"queue starts planning", StatusQueued, StatusPlanning, true},
		{"plan confirmation", StatusPlanned, StatusConfirmed, true},
		{"retry failed apply", StatusFailed, StatusApplying, true},
		{"retry blocked apply", StatusBlocked, StatusApplying, true},
		{"terminal completion", StatusApplying, StatusCompleted, true},
		{"same state metadata update", StatusApplying, StatusApplying, true},
		{"cannot skip confirmation", StatusPlanned, StatusApplying, false},
		{"cannot reopen completed", StatusCompleted, StatusApplying, false},
		{"cannot cancel completed", StatusCompleted, StatusCanceled, false},
		{"unspecified source", StatusUnspecified, StatusQueued, false},
		{"unknown target", StatusQueued, Status(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TransitionStatus(tt.from, tt.to)
			if got := err == nil; got != tt.want {
				t.Fatalf("TransitionStatus(%s, %s) error=%v, want allowed=%v", tt.from, tt.to, err, tt.want)
			}
		})
	}
}

func TestOperationTransitionKeepsStateOnInvalidEdge(t *testing.T) {
	op := Operation{Status: StatusCompleted}
	if err := op.Transition(StatusApplying); err == nil {
		t.Fatal("expected terminal reversal to be rejected")
	}
	if op.Status != StatusCompleted {
		t.Fatalf("invalid transition changed state to %s", op.Status)
	}
}
