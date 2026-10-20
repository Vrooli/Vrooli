package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsFromStorage(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)

	if got, want := ResolveConfigBasePath(), filepath.Join(root, "config", "vrooli", "system-monitor"); got != want {
		t.Fatalf("ResolveConfigBasePath() = %q, want %q", got, want)
	}
	if got, want := ResolvePromptBasePath(), filepath.Join(root, "config", "vrooli", "system-monitor", "prompts"); got != want {
		t.Fatalf("ResolvePromptBasePath() = %q, want %q", got, want)
	}
}

func TestResolveRuntimeStateBasePathUsesApiCoreStorage(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)

	want := filepath.Join(storageRoot, "state", "vrooli", "system-monitor")
	if got := ResolveRuntimeStateBasePath(); got != want {
		t.Fatalf("ResolveRuntimeStateBasePath() = %q, want %q", got, want)
	}
}

func TestResolveInvestigationWorkingDirUsesContractScenarioRoot(t *testing.T) {
	root := repoRootForPathsTest(t)
	t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(root, "scenarios", "system-monitor", "api"))
	t.Setenv("VROOLI_ROOT", "")

	if got, want := resolveInvestigationWorkingDir(), filepath.Join(root, "scenarios", "system-monitor"); got != want {
		t.Fatalf("resolveInvestigationWorkingDir() = %q, want %q", got, want)
	}
}

// repoRootForPathsTest anchors on the package directory, which `go test`
// always uses as the working directory. runtime.Caller is not usable here:
// under -trimpath (Test Genie's unit phase) it returns a module path.
func repoRootForPathsTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve test working directory: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..", "..", ".."))
}

func readLiveRepoContract(t *testing.T) string {
	t.Helper()
	root := repoRootForPathsTest(t)
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	return string(data)
}

func writePathsFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
