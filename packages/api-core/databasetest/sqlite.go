// Package databasetest provides test helpers for SQLite-backed repository tests.
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
//	        apidb.SchemaProviderFunc(notes.Schema),
//	    ))
//	    return d
//	}
//
// See `internal/notes/sqlite_test.go` for the worked example.
// The helper is intentionally inline at the consumer rather than
// exported from this package — `db` lives under `testutil` and the
// domain package is the consumer, so an exported helper would invert
// the dependency. Reach for it as a per-package convention; do not
// generalise across packages.
package databasetest

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"

	// Register the pure-Go SQLite driver used by test handles.
	_ "modernc.org/sqlite"
)

// NewSQLite returns a fresh SQLite handle backed by a file in the test's temp
// dir. The handle is closed automatically via t.Cleanup. The connection pool is
// capped at 1 to mirror production SQLite's single-writer constraint.
//
// The DSN comes from api-core/storage, the same seam production uses, so a test
// handle and a production handle open a file the same way. That property is the
// entire justification for a file-backed test handle over ":memory:", and it
// used to be maintained by hand: this helper carried its own pragma string, and
// dozens of scenario copies bore a comment instructing the reader to "tweak in
// lockstep with" it. They did not stay in lockstep. This helper had drifted to
// foreign_keys(1) with no synchronous or temp_store setting, so tests ran under
// different durability behaviour than the code they were validating.
//
// _txlock=immediate is kept deliberately: it makes BeginTx take the reserved
// lock up front, which surfaces write-ordering bugs in a test rather than
// leaving them to appear under production contention.
func NewSQLite(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	if err != nil {
		t.Fatalf("build sqlite dsn: %v", err)
	}

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = d.Close() })
	return d
}
