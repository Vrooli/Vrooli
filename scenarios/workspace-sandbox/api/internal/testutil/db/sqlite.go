// Package db provides test helpers for SQLite-backed repository
// tests. NewSQLite returns a connected handle with the production
// schema applied to a per-test temp dir.
package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/repository"
)

// NewSQLite returns a fresh, fully-initialized SQLite handle backed by
// a file in the test's temp dir. The handle is closed automatically
// via t.Cleanup. Connection pool is capped at 1 to mirror production.
//
// Schema is applied via repository.EnsureSchema (the same entry point
// main.go uses), so the returned handle is byte-identical to a fresh
// prod install — including the schema_version row.
func NewSQLite(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_txlock=immediate"

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	if err := repository.EnsureSchema(context.Background(), d, clock.System{}); err != nil {
		d.Close()
		t.Fatalf("ensure schema: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// NewSandboxRepository returns a real SandboxRepository backed by a
// per-test SQLite handle. Convenience for repository-integration tests
// that need the real production code path. Wires clock.System{} by
// default; tests that need deterministic timestamps should construct
// the repository themselves with a FakeClock.
func NewSandboxRepository(t *testing.T) *repository.SandboxRepository {
	return repository.NewSandboxRepository(NewSQLite(t), clock.System{})
}
