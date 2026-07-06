package validation

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

func TestSearchHubMaturitySpecLoads(t *testing.T) {
	spec := mustLoadSpec(t)
	if spec.Provider != "search-hub" {
		t.Fatalf("provider = %q, want search-hub", spec.Provider)
	}
	if spec.Phase != "search" {
		t.Fatalf("phase = %q, want search", spec.Phase)
	}
}

func TestSearchHubMaturitySpecSplitsSearchCapabilities(t *testing.T) {
	spec := mustLoadSpec(t)
	want := []string{
		"search_descriptor",
		"search_governance",
		"search_eval_performance",
		"search_operability",
	}
	if len(spec.Capabilities) != len(want) {
		t.Fatalf("capabilities = %d, want %d", len(spec.Capabilities), len(want))
	}
	for i, id := range want {
		if spec.Capabilities[i].ID != id {
			t.Fatalf("capability[%d] = %q, want %q", i, spec.Capabilities[i].ID, id)
		}
	}
}

func TestSearchHubMaturityFindingMappingsAreCapabilityScoped(t *testing.T) {
	spec := mustLoadSpec(t)
	levelsByCapability := map[string]map[string]bool{}
	for _, capability := range spec.Capabilities {
		levels := map[string]bool{}
		for _, level := range capability.Levels {
			levels[level.ID] = true
		}
		levelsByCapability[capability.ID] = levels
	}
	for code, mapping := range spec.Findings {
		if mapping.CapabilityID == "" {
			t.Fatalf("%s has no capability_id", code)
		}
		levels, ok := levelsByCapability[mapping.CapabilityID]
		if !ok {
			t.Fatalf("%s maps to unknown capability %q", code, mapping.CapabilityID)
		}
		if !levels[mapping.LocalLevelImpact] {
			t.Fatalf("%s maps to level %q outside capability %q", code, mapping.LocalLevelImpact, mapping.CapabilityID)
		}
		if mapping.EffectiveFixClass() == assessment.FixClassManual && mapping.FixReason == "" {
			t.Fatalf("%s is manual without reason", code)
		}
	}
}

func mustLoadSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	scenarioRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot)
	if err != nil {
		t.Fatalf("LoadSpecFromScenario: %v", err)
	}
	return spec
}
