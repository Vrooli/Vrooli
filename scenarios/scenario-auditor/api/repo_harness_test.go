package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"scenario-auditor/internal/repocontext"
)

type repoHarness struct {
	Root    string
	Context *repocontext.Context
}

func newRepoHarness(t *testing.T) *repoHarness {
	t.Helper()

	root := t.TempDir()
	liveRoot := liveRepoRootForTest(t)

	contractBytes, err := os.ReadFile(filepath.Join(liveRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read live repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractBytes, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}

	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatalf("load fixture repo contract: %v", err)
	}

	for _, dir := range contract.RootMarkers().RequiredDirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, file := range contract.RootMarkers().RequiredFiles {
		abs := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", file, err)
		}
		contents := []byte{}
		if filepath.Base(file) == "go.mod" {
			contents = []byte("module fixture\n\ngo 1.24.0\n")
		}
		if err := os.WriteFile(abs, contents, 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	h := &repoHarness{Root: root}
	h.WriteScenario(t, "scenario-auditor")
	h.WriteScenario(t, "demo")

	ctx, err := repocontext.FromRepoRoot(root)
	if err != nil {
		t.Fatalf("build repo context: %v", err)
	}
	h.Context = ctx
	return h
}

func (h *repoHarness) WriteScenario(t *testing.T, name string) string {
	t.Helper()

	scenarioRoot, err := h.ContextOrBuild(t).ResolveScenarioPath(name)
	if err != nil {
		t.Fatalf("resolve scenario root for %s: %v", name, err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir scenario manifest dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "api"), 0o755); err != nil {
		t.Fatalf("mkdir scenario api dir: %v", err)
	}
	writeJSONFile(t, filepath.Join(scenarioRoot, ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": name},
	})
	return scenarioRoot
}

func (h *repoHarness) ContextOrBuild(t *testing.T) *repocontext.Context {
	t.Helper()
	if h.Context != nil {
		return h.Context
	}
	ctx, err := repocontext.FromRepoRoot(h.Root)
	if err != nil {
		t.Fatalf("build repo context: %v", err)
	}
	h.Context = ctx
	return ctx
}

func (h *repoHarness) UseRepoContext(t *testing.T) {
	t.Helper()
	setRepoContext(h.ContextOrBuild(t))
	t.Cleanup(clearRepoContext)
}

func (h *repoHarness) UseAmbientResolution(t *testing.T) {
	t.Helper()
	clearRepoContext()
	t.Setenv("VROOLI_ROOT", h.Root)
}

func (h *repoHarness) ChdirToScenarioAuditorAPI(t *testing.T) {
	t.Helper()
	chdirForTest(t, filepath.Join(h.ContextOrBuild(t).ScenarioAuditorRoot(), "api"))
}

func liveRepoRootForTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func writeJSONFile(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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
	clearRepoContext()
}
