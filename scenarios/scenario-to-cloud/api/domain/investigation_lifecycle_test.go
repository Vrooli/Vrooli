package domain

import "testing"

func TestInvestigationTransitionStatusAcceptsLifecycleAndRejectsInvalidEdges(t *testing.T) {
	for _, tc := range []struct {
		name string
		from InvestigationStatus
		to   InvestigationStatus
		want InvestigationStatus
		bad  bool
	}{
		{name: "starts running", from: InvestigationStatusPending, to: InvestigationStatusRunning, want: InvestigationStatusRunning},
		{name: "pending can be cancelled", from: InvestigationStatusPending, to: InvestigationStatusCancelled, want: InvestigationStatusCancelled},
		{name: "running can complete", from: InvestigationStatusRunning, to: InvestigationStatusCompleted, want: InvestigationStatusCompleted},
		{name: "running can fail", from: InvestigationStatusRunning, to: InvestigationStatusFailed, want: InvestigationStatusFailed},
		{name: "same state replay", from: InvestigationStatusRunning, to: InvestigationStatusRunning, want: InvestigationStatusRunning},
		{name: "cannot skip running", from: InvestigationStatusPending, to: InvestigationStatusCompleted, bad: true},
		{name: "cannot revive terminal", from: InvestigationStatusFailed, to: InvestigationStatusRunning, bad: true},
		{name: "cannot replace terminal outcome", from: InvestigationStatusCompleted, to: InvestigationStatusFailed, bad: true},
		{name: "unknown source", from: InvestigationStatus("unknown"), to: InvestigationStatusCancelled, bad: true},
		{name: "unknown target", from: InvestigationStatusRunning, to: InvestigationStatus("unknown"), bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inv := &Investigation{Status: tc.from}
			err := inv.TransitionStatus(tc.to)
			if tc.bad {
				if err == nil {
					t.Fatalf("transition %q -> %q unexpectedly succeeded", tc.from, tc.to)
				}
				if inv.Status != tc.from {
					t.Fatalf("rejected transition changed status to %q", inv.Status)
				}
				return
			}
			if err != nil {
				t.Fatalf("transition %q -> %q: %v", tc.from, tc.to, err)
			}
			if inv.Status != tc.want {
				t.Fatalf("status = %q, want %q", inv.Status, tc.want)
			}
		})
	}
}
