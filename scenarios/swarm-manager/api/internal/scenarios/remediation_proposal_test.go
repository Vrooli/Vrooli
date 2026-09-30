package scenarios

import (
	"strings"
	"testing"
)

func TestBuildPhaseRemediationProposalIsPureAndGherkinValid(t *testing.T) {
	snapshot := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, SourceRunID: "run-1", Phases: []ScenarioHealthPhase{{Phase: "unit", CurrentRung: "L1", NextRung: "L2", PriorityCapabilityID: "coverage", PriorityCapabilityLabel: "Coverage"}}}
	proposal, err := BuildPhaseRemediationProposal(snapshot, RemediationTarget{Scenario: "swarm-manager", ProviderPhase: "unit", CapabilityID: "coverage"}, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Fingerprint == "" || !strings.HasPrefix(proposal.AcceptanceCriteria[0], "Given ") {
		t.Fatalf("proposal = %#v", proposal)
	}
	if strings.Contains(proposal.Description, "run-1") {
		t.Fatal("transient run id leaked into generated work")
	}
}

func TestBuildPhaseRemediationProposalRejectsDegradedOrMismatchedEvidence(t *testing.T) {
	_, err := BuildPhaseRemediationProposal(ScenarioHealthSnapshot{EvidenceState: HealthEvidenceDegraded}, RemediationTarget{Scenario: "s", ProviderPhase: "unit", CapabilityID: "coverage"}, "manual")
	if err == nil {
		t.Fatal("expected degraded evidence rejection")
	}
}

func TestBuildMaturityCampaignProposalRequiresFreshEvidenceAndDeclaresWorkflow(t *testing.T) {
	proposal, err := BuildMaturityCampaignProposalForTarget(ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}, MaturityCampaignTarget{Scenario: "swarm-manager", Target: "suite-green", ProviderPhases: []string{"unit"}})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.DeclaredWorkflow != "scenario-improvement-campaign" || !strings.HasPrefix(proposal.AcceptanceCriteria[0], "Given ") {
		t.Fatalf("proposal = %#v", proposal)
	}
}

func TestBuildMaturityCampaignProposalRequiresPresentedPhaseScope(t *testing.T) {
	_, err := BuildMaturityCampaignProposalForTarget(ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, Phases: []ScenarioHealthPhase{{Phase: "unit"}}}, MaturityCampaignTarget{Scenario: "swarm-manager", Target: "suite-green", ProviderPhases: []string{"missing"}})
	if err == nil {
		t.Fatal("expected campaign target with absent provider phase to be rejected")
	}
}
