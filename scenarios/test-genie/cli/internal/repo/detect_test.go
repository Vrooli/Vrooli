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

func TestRootAndScenarioDiscoveryUseRepositoryMarkers(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
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
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
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
