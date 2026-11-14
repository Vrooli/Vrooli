package retention

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// fixtureClock is a fixed reference point so age assertions are exact.
var fixtureClock = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

// newFixtureDB creates a WAL-mode database with a system_events-shaped table and
// rows spread across the given ages, newest last.
func newFixtureDB(t *testing.T, rows int, spacing time.Duration) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		t.Fatalf("enable wal: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE system_events (
		id INTEGER PRIMARY KEY,
		occurred_at TEXT NOT NULL,
		payload TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_occurred_at ON system_events(occurred_at)`); err != nil {
		t.Fatalf("create index: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO system_events (occurred_at, payload) VALUES (?, ?)`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	payload := strings.Repeat("x", 512)
	for i := range rows {
		// Row 0 is the oldest.
		at := fixtureClock.Add(-spacing * time.Duration(rows-i))
		if _, err := stmt.Exec(at.Format(time.RFC3339Nano), payload); err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("close stmt: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return db, path
}

func newFixturePruner(t *testing.T, db *sql.DB, path string, mutate func(*SQLiteTableConfig)) *SQLiteTablePruner {
	t.Helper()
	cfg := SQLiteTableConfig{
		DB:              db,
		Path:            path,
		Table:           "system_events",
		TimeColumn:      "occurred_at",
		BatchSize:       100,
		Now:             func() time.Time { return fixtureClock },
		AllowFullVacuum: true,
		FreeSpace:       func(string) (int64, error) { return 1 << 40, nil },
	}
	if mutate != nil {
		mutate(&cfg)
	}
	pruner, err := NewSQLiteTablePruner(cfg)
	if err != nil {
		t.Fatalf("NewSQLiteTablePruner: %v", err)
	}
	return pruner
}

func rowCount(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM system_events`).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

func TestSQLitePruneByAgeDeletesOnlyRowsPastTheHorizon(t *testing.T) {
	// 40 rows one hour apart: 20 are older than 20h, 20 are not.
	db, path := newFixtureDB(t, 40, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)

	result, err := pruner.Prune(context.Background(), Budget{Name: "system_events", MaxAge: 20 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if got := rowCount(t, db); got != 20 {
		t.Fatalf("rows remaining = %d, want 20", got)
	}
	if result.Deleted != 20 {
		t.Errorf("Deleted = %d, want 20", result.Deleted)
	}
	if result.BoundBy != BoundAge {
		t.Errorf("BoundBy = %v, want age", result.BoundBy)
	}

	// Every surviving row must be inside the horizon.
	var oldest string
	if err := db.QueryRow(`SELECT MIN(occurred_at) FROM system_events`).Scan(&oldest); err != nil {
		t.Fatalf("min occurred_at: %v", err)
	}
	cutoff := fixtureClock.Add(-20 * time.Hour)
	parsed, err := time.Parse(time.RFC3339Nano, oldest)
	if err != nil {
		t.Fatalf("parse %q: %v", oldest, err)
	}
	if parsed.Before(cutoff) {
		t.Fatalf("oldest surviving row %v is older than the %v horizon", parsed, cutoff)
	}
}

func TestSQLitePruneByAgeDeletesNothingWhenEverythingIsRecent(t *testing.T) {
	// This is the autoheal condition exactly: a correctly configured age policy
	// that runs and frees nothing because no row is old enough.
	db, path := newFixtureDB(t, 200, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)

	result, err := pruner.Prune(context.Background(), Budget{Name: "system_events", MaxAge: 30 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.Deleted != 0 {
		t.Fatalf("Deleted = %d, want 0; a 30d horizon must not touch rows minutes old", result.Deleted)
	}
	if got := rowCount(t, db); got != 200 {
		t.Fatalf("rows remaining = %d, want 200", got)
	}
	// And because only an age bound was declared, nothing reports a size bound.
	if result.BoundBy != BoundNone {
		t.Errorf("BoundBy = %v, want none", result.BoundBy)
	}
}

func TestSQLitePruneBySizeStopsAtTheBound(t *testing.T) {
	db, path := newFixtureDB(t, 2000, time.Second)
	pruner := newFixturePruner(t, db, path, nil)

	before, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	ceiling := before.Bytes / 4
	result, err := pruner.Prune(context.Background(), Budget{Name: "system_events", MaxBytes: ceiling})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	after, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure after: %v", err)
	}
	if after.Bytes > ceiling {
		t.Fatalf("after = %d bytes, want at or below the %d ceiling", after.Bytes, ceiling)
	}
	if after.Items == 0 {
		t.Fatal("pruned the table to empty; the size bound must stop at the ceiling, not below it")
	}
	if result.BoundBy != BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes", result.BoundBy)
	}
	if result.FreedBytes <= 0 {
		t.Errorf("FreedBytes = %d, want the space actually returned to the file", result.FreedBytes)
	}
}

func TestSQLitePruneBySizeBindsWhenAgeCannot(t *testing.T) {
	// Both bounds declared, every row inside the horizon: only the size ceiling
	// can do anything. This is the assertion the whole plan turns on.
	db, path := newFixtureDB(t, 2000, time.Second)
	pruner := newFixturePruner(t, db, path, nil)

	before, _ := pruner.Measure(context.Background())
	result, err := pruner.Prune(context.Background(), Budget{
		Name:     "system_events",
		MaxAge:   30 * 24 * time.Hour,
		MaxBytes: before.Bytes / 3,
	})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.BoundBy != BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes; the size ceiling is the only bound that could bind", result.BoundBy)
	}
	if result.Deleted == 0 {
		t.Fatal("Deleted = 0; the size ceiling freed nothing")
	}
}

func TestSQLitePruneAgeBindsWhenSizeIsAlreadySatisfied(t *testing.T) {
	db, path := newFixtureDB(t, 400, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)

	before, _ := pruner.Measure(context.Background())
	result, err := pruner.Prune(context.Background(), Budget{
		Name:     "system_events",
		MaxAge:   100 * time.Hour,
		MaxBytes: before.Bytes * 10,
	})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.BoundBy != BoundAge {
		t.Fatalf("BoundBy = %v, want age; the size ceiling was never approached", result.BoundBy)
	}
	if got := rowCount(t, db); got != 100 {
		t.Fatalf("rows remaining = %d, want 100", got)
	}
}

func TestSQLiteDeletesAreBatched(t *testing.T) {
	db, path := newFixtureDB(t, 1000, time.Hour)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.BatchSize = 100 })

	// 900 rows are past the horizon, which at 100 per batch cannot be one
	// statement. Deleting 846M rows in one statement is how the WAL becomes the
	// new disk problem.
	if _, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	batches := 0
	checkpoints := 0
	for _, stage := range pruner.Stages() {
		switch stage {
		case "prune_age_batch":
			batches++
		case "wal_checkpoint":
			checkpoints++
		}
	}
	if batches < 9 {
		t.Fatalf("ran %d delete batches for 900 rows at batch size 100, want at least 9", batches)
	}
	if checkpoints < batches {
		t.Fatalf("ran %d checkpoints for %d batches; the WAL must be truncated between batches", checkpoints, batches)
	}
}

func TestSQLiteCompactionRunsAfterPruningNeverBefore(t *testing.T) {
	db, path := newFixtureDB(t, 1000, time.Hour)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.BatchSize = 100 })

	if _, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour}); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	stages := pruner.Stages()
	firstPrune, firstCompact := -1, -1
	for i, stage := range stages {
		if firstPrune < 0 && strings.HasPrefix(stage, "prune_") {
			firstPrune = i
		}
		if firstCompact < 0 && strings.HasPrefix(stage, "compact_") {
			firstCompact = i
		}
	}
	if firstPrune < 0 {
		t.Fatalf("no prune stage recorded in %v", stages)
	}
	if firstCompact < 0 {
		t.Fatalf("no compaction stage recorded in %v", stages)
	}
	// Reversing this order is the failure that cannot even run on the host this
	// package was written for: a compact-first pass would need roughly 453 GB
	// against 226 GB available.
	if firstCompact < firstPrune {
		t.Fatalf("compaction at index %d ran before pruning at index %d: %v", firstCompact, firstPrune, stages)
	}
}

func TestSQLiteCompactionSkippedAndReportedWhenFreeSpaceIsShort(t *testing.T) {
	db, path := newFixtureDB(t, 1000, time.Hour)
	// One byte free: the projected copy cannot possibly fit.
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.FreeSpace = func(string) (int64, error) { return 1, nil }
	})

	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if !result.CompactSkipped {
		t.Fatal("CompactSkipped = false; compaction must refuse rather than fail part-way through a write")
	}
	if result.CompactSkipReason == "" {
		t.Fatal("CompactSkipReason is empty; a skipped compaction the caller cannot surface is a silent one")
	}
	// The pruning itself must still have happened: the guard protects the copy,
	// not the deletes.
	if result.Deleted == 0 {
		t.Fatal("Deleted = 0; the free-space guard must not stop pruning")
	}
	for _, stage := range pruner.Stages() {
		if stage == "compact_full" {
			t.Fatal("a full VACUUM ran despite insufficient free space")
		}
	}
}

func TestSQLiteFreeSpaceGuardIsWhatSkipsCompaction(t *testing.T) {
	// The mutation check, expressed as a test rather than a manual edit: with
	// abundant free space the same fixture compacts, so the skip above is caused
	// by the guard and nothing else. If the guard were removed, the short-space
	// case would behave like this one and TestSQLiteCompactionSkipped... fails.
	db, path := newFixtureDB(t, 1000, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)

	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.CompactSkipped {
		t.Fatalf("CompactSkipped = true with 1 TiB free: %s", result.CompactSkipReason)
	}
	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatalf("read auto_vacuum: %v", err)
	}
	if mode != autoVacuumIncremental {
		t.Fatalf("auto_vacuum = %d after compaction, want %d", mode, autoVacuumIncremental)
	}
}

func TestSQLiteCompactionReturnsSpaceToTheFilesystem(t *testing.T) {
	db, path := newFixtureDB(t, 4000, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)

	before, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if _, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	after, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure after: %v", err)
	}
	// Without compaction, deleting rows leaves the pages in the file and the
	// measured size does not move at all.
	if after.Bytes >= before.Bytes {
		t.Fatalf("size went from %d to %d; freed pages were not returned to the filesystem", before.Bytes, after.Bytes)
	}
}

func TestEnsureIncrementalAutoVacuumIsIdempotent(t *testing.T) {
	db, _ := newFixtureDB(t, 50, time.Hour)
	ctx := context.Background()

	for i := range 3 {
		if err := EnsureIncrementalAutoVacuum(ctx, db, nil); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		var mode int
		if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
			t.Fatalf("read auto_vacuum: %v", err)
		}
		if mode != autoVacuumIncremental {
			t.Fatalf("call %d: auto_vacuum = %d, want %d", i, mode, autoVacuumIncremental)
		}
		// The data must survive every repeat: a migration that is idempotent in
		// pragma but not in content is worse than one that fails.
		if got := rowCount(t, db); got != 50 {
			t.Fatalf("call %d: rows = %d, want 50", i, got)
		}
	}
}

func TestSQLiteCancelledContextStopsMidPruneAndReportsIncomplete(t *testing.T) {
	db, path := newFixtureDB(t, 2000, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	batches := 0
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.BatchSize = 50 })
	// Cancel after the first batch by watching the recorded stages through a
	// goroutine-free hook: run the prune with a context cancelled by a stage
	// counter wrapper.
	wrapped := &cancellingExecer{Execer: db, after: 3, cancel: cancel, counter: &batches}
	pruner.cfg.DB = wrapped

	result, err := pruner.Prune(ctx, Budget{Name: "b", MaxAge: time.Hour})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if !result.Incomplete {
		t.Fatal("Incomplete = false after a cancelled prune")
	}
	// Partial progress must be reported, not discarded: a table needing hours to
	// reach budget would otherwise never converge.
	if result.Deleted == 0 {
		t.Fatal("Deleted = 0; the batches completed before the cancel were discarded")
	}
	remaining := rowCount(t, db)
	if remaining == 2000 {
		t.Fatal("no rows were removed before the cancel")
	}
	if remaining == 0 {
		t.Fatal("the cancel did not stop the prune")
	}
}

// cancellingExecer cancels a context after a fixed number of DELETE statements,
// so cancellation lands mid-prune deterministically.
type cancellingExecer struct {
	Execer
	after   int
	cancel  context.CancelFunc
	counter *int
}

func (c *cancellingExecer) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := c.Execer.ExecContext(ctx, query, args...)
	if strings.HasPrefix(query, "DELETE") && err == nil {
		*c.counter++
		if *c.counter >= c.after {
			c.cancel()
		}
	}
	return res, err
}

func TestSQLitePrunerRejectsNonIdentifierNames(t *testing.T) {
	db, path := newFixtureDB(t, 1, time.Hour)
	// Identifiers are interpolated into SQL because SQLite cannot bind them as
	// parameters, so anything that is not a bare identifier must be refused.
	for _, bad := range []string{"", "system events", `events"; DROP TABLE x; --`, "1events", "events;"} {
		if _, err := NewSQLiteTablePruner(SQLiteTableConfig{DB: db, Path: path, Table: bad, TimeColumn: "occurred_at"}); err == nil {
			t.Errorf("table %q: expected rejection, got none", bad)
		}
		if _, err := NewSQLiteTablePruner(SQLiteTableConfig{DB: db, Path: path, Table: "system_events", TimeColumn: bad}); err == nil {
			t.Errorf("time column %q: expected rejection, got none", bad)
		}
	}
}

func TestSQLitePrunerRequiresPathForTheGuard(t *testing.T) {
	db, _ := newFixtureDB(t, 1, time.Hour)
	if _, err := NewSQLiteTablePruner(SQLiteTableConfig{DB: db, Table: "system_events", TimeColumn: "occurred_at"}); err == nil {
		t.Fatal("expected a missing Path to be rejected; without it the free-space guard has nothing to measure")
	}
}

func TestSQLitePrunerHandlesIntegerTimeColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "epoch.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE events (id INTEGER PRIMARY KEY, at INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	for i := range 100 {
		at := fixtureClock.Add(-time.Duration(100-i) * time.Hour).Unix()
		if _, err := db.Exec(`INSERT INTO events (at) VALUES (?)`, at); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	pruner, err := NewSQLiteTablePruner(SQLiteTableConfig{
		DB: db, Path: path, Table: "events", TimeColumn: "at",
		Now:       func() time.Time { return fixtureClock },
		FreeSpace: func(string) (int64, error) { return 1 << 40, nil },
	})
	if err != nil {
		t.Fatalf("NewSQLiteTablePruner: %v", err)
	}
	if _, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 50 * time.Hour}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 50 {
		t.Fatalf("rows = %d, want 50; an epoch-integer time column was compared as text", n)
	}
}

