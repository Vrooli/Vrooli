package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary SQLite database with schema and seed data.
// The database is automatically cleaned up when the test finishes.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })

	// Load and execute schema
	schemaPath := filepath.Join(findRepoRoot(t), "scenarios", "web-console", "initialization", "sqlite", "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("exec schema.sql: %v", err)
	}

	// Load and execute seed
	seedPath := filepath.Join(findRepoRoot(t), "scenarios", "web-console", "initialization", "sqlite", "seed.sql")
	seed, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("read seed.sql: %v", err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("exec seed.sql: %v", err)
	}

	return db
}

// findRepoRoot walks up from the current working directory to find the Vrooli
// repo root (identified by the presence of a CLAUDE.md file).
func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no CLAUDE.md found)")
		}
		dir = parent
	}
}
