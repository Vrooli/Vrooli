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

// [REQ:PH-TREND-001] CLS survives an insert/read round trip with its fractional
// magnitude intact. It is stored REAL rather than in one of the INTEGER
// millisecond columns because cumulative layout shift is a unitless ratio,
// typically well below 1 — an INTEGER column would persist every real reading
// as 0.
func TestCLSRoundTripsAsAFraction(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	store := NewStore(db)

	const want = 0.02939214801135392
	if err := store.Insert(context.Background(), Sample{Scenario: "demo", CLS: want, LCPMs: 248}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	latest, found, err := store.Latest(context.Background(), "demo")
	if err != nil || !found {
		t.Fatalf("expected a sample, got found=%v err=%v", found, err)
	}
	if latest.CLS != want {
		t.Errorf("CLS = %v, want %v (0 means the column or scan order regressed)", latest.CLS, want)
	}
	if latest.LCPMs != 248 {
		t.Errorf("LCPMs = %d, want 248 — a column-order slip would shift neighbouring values", latest.LCPMs)
	}
}

// [REQ:PH-TREND-001] EnsureColumns adds cls to a database created before the
// column existed, without disturbing rows already persisted. This is the
// upgrade path every existing install takes.
func TestEnsureColumnsAddsCLSToAnOlderDatabase(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// A pre-CLS table with one persisted sample.
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE perf_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scenario TEXT NOT NULL,
			flow TEXT NOT NULL DEFAULT '',
			captured_at TEXT NOT NULL,
			go_build_ms INTEGER NOT NULL DEFAULT 0,
			ui_build_ms INTEGER NOT NULL DEFAULT 0,
			bundle_bytes INTEGER NOT NULL DEFAULT 0,
			lcp_ms INTEGER NOT NULL DEFAULT 0,
			startup_ms INTEGER NOT NULL DEFAULT 0,
			slowest_component TEXT NOT NULL DEFAULT '',
			slowest_component_avg_ms REAL NOT NULL DEFAULT 0,
			slowest_component_max_ms REAL NOT NULL DEFAULT 0,
			drawn_fps REAL NOT NULL DEFAULT 0,
			dropped_frame_rate REAL NOT NULL DEFAULT 0,
			long_task_total_ms INTEGER NOT NULL DEFAULT 0,
			long_task_max_ms REAL NOT NULL DEFAULT 0,
			raster_total_ms REAL NOT NULL DEFAULT 0,
			layout_total_ms REAL NOT NULL DEFAULT 0,
			paint_total_ms REAL NOT NULL DEFAULT 0,
			input_event_count INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO perf_samples (scenario, captured_at, lcp_ms) VALUES ('demo', '2026-08-21T00:00:00Z', 512);`); err != nil {
		t.Fatalf("seed legacy table: %v", err)
	}

	if err := EnsureColumns(context.Background(), db); err != nil {
		t.Fatalf("EnsureColumns: %v", err)
	}
	// Idempotent on re-run.
	if err := EnsureColumns(context.Background(), db); err != nil {
		t.Fatalf("EnsureColumns re-run: %v", err)
	}

	latest, found, err := NewStore(db).Latest(context.Background(), "demo")
	if err != nil || !found {
		t.Fatalf("legacy row unreadable after migration: found=%v err=%v", found, err)
	}
	if latest.LCPMs != 512 {
		t.Errorf("pre-existing sample lost its LCP: got %d, want 512", latest.LCPMs)
	}
	if latest.CLS != 0 {
		t.Errorf("a row written before the column existed must default to 0, got %v", latest.CLS)
	}
	// The navigation columns are additive on the same upgrade path.
	if latest.LoadEventEndMs != 0 || latest.ResponseEndMs != 0 || latest.NavigationType != "" {
		t.Errorf("pre-navigation row must default to zero/empty, got load=%d response=%d type=%q",
			latest.LoadEventEndMs, latest.ResponseEndMs, latest.NavigationType)
	}
	// A sample written AFTER the migration must persist the new columns, which
	// proves the ALTERs landed rather than the reads merely defaulting.
	if err := NewStore(db).Insert(context.Background(), Sample{
		Scenario: "demo", LoadEventEndMs: 202, NavigationType: "navigate", CLS: 0.03,
	}); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
	after, _, err := NewStore(db).Latest(context.Background(), "demo")
	if err != nil {
		t.Fatalf("read after migration: %v", err)
	}
	if after.LoadEventEndMs != 202 || after.NavigationType != "navigate" || after.CLS != 0.03 {
		t.Errorf("migrated columns did not persist: load=%d type=%q cls=%v",
			after.LoadEventEndMs, after.NavigationType, after.CLS)
	}
}

// [REQ:PH-TREND-001] Navigation phases and the navigation type round-trip, and
// stay monotonic through the insert/scan column ordering. The ordering check is
// the cheap guard against a column-order slip: four plausible numbers in the
// wrong slots would otherwise look fine.
func TestNavigationTimingRoundTrips(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	store := NewStore(db)

	want := Sample{
		Scenario: "demo", LCPMs: 248, CLS: 0.03,
		ResponseEndMs: 4, DOMInteractiveMs: 14, DOMContentLoadedMs: 101, LoadEventEndMs: 202,
		NavigationType: "reload",
	}
	if err := store.Insert(context.Background(), want); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, found, err := store.Latest(context.Background(), "demo")
	if err != nil || !found {
		t.Fatalf("expected a sample, got found=%v err=%v", found, err)
	}
	for _, c := range []struct {
		name      string
		got, want int64
	}{
		{"ResponseEndMs", got.ResponseEndMs, want.ResponseEndMs},
		{"DOMInteractiveMs", got.DOMInteractiveMs, want.DOMInteractiveMs},
		{"DOMContentLoadedMs", got.DOMContentLoadedMs, want.DOMContentLoadedMs},
		{"LoadEventEndMs", got.LoadEventEndMs, want.LoadEventEndMs},
		{"LCPMs", got.LCPMs, want.LCPMs},
	} {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if got.NavigationType != "reload" {
		t.Errorf("NavigationType = %q, want reload — a reload is not comparable with a cold navigate", got.NavigationType)
	}
	if got.CLS != 0.03 {
		t.Errorf("CLS = %v, want 0.03 — neighbouring columns must not shift", got.CLS)
	}
	if !(got.ResponseEndMs <= got.DOMInteractiveMs &&
		got.DOMInteractiveMs <= got.DOMContentLoadedMs &&
		got.DOMContentLoadedMs <= got.LoadEventEndMs) {
		t.Errorf("navigation phases lost their ordering through the store: %d/%d/%d/%d",
			got.ResponseEndMs, got.DOMInteractiveMs, got.DOMContentLoadedMs, got.LoadEventEndMs)
	}
}
