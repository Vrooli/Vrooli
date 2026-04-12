package repo

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetRootCache() {
	rootOnce = sync.Once{}
	rootPath = ""
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test-repo\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs"}},
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
		t.Fatalf("write repo contract: %v", err)
	}
}

func TestRootAndScenarioDiscoveryUseRepositoryMarkers(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir scenario coverage: %v", err)
	}
	workDir := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}

	t.Chdir(workDir)
	resetRootCache()
	defer resetRootCache()

	if got := Root(); got != root {
		t.Fatalf("expected root %q, got %q", root, got)
	}

	paths := DiscoverScenarioPaths("demo")
	if paths.ScenarioDir != scenarioDir {
		t.Fatalf("expected scenario dir %q, got %q", scenarioDir, paths.ScenarioDir)
	}
	if paths.TestDir != filepath.Join(scenarioDir, "coverage") {
		t.Fatalf("expected coverage dir to be discovered, got %q", paths.TestDir)
	}

	if got := AbsPath("scenarios/demo"); got != filepath.Join(root, "scenarios", "demo") {
		t.Fatalf("expected relative path to resolve from repo root, got %q", got)
	}
}

func TestFileStateHandlesFilesDirectoriesAndMissingPaths(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Chdir(root)
	resetRootCache()
	defer resetRootCache()

	emptyFile := filepath.Join(root, "empty.txt")
	if err := os.WriteFile(emptyFile, nil, 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	exists, empty := FileState(emptyFile)
	if !exists || !empty {
		t.Fatalf("expected empty file to report exists=true empty=true, got exists=%v empty=%v", exists, empty)
	}

	dirPath := filepath.Join(root, "dir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	exists, empty = FileState(dirPath)
	if !exists || !empty {
		t.Fatalf("expected directories to be treated as empty existing paths, got exists=%v empty=%v", exists, empty)
	}

	if exists, empty = FileState("missing.txt"); exists || empty {
		t.Fatalf("expected missing path to report false,false, got exists=%v empty=%v", exists, empty)
	}
}
