package sqlitedb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveUsesPrimaryEnvVar(t *testing.T) {
	t.Setenv("SCENARIO_DATA_DIR", "")
	t.Setenv("SQLITE_DATABASE_PATH", "")
	t.Setenv("VROOLI_DATA", "")

	raw := filepath.Join(t.TempDir(), "custom", "test-genie.db")
	t.Setenv(PrimaryPathEnvVar, raw)

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	want, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if cfg.Path != want {
		t.Fatalf("expected resolved path %s, got %s", want, cfg.Path)
	}
	if cfg.DSN != BuildDSN(want) {
		t.Fatalf("expected DSN %s, got %s", BuildDSN(want), cfg.DSN)
	}
}

func TestResolveFallsBackToScenarioDataDir(t *testing.T) {
	t.Setenv(PrimaryPathEnvVar, "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	t.Setenv("SQLITE_DATABASE_PATH", "")
	t.Setenv("VROOLI_DATA", "")

	root := filepath.Join(t.TempDir(), "data")
	t.Setenv("SCENARIO_DATA_DIR", root)

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	want, err := filepath.Abs(filepath.Join(root, defaultDatabaseFile))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if cfg.Path != want {
		t.Fatalf("expected fallback path %s, got %s", want, cfg.Path)
	}
}

func TestResolveExpandsScenarioDataDirTemplate(t *testing.T) {
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")

	root := filepath.Join(t.TempDir(), "data")
	t.Setenv("SCENARIO_DATA_DIR", root)
	t.Setenv(PrimaryPathEnvVar, filepath.Join("${SCENARIO_DATA_DIR}", "custom.db"))

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	want, err := filepath.Abs(filepath.Join(root, "custom.db"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if cfg.Path != want {
		t.Fatalf("expected expanded path %s, got %s", want, cfg.Path)
	}
}

func TestResolveExplicitAcceptsFileDSN(t *testing.T) {
	rawPath := filepath.Join(t.TempDir(), "nested", "test-genie.db")
	dsn := BuildDSN(rawPath)

	cfg, err := ResolveExplicit(dsn)
	if err != nil {
		t.Fatalf("ResolveExplicit returned error: %v", err)
	}

	want, err := filepath.Abs(rawPath)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if cfg.Path != want {
		t.Fatalf("expected resolved path %s, got %s", want, cfg.Path)
	}
	if cfg.DSN != dsn {
		t.Fatalf("expected DSN to remain unchanged, got %s", cfg.DSN)
	}
	if _, err := os.Stat(filepath.Dir(want)); err != nil {
		t.Fatalf("expected sqlite directory to be created: %v", err)
	}
}

func TestResolveReturnsErrorWhenNoLocationConfigured(t *testing.T) {
	t.Setenv(PrimaryPathEnvVar, "")
	t.Setenv("SQLITE_PATH", "")
	t.Setenv("SQLITE_DB", "")
	t.Setenv("SCENARIO_DATA_DIR", "")
	t.Setenv("SQLITE_DATABASE_PATH", "")
	t.Setenv("VROOLI_DATA", "")

	_, err := Resolve()
	if err == nil {
		t.Fatal("expected Resolve to fail when no sqlite location is configured")
	}
}

func TestBuildDSNIncludesPortablePragmas(t *testing.T) {
	dsn := BuildDSN("/tmp/test-genie.db")
	for _, fragment := range []string{
		"file:/tmp/test-genie.db",
		"_pragma=foreign_keys(ON)",
		"_pragma=journal_mode(WAL)",
		"_pragma=busy_timeout(10000)",
		"_pragma=synchronous(NORMAL)",
	} {
		if !strings.Contains(dsn, fragment) {
			t.Fatalf("expected DSN to contain %q, got %s", fragment, dsn)
		}
	}
}
