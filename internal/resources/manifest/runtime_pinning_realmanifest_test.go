package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// gpuCapacityDebt lists resources that declare a gpu block without a capacity
// block, making their VRAM invisible to the capacity broker. Remove an entry
// when the resource gains a capacity claim (or a recorded decision that the
// broker must not manage it).
var gpuCapacityDebt = []string{
	"ollama", // manages its own model residency; broker integration undecided
}

func TestRealGPUResourcesDeclareCapacity(t *testing.T) {
	repoRoot := findRepoRoot(t)
	for _, m := range loadRealManifests(t, repoRoot) {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(m.raw, &raw); err != nil {
			t.Fatalf("%s: parse raw manifest: %v", m.manifest.Name, err)
		}
		_, hasGPU := raw["gpu"]
		_, hasCapacity := raw["capacity"]
		if hasGPU && !hasCapacity && !slices.Contains(gpuCapacityDebt, m.manifest.Name) {
			t.Errorf("%s: declares gpu without a capacity block; add a capacity claim or record it in gpuCapacityDebt with a rationale", m.manifest.Name)
		}
	}
}

type realManifest struct {
	dir      string
	raw      []byte
	manifest manifest.ResourceManifest
}

func loadRealManifests(t *testing.T, repoRoot string) []realManifest {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(repoRoot, "resources", "*", "resource.json"))
	if err != nil {
		t.Fatalf("glob resource manifests: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no resource manifests found")
	}
	manifests := make([]realManifest, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		m, err := manifest.Load(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		manifests = append(manifests, realManifest{dir: filepath.Dir(path), raw: raw, manifest: m})
	}
	return manifests
}
