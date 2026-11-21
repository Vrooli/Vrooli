package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

type cleanupRootCounter struct{ calls int }

func (r *cleanupRootCounter) Resolve(context.Context) (string, error) {
	r.calls++
	return "", errors.New("unavailable fixture root")
}
func (*cleanupRootCounter) RecordWrite(context.Context) {}

func TestRunStateCleanupScheduleSeparatesRetentionFromLiveness(t *testing.T) {
	r, _, _ := newRecoveryTestReconciler(t, domain.RunnerTypeCodex)
	root := &cleanupRootCounter{}
	r.runStateResolver = root
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	r.clock = func() time.Time { return now }
	r.cleanupRunStateDirsIfDue(context.Background())
	if root.calls != 1 {
		t.Fatalf("initial sweep calls=%d", root.calls)
	}
	for i := 0; i < 100; i++ {
		r.cleanupRunStateDirsIfDue(context.Background())
	}
	now = now.Add(runStateCleanupInterval - time.Nanosecond)
	r.cleanupRunStateDirsIfDue(context.Background())
	if root.calls != 1 {
		t.Fatalf("liveness cycles repeated retention work: calls=%d", root.calls)
	}
	now = now.Add(time.Nanosecond)
	r.cleanupRunStateDirsIfDue(context.Background())
	if root.calls != 2 {
		t.Fatalf("due sweep calls=%d", root.calls)
	}
	// A backward clock step cannot postpone cleanup until the old wall clock
	// catches up; start a new cadence immediately.
	now = now.Add(-time.Hour)
	r.cleanupRunStateDirsIfDue(context.Background())
	if root.calls != 3 {
		t.Fatalf("clock rollback prevented sweep: calls=%d", root.calls)
	}
}
