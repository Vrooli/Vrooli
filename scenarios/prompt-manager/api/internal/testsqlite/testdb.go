package testsqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	// Registers the sqlite driver so test binaries that open a routed DB
	// through this helper do not depend on package main's blank import.
	_ "modernc.org/sqlite"
)

func Open(t *testing.T) *database.RoutedDB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prompt-manager.db")
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          buildDSN(t, path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// buildDSN routes test handles through the same seam production uses, so a
// test opens a database the way the scenario does.
func buildDSN(t *testing.T, path string) string {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		t.Fatalf("build sqlite dsn: %v", err)
	}
	return dsn
}
