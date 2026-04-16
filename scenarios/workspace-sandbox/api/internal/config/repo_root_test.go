package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDefaultProjectRoot(t *testing.T) {
	t.Run("prefers explicit project root", func(t *testing.T) {
		t.Setenv("PROJECT_ROOT", "/tmp/custom")
		if got := ResolveDefaultProjectRoot(); got != "/tmp/custom" {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, "/tmp/custom")
		}
	})

	t.Run("resolves repo root from vrooli source root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("PROJECT_ROOT", "")
		t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))
		t.Setenv("VROOLI_ROOT", "")

		if got := ResolveDefaultProjectRoot(); got != repoRoot {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, repoRoot)
		}
	})

	t.Run("falls back to cwd repo root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("PROJECT_ROOT", "")
		t.Setenv("VROOLI_SOURCE_ROOT", "")
		t.Setenv("VROOLI_ROOT", "")
		chdirForTest(t, filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))

		if got := ResolveDefaultProjectRoot(); got != repoRoot {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, repoRoot)
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
