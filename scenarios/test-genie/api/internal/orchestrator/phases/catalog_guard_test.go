package phases

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/runnability"
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

func TestCuratedPresetsIncludeProto(t *testing.T) {
	presets := DefaultPresets()
	for _, preset := range []Preset{PresetQuick, PresetSmoke, PresetArchitectureAudit} {
		names, ok := presets[preset.String()]
		if !ok {
			t.Fatalf("preset %q missing from DefaultPresets", preset)
		}
		found := false
		for _, name := range names {
			if name == Proto.String() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("preset %q must include %q for proto contract feedback, got %v", preset, Proto, names)
		}
	}
}

// TestCapabilityManifestCoversEveryPhase is the anti-drift guard for the
// runnability capability manifest. Every catalog phase must carry a manifest
// whose Phase/Optional mirror the spec, and the surface declarations are pinned
// to the behavior the old hand-maintained runtimeNeeds switch encoded — so a
// future capability edit that silently changes which phases need UI/API breaks
// the build instead of changing runtime behavior unnoticed.
func TestCapabilityManifestCoversEveryPhase(t *testing.T) {
	// Pinned expectations transcribed from the pre-refactor runtimeNeeds switch
	// (smoke/playbooks/performance → UI, integration → API) plus the playbooks
	// DB-isolation/lifecycle-mutation contract.
	type want struct {
		ui, api, mutates, deferred bool
		dbiso                      runnability.DBIsolation
	}
	expected := map[Name]want{
		Smoke:       {ui: true},
		Performance: {ui: true},
		Integration: {api: true},
		Playbooks:   {ui: true, mutates: true, deferred: true, dbiso: runnability.DBIsolationRoutedOrRestart},
	}

	catalog := DefaultCatalog()
	for _, spec := range catalog.All() {
		caps := spec.Capabilities
		if caps.Phase != spec.Name.String() {
			t.Errorf("phase %q: Capabilities.Phase = %q, want lockstep with spec name", spec.Name, caps.Phase)
		}
		if caps.Optional != spec.Optional {
			t.Errorf("phase %q: Capabilities.Optional = %v, want %v (spec)", spec.Name, caps.Optional, spec.Optional)
		}
		w := expected[spec.Name] // zero value = static phase with no surface
		if caps.NeedsUI != w.ui || caps.NeedsAPI != w.api {
			t.Errorf("phase %q surfaces: NeedsUI=%v NeedsAPI=%v, want UI=%v API=%v",
				spec.Name, caps.NeedsUI, caps.NeedsAPI, w.ui, w.api)
		}
		if caps.MutatesLifecycle != w.mutates {
			t.Errorf("phase %q: MutatesLifecycle=%v, want %v", spec.Name, caps.MutatesLifecycle, w.mutates)
		}
		if caps.LifecycleDecisionDeferred != w.deferred {
			t.Errorf("phase %q: LifecycleDecisionDeferred=%v, want %v", spec.Name, caps.LifecycleDecisionDeferred, w.deferred)
		}
		if caps.DBIsolation != w.dbiso {
			t.Errorf("phase %q: DBIsolation=%v, want %v", spec.Name, caps.DBIsolation, w.dbiso)
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