func TestSQLitePrunerMeasureOnEmptyTable(t *testing.T) {
	db, path := newFixtureDB(t, 0, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)
	usage, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if usage.Items != 0 {
		t.Fatalf("Items = %d, want 0", usage.Items)
	}
	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: time.Hour, MaxBytes: 1 << 30})
	if err != nil {
		t.Fatalf("Prune on an empty table: %v", err)
	}
	if result.Deleted != 0 {
		t.Fatalf("Deleted = %d on an empty table", result.Deleted)
	}
}

func TestSQLiteFullVacuumRequiresExplicitPermission(t *testing.T) {
	// A full VACUUM rewrites the whole database. It must never happen as a side
	// effect of startup, only through an explicit operator command.
	db, path := newFixtureDB(t, 1000, time.Hour)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.AllowFullVacuum = false })

	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: 100 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	for _, stage := range pruner.Stages() {
		if stage == "compact_full" {
			t.Fatal("a full VACUUM ran without AllowFullVacuum")
		}
	}
	if !result.CompactSkipped || result.CompactSkipReason == "" {
		t.Fatal("a database left at auto_vacuum=0 must be reported, not silently left unshrunk")
	}
	// The pruning itself still happened; only the rewrite was withheld.
	if result.Deleted == 0 {
		t.Fatal("Deleted = 0; withholding the rewrite must not withhold the deletes")
	}
	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatalf("read auto_vacuum: %v", err)
	}
	if mode == autoVacuumIncremental {
		t.Fatal("the database was migrated to incremental auto-vacuum without permission")
	}
}

