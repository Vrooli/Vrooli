package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteFileDSNUsesFileURIAsIs(t *testing.T) {
	const dsn = "file:test.db?mode=memory"

	got, err := sqliteFileDSN(dsn)
	if err != nil {
		t.Fatalf("sqliteFileDSN() error = %v", err)
	}
	if got != dsn {
		t.Fatalf("sqliteFileDSN() = %q, want %q", got, dsn)
	}
}

func TestSQLiteFileDSNCreatesParentAndAddsPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "tech-tree-designer.db")

	got, err := sqliteFileDSN(dbPath)
	if err != nil {
		t.Fatalf("sqliteFileDSN() error = %v", err)
	}

	if _, err := os.Stat(filepath.Dir(dbPath)); err != nil {
		t.Fatalf("expected parent directory to exist: %v", err)
	}
	if !strings.HasPrefix(got, "file:"+dbPath+"?") {
		t.Fatalf("sqliteFileDSN() = %q, want file URI for %q", got, dbPath)
	}
	for _, pragma := range []string{
		"_pragma=foreign_keys(ON)",
		"_pragma=journal_mode(WAL)",
		"_pragma=busy_timeout(10000)",
	} {
		if !strings.Contains(got, pragma) {
			t.Fatalf("sqliteFileDSN() = %q, missing %q", got, pragma)
		}
	}
}
