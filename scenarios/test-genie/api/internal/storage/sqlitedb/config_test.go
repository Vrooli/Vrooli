package sqlitedb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveIgnoresInheritedDatabasePaths is the regression test for the
// defect that put 146 Test Genie runs inside vrooli-autoheal's database. Test
// Genie starts other scenarios, so it is the process most exposed to an
// inherited environment — and it used to prefer three database-path variables
// over its own identity.
func TestResolveIgnoresInheritedDatabasePaths(t *testing.T) {
	inherited := filepath.Join(t.TempDir(), "autoheal.sqlite")
	t.Setenv("TEST_GENIE_SQLITE_PATH", inherited)
	t.Setenv("SQLITE_PATH", inherited)
	t.Setenv("SQLITE_DB", inherited)
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if cfg.Path == inherited || strings.Contains(cfg.Path, "autoheal") {
		t.Fatalf("an inherited database path redirected the run ledger: %s", cfg.Path)
	}
	if filepath.Base(cfg.Path) != databaseFile {
		t.Fatalf("expected %s, got %s", databaseFile, cfg.Path)
	}
	if _, err := os.Stat(inherited); err == nil {
		t.Fatalf("the inherited path was created: %s", inherited)
	}
}

// TestResolveUsesLifecycleDataDir covers the ordinary live case: a
// lifecycle-launched Test Genie keeps its ledger exactly where it already is.
func TestResolveUsesLifecycleDataDir(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("SCENARIO_NAME", scenarioID)
	t.Setenv("VROOLI_SCENARIO", scenarioID)
	t.Setenv("SCENARIO_DATA_DIR", dataRoot)
	t.Setenv("VROOLI_VARIANT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	want := filepath.Join(dataRoot, databaseFile)
	if cfg.Path != want {
		t.Fatalf("expected %s, got %s", want, cfg.Path)
	}
}

// TestResolveRejectsForeignLifecycleDataDir covers the same variable arriving
// from a supervisor rather than from the lifecycle. The identity guard must
// reject it, because it names the supervisor rather than Test Genie.
func TestResolveRejectsForeignLifecycleDataDir(t *testing.T) {
	supervisorData := t.TempDir()
	t.Setenv("SCENARIO_NAME", "vrooli-autoheal")
	t.Setenv("VROOLI_SCENARIO", "vrooli-autoheal")
	t.Setenv("SCENARIO_DATA_DIR", supervisorData)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if strings.HasPrefix(cfg.Path, supervisorData) {
		t.Fatalf("the run ledger landed in the supervisor's data directory: %s", cfg.Path)
	}
	if !strings.Contains(cfg.Path, scenarioID) {
		t.Fatalf("expected a path scoped to %s, got %s", scenarioID, cfg.Path)
	}
}

func TestResolveUsesVariantAwareStorageNamespace(t *testing.T) {
	dataRoot := t.TempDir()
	t.Setenv("SCENARIO_NAME", scenarioID)
	t.Setenv("VROOLI_SCENARIO", scenarioID)
	t.Setenv("SCENARIO_DATA_DIR", dataRoot)
	t.Setenv("VROOLI_VARIANT", "shadow")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", scenarioID+"_shadow")
	t.Setenv("VROOLI_STORAGE_ROOT", "")

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if strings.HasPrefix(cfg.Path, dataRoot) {
		t.Fatalf("the shadow reused live's data directory: %s", cfg.Path)
	}
	if !strings.Contains(cfg.Path, scenarioID+"_shadow") {
		t.Fatalf("path %q is not variant-scoped", cfg.Path)
	}
}

// TestResolveHonoursStorageRootOverride covers the harness lever: a storage
// root redirects the whole class tree and must outrank the lifecycle-assigned
// data directory, or a scenario under test would run on production data.
func TestResolveHonoursStorageRootOverride(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SCENARIO_NAME", scenarioID)
	t.Setenv("VROOLI_SCENARIO", scenarioID)
	t.Setenv("SCENARIO_DATA_DIR", t.TempDir())
	t.Setenv("VROOLI_VARIANT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_STORAGE_ROOT", root)

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !strings.HasPrefix(cfg.Path, root) {
		t.Fatalf("expected the storage root override to win, got %s", cfg.Path)
	}
}

func TestResolveExplicitAcceptsFileDSN(t *testing.T) {
	rawPath := filepath.Join(t.TempDir(), "nested", "test-genie.db")
	dsn, err := BuildDSN(rawPath)
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}

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

func TestResolveExplicitAcceptsPath(t *testing.T) {
	rawPath := filepath.Join(t.TempDir(), "nested", "test-genie.db")

	cfg, err := ResolveExplicit(rawPath)
	if err != nil {
		t.Fatalf("ResolveExplicit returned error: %v", err)
	}
	if cfg.Path != rawPath {
		t.Fatalf("expected %s, got %s", rawPath, cfg.Path)
	}
	if !strings.HasPrefix(cfg.DSN, "file:"+rawPath+"?") {
		t.Fatalf("unexpected DSN: %s", cfg.DSN)
	}
}

func TestBuildDSNCarriesReadOrientedTuning(t *testing.T) {
	dsn, err := BuildDSN("/tmp/test-genie.db")
	if err != nil {
		t.Fatalf("BuildDSN returned error: %v", err)
	}
	for _, fragment := range []string{
		"file:/tmp/test-genie.db",
		"_pragma=foreign_keys(ON)",
		"_pragma=journal_mode(WAL)",
		"_pragma=busy_timeout(10000)",
		"_pragma=synchronous(NORMAL)",
		// The ledger is read far more often than written.
		"_pragma=page_size(4096)",
		"_pragma=mmap_size(268435456)",
	} {
		if !strings.Contains(dsn, fragment) {
			t.Fatalf("expected DSN to contain %q, got %s", fragment, dsn)
		}
	}
}

func TestBuildDSNRejectsAssembledDSN(t *testing.T) {
	if _, err := BuildDSN("file:/tmp/test-genie.db?_pragma=journal_mode(WAL)"); err == nil {
		t.Fatal("expected an already-assembled DSN to be rejected")
	}
}
