package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"architecture-cartographer/internal/clock"

	"github.com/vrooli/api-core/retention"
	_ "modernc.org/sqlite"
)

// newRetentionDB opens a real SQLite file with the production pragmas.
//
// These tests use a real database rather than a fake on purpose. Every hard
// part of retention — the window function's cost, WAL growth during a large
// delete, and whether freed pages actually return to the filesystem — is a
// property of SQLite, not of the Go code. A fake would assert the parts that
// were never in doubt.
func newRetentionDB(t *testing.T) (*sql.DB, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "cartographer.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// A single connection keeps pragma state (notably auto_vacuum) stable
	// across statements, matching how the scenario opens its database.
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), snapshotSchemaForTest); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db, path
}

const snapshotSchemaForTest = `
CREATE TABLE IF NOT EXISTS graph_snapshots (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  source_fingerprint TEXT NOT NULL DEFAULT '',
  payload BLOB NOT NULL,
  payload_codec TEXT NOT NULL DEFAULT '',
  extracted_at TEXT NOT NULL,
  extraction_ms INTEGER NOT NULL DEFAULT 0,
  UNIQUE(scenario, content_hash)
);
CREATE TABLE IF NOT EXISTS unrelated_table (
  id INTEGER PRIMARY KEY,
  value TEXT NOT NULL
);
`

// insertSnapshot writes one row directly, so tests control extracted_at.
func insertSnapshot(t *testing.T, db *sql.DB, scenario, hash string, at time.Time, payloadSize int) {
	t.Helper()
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte('a' + i%26)
	}
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO graph_snapshots (id, scenario, content_hash, source_fingerprint, payload, extracted_at, extraction_ms) VALUES (?,?,?,?,?,?,?)`,
		scenario+"-"+hash, scenario, hash, "", payload, at.Format(snapshotTimeFormat), 1)
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
}

func snapshotCount(t *testing.T, db *sql.DB, scenario string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM graph_snapshots WHERE scenario = ?`, scenario).Scan(&count); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	return count
}

// TestPruneSnapshots_BoundsRepeatedExtraction is the test the plan names as the
// one that would have caught the original defect.
//
// The unique index is on (scenario, content_hash), so every distinct code state
// creates a new row and none are ever removed. This loop reproduces exactly
// that: extract changing content over and over, and assert the table stops
// growing. Before retention existed, this same activity reached 2,469 rows and
// 77.2 GB and filled the host disk.
func TestPruneSnapshots_BoundsRepeatedExtraction(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()
	policy := RetentionPolicy{KeepPerScenario: 3}

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 50; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("hash-%03d", i), base.Add(time.Duration(i)*time.Hour), 512)

		if _, err := repo.PruneSnapshots(ctx, policy); err != nil {
			t.Fatalf("prune after insert %d: %v", i, err)
		}
		if got := snapshotCount(t, db, "demo"); got > 3 {
			t.Fatalf("after %d extractions the table holds %d rows, want at most 3 — growth is unbounded again", i+1, got)
		}
	}

	if got := snapshotCount(t, db, "demo"); got != 3 {
		t.Errorf("final row count = %d, want exactly the configured 3", got)
	}
}

// TestPruneSnapshots_KeepsNewestRemovesOldest asserts retention keeps the rows
// a reader would consider current.
func TestPruneSnapshots_KeepsNewestRemovesOldest(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 6; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("hash-%d", i), base.Add(time.Duration(i)*time.Hour), 128)
	}

	if _, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: 2}); err != nil {
		t.Fatalf("prune: %v", err)
	}

	rows, err := db.QueryContext(ctx, `SELECT content_hash FROM graph_snapshots ORDER BY extracted_at DESC`)
	if err != nil {
		t.Fatalf("query survivors: %v", err)
	}
	defer rows.Close()

	var survivors []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			t.Fatalf("scan: %v", err)
		}
		survivors = append(survivors, hash)
	}

	want := []string{"hash-5", "hash-4"}
	if len(survivors) != len(want) {
		t.Fatalf("survivors = %v, want %v", survivors, want)
	}
	for i := range want {
		if survivors[i] != want[i] {
			t.Errorf("survivor %d = %q, want %q", i, survivors[i], want[i])
		}
	}
}

// TestPruneSnapshots_IsPerScenario asserts one busy scenario cannot evict
// another's history.
func TestPruneSnapshots_IsPerScenario(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		insertSnapshot(t, db, "busy", fmt.Sprintf("h-%d", i), base.Add(time.Duration(i)*time.Hour), 128)
	}
	insertSnapshot(t, db, "quiet", "only", base, 128)

	result, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: 3})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}

	if got := snapshotCount(t, db, "busy"); got != 3 {
		t.Errorf("busy scenario kept %d rows, want 3", got)
	}
	if got := snapshotCount(t, db, "quiet"); got != 1 {
		t.Errorf("quiet scenario kept %d rows, want its single row untouched", got)
	}
	if result.ScenariosScanned != 2 {
		t.Errorf("ScenariosScanned = %d, want 2", result.ScenariosScanned)
	}
	if result.RowsRemoved != 7 {
		t.Errorf("RowsRemoved = %d, want 7", result.RowsRemoved)
	}
}

