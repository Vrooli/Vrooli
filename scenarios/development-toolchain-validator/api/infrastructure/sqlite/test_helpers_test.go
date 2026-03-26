package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// setupTestDB creates a fresh SQLite database in a temp directory for testing.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDBAt(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
