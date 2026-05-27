package phases

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scenarioRoot resolves scenarios/test-genie from this test file's location so
// doc-existence checks don't depend on the working directory.
func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../scenarios/test-genie/api/internal/orchestrator/phases/<file>
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

// TestPresetsResolveAgainstCatalog is the anti-drift guard for presets: every
// phase referenced by a built-in preset must exist in the catalog.
func TestPresetsResolveAgainstCatalog(t *testing.T) {
	if err := ValidatePresets(DefaultCatalog()); err != nil {
		t.Fatalf("default presets must resolve against the catalog: %v", err)
	}
	valid := make(map[string]struct{})
	for _, n := range ValidPhaseNames() {
		valid[n] = struct{}{}
	}
	for preset, phases := range DefaultPresets() {
		for _, p := range phases {
			if _, ok := valid[p]; !ok {
				t.Errorf("preset %q references unknown phase %q", preset, p)
			}
		}
	}
}

// TestDocPathsCoverEveryCatalogPhase is the anti-drift guard for documentation:
// every catalog phase resolves to a doc path that exists on disk, and unknown
// phases resolve to nothing.
func TestDocPathsCoverEveryCatalogPhase(t *testing.T) {
	root := scenarioRoot(t)
	for _, name := range ValidPhaseNames() {
		docs := DocPaths(name)
		if len(docs) == 0 {
			t.Errorf("phase %q has no documentation path", name)
			continue
		}
		for _, rel := range docs {
			abs := filepath.Join(root, strings.TrimPrefix(rel, "scenarios/test-genie/"))
			if _, err := os.Stat(abs); err != nil {
				t.Errorf("phase %q doc %q missing on disk (%s): %v", name, rel, abs, err)
			}
		}
	}
	if got := DocPaths("nonexistent-phase"); got != nil {
		t.Errorf("DocPaths(nonexistent) = %v, want nil", got)
	}
}
