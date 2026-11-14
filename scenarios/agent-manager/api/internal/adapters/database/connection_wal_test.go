package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage"
)

func TestSQLiteDSNPreservesCanonicalPathAndPragmasOnEveryConnection(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store space & plus+ ü")
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("SCENARIO_DATA_DIR", "")
	dsn, err := SQLiteDSN(nil)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err := SQLiteFilePath()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	ctx := context.Background()
	// Hold both connections open so the second assertion exercises a fresh
	// driver connection rather than reusing a previously configured handle.
	for range 2 {
		conn, err := raw.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		for pragma, want := range map[string]int64{
			"journal_size_limit": walJournalSizeLimit,
			"foreign_keys":       1,
			"busy_timeout":       storage.DefaultSQLiteBusyTimeout.Milliseconds(),
			"page_size":          4096,
			"cache_size":         -storage.DefaultSQLiteCacheSizeKiB,
			"synchronous":        1,
			"temp_store":         2,
		} {
			var got int64
			if err := conn.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("PRAGMA %s = %d, want %d", pragma, got, want)
			}
		}
		var mode string
		if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if mode != "wal" {
			t.Fatalf("journal mode = %q, want wal", mode)
		}
		var seq int
		var name, gotPath string
		if err := conn.QueryRowContext(ctx, "PRAGMA database_list").Scan(&seq, &name, &gotPath); err != nil {
			t.Fatal(err)
		}
		if gotPath != wantPath {
			t.Fatalf("opened %q, want canonical path %q", gotPath, wantPath)
		}
	}
}

func TestSQLiteDSNRejectsPathsThatCouldRedirectTheDatabase(t *testing.T) {
	for _, suffix := range []string{"query?path", "fragment#path"} {
		t.Run(suffix, func(t *testing.T) {
			t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(t.TempDir(), suffix))
			t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
			t.Setenv("SCENARIO_DATA_DIR", "")
			if _, err := SQLiteDSN(nil); err == nil {
				t.Fatal("expected unsafe DSN path to be rejected")
			}
		})
	}
}