func TestSQLiteByteBoundStopsWhenDeletingCannotShrinkTheFile(t *testing.T) {
	// Without the rewrite, an auto_vacuum=0 database never returns pages. The
	// pruner must report the overage rather than delete the table to nothing
	// chasing a file size that will not move.
	db, path := newFixtureDB(t, 2000, time.Second)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.AllowFullVacuum = false })

	before, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxBytes: before.Bytes / 8})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	after, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure after: %v", err)
	}
	if after.Items == 0 {
		t.Fatal("the table was emptied chasing a ceiling the file could not reach")
	}
	if !result.OverBudget(Budget{MaxBytes: before.Bytes / 8}) {
		t.Fatal("the target is within its ceiling, so this test is asserting nothing")
	}
}

func TestSQLitePruneStopsAtItsWallClockAllowanceAndReportsIncomplete(t *testing.T) {
	// On a table with hundreds of millions of rows to remove, an unbounded cycle
	// would hold the write lock for hours and starve the ingest path.
	db, path := newFixtureDB(t, 5000, time.Hour)
	now := fixtureClock
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) {
		cfg.BatchSize = 10
		cfg.MaxDuration = 50 * time.Millisecond
		// Each clock read advances time, so the allowance is consumed
		// deterministically rather than by racing a real timer.
		cfg.Now = func() time.Time {
			now = now.Add(10 * time.Millisecond)
			return now
		}
	})

	result, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if !result.Incomplete {
		t.Fatal("Incomplete = false after the wall-clock allowance was exhausted")
	}
	remaining := rowCount(t, db)
	if remaining == 0 {
		t.Fatal("the allowance did not stop the prune")
	}
	if remaining == 5000 {
		t.Fatal("no progress was made before the allowance ran out")
	}
}

