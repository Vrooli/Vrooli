package main

import (
	"testing"
)

// TestApplyColumnMigrations_BackfillsOriginToUI proves that adding the origin
// column to a pre-existing sessions table backfills every historical row to
// 'ui' (they were all opened from the web UI) and preserves existing data.
func TestApplyColumnMigrations_BackfillsOriginToUI(t *testing.T) {
	db := openMigrationDB(t)
	// applyColumnMigrations also alters workspace_panes; give it a table so the
	// ALTER does not fail with "no such table".
	if _, err := db.Exec(`CREATE TABLE workspace_panes (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	// oldSessionsTable (from agent_type_migration_test.go) predates the origin
	// column.
	if _, err := db.Exec(oldSessionsTable); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, agent_type, agent_session_id) VALUES ('legacy', 'codex', 'cid')`); err != nil {
		t.Fatal(err)
	}

	if err := applyColumnMigrations(db); err != nil {
		t.Fatalf("applyColumnMigrations: %v", err)
	}

	var origin, owner, label, agentSessionID string
	err := db.QueryRow(
		`SELECT origin, owner, display_label, agent_session_id FROM sessions WHERE id = 'legacy'`,
	).Scan(&origin, &owner, &label, &agentSessionID)
	if err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if origin != "ui" {
		t.Errorf("pre-existing row origin = %q, want 'ui'", origin)
	}
	if owner != "" || label != "" {
		t.Errorf("owner/label = %q/%q, want empty", owner, label)
	}
	if agentSessionID != "cid" {
		t.Errorf("existing data lost: agent_session_id = %q, want 'cid'", agentSessionID)
	}

	// Idempotent: a second pass must not error on the now-existing columns.
	if err := applyColumnMigrations(db); err != nil {
		t.Fatalf("applyColumnMigrations (second pass): %v", err)
	}
}
