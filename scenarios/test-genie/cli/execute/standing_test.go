package execute

import (
	"encoding/json"
	"strings"
	"testing"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func sampleEvent() *runspb.RunEvent {
	return &runspb.RunEvent{
		Event:  "phase_completed",
		Phase:  "contracts",
		Status: "passed",
		MaturityStanding: &runspb.PhaseMaturityStanding{
			Provider:                "cli-health",
			Phase:                   "contracts",
			CurrentLevel:            "L3",
			CurrentLevelLabel:       "Ready",
			NextLevel:               "L4",
			CeilingLevel:            "L4",
			NorthStar:               "Verified renderer-separated primitives.",
			NextMove:                "Prove each declared primitive with cli-core evidence.",
			NextMoveReason:          "highest-unlock capability gap",
			PriorityCapabilityId:    "command_architecture",
			PriorityCapabilityLabel: "Command Architecture",
			BlockingFindingCodes:    []string{"arch.primitive_unverified"},
			Capabilities: []*runspb.PhaseCapabilityStanding{
				{Id: "command_architecture", Label: "Command Architecture", CurrentLevel: "L3", NextLevel: "L4"},
			},
		},
		FindingsSummary: &runspb.PhaseFindingsSummary{Warnings: 1, Total: 1},
	}
}

// TestPhaseAndJSONDeriveOneStanding proves the human and --json paths derive an
// identical standing from a single server payload: phaseFromEvent maps the proto
// standing once (StandingFromProto), the --json Response marshals exactly that
// object, and the human scorecard reads the same Phase.MaturityStanding field.
func TestPhaseAndJSONDeriveOneStanding(t *testing.T) {
	ev := sampleEvent()
	phase := phaseFromEvent(ev)

	if phase.MaturityStanding == nil {
		t.Fatal("phaseFromEvent dropped the maturity standing")
	}
	// The single mapping used by both output modes.
	want := StandingFromProto(ev.GetMaturityStanding())
	got, err := json.Marshal(phase.MaturityStanding)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
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
	if !strings.Contains(string(blob), `"currentLevel":"L3"`) || !strings.Contains(string(blob), `"northStar":"Verified renderer-separated primitives."`) {
		t.Fatalf("--json response missing the standing: %s", blob)
	}
	if phase.FindingsSummary == nil || phase.FindingsSummary.Total != 1 {
		t.Fatalf("findings summary not carried: %+v", phase.FindingsSummary)
	}
}

func TestStandingCarriesDocSearchTopics(t *testing.T) {
	st := StandingFromProto(sampleEvent().GetMaturityStanding())
	if len(st.DocSearchTopics) == 0 {
		t.Fatal("expected runnable doc-search topics for a non-max phase")
	}
	joined := strings.Join(st.DocSearchTopics, "|")
	if !strings.Contains(joined, "contracts") || !strings.Contains(joined, "arch.primitive_unverified") {
		t.Fatalf("doc topics should reference the phase + blocking codes: %v", st.DocSearchTopics)
	}
}

func TestStandingAtMaximumSuppressesTopics(t *testing.T) {
	ev := sampleEvent()
	ev.MaturityStanding.AtMaximum = true
	ev.MaturityStanding.NextLevel = ""
	ev.MaturityStanding.BlockingFindingCodes = nil
	st := StandingFromProto(ev.GetMaturityStanding())
	if len(st.DocSearchTopics) != 0 {
		t.Fatalf("no doc topics expected at maximum maturity, got %v", st.DocSearchTopics)
	}
}

func TestStandingNilWhenAbsent(t *testing.T) {
	phase := phaseFromEvent(&runspb.RunEvent{Event: "phase_completed", Phase: "native", Status: "passed"})
	if phase.MaturityStanding != nil {
		t.Fatalf("expected nil standing for a phase with no ladder, got %+v", phase.MaturityStanding)
	}
}
