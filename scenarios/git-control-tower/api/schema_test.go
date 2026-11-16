package main

import (
	"context"
	"testing"

	testdb "github.com/vrooli/api-core/databasetest"
)

func TestCommitCheckIdentityMigrationAddsColumnsToExistingStore(t *testing.T) {
	db := testdb.OpenSQLiteMemory(t)
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE git_commit_check_runs (
		id INTEGER PRIMARY KEY, repo_path TEXT NOT NULL, commit_hash TEXT NOT NULL,
		kind TEXT NOT NULL, status TEXT NOT NULL, command TEXT NOT NULL DEFAULT '',
		exit_code INTEGER NOT NULL DEFAULT 0, summary TEXT NOT NULL DEFAULT '',
		stdout TEXT, stderr TEXT, duration_ms INTEGER NOT NULL DEFAULT 0,
		timestamp TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("create legacy store: %v", err)
	}
	if err := ensureCommitCheckIdentityColumns(context.Background(), db); err != nil {
		t.Fatalf("migrate commit check identity: %v", err)
	}
	cols, err := tableColumns(context.Background(), db, "git_commit_check_runs")
	if err != nil {
		t.Fatalf("inspect migrated store: %v", err)
	}
	for _, name := range []string{"check_job_id", "candidate_digest", "config_digest"} {
		if !cols[name] {
			t.Errorf("migration omitted %q", name)
		}
	}
}

func TestPrecommitDeclaredInputsMigrationAddsEmptyListToLegacyStore(t *testing.T) {
	db := testdb.OpenSQLiteMemory(t)
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE git_repo_precommit (
		repo_path TEXT PRIMARY KEY, enabled INTEGER NOT NULL DEFAULT 0, command TEXT NOT NULL DEFAULT '',
		working_directory TEXT NOT NULL DEFAULT '', timeout_seconds INTEGER NOT NULL DEFAULT 300,
		run_before_commit INTEGER NOT NULL DEFAULT 1, allow_override INTEGER NOT NULL DEFAULT 1,
		last_status TEXT, last_exit_code INTEGER, last_summary TEXT, last_stdout TEXT, last_stderr TEXT,
		last_duration_ms INTEGER, last_timestamp TEXT, hook_install_status TEXT, hook_install_reason TEXT,
		hook_existing_kind TEXT, hook_installed_at TEXT, updated_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("create legacy precommit store: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO git_repo_precommit (repo_path) VALUES ('/legacy/repo')`); err != nil {
		t.Fatalf("seed legacy config: %v", err)
	}
	if err := ensurePrecommitHookColumns(context.Background(), db); err != nil {
		t.Fatalf("migrate precommit config: %v", err)
	}
	cols, err := tableColumns(context.Background(), db, "git_repo_precommit")
	if err != nil {
		t.Fatalf("inspect migrated config: %v", err)
	}
	if !cols["declared_inputs_json"] {
		t.Fatal("migration omitted declared input column")
	}
	svc := NewPrecommitService(db)
	loaded, err := svc.Get(context.Background(), "/legacy/repo")
	if err != nil || len(loaded.DeclaredInputs) != 0 {
		t.Fatalf("legacy config did not default to no declared inputs: %#v err=%v", loaded, err)
	}
}
