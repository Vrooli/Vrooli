package scoring

import (
	"context"
	"testing"
	"time"

	testdb "scenario-completeness-scoring/internal/testutil/db"
)

// TestUpsertSnapshotPersistsRecency verifies the scenario-level recency columns
// round-trip through the snapshot row.
func TestUpsertSnapshotPersistsRecency(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	created := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	lastRun := time.Date(2026, 6, 10, 9, 30, 0, 0, time.UTC)

	if _, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario:   "cli-health",
		Digest:     "td:one",
		Composite:  72,
		CreatedAt:  created,
		LastRunAt:  lastRun,
		LastStatus: "passed",
	}); err != nil {
		t.Fatalf("UpsertSnapshot() error = %v", err)
	}

	got, ok, err := repo.LatestFor(ctx, "cli-health")
	if err != nil || !ok {
		t.Fatalf("LatestFor() ok=%v err=%v", ok, err)
	}
	if !got.LastRunAt.Equal(lastRun) {
		t.Fatalf("LastRunAt = %v, want %v", got.LastRunAt, lastRun)
	}
	if got.LastStatus != "passed" {
		t.Fatalf("LastStatus = %q, want passed", got.LastStatus)
	}
}

// TestUpsertSnapshotAdvancesRecencyOnUnchangedDigest is the keystone behavior:
// a new test run can complete without the tree (digest) changing, so the
// digest-deduplicated score row is not re-inserted — but its recency must still
// advance, and never regress to an older run.
func TestUpsertSnapshotAdvancesRecencyOnUnchangedDigest(t *testing.T) {
	repo := newTestSnapshotRepo(t)
	ctx := context.Background()
	created := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	first := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC)
	older := time.Date(2026, 6, 9, 9, 0, 0, 0, time.UTC)

	inserted, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario: "cli-health", Digest: "td:same", Composite: 72,
		CreatedAt: created, LastRunAt: first, LastStatus: "passed",
	})
	if err != nil || !inserted {
		t.Fatalf("first upsert inserted=%v err=%v", inserted, err)
	}

	// Same digest, newer run -> not a new row, but recency advances.
	inserted, err = repo.UpsertSnapshot(ctx, Snapshot{
		Scenario: "cli-health", Digest: "td:same", Composite: 72,
		CreatedAt: created.Add(time.Hour), LastRunAt: newer, LastStatus: "failed",
	})
	if err != nil {
		t.Fatalf("second upsert error = %v", err)
	}
	if inserted {
		t.Fatalf("second upsert inserted = true, want false (digest unchanged)")
	}
	got, _, _ := repo.LatestFor(ctx, "cli-health")
	if !got.LastRunAt.Equal(newer) || got.LastStatus != "failed" {
		t.Fatalf("after newer run: LastRunAt=%v LastStatus=%q, want %v/failed", got.LastRunAt, got.LastStatus, newer)
	}
	// Composite is unchanged (digest dedup preserved the original score row).
	if got.Composite != 72 {
		t.Fatalf("Composite = %d, want 72 (score row preserved)", got.Composite)
	}

	// Same digest, OLDER run -> recency must not regress.
	if _, err := repo.UpsertSnapshot(ctx, Snapshot{
		Scenario: "cli-health", Digest: "td:same", Composite: 72,
		CreatedAt: created.Add(2 * time.Hour), LastRunAt: older, LastStatus: "passed",
	}); err != nil {
		t.Fatalf("third upsert error = %v", err)
	}
	got, _, _ = repo.LatestFor(ctx, "cli-health")
	if !got.LastRunAt.Equal(newer) || got.LastStatus != "failed" {
		t.Fatalf("after older run: recency regressed to %v/%q, want %v/failed", got.LastRunAt, got.LastStatus, newer)
	}
}

// TestMigrateAddsRecencyColumns proves the brownfield path: a score_snapshots
// table created before the recency columns existed is ALTERed in place (data
// preserved), and the migration is idempotent.
func TestMigrateAddsRecencyColumns(t *testing.T) {
	ctx := context.Background()
	d := testdb.NewSQLite(t)

	// Old-shape table (no last_run_at/last_status), as a pre-recency database
	// would have it.
	if _, err := d.ExecContext(ctx, `CREATE TABLE score_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scenario TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'utility',
		digest TEXT NOT NULL,
		composite INTEGER NOT NULL,
		classification TEXT NOT NULL,
		working_rung TEXT NOT NULL DEFAULT '',
		breakdown_json TEXT NOT NULL DEFAULT '{}',
		importance REAL,
		source TEXT NOT NULL DEFAULT 'sweeper',
		created_at TEXT NOT NULL,
		UNIQUE (scenario, digest)
	)`); err != nil {
		t.Fatalf("create old table: %v", err)
	}
	if _, err := d.ExecContext(ctx, `INSERT INTO score_snapshots
		(scenario, category, digest, composite, classification, created_at)
		VALUES ('legacy', 'utility', 'td:legacy', 50, 'partial', '2026-06-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	for _, col := range recencyColumns {
		if has, _ := columnExists(ctx, d, "score_snapshots", col); has {
			t.Fatalf("column %s present before migration", col)
		}
	}

	if err := Migrate(ctx, d); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	for _, col := range recencyColumns {
		if has, _ := columnExists(ctx, d, "score_snapshots", col); !has {
			t.Fatalf("column %s missing after migration", col)
		}
	}

	// Pre-existing data preserved and readable through the repository.
	repo := NewSQLiteSnapshotRepository(d)
	got, ok, err := repo.LatestFor(ctx, "legacy")
	if err != nil || !ok {
		t.Fatalf("LatestFor(legacy) ok=%v err=%v", ok, err)
	}
	if got.Composite != 50 || got.LastStatus != "" || !got.LastRunAt.IsZero() {
		t.Fatalf("legacy row = %+v, want composite=50 empty recency", got)
	}

	// Idempotent: a second run is a clean no-op.
	if err := Migrate(ctx, d); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
}

// TestMigrateNoTableIsNoop proves a fresh database (no table yet) is handled
// gracefully — EnsureSchemas creates the table complete afterward.
func TestMigrateNoTableIsNoop(t *testing.T) {
	ctx := context.Background()
	d := testdb.NewSQLite(t)
	if err := Migrate(ctx, d); err != nil {
		t.Fatalf("Migrate() on empty db error = %v", err)
	}
}
