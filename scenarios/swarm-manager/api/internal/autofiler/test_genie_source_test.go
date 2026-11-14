package autofiler

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/scenarios"
)

type staticHealthSource struct {
	snapshot scenarios.ScenarioHealthSnapshot
}

func (s staticHealthSource) Snapshot(context.Context, string) scenarios.ScenarioHealthSnapshot {
	return s.snapshot
}

func TestTestGenieFindingSourceUsesSharedPhaseIdentity(t *testing.T) {
	snapshot := scenarios.ScenarioHealthSnapshot{EvidenceState: scenarios.HealthEvidenceFresh, Phases: []scenarios.ScenarioHealthPhase{{Phase: "unit", PriorityCapabilityID: "coverage", PriorityCapabilityLabel: "Coverage", CurrentRung: "L1", NextRung: "L2"}}}
	found, err := (TestGenieFindingSource{Health: staticHealthSource{snapshot}}).Findings(context.Background(), Target{Scenario: "demo"})
	if err != nil || len(found) != 1 {
		t.Fatalf("Findings = %#v, %v", found, err)
	}
	manual, err := scenarios.BuildPhaseRemediationProposal(snapshot, scenarios.RemediationTarget{Scenario: "demo", ProviderPhase: "unit", CapabilityID: "coverage"}, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if found[0].ID != manual.Fingerprint || stableItemName(found[0]) != scenarios.RemediationItemName(manual.Fingerprint) {
		t.Fatalf("auto-filer identity differs from manual proposal: %#v / %#v", found[0], manual)
	}
	item := itemForFinding(found[0], FileOptions{}, time.Time{})
	if item.Title != manual.Title || item.Description != manual.Description+"\n\nAcceptance:\n- "+manual.AcceptanceCriteria[0] {
		t.Fatalf("auto-filer work shape differs: %#v", item)
	}
}

func TestTestGenieFindingSourceDoesNotSuggestStaleEvidence(t *testing.T) {
	found, err := (TestGenieFindingSource{Health: staticHealthSource{scenarios.ScenarioHealthSnapshot{EvidenceState: scenarios.HealthEvidenceStale}}}).Findings(context.Background(), Target{Scenario: "demo"})
	if err == nil || found != nil {
		t.Fatalf("stale evidence must not become a suggestion: %#v, %v", found, err)
	}
}
