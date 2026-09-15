package readiness

import (
	"strings"
	"time"
)

import "testing"

func TestCheckReleaseReadinessHasTypedRefusals(t *testing.T) {
	cases := []struct {
		name   string
		record *ReleaseRecord
		want   string
	}{
		{"missing", nil, ReasonVerdictMissing},
		{"stale", &ReleaseRecord{VerdictPresent: true, ApprovedAtCommit: "old", GoalClosed: true}, ReasonVerdictStale},
		{"open", &ReleaseRecord{VerdictPresent: true, ApprovedAtCommit: "new", GoalRef: "goal-1"}, ReasonGoalOpen},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckReleaseReadiness("new", tc.record); err == nil || err.Error()[:len(tc.want)] != tc.want {
				t.Fatalf("error = %v, want prefix %q", err, tc.want)
			}
		})
	}
}

func TestCheckReleaseReadinessAllowsCurrentClosedGoal(t *testing.T) {
	if err := CheckReleaseReadiness("new", &ReleaseRecord{VerdictPresent: true, ApprovedAtCommit: "new", GoalRef: "goal-1", GoalClosed: true}); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReleaseReadinessAllowsValidCommitScopedWaiver(t *testing.T) {
	err := CheckReleaseReadiness("new", &ReleaseRecord{Waiver: &Waiver{Reason: "operator accepted documented limitation", Actor: "operator-1", Commit: "new", At: time.Now()}})
	if err != nil {
		t.Fatalf("valid waiver rejected: %v", err)
	}
}

func TestCheckReleaseReadinessRejectsReasonlessWaiver(t *testing.T) {
	err := CheckReleaseReadiness("new", &ReleaseRecord{Waiver: &Waiver{Actor: "operator-1", Commit: "new", At: time.Now()}})
	if err == nil || !strings.HasPrefix(err.Error(), ReasonWaiverInvalid) {
		t.Fatalf("expected %s, got %v", ReasonWaiverInvalid, err)
	}
}
