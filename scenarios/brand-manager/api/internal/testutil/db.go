// Package testutil provides shared test infrastructure for brand-manager.
//
// It centralises database setup, test helpers, and assertions so that
// test files across packages use a single, consistent foundation.
package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	"brand-manager/config"
	"brand-manager/database"
)

// SetupTestDB creates a temporary SQLite database for testing.
// The database is automatically cleaned up when the test finishes.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := config.Default()
	cfg.SQLitePath = dbPath

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("setup test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
