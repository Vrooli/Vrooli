package execute

import (
	"encoding/json"
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func sampleEvent() *runspb.RunEvent {
	return &runspb.RunEvent{
		Event:  "phase_completed",
		Phase:  "contracts",
		Status: "passed",
		PhasePresentation: &commonv1.PhasePresentation{
			Provider:             "cli-health",
			Phase:                "contracts",
			CurrentLevel:         "L3",
			CurrentLevelLabel:    "Ready",
			NextLevel:            "L4",
			CeilingLevel:         "L4",
			NorthStar:            "Verified renderer-separated primitives.",
			NextAction:           "Prove each declared primitive with cli-core evidence.",
			NextActionReason:     "highest-unlock capability gap",
			FocusCapabilityId:    "command_architecture",
			FocusCapabilityLabel: "Command Architecture",
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			Capabilities: []*commonv1.PhaseCapabilityPresentation{
				{Id: "command_architecture", Label: "Command Architecture", CurrentLevel: "L3", NextLevel: "L4"},
			},
		},
		FindingsSummary: &runspb.PhaseFindingsSummary{Warnings: 1, Total: 1},
	}
}

// TestPhaseAndJSONCarryOnePresentation proves the human and --json paths carry
// the exact provider object from one server payload.
func TestPhaseAndJSONDeriveOneStanding(t *testing.T) {
	ev := sampleEvent()
	phase := phaseFromEvent(ev)

	if phase.PhasePresentation == nil {
		t.Fatal("phaseFromEvent dropped the maturity standing")
	}
	got, err := json.Marshal(phase.PhasePresentation)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(ev.GetPhasePresentation())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(wantJSON) {
		t.Fatalf("--json standing diverges from the shared mapping:\n got=%s\nwant=%s", got, wantJSON)
	}

	// The --json Response carries the standing verbatim (no separate derivation).
	resp := buildResponse(&runspb.RunEvent{Event: "run_completed", Success: true}, []Phase{phase})
	blob, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), `"current_level":"L3"`) || !strings.Contains(string(blob), `"north_star":"Verified renderer-separated primitives."`) {
		t.Fatalf("--json response missing the standing: %s", blob)
	}
	if phase.FindingsSummary == nil || phase.FindingsSummary.Total != 1 {
		t.Fatalf("findings summary not carried: %+v", phase.FindingsSummary)
	}
}

func TestPresentationCarriesProviderDocumentationTopics(t *testing.T) {
	ev := sampleEvent()
	ev.PhasePresentation.DocumentationTopics = []string{"contracts maturity next move", "contracts arch.primitive_unverified canonical fix"}
	st := phaseFromEvent(ev).PhasePresentation
	if len(st.GetDocumentationTopics()) == 0 {
		t.Fatal("expected runnable doc-search topics for a non-max phase")
	}
	joined := strings.Join(st.GetDocumentationTopics(), "|")
	if !strings.Contains(joined, "contracts") || !strings.Contains(joined, "arch.primitive_unverified") {
		t.Fatalf("doc topics should reference the phase + blocking codes: %v", st.GetDocumentationTopics())
	}
}

func TestStandingAtMaximumSuppressesTopics(t *testing.T) {
	ev := sampleEvent()
	ev.PhasePresentation.AtMaximum = true
	ev.PhasePresentation.NextLevel = ""
	ev.PhasePresentation.BlockingFindingCodes = nil
	st := phaseFromEvent(ev).PhasePresentation
	if len(st.GetDocumentationTopics()) != 0 {
		t.Fatalf("no doc topics expected at maximum maturity, got %v", st.GetDocumentationTopics())
	}
}

func TestStandingNilWhenAbsent(t *testing.T) {
	phase := phaseFromEvent(&runspb.RunEvent{Event: "phase_completed", Phase: "native", Status: "passed"})
	if phase.PhasePresentation != nil {
		t.Fatalf("expected nil standing for a phase with no ladder, got %+v", phase.PhasePresentation)
	}
}

func TestHistoricalStandingIsExplicitlyNonCanonical(t *testing.T) {
	phases := phasesFromTerminalRun(&runspb.RunInfo{Phases: []*runspb.PhaseInfo{{
		Name: "structure",
		MaturityStanding: &runspb.PhaseMaturityStanding{
			Provider: "structure-health",
			Phase:    "structure",
		},
	}}})
	if len(phases) != 1 || phases[0].PhasePresentation != nil || phases[0].PresentationState != "legacy_maturity_standing" {
		t.Fatalf("historical standing must remain explicit non-canonical evidence: %+v", phases)
	}
}

func TestTopPriorityFromPhasesSelectsLowestRungHighestUnlock(t *testing.T) {
	priority := TopPriorityFromPhases([]Phase{
		{
			Name: "unit",
			PhasePresentation: &commonv1.PhasePresentation{
				Phase:                "unit",
				Provider:             "unit-health",
				CurrentLevel:         "L3",
				NextLevel:            "L4",
				NextAction:           "Harden unit coverage.",
				BlockingFindingCodes: []string{"unit.coverage_gap"},
				Capabilities: []*commonv1.PhaseCapabilityPresentation{{
					Id:           "coverage",
					PriorityRank: 1,
				}},
				DocumentationTopics: []string{"unit maturity next move"},
			},
		},
		{
			Name: "architecture",
			PhasePresentation: &commonv1.PhasePresentation{
				Phase:                "architecture",
				Provider:             "cli-health",
				CurrentLevel:         "L2",
				NextLevel:            "L3",
				NextAction:           "Prove command primitives.",
				BlockingFindingCodes: []string{"arch.primitive_unverified"},
				Capabilities: []*commonv1.PhaseCapabilityPresentation{{
					Id:           "command_architecture",
					PriorityRank: 2,
				}},
				DocumentationTopics: []string{"architecture maturity next move"},
			},
		},
	})
	if priority == nil {
		t.Fatal("expected a priority")
	}
	if priority.Phase != "architecture" || priority.NextMove != "Prove command primitives." {
		t.Fatalf("priority = %+v, want architecture lowest-rung move", priority)
	}
}

func TestTopPriorityFromPhasesDeterministicTieAndCeiling(t *testing.T) {
	priority := TopPriorityFromPhases([]Phase{
		{Name: "zeta", PhasePresentation: &commonv1.PhasePresentation{Phase: "zeta", CurrentLevel: "L2", NextLevel: "L3", NextAction: "Fix zeta."}},
		{Name: "alpha", PhasePresentation: &commonv1.PhasePresentation{Phase: "alpha", CurrentLevel: "L2", NextLevel: "L3", NextAction: "Fix alpha."}},
	})
	if priority == nil || priority.Phase != "alpha" {
		t.Fatalf("tie should resolve by phase name, got %+v", priority)
	}
	if got := TopPriorityFromPhases([]Phase{{Name: "unit", PhasePresentation: &commonv1.PhasePresentation{Phase: "unit", CurrentLevel: "L4", AtMaximum: true}}}); got != nil {
		t.Fatalf("all-at-ceiling should produce no priority, got %+v", got)
	}
}
