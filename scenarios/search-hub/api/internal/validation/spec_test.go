package validation

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
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

func TestLiveEvidenceKeepsCertificationHonest(t *testing.T) {
	cases := []struct {
		name        string
		health      *routingv1.ProviderHealth
		wantCode    string
		wantFinding bool
	}{
		{name: "healthy full corpus", health: &routingv1.ProviderHealth{ProviderId: "leaf", Reachable: true, TotalHits: 4, TimesRouted: 8}},
		{name: "healthy thin corpus", health: &routingv1.ProviderHealth{ProviderId: "leaf", Reachable: true, TotalHits: 1, TimesRouted: 2}},
		{name: "high degradation", health: &routingv1.ProviderHealth{ProviderId: "leaf", Degraded: true, Reachable: false, Reachability: "timeout"}, wantCode: CodeLiveDegraded, wantFinding: true},
		{name: "open circuit", health: &routingv1.ProviderHealth{ProviderId: "leaf", Degraded: true, CircuitState: "open", Reachability: "circuit_open"}, wantCode: CodeLiveDegraded, wantFinding: true},
		{name: "zero yield", health: &routingv1.ProviderHealth{ProviderId: "leaf", Reachable: true, TimesRouted: 5}, wantCode: CodeLiveZeroYield, wantFinding: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			finding := liveEvidenceFinding("leaf", tc.health)
			if !tc.wantFinding {
				if finding != nil {
					t.Fatalf("unexpected finding: %+v", finding)
				}
				return
			}
			if finding == nil || finding.Code != tc.wantCode {
				t.Fatalf("finding = %+v, want code %s", finding, tc.wantCode)
			}
		})
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

func TestSearchHubMaturitySpecDefinesProductionReadinessRung(t *testing.T) {
	spec := mustLoadSpec(t)
	if got := spec.Levels[len(spec.Levels)-1].ID; got != "L4" {
		t.Fatalf("top-level max level = %q, want L4 production readiness", got)
	}
	for _, capability := range spec.Capabilities {
		if got := capability.Levels[len(capability.Levels)-1].StatusLabel; got != "Production" {
			t.Fatalf("%s max status label = %q, want Production", capability.ID, got)
		}
	}
	for _, code := range []string{
		CodeEvalCorpusThin,
		CodeEvalCorpusCoverage,
		CodeStatusEndpointMissing,
		CodeControlEndpointMissing,
		CodePerfBudgetBreach,
		CodePerfDegraded,
	} {
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("missing finding mapping %s", code)
		}
		if mapping.CleanRequirement != string(assessment.CleanRequirementRequired) {
			t.Fatalf("%s clean requirement = %q, want required", code, mapping.CleanRequirement)
		}
		if mapping.LocalLevelImpact != "L3" {
			t.Fatalf("%s local level impact = %q, want L3 production blocker", code, mapping.LocalLevelImpact)
		}
	}
	if got := spec.Findings[CodePerfSamplesUnproven].CleanRequirement; got != string(assessment.CleanRequirementRequired) {
		t.Fatalf("%s clean requirement = %q, want required", CodePerfSamplesUnproven, got)
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
