// Package dbtest opens throwaway SQLite databases for repository tests.
//
// Each call gets its own file under t.TempDir(). A file is used rather than
// ":memory:" because modernc.org/sqlite gives every pooled connection its own
// private in-memory database, so a pool of more than one connection would see
// inconsistent state.
package dbtest

import (
	"context"
	"path/filepath"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	_ "modernc.org/sqlite" // registers the "sqlite" driver for tests
)

// New returns a routed SQLite database with every provider's schema applied.
// The database is closed when the test finishes.
func New(t *testing.T, providers ...apidb.SchemaProvider) *apidb.RoutedDB {
	t.Helper()

	db, err := apidb.Open(context.Background(), apidb.Config{
		Driver:       apidb.DriverSQLite,
		DSN:          filepath.Join(t.TempDir(), "test.db"),
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := apidb.EnsureSchemas(context.Background(), db.Primary(), providers...); err != nil {
		t.Fatalf("apply schemas: %v", err)
	}
	return db
}
