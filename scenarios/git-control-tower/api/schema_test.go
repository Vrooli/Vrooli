package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteDSNUsesVariantAwareStorageNamespace(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("GCT_SQLITE_PATH", "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SCENARIO_DATA_DIR", dataRoot)
	t.Setenv("VROOLI_SCENARIO", "git-control-tower")
	t.Setenv("VROOLI_VARIANT", "shadow")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "git-control-tower_shadow")

	dsn, err := sqliteDSN()
	if err != nil {
		t.Fatalf("sqliteDSN() error: %v", err)
	}
	if strings.HasPrefix(dsn, "file:"+filepath.Join(dataRoot, "git-control-tower.db")) {
		t.Fatalf("shadow database reused live SCENARIO_DATA_DIR: %s", dsn)
	}
	if !strings.Contains(dsn, "git-control-tower_shadow") {
		t.Fatalf("variant-aware database DSN %q does not contain the shadow namespace", dsn)
	}
}

func TestSQLiteDSNHonorsExplicitOverride(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "explicit.db")
	t.Setenv("GCT_SQLITE_PATH", explicit)
	t.Setenv("VROOLI_VARIANT", "shadow")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "git-control-tower_shadow")

	dsn, err := sqliteDSN()
	if err != nil {
		t.Fatalf("sqliteDSN() error: %v", err)
	}
	if !strings.HasPrefix(dsn, "file:"+explicit) {
		t.Fatalf("sqliteDSN() = %q, want explicit path %q", dsn, explicit)
	}
}
