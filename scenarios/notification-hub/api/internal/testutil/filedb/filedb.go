// Package filedb opens file-backed SQLite databases for tests that need the real
// routed pool (leases, several connections) rather than databasetest's single
// connection.
//
// The DSN comes from the production seam, storage.SQLiteDSNAt. A bare path as a
// DSN carries no busy timeout and no WAL, so a worker and the test's own reads
// fail with SQLITE_BUSY as soon as they overlap. The package imports nothing from
// the hub so that the hub's own tests can use it too.
package filedb

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// DSN is the DSN of a fresh database file called name in the test's temp dir.
func DSN(t testing.TB, name string) string {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), name), storage.SQLiteTuning{})
	require.NoError(t, err)
	return dsn
}

// Open opens a routed SQLite database on a DSN.
func Open(t testing.TB, name string) *database.RoutedDB {
	t.Helper()
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: DSN(t, name)})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	return db
}