// TestPruneSnapshots_LeavesOtherTablesAlone asserts retention is scoped to
// graph_snapshots.
//
// The rejected alternative during design was a cleanup provider that truncates
// the database file. That would destroy the other twelve tables, including the
// 752,620 analytics rows the incident cleanup verified intact.
func TestPruneSnapshots_LeavesOtherTablesAlone(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()

	for i := 0; i < 25; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO unrelated_table (value) VALUES (?)`, fmt.Sprintf("row-%d", i)); err != nil {
			t.Fatalf("seed unrelated table: %v", err)
		}
	}
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%d", i), base.Add(time.Duration(i)*time.Hour), 256)
	}

	if _, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: 1}); err != nil {
		t.Fatalf("prune: %v", err)
	}

	var unrelated int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM unrelated_table`).Scan(&unrelated); err != nil {
		t.Fatalf("count unrelated: %v", err)
	}
	if unrelated != 25 {
		t.Errorf("unrelated_table holds %d rows, want all 25 — retention touched a table it does not own", unrelated)
	}
}

// TestPruneSnapshots_ZeroKeepFallsBackToDefault asserts a misconfigured zero
// cannot be used to empty the table.
func TestPruneSnapshots_ZeroKeepFallsBackToDefault(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%d", i), base.Add(time.Duration(i)*time.Hour), 128)
	}

	for _, keep := range []int{0, -1, -100} {
		if _, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: keep}); err != nil {
			t.Fatalf("prune with keep=%d: %v", keep, err)
		}
		if got := snapshotCount(t, db, "demo"); got != DefaultSnapshotRetentionKeep {
			t.Fatalf("keep=%d left %d rows, want the default %d — a bad config must not empty the table",
				keep, got, DefaultSnapshotRetentionKeep)
		}
	}
}

// TestPruneSnapshots_BatchesLargeDeletes asserts a prune far larger than one
// batch still removes everything beyond the floor.
func TestPruneSnapshots_BatchesLargeDeletes(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	const total = 250
	for i := 0; i < total; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%04d", i), base.Add(time.Duration(i)*time.Minute), 64)
	}

	result, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: 3, BatchSize: 10})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if result.RowsRemoved != total-3 {
		t.Errorf("RowsRemoved = %d, want %d", result.RowsRemoved, total-3)
	}
	if got := snapshotCount(t, db, "demo"); got != 3 {
		t.Errorf("final count = %d, want 3", got)
	}
}

// TestPruneSnapshots_ReturnsPagesToFilesystem asserts retention plus
// incremental vacuum actually shrinks the file.
//
// This is the property the incident proved was missing. After the manual prune
// the live payload was 3.26 GB while the file still measured 73 GB, because
// deleted pages go on the freelist rather than back to the filesystem.
func TestPruneSnapshots_ReturnsPagesToFilesystem(t *testing.T) {
	db, path := newRetentionDB(t)
	ctx := context.Background()

	if err := retention.EnsureIncrementalAutoVacuum(ctx, db, nil); err != nil {
		t.Fatalf("enable incremental auto-vacuum: %v", err)
	}

	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)

	// Write enough payload that the file growth is unambiguous.
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	const rows = 60
	for i := 0; i < rows; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%03d", i), base.Add(time.Duration(i)*time.Hour), 128*1024)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	before := fileSize(t, path)

	result, err := repo.PruneSnapshots(ctx, RetentionPolicy{KeepPerScenario: 2})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if result.RowsRemoved != rows-2 {
		t.Fatalf("RowsRemoved = %d, want %d", result.RowsRemoved, rows-2)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatalf("checkpoint after prune: %v", err)
	}

	after := fileSize(t, path)
	if after >= before {
		t.Errorf("database file did not shrink: %d bytes before, %d after — retention pruned rows but returned no pages", before, after)
	}
	if result.PagesFreed <= 0 {
		t.Errorf("PagesFreed = %d, want a positive count", result.PagesFreed)
	}
}

// TestReclaimableSnapshotBytes_ReportsWithoutDeleting asserts the estimate is
// non-destructive and reports zero at the floor.
func TestReclaimableSnapshotBytes_ReportsWithoutDeleting(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	ctx := context.Background()
	policy := RetentionPolicy{KeepPerScenario: 3}

	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%d", i), base.Add(time.Duration(i)*time.Hour), 1024)
	}

	bytes, rows, err := repo.ReclaimableSnapshotBytes(ctx, policy)
	if err != nil {
		t.Fatalf("ReclaimableSnapshotBytes: %v", err)
	}
	if rows != 4 {
		t.Errorf("reclaimable rows = %d, want 4", rows)
	}
	if bytes != 4*1024 {
		t.Errorf("reclaimable bytes = %d, want %d", bytes, 4*1024)
	}
	// Estimating must not delete.
	if got := snapshotCount(t, db, "demo"); got != 7 {
		t.Fatalf("estimating removed rows: %d remain, want 7", got)
	}

	if _, err := repo.PruneSnapshots(ctx, policy); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// At the floor there is nothing left to reclaim.
	bytes, rows, err = repo.ReclaimableSnapshotBytes(ctx, policy)
	if err != nil {
		t.Fatalf("ReclaimableSnapshotBytes at floor: %v", err)
	}
	if bytes != 0 || rows != 0 {
		t.Errorf("at the retention floor reclaimable = %d bytes / %d rows, want 0/0", bytes, rows)
	}
}

func fileSize(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Size()
}
