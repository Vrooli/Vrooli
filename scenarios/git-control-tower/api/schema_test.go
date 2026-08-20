package main

import (
	"strings"
	"testing"
)

func TestSQLiteDSNUsesVariantAwareStorageNamespace(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", t.TempDir())
	t.Setenv("VROOLI_SCENARIO", "git-control-tower")
	t.Setenv("VROOLI_VARIANT", "shadow")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "git-control-tower_shadow")

	dsn, err := sqliteDSN()
	if err != nil {
		t.Fatalf("sqliteDSN() error: %v", err)
	}
	if !strings.Contains(dsn, "git-control-tower_shadow") {
		t.Fatalf("variant-aware database DSN %q does not contain the shadow namespace", dsn)
	}
}

// TestSQLiteDSNIgnoresInheritedPathOverrides is the regression test for the
// cross-scenario database hijack. This scenario previously honoured
// GCT_SQLITE_PATH, SQLITE_PATH, SQLITE_DB, and a "file:" DATABASE_URL ahead of
// its own identity, so any process that exported one and then started this
// scenario redirected its database. None of them may be read now.
func TestSQLiteDSNIgnoresInheritedPathOverrides(t *testing.T) {
	for _, key := range []string{"GCT_SQLITE_PATH", "SQLITE_PATH", "SQLITE_DB"} {
		t.Setenv(key, "/inherited/autoheal.sqlite")
	}
	t.Setenv("DATABASE_URL", "file:/inherited/autoheal.sqlite")

	dsn, err := sqliteDSN()
	if err != nil {
		t.Fatalf("sqliteDSN() error: %v", err)
	}
	if strings.Contains(dsn, "inherited") || strings.Contains(dsn, "autoheal") {
		t.Fatalf("an inherited environment redirected the database: %s", dsn)
	}
	if !strings.Contains(dsn, "git-control-tower") {
		t.Fatalf("expected this scenario's own database, got %s", dsn)
	}
}

func TestSQLiteDSNKeepsReadOrientedTuning(t *testing.T) {
	dsn, err := sqliteDSN()
	if err != nil {
		t.Fatalf("sqliteDSN() error: %v", err)
	}
	for _, want := range []string{"page_size(4096)", "mmap_size(268435456)", "journal_mode(WAL)"} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("expected %q in %s", want, dsn)
		}
	}
}
