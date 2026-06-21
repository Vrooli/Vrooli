package trend

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return NewStore(db)
}

// [REQ:PH-TREND-001] Trend samples persist and read back newest-first; writes
// are additive.
func TestInsertAndSeries(t *testing.T) {
	store := newStore(t)
	for i, ms := range []int64{1000, 2000, 3000} {
		if err := store.Insert(context.Background(), Sample{
			Scenario:   "demo",
			CapturedAt: time.Now().Add(time.Duration(i) * time.Second),
			GoBuildMs:  ms,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	series, err := store.Series(context.Background(), "demo", 10)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(series) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(series))
	}
	if series[0].GoBuildMs != 3000 {
		t.Fatalf("expected newest-first, got %d", series[0].GoBuildMs)
	}
}

func TestSeriesRequiresScenario(t *testing.T) {
	store := newStore(t)
	if _, err := store.Series(context.Background(), "", 0); err == nil {
		t.Fatal("expected error for empty scenario")
	}
}

// [REQ:PH-TREND-001] Latest returns the newest sample, or found=false when none.
func TestLatest(t *testing.T) {
	store := newStore(t)
	if _, found, err := store.Latest(context.Background(), "demo"); err != nil || found {
		t.Fatalf("expected no sample initially, got found=%v err=%v", found, err)
	}
	for i, ms := range []int64{1000, 2000} {
		if err := store.Insert(context.Background(), Sample{
			Scenario: "demo", CapturedAt: time.Now().Add(time.Duration(i) * time.Second), GoBuildMs: ms,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	latest, found, err := store.Latest(context.Background(), "demo")
	if err != nil || !found {
		t.Fatalf("expected a sample, got found=%v err=%v", found, err)
	}
	if latest.GoBuildMs != 2000 {
		t.Fatalf("expected newest sample 2000, got %d", latest.GoBuildMs)
	}
}

// [REQ:PH-TREND-001] EnsureColumns is a no-op on a fresh DB (no table yet) and
// idempotent once columns exist; it never errors on re-run.
func TestEnsureColumns(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Fresh DB, table not created yet: must be a no-op (no error).
	if err := EnsureColumns(context.Background(), db); err != nil {
		t.Fatalf("EnsureColumns on fresh DB: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	// After create, idempotent (every column already exists).
	if err := EnsureColumns(context.Background(), db); err != nil {
		t.Fatalf("EnsureColumns after create: %v", err)
	}
}