func TestSQLiteCompactionProjectsFromLivePayloadNotAllocatedSize(t *testing.T) {
	// The guard must project the VACUUM copy from the LIVE payload. Immediately
	// after a large delete on an auto_vacuum=0 database the allocated size has
	// not moved at all — every freed page is still in the file — so projecting
	// from the allocated size would refuse the very compaction that pruning just
	// made possible. That is the 453 GB case this whole ordering exists for.
	db, path := newFixtureDB(t, 4000, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)

	allocatedBefore, err := pruner.databaseBytes(context.Background())
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	// Delete almost everything, leaving the pages in the file.
	if _, err := db.Exec(`DELETE FROM system_events WHERE id <= 3900`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	allocatedAfter, err := pruner.databaseBytes(context.Background())
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	live, err := pruner.liveBytes(context.Background())
	if err != nil {
		t.Fatalf("liveBytes: %v", err)
	}
	if allocatedAfter < allocatedBefore/2 {
		t.Fatalf("allocated size fell to %d from %d; this fixture is not reproducing the freed-pages-retained condition", allocatedAfter, allocatedBefore)
	}
	if live >= allocatedAfter {
		t.Fatalf("liveBytes = %d is not below the allocated %d; the freelist was not accounted for", live, allocatedAfter)
	}

	// Free space that fits the live payload but NOT the allocated size. A guard
	// projecting from the allocated size would refuse here.
	pruner.cfg.FreeSpace = func(string) (int64, error) { return allocatedAfter / 2, nil }
	skipped, reason, err := pruner.compact(context.Background())
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if skipped {
		t.Fatalf("compaction was refused with room for the live payload: %s", reason)
	}
	final, err := pruner.databaseBytes(context.Background())
	if err != nil {
		t.Fatalf("databaseBytes after: %v", err)
	}
	if final >= allocatedAfter {
		t.Fatalf("file is %d after compaction, want below the %d it held", final, allocatedAfter)
	}
}

func TestSQLiteByteBoundEstimatesFromLivePayloadNotAllocatedSize(t *testing.T) {
	// The regression this guards: after a large delete on a database that has
	// not reclaimed, allocated size is unchanged while the row count has
	// collapsed. Estimating bytes-per-row from allocated/rows then overstates it
	// by orders of magnitude, and the next pass deletes nearly every surviving
	// row to shrink a file that only needed compacting.
	db, path := newFixtureDB(t, 4000, time.Hour)
	pruner := newFixturePruner(t, db, path, nil)
	ctx := context.Background()

	// Simulate the aftermath of a big prune: most rows gone, pages still held.
	if _, err := db.Exec(`DELETE FROM system_events WHERE id <= 3600`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	allocated, err := pruner.databaseBytes(ctx)
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	live, err := pruner.liveBytes(ctx)
	if err != nil {
		t.Fatalf("liveBytes: %v", err)
	}
	if live >= allocated {
		t.Fatalf("fixture is not reproducing the unreclaimed condition: live=%d allocated=%d", live, allocated)
	}

	// A ceiling that the 400 surviving rows already fit inside, but which the
	// still-inflated allocated size does not.
	ceiling := (live + allocated) / 2
	before := rowCount(t, db)
	if _, err := pruner.Prune(ctx, Budget{Name: "b", MaxBytes: ceiling}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	after := rowCount(t, db)
	if after == 0 {
		t.Fatal("every surviving row was deleted to shrink a file that only needed compacting")
	}
	if after < before/2 {
		t.Fatalf("rows fell from %d to %d; the per-row estimate came from allocated size, not live payload", before, after)
	}
}

func TestRebuildToBudgetKeepsNewestRowsAndShrinksTheFile(t *testing.T) {
	db, path := newFixtureDB(t, 5000, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)
	ctx := context.Background()

	before, err := pruner.databaseBytes(ctx)
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	newestBefore := maxOccurredAt(t, db)

	result, err := pruner.RebuildToBudget(ctx, Budget{Name: "system_events", MaxBytes: before / 8})
	if err != nil {
		t.Fatalf("RebuildToBudget: %v", err)
	}

	after, err := pruner.databaseBytes(ctx)
	if err != nil {
		t.Fatalf("databaseBytes after: %v", err)
	}
	if after >= before {
		t.Fatalf("file is %d after rebuild, want below the %d it held", after, before)
	}
	remaining := rowCount(t, db)
	if remaining == 0 {
		t.Fatal("the rebuild kept no rows")
	}
	if remaining >= 5000 {
		t.Fatalf("%d rows remain of 5000; nothing was removed", remaining)
	}
	if result.FreedBytes <= 0 {
		t.Errorf("FreedBytes = %d, want the space returned to the file", result.FreedBytes)
	}

	// The NEWEST rows must survive, not the oldest.
	if got := maxOccurredAt(t, db); got != newestBefore {
		t.Fatalf("newest surviving row is %q, want the %q that was newest before", got, newestBefore)
	}
	oldest := minOccurredAt(t, db)
	if oldest == firstOccurredAt(t) {
		t.Fatal("the oldest row survived; the rebuild kept the wrong end of the table")
	}
}

func TestRebuildToBudgetPreservesSchemaAndIndexes(t *testing.T) {
	db, path := newFixtureDB(t, 2000, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)
	ctx := context.Background()

	tableBefore := schemaSQL(t, db, "table", "system_events")
	indexesBefore := schemaSQL(t, db, "index", "idx_occurred_at")

	size, _ := pruner.databaseBytes(ctx)
	if _, err := pruner.RebuildToBudget(ctx, Budget{Name: "b", MaxBytes: size / 4}); err != nil {
		t.Fatalf("RebuildToBudget: %v", err)
	}

	if got := schemaSQL(t, db, "table", "system_events"); got != tableBefore {
		t.Errorf("table DDL changed:\n before: %s\n after:  %s", tableBefore, got)
	}
	if got := schemaSQL(t, db, "index", "idx_occurred_at"); got != indexesBefore {
		t.Errorf("index DDL changed:\n before: %s\n after:  %s", indexesBefore, got)
	}
	// The transient table must not survive.
	if schemaSQL(t, db, "table", "system_events"+rebuildSuffix) != "" {
		t.Error("the transient rebuild table was left behind")
	}
	// And the rebuilt table must still be usable through its index.
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM system_events WHERE occurred_at > ?`, "2000-01-01").Scan(&n); err != nil {
		t.Fatalf("query rebuilt table: %v", err)
	}
	if n == 0 {
		t.Fatal("the rebuilt table returned no rows through its index")
	}
}

func TestRebuildToBudgetRequiresAByteCeiling(t *testing.T) {
	db, path := newFixtureDB(t, 10, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)
	if _, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "b", MaxAge: time.Hour}); err == nil {
		t.Fatal("expected an age-only budget to be refused: it cannot size the surviving set")
	}
}

func TestRebuildToBudgetLeavesTheTableIntactOnFailure(t *testing.T) {
	// The rebuild runs in one transaction, so a failure must leave the original
	// table exactly as it was rather than half-migrated.
	db, path := newFixtureDB(t, 500, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)
	ctx := context.Background()
	before := rowCount(t, db)

	// A table that does not exist fails the copy statement mid-transaction.
	broken := *pruner
	broken.cfg.Table = "no_such_table"
	size, _ := pruner.databaseBytes(ctx)
	if _, err := broken.RebuildToBudget(ctx, Budget{Name: "b", MaxBytes: size / 4}); err == nil {
		t.Fatal("expected the rebuild to fail on a missing time column")
	}

	if got := rowCount(t, db); got != before {
		t.Fatalf("%d rows after a failed rebuild, want the original %d", got, before)
	}
	if schemaSQL(t, db, "index", "idx_occurred_at") == "" {
		t.Fatal("the original index was not restored after a failed rebuild")
	}
}

func TestIndexNamesParsesCreateStatements(t *testing.T) {
	cases := map[string]string{
		`CREATE INDEX idx_a ON t(c)`:                     "idx_a",
		`CREATE INDEX IF NOT EXISTS idx_b ON t (c DESC)`: "idx_b",
		`CREATE UNIQUE INDEX "idx_c" ON t(c)`:            "idx_c",
	}
	for ddl, want := range cases {
		got := indexNames([]string{ddl})
		if len(got) != 1 || got[0] != want {
			t.Errorf("indexNames(%q) = %v, want [%s]", ddl, got, want)
		}
	}
}

func maxOccurredAt(t *testing.T, db *sql.DB) string {
	t.Helper()
	var v string
	if err := db.QueryRow(`SELECT MAX(occurred_at) FROM system_events`).Scan(&v); err != nil {
		t.Fatalf("max occurred_at: %v", err)
	}
	return v
}

func minOccurredAt(t *testing.T, db *sql.DB) string {
	t.Helper()
	var v string
	if err := db.QueryRow(`SELECT MIN(occurred_at) FROM system_events`).Scan(&v); err != nil {
		t.Fatalf("min occurred_at: %v", err)
	}
	return v
}

func firstOccurredAt(t *testing.T) string {
	t.Helper()
	return fixtureClock.Add(-time.Minute * 5000).Format(time.RFC3339Nano)
}

func schemaSQL(t *testing.T, db *sql.DB, objType, name string) string {
	t.Helper()
	var ddl sql.NullString
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type=? AND name=?`, objType, name).Scan(&ddl)
	if errors.Is(err, sql.ErrNoRows) {
		return ""
	}
	if err != nil {
		t.Fatalf("read %s %s: %v", objType, name, err)
	}
	return ddl.String
}

func TestSQLitePrunerRejectsATimeColumnThatDoesNotExist(t *testing.T) {
	// SQLite reinterprets a double-quoted identifier that matches no column as a
	// string literal, so `ORDER BY "occured_at"` sorts every row by the same
	// constant and "oldest first" silently becomes an arbitrary order. A typo in
	// a manifest must fail loudly instead.
	db, path := newFixtureDB(t, 100, time.Minute)
	pruner := newFixturePruner(t, db, path, func(cfg *SQLiteTableConfig) { cfg.TimeColumn = "occured_at" })

	_, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxAge: time.Hour})
	if !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("Prune error = %v, want ErrInvalidTarget", err)
	}
	if got := rowCount(t, db); got != 100 {
		t.Fatalf("%d rows remain of 100; a misspelled time column deleted rows", got)
	}

	size, _ := pruner.databaseBytes(context.Background())
	if _, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "b", MaxBytes: size / 4}); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("RebuildToBudget error = %v, want ErrInvalidTarget", err)
	}
}

