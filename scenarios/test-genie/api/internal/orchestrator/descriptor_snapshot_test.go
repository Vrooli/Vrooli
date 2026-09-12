package orchestrator

import (
	"encoding/json"
	"testing"

	"test-genie/internal/orchestrator/applicability"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerdescriptor"
)

func TestBuildRunDescriptorSnapshotFreezesCatalogEvolution(t *testing.T) { // [REQ:TESTGENIE-DESCRIPTOR-SNAPSHOT-P0]
	unitDef := phases.Definition{Name: phases.Name("unit"), DisplayName: "Unit", ProviderScenario: "unit-health", Policy: phasepolicy.RequiredProviderPolicy()}
	searchDef := phases.Definition{Name: phases.Name("search"), DisplayName: "Search", ProviderScenario: "search-hub", Policy: phasepolicy.RequiredProviderPolicy()}
	unitDescriptor := providerdescriptor.Descriptor{
		SchemaVersion: "1.0.0", Scenario: "unit-health", Phase: "unit", DisplayName: "Unit Health",
		Description: "Validates unit tests.", OrderHint: 20, PhaseClass: "quality", RuntimeClass: "execution",
		Dimensions: []string{"testing"}, FindingSource: "coverage", EvidenceKinds: []string{"coverage.report"},
		Docs:     providerdescriptor.Docs{Path: "scenarios/unit-health/docs/README.md"},
		Maturity: json.RawMessage(`{"version":"2.0.0"}`), Policy: providerdescriptor.Policy{Policy: phasepolicy.RequiredProviderPolicy()},
		Applicability: providerdescriptor.Applicability{Default: "applies"},
	}
	searchDescriptor := providerdescriptor.Descriptor{
		SchemaVersion: "1.0.0", Scenario: "search-hub", Phase: "search", DisplayName: "Search Health",
		Description: "Validates search.", OrderHint: 30, PhaseClass: "capability", RuntimeClass: "execution",
		Aliases: []string{"legacy-search"}, Supersedes: []string{"search-v0"},
		Policy: phasepolicyDescriptor(), Applicability: providerdescriptor.Applicability{Default: "not_applicable"},
	}
	plan := &phasePlan{
		Definitions: []phases.Definition{unitDef, searchDef}, Selected: []phases.Definition{unitDef},
		Applicability: map[string]phaseApplicabilityNotice{
			"unit": {
				Definition: unitDef, Descriptor: unitDescriptor,
				Result: applicability.Result{Phase: "unit", Status: applicability.StatusApplies, Reasons: []applicability.Reason{{Code: "applicability.default_applies", Message: "applies"}}},
			},
			"search": {
				Definition: searchDef, Descriptor: searchDescriptor,
				Result: applicability.Result{Phase: "search", Status: applicability.StatusNotApplicable, Reasons: []applicability.Reason{{Code: "applicability.default_not_applicable", Message: "not applicable"}}},
			},
		},
	}

	oldSnapshot, err := buildRunDescriptorSnapshot(plan)
	if err != nil {
		t.Fatalf("build old snapshot: %v", err)
	}
	plan.Definitions = []phases.Definition{searchDef, unitDef}
	plan.Applicability["unit"] = phaseApplicabilityNotice{
		Definition: unitDef,
		Descriptor: func() providerdescriptor.Descriptor {
			changed := unitDescriptor
			changed.DisplayName = "Verification"
			changed.OrderHint = 99
			changed.Scenario = "next-unit-health"
			return changed
		}(),
		Result: applicability.Result{Phase: "unit", Status: applicability.StatusApplies},
	}
	newSnapshot, err := buildRunDescriptorSnapshot(plan)
	if err != nil {
		t.Fatalf("build evolved snapshot: %v", err)
	}

	if oldSnapshot.Phases[0].DisplayName != "Unit Health" || oldSnapshot.Phases[0].Provider != "unit-health" || oldSnapshot.Phases[0].OrderHint != 20 {
		t.Fatalf("old snapshot was rewritten by live catalog evolution: %+v", oldSnapshot.Phases[0])
	}
	if oldSnapshot.Phases[1].Applicability.Status != "not_applicable" || oldSnapshot.Phases[1].Applicability.Planned {
		t.Fatalf("captured applicability = %+v", oldSnapshot.Phases[1].Applicability)
	}
	if newSnapshot.Phases[1].DisplayName != "Verification" || newSnapshot.Digest == oldSnapshot.Digest {
		t.Fatalf("evolved snapshot = %+v digest=%q, old digest=%q", newSnapshot.Phases[1], newSnapshot.Digest, oldSnapshot.Digest)
	}
}

func phasepolicyDescriptor() providerdescriptor.Policy {
	return providerdescriptor.Policy{Policy: phasepolicy.RequiredProviderPolicy()}
}
