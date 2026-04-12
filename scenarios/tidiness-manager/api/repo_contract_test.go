package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		".vrooli",
		"scenarios",
		"resources",
		"packages",
		"cmd",
		"internal",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("Failed to create repo dir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test-repo\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "api": "api", "ui": "ui", "cli": "cli", "initialization": "initialization", "requirements": "requirements", "docs": "docs"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {
    "fixture": {
      "description": "test fixture profile",
      "parameters": [],
      "include": ["scenarios"],
      "optional_include": ["go.mod"],
      "exclude": [".git/**"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("Failed to write repo contract: %v", err)
	}
}
