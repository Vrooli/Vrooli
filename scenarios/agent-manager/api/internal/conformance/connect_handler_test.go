package conformance

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestBuildAssessmentRepresentsFullLadderAndBlockingFindings(t *testing.T) {
	clean := buildAssessment(Report{Scenario: "consumer"})
	if clean.GetLocal().GetCurrentLevel() != "L4" || !clean.GetLocal().GetClean() || clean.GetLocal().GetNextLevel() != "" {
		t.Fatalf("clean assessment = %#v, want clean L4", clean.GetLocal())
	}

	assessment := buildAssessment(Report{Scenario: "consumer", Findings: []Finding{
		{Code: CodeRoleUnresolved, Severity: "SEVERITY_ERROR"},
		{Code: CodeDirectSpawnBypass, Severity: "SEVERITY_ERROR"},
	}})
	if assessment.GetLocal().GetCurrentLevel() != "L2" || assessment.GetLocal().GetNextLevel() != "L3" {
		t.Fatalf("assessment level = %#v, want L2 -> L3", assessment.GetLocal())
	}
	if got := assessment.GetFindingsBySeverity(); got["SEVERITY_ERROR"] != 2 {
		t.Fatalf("severity counts = %#v", got)
	}
	if got := assessment.GetLocal().GetBlockingFindingCodes(); len(got) != 2 || got[0] != CodeRoleUnresolved || got[1] != CodeDirectSpawnBypass {
		t.Fatalf("blocking findings = %#v", got)
	}
	if got := assessment.GetFindings()[0].GetMaturity().GetGlobalImpact(); got != commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP {
		t.Fatalf("role finding global impact = %s, want evolvability gap", got)
	}
	if got := globalImpactFor(Finding{Code: CodePermissionPosture}); got != commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER {
		t.Fatalf("permission posture global impact = %s, want safety blocker", got)
	}
	if got := assessment.GetPresentation(); got == nil || got.GetContractVersion() != "v1" || got.GetProvider() != "agent-manager" || got.GetPhase() != "agent-conformance" {
		t.Fatalf("presentation = %#v, want contract-valid agent-conformance presentation", got)
	}
	if got := assessment.GetPresentation().GetBlockingFindingCodes(); len(got) != 2 || got[0] != CodeDirectSpawnBypass || got[1] != CodeRoleUnresolved {
		t.Fatalf("presentation blocking findings = %#v", got)
	}
}

func TestValidateScenarioIncludesExecutionMetrics(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	response, err := NewHandler(repoRoot).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "agent-inbox"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetMetrics() == nil || response.Msg.GetMetrics().GetCompletedAt() == nil || response.Msg.GetMetrics().GetStartedAt() == nil {
		t.Fatalf("metrics = %#v, want measured validation timing", response.Msg.GetMetrics())
	}
}
