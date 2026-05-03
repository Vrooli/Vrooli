// Package db provides test helpers for SQLite-backed repository tests.
// NewSQLite returns a connected handle backed by a per-test temp-dir
// file using modernc.org/sqlite (pure-Go, CGO-clean, cross-platform).
//
// # Why a real file, not :memory:
//
// File-backed SQLite shares the same file-locking semantics as the
// production driver path. Bugs around WAL mode, busy timeouts, and
// foreign-key enforcement surface here that would not in :memory:.
//
// # Schema
//
// The template ships zero tables, so this helper currently returns a
// blank database. Scenarios that add tables wrap NewSQLite with a
// domain-specific helper that applies their schema, e.g.:
//
//	func NewTaskDB(t *testing.T) *sql.DB {
//	    db := db.NewSQLite(t)
//	    if err := repository.EnsureSchema(context.Background(), db); err != nil {
//	        t.Fatalf("ensure schema: %v", err)
//	    }
//	    return db
//	}
//
// Apply the same schema entry point main.go uses so test handles are
// byte-identical to a fresh production install.
package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// NewSQLite returns a fresh, fully-initialized SQLite handle backed by
// a file in the test's temp dir. The handle is closed automatically
// via t.Cleanup. Connection pool is capped at 1 to mirror production
// SQLite's single-writer constraint.
func NewSQLite(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_txlock=immediate"

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })
	return d
}
