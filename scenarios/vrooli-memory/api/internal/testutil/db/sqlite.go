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
// NewSQLite returns a *blank* handle. The canonical compose pattern
// for repository tests pairs it with the production schema entry
// point (`apidb.EnsureSchemas` from `api-core/database`) over the
// system + per-domain providers, so tests exercise the same shape
// `main.go` ships:
//
//	func newSchemaDB(t *testing.T) *sql.DB {
//	    d := db.NewSQLite(t)
//	    require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
//	        apidb.SchemaProviderFunc(localdb.SystemSchema),
//	        apidb.SchemaProviderFunc(journal.Schema),
//	    ))
//	    return d
//	}
//
// The helper is intentionally inline at the consumer rather than
// exported from this package — `db` lives under `testutil` and the
// domain package is the consumer, so an exported helper would invert
// the dependency. Reach for it as a per-package convention; do not
// generalise across packages.
package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	// Register the pure-Go SQLite driver used by test handles.
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
