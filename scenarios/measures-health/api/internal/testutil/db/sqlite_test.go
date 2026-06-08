package db

import (
	"database/sql"
	"strings"
	"testing"
)

func TestNewSQLite_HandleUsable(t *testing.T) {
	d := NewSQLite(t)
	var got int
	if err := d.QueryRow("SELECT 1").Scan(&got); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if got != 1 {
		t.Fatalf("SELECT 1 = %d, want 1", got)
	}
}

// TestNewSQLite_PragmasApplied is the contract gate for the DSN pragma
// string. If a future caller drops a pragma — e.g. busy_timeout
// regressing to 0 — handler tests that previously caught lock
// contention silently pass instead.
func TestNewSQLite_PragmasApplied(t *testing.T) {
	d := NewSQLite(t)

	var journal string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if !strings.EqualFold(journal, "wal") {
		t.Errorf("journal_mode = %q, want wal", journal)
	}

	var fk int
	if err := d.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}

	var busy int
	if err := d.QueryRow("PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busy <= 0 {
		t.Errorf("busy_timeout = %d, want > 0", busy)
	}
}

// TestNewSQLite_FreshPerCall proves each call returns an independent
// database backed by its own file. Without this, tests that share a t
// would leak state between sub-cases via the same backing file.
func TestNewSQLite_FreshPerCall(t *testing.T) {
	a := NewSQLite(t)
	if _, err := a.Exec(`CREATE TABLE marker (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("create marker: %v", err)
	}

	b := NewSQLite(t)
	var name string
	row := b.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='marker'`)
	err := row.Scan(&name)
	if err == nil {
		t.Fatalf("expected no marker table in fresh handle, found %q", name)
	}
}

// TestNewSQLite_ClosedOnCleanup proves the t.Cleanup wiring closes the
// handle so tests don't leak open file descriptors.
func TestNewSQLite_ClosedOnCleanup(t *testing.T) {
	var captured *sql.DB
	t.Run("inner", func(tt *testing.T) {
		captured = NewSQLite(tt)
	})
	// After the subtest finishes, t.Cleanup has run and Close has been
	// called on the handle. A subsequent operation must error.
	if _, err := captured.Exec("SELECT 1"); err == nil {
		t.Fatal("handle still usable after cleanup; close not wired")
	}
}
