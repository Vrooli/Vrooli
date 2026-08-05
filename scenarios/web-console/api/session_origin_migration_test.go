package main

import (
	"context"
	"testing"
)

// TestApplyColumnMigrations_BackfillsOriginToUI proves that adding the origin
// column to a pre-existing sessions table backfills every historical row to
// 'ui' (they were all opened from the web UI) and preserves existing data.
func TestApplyColumnMigrations_BackfillsOriginToUI(t *testing.T) {
	db := openMigrationDB(t)
	// applyColumnMigrations also alters workspace_panes and indexes
	// conversation_events; give it both so those statements do not fail with
	// "no such table" and mask the assertions below.
	if _, err := db.Exec(`CREATE TABLE workspace_panes (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`CREATE TABLE conversation_events (session_id TEXT NOT NULL, sequence INTEGER NOT NULL)`,
	); err != nil {
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

	if err := applyColumnMigrations(context.Background(), db); err != nil {
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
	if err := applyColumnMigrations(context.Background(), db); err != nil {
		t.Fatalf("applyColumnMigrations (second pass): %v", err)
	}
}

// TestInitSchema_DriftCheckRunsAfterMigrations pins the boot ordering.
//
// schema.sql declares every current column, but on a database that already has
// the table `CREATE TABLE IF NOT EXISTS` is a no-op — the new column only
// lands via applyColumnMigrations. Running api-core's declared-column drift
// check before those migrations therefore fails boot on exactly the drift they
// exist to repair, which is what adding workspace_panes.manually_unread hit:
// the API refused to start against a real, pre-existing database.
func TestInitSchema_DriftCheckRunsAfterMigrations(t *testing.T) {
	db := openMigrationDB(t)

	// A workspace_panes table from before manually_unread existed. Its columns
	// mirror the pre-migration shape, so the drift check has something real to
	// object to if it ever runs too early again.
	if _, err := db.Exec(`
		CREATE TABLE workspace_panes (
			session_id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT 'terminal',
			header_color TEXT NOT NULL DEFAULT 'transparent',
			theme_id TEXT NOT NULL DEFAULT 'default',
			font_size INTEGER NOT NULL DEFAULT 14,
			sort_order INTEGER NOT NULL DEFAULT 0,
			group_id TEXT,
			is_active INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO workspace_panes (session_id, name) VALUES ('legacy', 'my-terminal')`,
	); err != nil {
		t.Fatal(err)
	}

	if err := initSchemaWithProviders(context.Background(), db, repoSchemaProviders(t)); err != nil {
		t.Fatalf("initSchema against a pre-existing database: %v", err)
	}

	// The column landed, backfilled false, and the existing row survived.
	var name string
	var manuallyUnread int
	if err := db.QueryRow(
		`SELECT name, manually_unread FROM workspace_panes WHERE session_id = 'legacy'`,
	).Scan(&name, &manuallyUnread); err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if name != "my-terminal" {
		t.Errorf("existing data lost: name = %q, want 'my-terminal'", name)
	}
	if manuallyUnread != 0 {
		t.Errorf("manually_unread backfilled to %d, want 0 — pre-existing panes must read as read", manuallyUnread)
	}
}
