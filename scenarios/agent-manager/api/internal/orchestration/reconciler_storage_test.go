package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

type evictRunsFake struct {
	repository.RunRepository
	ids []uuid.UUID
}

func (f evictRunsFake) ListImportedRunIDsOldestFirst(_ context.Context, limit int) ([]uuid.UUID, error) {
	if limit < len(f.ids) {
		return f.ids[:limit], nil
	}
	return f.ids, nil
}

func (f evictRunsFake) DeleteImportedRun(context.Context, uuid.UUID) error { return nil }

func (f evictRunsFake) DeleteEmptyImportedRuns(context.Context) (int, error) { return 0, nil }

type evictEventStoreFake struct {
	event.Store
	onDelete func(uuid.UUID)
}

func (f *evictEventStoreFake) Delete(_ context.Context, id uuid.UUID) error {
	if f.onDelete != nil {
		f.onDelete(id)
	}
	return nil
}

// Eviction removes oldest imported runs until the live-byte measure is back
// under budget, dropping each run's search docs, and stops as soon as the
// target is met rather than draining the whole batch.
func TestEvictImportedCorpusStopsAtBudget(t *testing.T) {
	live := int64(100)
	fakeEvents := &evictEventStoreFake{onDelete: func(uuid.UUID) { live -= 30 }}
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	var searchDeleted []string

	r := NewReconciler(evictRunsFake{ids: ids}, nil,
		WithReconcilerEvents(fakeEvents),
		WithReconcilerLiveBytes(func(context.Context) (int64, error) { return live, nil }),
		WithReconcilerSearchDeleteRun(func(_ context.Context, id string) error {
			searchDeleted = append(searchDeleted, id)
			return nil
		}),
	)
	r.levers.Storage.ImportedCorpusMaxBytes = 50
	r.levers.Storage.ImportedCorpusEvictionBatch = 10

	// 100 -> delete(70) -> delete(40<=50) stop: exactly 2 evictions.
	n, err := r.evictImportedCorpus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("evicted = %d, want 2 (stop as soon as under budget)", n)
	}
	if len(searchDeleted) != 2 {
		t.Fatalf("search deletes = %d, want 2 (each evicted run drops its docs)", len(searchDeleted))
	}
}

// Under budget, eviction is a no-op — canonical and recent imported data stays.
func TestEvictImportedCorpusNoopUnderBudget(t *testing.T) {
	evicted := 0
	r := NewReconciler(evictRunsFake{ids: []uuid.UUID{uuid.New()}}, nil,
		WithReconcilerEvents(&evictEventStoreFake{onDelete: func(uuid.UUID) { evicted++ }}),
		WithReconcilerLiveBytes(func(context.Context) (int64, error) { return 10, nil }),
		WithReconcilerSearchDeleteRun(func(context.Context, string) error { return nil }),
	)
	r.levers.Storage.ImportedCorpusMaxBytes = 50

	if n, err := r.evictImportedCorpus(context.Background()); err != nil || n != 0 {
		t.Fatalf("evictImportedCorpus = %d, %v; want 0, nil under budget", n, err)
	}
	if evicted != 0 {
		t.Fatalf("deleted %d runs while under budget; want 0", evicted)
	}
}

// Search-index reclaim (a full FTS merge) runs only under storage pressure:
// fired when live bytes approach the budget, skipped when comfortably under.
func TestReclaimSearchIndexGatedByPressure(t *testing.T) {
	cases := []struct {
		name string
		live int64
		want bool
	}{
		{"under budget: skip", 4 << 30, false},  // 4 GiB vs 8 GiB target → <70%
		{"near budget: reclaim", 7 << 30, true}, // 7 GiB vs 8 GiB target → ≥70%
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			optimized := false
			r := NewReconciler(evictRunsFake{}, nil,
				WithReconcilerLiveBytes(func(context.Context) (int64, error) { return tc.live, nil }),
			)
			r.SetSearchOptimize(func(context.Context) error { optimized = true; return nil })
			r.levers.Storage.ImportedCorpusMaxBytes = 8 << 30
			if err := r.reclaimSearchIndex(context.Background()); err != nil {
				t.Fatal(err)
			}
			if optimized != tc.want {
				t.Fatalf("optimized=%v, want %v at live=%d", optimized, tc.want, tc.live)
			}
		})
	}
}

// A zero budget disables eviction entirely.
func TestEvictImportedCorpusDisabledWhenBudgetZero(t *testing.T) {
	r := NewReconciler(evictRunsFake{ids: []uuid.UUID{uuid.New()}}, nil,
		WithReconcilerEvents(&evictEventStoreFake{}),
		WithReconcilerLiveBytes(func(context.Context) (int64, error) { return 1 << 40, nil }),
	)
	r.levers.Storage.ImportedCorpusMaxBytes = 0

	if n, err := r.evictImportedCorpus(context.Background()); err != nil || n != 0 {
		t.Fatalf("evictImportedCorpus = %d, %v; want 0, nil when disabled", n, err)
	}
}

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
