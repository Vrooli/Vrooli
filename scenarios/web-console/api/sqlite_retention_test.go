package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/retention/sqlitemaintenance"
)

func TestConsoleSQLiteUsesIncrementalVacuumOnNewFiles(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "web-console.db")
	db, err := openConsoleDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := initSchema(ctx, db.Primary()); err != nil {
		t.Fatal(err)
	}
	assertConsoleSQLiteVacuumMode(t, db.Primary(), "incremental")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openConsoleDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	assertConsoleSQLiteVacuumMode(t, reopened.Primary(), "incremental")
	if !reopened.HasReadPool() {
		t.Fatal("expected the query-only read pool to stay enabled")
	}
}

func TestConsoleSQLiteMaintenanceKeepsLegacyRowIDs(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "web-console.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = legacy.Close() })
	// Exercise the owner path above the real reclamation trigger. The FTS
	// index is external-content, just like the console's conversation index.
	triggerBytes := sqlitemaintenance.DefaultPolicy().FreeTriggerBytes
	payloadBytes := triggerBytes + 1<<20
	if _, err := legacy.ExecContext(ctx, `
		CREATE TABLE conversation_events (id TEXT PRIMARY KEY, text TEXT);
		CREATE VIRTUAL TABLE conversation_events_fts USING fts5(text, content='conversation_events', content_rowid='rowid');
		INSERT INTO conversation_events (rowid, id, text) VALUES (123450, 'keep', 'proof');
		INSERT INTO conversation_events_fts(rowid, text) VALUES (123450, 'proof');
		CREATE TABLE disposable (data BLOB);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(ctx, "INSERT INTO disposable VALUES (zeroblob(?))", payloadBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.ExecContext(ctx, "DROP TABLE disposable"); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := openConsoleDatabase(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	before := assertConsoleSQLiteVacuumMode(t, db.Primary(), "none")
	if before.FreeBytes <= triggerBytes {
		t.Fatalf("legacy fixture free bytes = %d, want above maintenance trigger %d", before.FreeBytes, triggerBytes)
	}
	maintain := newConsoleSQLiteMaintenance(db.Primary())
	maintain(ctx)
	assertConsoleSQLiteVacuumMode(t, db.Primary(), "none")

	var rowID int64
	err = db.Reader().QueryRowContext(ctx, `
		SELECT e.rowid
		FROM conversation_events_fts
		JOIN conversation_events e ON e.rowid = conversation_events_fts.rowid
		WHERE conversation_events_fts MATCH 'proof'
	`).Scan(&rowID)
	if err != nil || rowID != 123450 {
		t.Fatalf("legacy FTS row ID = %d, %v; want 123450", rowID, err)
	}
}

func assertConsoleSQLiteVacuumMode(t *testing.T, db *sql.DB, want string) sqlitemaintenance.Stats {
	t.Helper()
	stats, err := sqlitemaintenance.Measure(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if stats.AutoVacuum != want {
		t.Fatalf("auto-vacuum mode = %q, want %q", stats.AutoVacuum, want)
	}
	return stats
}
