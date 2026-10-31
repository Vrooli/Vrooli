package adapterregistry

import (
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
	"unit-health/internal/adapters/reactvitest"
)

func TestRegistryHelpersAndResolution(t *testing.T) {
	defaults := Default()
	if source, ok := defaults.ResolveSource(adapters.Identity{ID: "go", Version: "1.0.0"}, adapters.Match{Language: "go"}); !ok || source.Identity().ID != "go" {
		t.Fatalf("default Go source analyzer = %v,%v", source, ok)
	}
	if source, ok := defaults.ResolveSource(adapters.Identity{ID: "missing"}, adapters.Match{Language: "go"}); ok || source != nil {
		t.Fatal("unknown source analyzer resolved")
	}
	if NormalizeFramework(" react-vite ") != "vite" || NormalizeFramework("Vitest") != "vitest" {
		t.Fatal("framework normalization failed")
	}
	if DefaultFramework("go") != "go test" || DefaultFramework("typescript") != "vitest" || DefaultFramework("other") != "" {
		t.Fatal("default framework mapping failed")
	}
	if MinimumCoverageFloor("react-vitest", "", "", "") != 85 || MinimumCoverageFloor("", "", "go", "") != 75 || MinimumCoverageFloor("", "", "rust", "") != 0 {
		t.Fatal("coverage floor mapping failed")
	}
	for _, name := range []string{"testing.json", "vite.config.ts", "vitest.config.js"} {
		if !IsConfigFile(name) {
			t.Errorf("%s should be config", name)
		}
	}
	for _, name := range []string{"go.sum", "Cargo.lock", "poetry.lock", "Pipfile.lock", "pnpm-lock.yaml"} {
		if !IsLockFile(name) {
			t.Errorf("%s should be lock", name)
		}
	}
	root := t.TempDir()
	if HasMarkerFile(root, "rust") {
		t.Fatal("empty root has rust marker")
	}
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasMarkerFile(root, "rust") {
		t.Fatal("rust marker not detected")
	}
	r := New()
	uiAnalyzer := reactvitest.Analyzer{}
	if err := r.Register(uiAnalyzer); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Resolve("react-vitest", "typescript", "vitest"); !ok {
		t.Fatal("registered analyzer not resolved")
	}
	if _, ok := r.Resolve("react-vitest", "go", "go test"); ok {
		t.Fatal("mismatched analyzer resolved")
	}
	if err := r.Register(uiAnalyzer); err == nil {
		t.Fatal("duplicate analyzer accepted")
	}
	if err := (*Registry)(nil).Register(uiAnalyzer); err == nil {
		t.Fatal("nil registry accepted")
	}
	if _, ok := (*Registry)(nil).Resolve("react-vitest", "typescript", ""); ok {
		t.Fatal("nil registry resolved")
	}
	if _, ok := r.Resolve("", "typescript", "vite"); !ok {
		t.Fatal("compatibility resolution should use the registered react adapter")
	}
}
