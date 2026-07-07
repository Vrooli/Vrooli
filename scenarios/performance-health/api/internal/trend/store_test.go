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
	series, err := store.Series(context.Background(), "demo", "", 10)
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
	if _, err := store.Series(context.Background(), "", "", 0); err == nil {
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

// [REQ:PH-TREND-002] Samples are scoped by (scenario, flow): scenario-level
// reads (flow="") never see flow-tagged rows and vice versa; LatestFlow returns
// the newest sample for one flow.
func TestSeriesAndLatestFilterByFlow(t *testing.T) {
	store := newStore(t)
	base := time.Now()
	mustInsert := func(flow string, lcp int64, at time.Time) {
		if err := store.Insert(context.Background(), Sample{
			Scenario: "demo", Flow: flow, LCPMs: lcp, CapturedAt: at,
			DrawnFPS: 48, DroppedFrameRate: 0.12, LongTaskTotalMs: 90, LongTaskMaxMs: 55,
			RasterTotalMs: 120, LayoutTotalMs: 30, PaintTotalMs: 20, InputEventCount: 24,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}
	mustInsert("", 100, base)
	mustInsert("scroll-list", 200, base.Add(1*time.Second))
	mustInsert("scroll-list", 300, base.Add(2*time.Second))
	mustInsert("drag-divider", 400, base.Add(3*time.Second))

	scenarioLevel, err := store.Series(context.Background(), "demo", "", 50)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(scenarioLevel) != 1 || scenarioLevel[0].LCPMs != 100 {
		t.Fatalf("scenario-level read must exclude flow rows, got %#v", scenarioLevel)
	}

	scroll, err := store.Series(context.Background(), "demo", "scroll-list", 50)
	if err != nil {
		t.Fatalf("Series flow: %v", err)
	}
	if len(scroll) != 2 {
		t.Fatalf("expected 2 scroll-list rows, got %d", len(scroll))
	}

	latest, found, err := store.LatestFlow(context.Background(), "demo", "scroll-list")
	if err != nil || !found {
		t.Fatalf("LatestFlow: found=%v err=%v", found, err)
	}
	if latest.LCPMs != 300 {
		t.Fatalf("LatestFlow must be newest-first, got %d", latest.LCPMs)
	}
	if latest.DrawnFPS != 48 || latest.DroppedFrameRate != 0.12 || latest.InputEventCount != 24 {
		t.Fatalf("interaction metrics must round-trip, got %#v", latest)
	}

	// Scenario-level Latest still ignores flow rows.
	sl, found, err := store.Latest(context.Background(), "demo")
	if err != nil || !found || sl.LCPMs != 100 {
		t.Fatalf("scenario-level Latest must read flow='' row, got %#v found=%v err=%v", sl, found, err)
	}
}

// [REQ:PH-TREND-002] EnsureColumns adds the flow column to an older,
// pre-flow perf_samples table without dropping existing rows.
func TestEnsureColumnsAddsFlow(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/old.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// Older shape: no flow column, one pre-existing row.
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE perf_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scenario TEXT NOT NULL,
			captured_at TEXT NOT NULL,
			go_build_ms INTEGER NOT NULL DEFAULT 0,
			ui_build_ms INTEGER NOT NULL DEFAULT 0,
			bundle_bytes INTEGER NOT NULL DEFAULT 0,
			lcp_ms INTEGER NOT NULL DEFAULT 0,
			startup_ms INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO perf_samples (scenario, captured_at, lcp_ms) VALUES ('demo','2026-01-01T00:00:00Z', 123);`); err != nil {
		t.Fatalf("seed old table: %v", err)
	}
	if err := EnsureColumns(context.Background(), db); err != nil {
		t.Fatalf("EnsureColumns: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("ensure schema after migration: %v", err)
	}
	store := NewStore(db)
	got, found, err := store.Latest(context.Background(), "demo")
	if err != nil || !found || got.LCPMs != 123 || got.Flow != "" {
		t.Fatalf("pre-existing row must survive with flow='', got %#v found=%v err=%v", got, found, err)
	}
}
