// Package db contains shared SQLite helpers for API tests.
package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retry"
	_ "modernc.org/sqlite" // Register the pure-Go SQLite driver used by api-core/database in tests.
)

// OpenSQLiteFile opens a SQLite database file under t.TempDir and closes it
// automatically during test cleanup.
func OpenSQLiteFile(t *testing.T, filename string) *sql.DB {
	t.Helper()

	return openSQLite(t, "file:"+filepath.Join(t.TempDir(), filename))
}

// OpenSQLiteMemory opens a shared in-memory SQLite database and closes it
// automatically during test cleanup.
func OpenSQLiteMemory(t *testing.T) *sql.DB {
	t.Helper()

	return openSQLite(t, "file::memory:?cache=shared")
}

func openSQLite(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	handle, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Retry: &retry.Config{
			MaxAttempts:    1,
			BaseDelay:      time.Millisecond,
			MaxDelay:       time.Millisecond,
			JitterFraction: 0,
		},
	})
	if err != nil {
		t.Fatalf("open sqlite %s: %v", dsn, err)
	}
	t.Cleanup(func() {
		_ = handle.Close()
	})
	return handle
}
