package store

import (
	"context"
	"path/filepath"
	"testing"

	"database/sql"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "schema_test.db") +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// TestEnsureSchema_AppliesNotesTable pins the canonical bootstrap
// contract: after EnsureSchema runs, every table the template ships is
// present in sqlite_master. The notes table is the canonical CRUD
// reference; if it's missing, every downstream test (handler, repo)
// fails opaquely on missing-table errors.
func TestEnsureSchema_AppliesNotesTable(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, EnsureSchema(context.Background(), db))

	tables := listTables(t, db)
	require.Contains(t, tables, "notes",
		"EnsureSchema must create the notes table; got tables=%v", tables)
}

// TestEnsureSchema_Idempotent verifies the script can run twice without
// erroring. main.go invokes EnsureSchema on every boot, so a second
// call against an already-populated database must succeed.
func TestEnsureSchema_Idempotent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	require.NoError(t, EnsureSchema(ctx, db))
	require.NoError(t, EnsureSchema(ctx, db),
		"EnsureSchema must be idempotent (uses IF NOT EXISTS guards)")
}

func listTables(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	require.NoError(t, err)
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		out = append(out, name)
	}
	require.NoError(t, rows.Err())
	return out
}
