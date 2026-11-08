package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/orchestration/testutil"
)

type storageMaintainerStub struct {
	calls int
	err   error
}

func (s *storageMaintainerStub) Maintain(context.Context) error {
	s.calls++
	return s.err
}

// TestReconcilerRunOnceMaintainsStorageAfterRetention proves every reconcile
// cycle hands the storage owner a bounded pass after the retention steps, and
// that its failure is reported without stopping the cycle.
func TestReconcilerRunOnceMaintainsStorageAfterRetention(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	reconciler := NewReconciler(repos.Runs, nil, WithReconcilerConfig(ReconcilerConfig{OrphanGracePeriod: 24 * time.Hour}))
	baseline := reconciler.RunOnce(context.Background())

	maintainer := &storageMaintainerStub{err: errors.New("database is locked")}
	reconciler.SetStorageMaintainer(maintainer)
	stats := reconciler.RunOnce(context.Background())

	if maintainer.calls != 1 {
		t.Fatalf("storage maintainer calls=%d, want 1 per cycle", maintainer.calls)
	}
	if len(stats.Errors) != len(baseline.Errors)+1 || !strings.Contains(strings.Join(stats.Errors, "\n"), "storage maintenance: database is locked") {
		t.Fatalf("storage failure not reported: %v (baseline %v)", stats.Errors, baseline.Errors)
	}
}
