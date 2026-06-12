package effectiveness

import (
	"sync"
	"testing"

	"github.com/vrooli/maturity-go/dimensions"

	"github.com/ecosystem-manager/api/pkg/internal/testdb"
)

func newStore(t *testing.T) *SQLiteStore {
	t.Helper()
	return NewSQLiteStore(testdb.NewSQLite(t, Schema()))
}

func TestSQLiteStoreRoundTrip(t *testing.T) { // [REQ:EM-P1-002]
	store := newStore(t)

	if err := store.Record(CreditEvent{
		SkillID:               "lint-fix",
		TargetDimension:       standards,
		ClosedByDimension:     map[dimensions.Dimension]int{standards: 3},
		IntroducedByDimension: map[dimensions.Dimension]int{"tests": 1},
		Tokens:                1500,
	}); err != nil {
		t.Fatalf("record: %v", err)
	}

	got, ok, err := store.Get("lint-fix", standards)
	if err != nil || !ok {
		t.Fatalf("get standards: ok=%v err=%v", ok, err)
	}
	if got.ClosedCount != 3 || got.TotalRuns != 1 || got.TotalTokens != 1500 {
		t.Fatalf("unexpected target stat: %+v", got)
	}

	tdebt, ok, _ := store.Get("lint-fix", "tests")
	if !ok || tdebt.IntroducedCount != 1 || tdebt.TotalRuns != 0 {
		t.Fatalf("expected collateral tests debt with zero runs, got %+v (ok=%v)", tdebt, ok)
	}

	// No row for an unobserved pair (cold start).
	if _, ok, _ := store.Get("lint-fix", "security"); ok {
		t.Fatal("expected no row for unobserved (skill, dimension)")
	}
}

func TestSQLiteStoreConcurrentUpsert(t *testing.T) {
	store := newStore(t)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = store.Record(CreditEvent{
				SkillID:           "refactor",
				TargetDimension:   standards,
				ClosedByDimension: map[dimensions.Dimension]int{standards: 1},
				Tokens:            10,
			})
		}()
	}
	wg.Wait()

	got, ok, err := store.Get("refactor", standards)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.TotalRuns != n || got.ClosedCount != n || got.TotalTokens != n*10 {
		t.Fatalf("concurrent upserts lost increments: %+v", got)
	}
}

// max(last_run_at, excluded.last_run_at) must keep the most recent run time
// across upserts — the SQLite analogue of the Postgres GREATEST() it replaced.
func TestSQLiteStoreLastRunAtMonotonic(t *testing.T) {
	store := newStore(t)
	for i := 0; i < 3; i++ {
		if err := store.Record(CreditEvent{
			SkillID:           "doc-skill",
			TargetDimension:   standards,
			ClosedByDimension: map[dimensions.Dimension]int{standards: 1},
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	got, ok, err := store.Get("doc-skill", standards)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.TotalRuns != 3 {
		t.Fatalf("expected 3 runs accumulated, got %d", got.TotalRuns)
	}
	if got.LastRunAt.IsZero() {
		t.Fatal("last_run_at should be populated after records")
	}
}
