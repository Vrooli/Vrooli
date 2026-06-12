// Package testdb provides a canonical temp-SQLite database for tests. It opens
// a file-backed database with the same pragmas as production (pkg/storagepaths)
// and applies the supplied domain schemas through database.EnsureSchemas, so
// tests exercise the real DDL and driver behaviour rather than an ad-hoc shape.
//
// Callers pass the schema text they need (e.g. autosteer.Schema()) to avoid an
// import cycle with the central pkg/dbschema registry.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	// Register the sqlite driver used by database.Connect in tests.
	_ "modernc.org/sqlite"
)

// NewSQLite opens a fresh temp SQLite database, applies the given schemas, and
// registers cleanup. The pool is capped at a single connection so writers
// serialize (matching the production single-writer model) and concurrent tests
// never hit "database is locked".
func NewSQLite(t *testing.T, schemas ...string) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	)

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	providers := make([]database.SchemaProvider, 0, len(schemas))
	for _, s := range schemas {
		providers = append(providers, database.SchemaProviderFunc(func() string { return s }))
	}
	if err := database.EnsureSchemas(context.Background(), db, providers...); err != nil {
		t.Fatalf("apply schemas: %v", err)
	}
	return db
}
