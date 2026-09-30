package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func resetRootCache() {
	rootOnce = sync.Once{}
	rootPath = ""
}

// writeRepoContractFixture builds a throwaway repo whose .vrooli/repo-contract.json
// is the live repo contract copied verbatim — never a hand-typed copy — so the
// fixture cannot drift from the schema's single source of truth (e.g. when a
// required field such as runtime_home is added). Required directories come from
// the live contract's root.markers.
func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	contract := liveRepoContract(t)
	var parsed struct {
		Root struct {
			Markers struct {
				RequiredDirs []string `json:"required_dirs"`
			} `json:"markers"`
		} `json:"root"`
	}
	if err := json.Unmarshal(contract, &parsed); err != nil {
		t.Fatalf("parse live repo contract: %v", err)
	}
	dirs := parsed.Root.Markers.RequiredDirs
	if len(dirs) == 0 {
		dirs = []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal", "docs"}
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test-repo\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
}

// liveRepoContract reads the repository's authoritative
// .vrooli/repo-contract.json from the current repository. This remains valid
// under `go test -trimpath`, where runtime.Caller source paths are intentionally
// not absolute.
func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("locate live repository contract: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read live repository contract: %v", err)
	}
	return data
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
