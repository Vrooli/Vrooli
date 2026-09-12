package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage"
)

// The DSN construction itself is owned and exhaustively tested by
// api-core/storage. What is worth asserting HERE is the property that is
// specific to this scenario: that it opens ITS OWN database, and that nothing
// arriving in the environment can move it.

func TestScenarioResolvesItsOwnDatabase(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())

	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "tech-tree-designer"})
	if err != nil {
		t.Fatalf("SQLiteDSN returned error: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:") {
		t.Fatalf("expected a file DSN, got %q", dsn)
	}
	if !strings.Contains(dsn, "tech-tree-designer") {
		t.Fatalf("DSN is not scoped to this scenario: %q", dsn)
	}
	for _, want := range []string{"journal_mode(WAL)", "foreign_keys(ON)", "busy_timeout(10000)"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN missing %s: %q", want, dsn)
		}
	}
}

// TestScenarioIgnoresInheritedDatabasePath is the regression test for the
// cross-scenario database hijack: a supervisor that exports a database path and
// then restarts this scenario must not redirect its storage.
func TestScenarioIgnoresInheritedDatabasePath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	inherited := filepath.Join(root, "inherited", "autoheal.sqlite")
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DB"} {
		t.Setenv(key, inherited)
	}
	t.Setenv("SCENARIO_NAME", "vrooli-autoheal")
	t.Setenv("VROOLI_SCENARIO", "vrooli-autoheal")
	t.Setenv("SCENARIO_DATA_DIR", filepath.Join(root, "autoheal-data"))

	dsn, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "tech-tree-designer"})
	if err != nil {
		t.Fatalf("SQLiteDSN returned error: %v", err)
	}
	if strings.Contains(dsn, "autoheal") || strings.Contains(dsn, "inherited") {
		t.Fatalf("an inherited environment redirected this scenario: %q", dsn)
	}
	if !strings.Contains(dsn, "tech-tree-designer") {
		t.Fatalf("expected this scenario's own database, got %q", dsn)
	}
}
