// Package testutil provides shared test utilities for the lifestyle dashboard API.
// This centralizes common test infrastructure to avoid duplication and ensure
// consistent test setup patterns across all packages.
//
// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md#Testing-Seams
package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/retry"
	"github.com/vrooli/api-core/storage"

	"lifestyle-dashboard/domain"
)

// SetupTestDB creates a file-based SQLite database with schema for testing.
// It returns the database connection and a cleanup function.
//
// Usage:
//
//	db, cleanup := testutil.SetupTestDB(t)
//	defer cleanup()
//
// The database is configured with:
//   - MaxOpenConns=1 (SQLite single-writer constraint)
//   - MaxIdleConns=1 (minimize resource usage)
//   - WAL mode and busy timeout for concurrent read support
//   - Uses api-core/database.Connect for consistency with production
func SetupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "lifestyle-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	// The path is passed as an argument rather than exported into the process
	// environment, so parallel tests cannot overwrite each other's database and
	// no child process can inherit one.
	dsn, err := storage.SQLiteDSNAt(dbPath, storage.SQLiteTuning{})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to build test database DSN: %v", err)
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Retry: &retry.Config{
			MaxAttempts: 1, // Single attempt for tests (fail fast)
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := domain.InitSchema(db); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

// SetupInMemoryDB creates an in-memory SQLite database for quick unit tests.
// Use this when you don't need WAL mode or file-based persistence.
//
// Note: In-memory databases are destroyed when the connection closes,
// making them ideal for isolated unit tests.
// Uses api-core/database.Connect for consistency with production.
func SetupInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()
	db, err := database.Connect(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file::memory:?_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		Retry: &retry.Config{
			MaxAttempts: 1, // Single attempt for tests (fail fast)
			BaseDelay:   10 * time.Millisecond,
			MaxDelay:    50 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("Failed to connect to in-memory database: %v", err)
	}

	if err := domain.InitSchema(db); err != nil {
		db.Close()
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
