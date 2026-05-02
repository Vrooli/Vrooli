package store

import (
	"context"
	"path/filepath"
	"testing"

	"database/sql"

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

// TestEnsureSchema_EmptyPlaceholderIsNoop documents the contract: a
// placeholder schema (comments / blank lines only) is a no-op. The
// template ships in this state, so a freshly-generated scenario must
// boot without erroring on schema apply.
func TestEnsureSchema_EmptyPlaceholderIsNoop(t *testing.T) {
	db := openTestDB(t)
	if err := EnsureSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSchema on placeholder schema: %v", err)
	}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		_ = rows.Scan(&name)
		t.Errorf("placeholder schema unexpectedly created table %q", name)
	}
}

// TestStripComments verifies the comment-only detection so future
// placeholder edits don't accidentally trigger Exec on a script that
// looks empty to a human.
func TestStripComments(t *testing.T) {
	cases := map[string]string{
		"":                                       "",
		"-- comment only":                        "",
		"\n\n\n  -- with blanks\n\n":             "",
		"CREATE TABLE x(id);":                    "CREATE TABLE x(id);",
		"-- header\nCREATE TABLE x(id);\n-- end": "CREATE TABLE x(id);",
	}
	for in, want := range cases {
		if got := stripComments(in); got != want {
			t.Errorf("stripComments(%q) = %q, want %q", in, got, want)
		}
	}
}
