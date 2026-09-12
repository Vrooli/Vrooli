package remediation

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

const oldRemediationJobsSchema = `
CREATE TABLE remediation_jobs (
    id TEXT PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    status TEXT NOT NULL,
    source_json TEXT NOT NULL,
    selected_finding_ids_json TEXT NOT NULL,
    additional_context TEXT NOT NULL DEFAULT '',
    attribution_json TEXT NOT NULL DEFAULT '{}',
    verification_json TEXT NOT NULL DEFAULT '{}',
    failure TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    cancelled_at TEXT
);`

func openOldRemediationDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "old-remediation.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(oldRemediationJobsSchema); err != nil {
		t.Fatalf("create old remediation schema: %v", err)
	}
	return db
}

func TestMigratePreservesExistingJobsAndAddsSelectedRequirements(t *testing.T) {
	ctx := context.Background()
	db := openOldRemediationDB(t)
	// Runtime applies idempotent domain DDL before guarded column migrations.
	// The existing jobs table is retained while the new attempts table appears.
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply current domain schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO remediation_jobs
        (id, scenario_name, status, source_json, selected_finding_ids_json, created_at, updated_at)
        VALUES ('job-before-migration', 'demo', 'created', '{}', '["afid:1"]', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed old job: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}

	var requirements string
	if err := db.QueryRowContext(ctx, `SELECT selected_requirement_ids_json FROM remediation_jobs WHERE id = 'job-before-migration'`).Scan(&requirements); err != nil {
		t.Fatalf("read migrated job: %v", err)
	}
	if requirements != "[]" {
		t.Fatalf("requirements default = %q, want []", requirements)
	}

	repo := NewSQLiteRepository(db)
	stored, err := repo.Get(ctx, "job-before-migration")
	if err != nil {
		t.Fatalf("read migrated job through repository: %v", err)
	}
	if len(stored.SelectedRequirementIDs) != 0 || stored.SelectedFindingIDs[0] != "afid:1" || stored.LaunchAttempt != 0 {
		t.Fatalf("migrated job = %+v", stored)
	}
	// launch_pending is active after the migration, so an upgrade cannot admit
	// a duplicate recovery job for the same scenario.
	if _, err := db.ExecContext(ctx, `INSERT INTO remediation_jobs (id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, created_at, updated_at) VALUES ('pending-a', 'pending-demo', 'launch_pending', '{}', 'a', '[]', '[]', 'b', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed pending job: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO remediation_jobs (id, scenario_name, status, source_json, source_hash, selected_finding_ids_json, selected_requirement_ids_json, selection_hash, created_at, updated_at) VALUES ('pending-b', 'pending-demo', 'launch_pending', '{}', 'a', '[]', '[]', 'b', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err == nil {
		t.Fatal("launch_pending duplicate should be rejected by rebuilt active-job index")
	}
}
