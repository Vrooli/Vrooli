package validationprovider

import (
	"testing"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

func buildStandingClone(t *testing.T, raw []byte) *runspb.PhaseMaturityStanding {
	t.Helper()
	out := &runspb.PhaseMaturityStanding{}
	if err := proto.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func sampleAssessment() *commonv1.MaturityAssessment {
	levels := []*commonv1.LocalMaturityLevel{
		{Id: "L0", Name: "Blocked", StatusLabel: "Unavailable", CapabilitySummary: "No inspectable surface.", NextUnlock: "A runnable shell."},
		{Id: "L3", Name: "Declared, not yet verified", StatusLabel: "Ready", CapabilitySummary: "Declared but unverified.", NextUnlock: "Prove the declaration."},
		{Id: "L4", Name: "Verified primitives", StatusLabel: "Complete", CapabilitySummary: "Every command is a verified renderer-separated primitive — the North Star.", NextUnlock: ""},
	}
	return &commonv1.MaturityAssessment{
		Scenario: "cli-health",
		Provider: "cli-health",
		Phase:    "contracts",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel:         "L3",
			NextLevel:            "L4",
			Levels:               levels,
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			Clean:                false,
			UnknownCount:         1,
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{
			{
				Id:                   "command_architecture",
				Label:                "Command Architecture",
				CurrentLevel:         "L3",
				NextLevel:            "L4",
				Levels:               levels,
				CurrentSummary:       "Declared but unverified.",
				NextUnlock:           "Prove each declared primitive with cli-core evidence.",
				BlockingFindingCodes: []string{"arch.primitive_unverified"},
				PriorityRank:         1,
				PriorityReason:       "highest-unlock capability gap",
			},
		},
		HighestPriorityCapability: &commonv1.PriorityFocus{
			CapabilityId:    "command_architecture",
			CapabilityLabel: "Command Architecture",
			CurrentLevel:    "L3",
			NextLevel:       "L4",
			Reason:          "highest-unlock capability gap",
		},
	}
}

func TestBuildStandingProjectsProviderAssessment(t *testing.T) {
	provider := Provider{Phase: "contracts", ProviderScenario: "cli-health"}
	st := buildStanding(provider, sampleAssessment())
	if st == nil {
		t.Fatal("expected a standing for an assessment with a local ladder")
	}
	if st.GetProvider() != "cli-health" || st.GetPhase() != "contracts" {
		t.Fatalf("identity not carried: provider=%q phase=%q", st.GetProvider(), st.GetPhase())
	}
	if st.GetCurrentLevel() != "L3" || st.GetNextLevel() != "L4" {
		t.Fatalf("rung mismatch: current=%q next=%q", st.GetCurrentLevel(), st.GetNextLevel())
	}
	if st.GetCeilingLevel() != "L4" {
		t.Fatalf("ceiling should be the top rung, got %q", st.GetCeilingLevel())
	}
	if st.GetCurrentLevelLabel() != "Ready" {
		t.Fatalf("current level label should be the status label, got %q", st.GetCurrentLevelLabel())
	}
	if st.GetNorthStar() == "" || st.GetNorthStar() != "Every command is a verified renderer-separated primitive — the North Star." {
		t.Fatalf("north star should be the top rung aspiration, got %q", st.GetNorthStar())
	}
	if len(st.GetBlockingFindingCodes()) != 1 || st.GetBlockingFindingCodes()[0] != "arch.primitive_unverified" {
		t.Fatalf("blocking codes not carried: %v", st.GetBlockingFindingCodes())
	}
	// The single next move is the priority capability's next_unlock + its reason.
	if st.GetNextMove() != "Prove each declared primitive with cli-core evidence." {
		t.Fatalf("next move should be the priority capability's next_unlock, got %q", st.GetNextMove())
	}
	if st.GetNextMoveReason() != "highest-unlock capability gap" {
		t.Fatalf("next move reason mismatch: %q", st.GetNextMoveReason())
	}
	if st.GetPriorityCapabilityId() != "command_architecture" {
		t.Fatalf("priority capability id mismatch: %q", st.GetPriorityCapabilityId())
	}
	if st.GetAtMaximum() {
		t.Fatal("a non-clean L3 phase is not at maximum")
	}
	if len(st.GetCapabilities()) != 1 || st.GetCapabilities()[0].GetId() != "command_architecture" {
		t.Fatalf("capability depth not carried: %v", st.GetCapabilities())
	}
}

func TestBuildStandingAtMaximum(t *testing.T) {
	a := sampleAssessment()
	a.Local.CurrentLevel = "L4"
	a.Local.NextLevel = ""
	a.Local.Clean = true
	a.Local.BlockingFindingCodes = nil
	a.HighestPriorityCapability = nil
	a.Capabilities = nil
	st := buildStanding(Provider{Phase: "contracts", ProviderScenario: "cli-health"}, a)
	if st == nil {
		t.Fatal("expected a standing")
	}
	if !st.GetAtMaximum() {
		t.Fatal("a clean phase at the top rung with no next level should be at maximum")
	}
	// With no priority capability, the next move falls back to the current rung's
	// next_unlock — which is empty at the ceiling.
	if st.GetNextMove() != "" {
		t.Fatalf("no next move expected at maximum, got %q", st.GetNextMove())
	}
}

func TestBuildStandingNilWhenNoLadder(t *testing.T) {
	if got := buildStanding(Provider{Phase: "native"}, &commonv1.MaturityAssessment{}); got != nil {
		t.Fatalf("expected nil standing when the assessment declares no local ladder, got %+v", got)
	}
	if got := buildStanding(Provider{Phase: "native"}, nil); got != nil {
		t.Fatalf("expected nil standing for a nil assessment, got %+v", got)
	}
}

func TestPhaseMaturityStandingProtoRoundTrip(t *testing.T) {
	st := buildStanding(Provider{Phase: "contracts", ProviderScenario: "cli-health"}, sampleAssessment())
	if st == nil {
		t.Fatal("expected a standing")
	}
	raw, err := proto.Marshal(st)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := buildStandingClone(t, raw)
	if !proto.Equal(st, got) {
		t.Fatalf("proto round-trip lost fields:\n before=%+v\n after=%+v", st, got)
	}
	// The new fields survive the wire.
	if got.GetNorthStar() != st.GetNorthStar() || got.GetCeilingLevel() != "L4" || got.GetNextMove() == "" {
		t.Fatalf("new standing fields not stable on the wire: %+v", got)
	}
	if len(got.GetCapabilities()) != 1 {
		t.Fatalf("capability depth lost on the wire: %+v", got.GetCapabilities())
	}
}

func TestBuildFindingsSummaryTallies(t *testing.T) {
	fs := buildFindingsSummary(Summary{Blockers: 1, Errors: 2, Warnings: 3, Infos: 4})
	if fs.GetTotal() != 10 {
		t.Fatalf("total should sum severities, got %d", fs.GetTotal())
	}
	if fs.GetBlockers() != 1 || fs.GetErrors() != 2 || fs.GetWarnings() != 3 || fs.GetInfos() != 4 {
		t.Fatalf("severity tally mismatch: %+v", fs)
	}
}
