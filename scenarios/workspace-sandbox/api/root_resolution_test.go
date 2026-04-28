package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestResolveWorkspaceSandboxScenarioDir(t *testing.T) {
	t.Run("resolves from contract-aware env root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))
		t.Setenv("VROOLI_ROOT", "")

		got, err := resolveWorkspaceSandboxScenarioDir()
		if err != nil {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir: %v", err)
		}
		want := filepath.Join(repoRoot, "scenarios", "workspace-sandbox")
		if got != want {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir = %q, want %q", got, want)
		}
	})

	t.Run("falls back to cwd repo root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("VROOLI_SOURCE_ROOT", "")
		t.Setenv("VROOLI_ROOT", "")
		chdirForTest(t, filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))

		got, err := resolveWorkspaceSandboxScenarioDir()
		if err != nil {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir: %v", err)
		}
		want := filepath.Join(repoRoot, "scenarios", "workspace-sandbox")
		if got != want {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir = %q, want %q", got, want)
		}
	})
}

func writeRepoContractFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "workspace-sandbox", "api"), 0o755); err != nil {
		t.Fatalf("mkdir workspace-sandbox api: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "workspace-sandbox", ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir workspace-sandbox config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "workspace-sandbox", ".vrooli", "service.json"), []byte(`{"service":{"name":"workspace-sandbox"}}`), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "template_dir": "templates", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "docs": "docs", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "initialization": "initialization"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {
    "fixture": {
      "description": "fixture profile",
      "parameters": ["scenario"],
      "include": ["scenarios/{scenario}"],
      "optional_include": ["go.mod"],
      "exclude": [".git/**"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
	return root
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
}

func TestEnsureSchemaRequiresLifecycleBootstrap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE SCHEMA IF NOT EXISTS workspace_sandbox").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SET search_path TO workspace_sandbox, public").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("workspace_sandbox").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	err = ensureSchema(db)
	if err == nil {
		t.Fatal("ensureSchema() expected error, got nil")
	}
	if got := err.Error(); got == "" || !containsAll(got, "workspace_sandbox.sandboxes", "rerun scenario bootstrap") {
		t.Fatalf("ensureSchema() error = %q, want actionable bootstrap guidance", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestEnsureSchemaAcceptsBootstrappedDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE SCHEMA IF NOT EXISTS workspace_sandbox").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SET search_path TO workspace_sandbox, public").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// sandboxes table existence
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("workspace_sandbox").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// applied_changes table existence
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("workspace_sandbox").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// applied_changes.agent_manager_run_id column
	mock.ExpectQuery("SELECT EXISTS \\(").
		WithArgs("workspace_sandbox").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	// sandbox-provenance v1.0.0 columns (5 of them).
	for _, col := range []string{"schema_version", "run_outcome", "provenance_state", "conversation_id", "cost_usd"} {
		mock.ExpectQuery("SELECT EXISTS \\(").
			WithArgs("workspace_sandbox", col).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	}

	if err := ensureSchema(db); err != nil {
		t.Fatalf("ensureSchema() unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