func TestRebuildCopiesBeforeDroppingTheOriginalIndexes(t *testing.T) {
	// The original's index on the time column is what makes the copy a bounded
	// index walk. Dropping the indexes first — the obvious way to free their
	// names — removes it, and ORDER BY silently falls back to scanning every row
	// and sorting externally. On the 455 GiB database that meant reading the
	// whole file and spilling a comparable sorter onto a disk with 208 GiB free.
	//
	// Small fixtures sort fast, so a size assertion cannot catch this. The order
	// itself is the property, so the order itself is what is asserted.
	db, path := newFixtureDB(t, 2000, time.Minute)
	pruner := newFixturePruner(t, db, path, nil)
	size, _ := pruner.databaseBytes(context.Background())

	if _, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "b", MaxBytes: size / 4}); err != nil {
		t.Fatalf("RebuildToBudget: %v", err)
	}

	stages := pruner.Stages()
	copyAt, dropAt := -1, -1
	for i, stage := range stages {
		if stage == "rebuild_copy" && copyAt < 0 {
			copyAt = i
		}
		if stage == "rebuild_drop_original_indexes" && dropAt < 0 {
			dropAt = i
		}
	}
	if copyAt < 0 || dropAt < 0 {
		t.Fatalf("stages = %v, want both a copy and an index drop", stages)
	}
	if dropAt < copyAt {
		t.Fatalf("original indexes dropped at %d, before the copy at %d: %v", dropAt, copyAt, stages)
	}
}

