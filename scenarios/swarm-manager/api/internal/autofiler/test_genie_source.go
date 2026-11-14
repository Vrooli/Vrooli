package autofiler

import (
	"context"
	"fmt"
	"strings"

	"swarm-manager/internal/scenarios"
)

// TestGenieFindingSource projects one provider-priority phase target through
// the same factory used by the operator path. It never reads raw findings or
// creates broad campaigns.
type TestGenieFindingSource struct{ Health scenarios.HealthSource }

func (s TestGenieFindingSource) Findings(ctx context.Context, target Target) ([]Finding, error) {
	if s.Health == nil {
		return nil, fmt.Errorf("Test Genie health source is not configured")
	}
	snapshot := s.Health.Snapshot(ctx, target.Scenario)
	if !snapshot.IsActionable() {
		return nil, fmt.Errorf("Test Genie evidence is %s", snapshot.EvidenceState)
	}
	for _, phase := range snapshot.Phases {
		if strings.TrimSpace(phase.PriorityCapabilityID) == "" {
			continue
		}
		proposal, err := scenarios.BuildPhaseRemediationProposal(snapshot, scenarios.RemediationTarget{Scenario: target.Scenario, ProviderPhase: phase.Phase, CapabilityID: phase.PriorityCapabilityID}, "auto-filer")
		if err != nil {
			continue
		}
		return []Finding{{ID: proposal.Fingerprint, Scenario: target.Scenario, Dimension: phase.Phase, Severity: SeverityYellow, Title: proposal.Title, Description: proposal.Description, Details: "Acceptance:\n- " + strings.Join(proposal.AcceptanceCriteria, "\n- "), RecommendedSkillIDs: proposal.RecommendedWorkflows}}, nil
	}
	return []Finding{}, nil
}
