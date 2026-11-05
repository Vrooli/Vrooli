package domain

import "testing"

func TestTaskAttemptStatusTransitionsKeepExecutionCompletionNeutral(t *testing.T) {
	cases := []struct {
		from, to TaskStatus
		want     bool
	}{
		{TaskStatusRunning, TaskStatusCompleted, true},
		{TaskStatusNeedsReview, TaskStatusCompleted, true},
		{TaskStatusCompleted, TaskStatusRunning, true},
		{TaskStatusFailed, TaskStatusRunning, true},
		{TaskStatusCancelled, TaskStatusRunning, false},
		{TaskStatusApproved, TaskStatusRunning, false},
		{TaskStatusRejected, TaskStatusRunning, false},
	}
	for _, tc := range cases {
		if got, _ := tc.from.CanTransitionTo(tc.to); got != tc.want {
			t.Errorf("%s -> %s allowed=%v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
	for _, tc := range []struct {
		status   TaskStatus
		terminal bool
	}{
		{TaskStatusCompleted, false},
		{TaskStatusFailed, false},
		{TaskStatusCancelled, true},
		{TaskStatusApproved, true},
		{TaskStatusRejected, true},
	} {
		if got := tc.status.IsTerminal(); got != tc.terminal {
			t.Errorf("%s terminal=%v, want %v", tc.status, got, tc.terminal)
		}
	}
}