func TestRebuildCopyStatementUsesTheTimeIndex(t *testing.T) {
	// The companion to the ordering test: it pins WHY the order matters, so a
	// future change to the copy statement that makes it unindexable is caught.
	db, _ := newFixtureDB(t, 500, time.Minute)
	copyShape := `SELECT * FROM system_events ORDER BY "occurred_at" DESC LIMIT 100`

	planWithIndex := queryPlan(t, db, copyShape)
	if strings.Contains(planWithIndex, "TEMP B-TREE") {
		t.Fatalf("the copy sorts even with the index present: %s", planWithIndex)
	}
	if !strings.Contains(planWithIndex, "idx_occurred_at") {
		t.Fatalf("the copy does not use the time index: %s", planWithIndex)
	}

	// Removing the index is exactly what dropping it before the copy did.
	if _, err := db.Exec(`DROP INDEX idx_occurred_at`); err != nil {
		t.Fatalf("drop index: %v", err)
	}
	planWithout := queryPlan(t, db, copyShape)
	if !strings.Contains(planWithout, "TEMP B-TREE") {
		t.Fatalf("expected an external sort without the index, got: %s", planWithout)
	}
}

func queryPlan(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN " + query)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(detail)
		plan.WriteString("; ")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}
	return plan.String()
}

