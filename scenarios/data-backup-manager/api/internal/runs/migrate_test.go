package runs

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureColumnsWidensIncidentKeyWithTargetIdentity(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE runs (
			id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, trigger TEXT NOT NULL,
			status TEXT NOT NULL, started_at TEXT NOT NULL
		)`,
		`CREATE TABLE run_incidents (
			run_id TEXT NOT NULL, code TEXT NOT NULL, category TEXT NOT NULL,
			scope TEXT NOT NULL, message TEXT NOT NULL DEFAULT '',
			next_action TEXT NOT NULL DEFAULT '', destination_id TEXT NOT NULL DEFAULT '',
			target_ids TEXT NOT NULL DEFAULT '[]', first_observed TEXT NOT NULL DEFAULT '',
			last_observed TEXT NOT NULL DEFAULT '', last_known_good TEXT NOT NULL DEFAULT '',
			retryable INTEGER NOT NULL DEFAULT 0, retry_after_seconds INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (run_id, code, scope, destination_id)
		)`,
		`CREATE TABLE run_outcomes (
			run_id TEXT NOT NULL, target_id TEXT NOT NULL, destination_id TEXT NOT NULL,
			status TEXT NOT NULL, snapshot_id TEXT NOT NULL DEFAULT '', bytes INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '', started_at TEXT NOT NULL, finished_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (run_id, target_id, destination_id)
		)`,
		`INSERT INTO runs (id, plan_id, trigger, status, started_at) VALUES ('r1', 'p1', 'manual', 'failed', '2026-11-19T00:00:00Z')`,
		`INSERT INTO run_incidents (run_id, code, category, scope, destination_id, target_ids, message)
			VALUES ('r1', 'source_missing', 'source', 'target', '', '["old"]', 'old target')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := EnsureColumns(ctx, db); err != nil {
		t.Fatalf("EnsureColumns: %v", err)
	}
	key, err := primaryKeyColumns(ctx, db, "run_incidents")
	if err != nil {
		t.Fatal(err)
	}
	if !key["target_ids"] {
		t.Fatalf("run_incidents key = %v, want target_ids included", key)
	}

	for _, target := range []string{"one", "two"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_incidents
			(run_id, code, category, scope, destination_id, target_ids, message)
			VALUES ('r1', 'source_missing', 'source', 'target', '', ?, ?)`, `["`+target+`"]`, target+" unreadable"); err != nil {
			t.Fatalf("two unreadable targets must both persist (%s): %v", target, err)
		}
	}
	var copied int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_incidents WHERE target_ids = '["old"]'`).Scan(&copied); err != nil {
		t.Fatal(err)
	}
	if copied != 1 {
		t.Fatalf("old incident row was not preserved: count = %d", copied)
	}
}
