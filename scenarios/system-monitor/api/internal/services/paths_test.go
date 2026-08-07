package services

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePathsFromRepoContract(t *testing.T) {
	root := repoRootForPathsTest(t)
	t.Setenv("VROOLI_SOURCE_ROOT", root)
	t.Setenv("VROOLI_ROOT", "")

	if got, want := ResolveConfigBasePath(), filepath.Join(root, "scenarios", "system-monitor", ".vrooli"); got != want {
		t.Fatalf("ResolveConfigBasePath() = %q, want %q", got, want)
	}
	if got, want := ResolvePromptBasePath(), filepath.Join(root, "scenarios", "system-monitor", "prompts"); got != want {
		t.Fatalf("ResolvePromptBasePath() = %q, want %q", got, want)
	}
	if got, want := ResolveScriptsDir(), filepath.Join(root, "scenarios", "system-monitor", "investigations", "active"); got != want {
		t.Fatalf("ResolveScriptsDir() = %q, want %q", got, want)
	}
}

func TestResolveScriptsDirReturnsScenarioPathEvenWhenMissing(t *testing.T) {
	repoRoot := t.TempDir()
	writePathsFixture(t, repoRoot, ".vrooli/repo-contract.json", readLiveRepoContract(t))
	writePathsFixture(t, repoRoot, "go.mod", "module example.com/test\n\ngo 1.24.0\n")
	for _, dir := range []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if dir == ".vrooli" {
			continue
		}
		if err := os.MkdirAll(filepath.Join(repoRoot, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writePathsFixture(t, repoRoot, filepath.Join("scenarios", "system-monitor", ".vrooli", "service.json"), `{"service":{"name":"system-monitor"}}`)

	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv("VROOLI_ROOT", "")
	if got, want := ResolveScriptsDir(), filepath.Join(repoRoot, "scenarios", "system-monitor", "investigations", "active"); got != want {
		t.Fatalf("ResolveScriptsDir() = %q, want %q", got, want)
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

func repoRootForPathsTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
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