func TestRebuildLandsOnTheCeilingWithWidelyVaryingRowSizes(t *testing.T) {
	// The live failure this guards: bytes-per-row was inferred from the rowid
	// range, which on an AUTOINCREMENT table with a long history of deletes
	// overstated the live row count ~8x. The rebuild kept eight times too many
	// rows and landed at 15.8GiB against a 2GiB ceiling.
	//
	// Row size is deliberately uneven here so any arithmetic that assumes a
	// uniform per-row cost derived from the whole file gets it wrong.
	path := filepath.Join(t.TempDir(), "varied.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		occurred_at TEXT NOT NULL,
		payload TEXT NOT NULL)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_at ON events(occurred_at DESC)`); err != nil {
		t.Fatalf("index: %v", err)
	}

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO events (occurred_at, payload) VALUES (?, ?)`)
	for i := range 12000 {
		// Newer rows are much fatter than older ones.
		size := 64
		if i > 8000 {
			size = 4096
		}
		at := fixtureClock.Add(-time.Duration(12000-i) * time.Second)
		if _, err := stmt.Exec(at.Format(time.RFC3339Nano), strings.Repeat("y", size)); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	// Inflate the id sequence the way weeks of dedup-and-delete would, so the
	// rowid range no longer describes the live row count.
	if _, err := db.Exec(`UPDATE sqlite_sequence SET seq = 90000000 WHERE name='events'`); err != nil {
		t.Fatalf("bump sequence: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO events (occurred_at, payload) VALUES (?, 'z')`,
		fixtureClock.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert sentinel: %v", err)
	}

	pruner, err := NewSQLiteTablePruner(SQLiteTableConfig{
		DB: db, Path: path, Table: "events", TimeColumn: "occurred_at",
		AllowFullVacuum: true,
		FreeSpace:       func(string) (int64, error) { return 1 << 40, nil },
	})
	if err != nil {
		t.Fatalf("NewSQLiteTablePruner: %v", err)
	}

	before, _ := pruner.databaseBytes(context.Background())
	ceiling := before / 6
	result, err := pruner.RebuildToBudget(context.Background(), Budget{Name: "events", MaxBytes: ceiling})
	if err != nil {
		t.Fatalf("RebuildToBudget: %v", err)
	}

	after, err := pruner.databaseBytes(context.Background())
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	if after > ceiling {
		t.Fatalf("rebuild landed at %s against a %s ceiling; per-row cost was estimated, not measured",
			FormatBytes(after), FormatBytes(ceiling))
	}
	if result.After.Items == 0 {
		t.Fatal("the rebuild kept no rows at all")
	}
	// And it kept the NEWEST rows, which are the fat ones.
	var newest string
	if err := db.QueryRow(`SELECT MAX(occurred_at) FROM events`).Scan(&newest); err != nil {
		t.Fatalf("max: %v", err)
	}
	if newest != fixtureClock.Format(time.RFC3339Nano) {
		t.Fatalf("newest surviving row is %q, want the sentinel at %q", newest, fixtureClock.Format(time.RFC3339Nano))
	}
}

func TestSeveralByteCeilingsOnOneDatabaseAreIndependent(t *testing.T) {
	// Per-table measurement makes this correct, and it is the shape autoheal
	// actually needs: system_events held 0.86GiB of a 15.1GiB file while
	// health_results held 12.2GiB, so bounding only one of them bounds nothing.
	manifest := `{"retention":{"budgets":{
	  "events":{"target":{"kind":"sqlite_table","database":"a.sqlite","table":"events","time_column":"at"},"max_bytes":"5GiB"},
	  "health":{"target":{"kind":"sqlite_table","database":"a.sqlite","table":"health","time_column":"at"},"max_bytes":"256MiB"}}}}`
	specs, err := ParseManifest([]byte(manifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("got %d specs, want 2", len(specs))
	}
	if specs[0].Budget.MaxBytes != 5<<30 || specs[1].Budget.MaxBytes != 256<<20 {
		t.Fatalf("ceilings = %d and %d, want each budget to keep its own",
			specs[0].Budget.MaxBytes, specs[1].Budget.MaxBytes)
	}
}

func TestMeasureReportsTheTableNotTheWholeFile(t *testing.T) {
	// The live failure: a 2GiB ceiling on system_events was compared against the
	// whole 15.1GiB file, so it could never be satisfied and the engine deleted
	// the table toward empty chasing it.
	db, path := newFixtureDB(t, 500, time.Minute)

	// A second, much larger table in the same database.
	if _, err := db.Exec(`CREATE TABLE bulk (id INTEGER PRIMARY KEY, blob TEXT NOT NULL)`); err != nil {
		t.Fatalf("create bulk: %v", err)
	}
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO bulk (blob) VALUES (?)`)
	for range 20000 {
		if _, err := stmt.Exec(strings.Repeat("q", 1024)); err != nil {
			t.Fatalf("insert bulk: %v", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	pruner := newFixturePruner(t, db, path, nil)
	usage, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	fileBytes, err := pruner.databaseBytes(context.Background())
	if err != nil {
		t.Fatalf("databaseBytes: %v", err)
	}
	if usage.Bytes >= fileBytes/2 {
		t.Fatalf("Measure reported %s of a %s file; the sibling table's pages are being charged to this budget",
			FormatBytes(usage.Bytes), FormatBytes(fileBytes))
	}
	if usage.Items != 500 {
		t.Errorf("Items = %d, want the 500 rows of the budgeted table", usage.Items)
	}
}

func TestByteBoundDoesNotEmptyATableItCannotShrinkTheFileWith(t *testing.T) {
	// With a per-table ceiling the budgeted table stops at its own bound and
	// leaves a large sibling alone, instead of deleting itself to nothing.
	db, path := newFixtureDB(t, 3000, time.Minute)
	if _, err := db.Exec(`CREATE TABLE bulk (id INTEGER PRIMARY KEY, blob TEXT NOT NULL)`); err != nil {
		t.Fatalf("create bulk: %v", err)
	}
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO bulk (blob) VALUES (?)`)
	for range 30000 {
		if _, err := stmt.Exec(strings.Repeat("q", 1024)); err != nil {
			t.Fatalf("insert bulk: %v", err)
		}
	}
	stmt.Close()
	tx.Commit()

	pruner := newFixturePruner(t, db, path, nil)
	tableBefore, err := pruner.tableBytes(context.Background())
	if err != nil {
		t.Fatalf("tableBytes: %v", err)
	}

	// A ceiling well under the file size but reachable for this table.
	if _, err := pruner.Prune(context.Background(), Budget{Name: "b", MaxBytes: tableBefore / 2}); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	remaining := rowCount(t, db)
	if remaining == 0 {
		t.Fatal("the table was emptied; the ceiling was compared against the whole file again")
	}
	if remaining >= 3000 {
		t.Fatalf("%d rows remain of 3000; nothing was pruned", remaining)
	}
	var bulk int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM bulk`).Scan(&bulk); err != nil {
		t.Fatalf("count bulk: %v", err)
	}
	if bulk != 30000 {
		t.Fatalf("the sibling table lost rows (%d of 30000); a budget must not touch tables it does not name", bulk)
	}
}
