package skillsetvalidation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestValidateScenarioFailureCarriesTypedErrorFindings(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "scenarios", "demo")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{"skills":{"improve":{"source":"skills/improve/SKILL.md"}}}`)
	writeFile(t, filepath.Join(root, "cli", "manifest.json"), `{}`)

	resp, err := (&handler{repoRoot: repo}).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status = %v, want FAILED", resp.Msg.GetStatus())
	}
	assessment := resp.Msg.GetAssessment()
	findings := assessment.GetFindings()
	if got := assessment.GetFindingsBySeverity()["SEVERITY_ERROR"]; got != int32(len(findings)) || got != 2 {
		t.Fatalf("SEVERITY_ERROR count = %d with %d findings, want 2 matching", got, len(findings))
	}
	codes := map[string]string{}
	for _, finding := range findings {
		if finding.GetSeverity() != "SEVERITY_ERROR" || finding.GetMessage() == "" || finding.GetRemediation() == "" {
			t.Fatalf("finding %+v lacks severity, message or remediation", finding)
		}
		codes[finding.GetCode()] = finding.GetLocation()
	}
	if codes["skill-set.role_missing"] != ".vrooli/service.json skills.usage" {
		t.Fatalf("usage role_missing location = %q", codes["skill-set.role_missing"])
	}
	if codes["skill-set.source_missing"] != "skills/improve/SKILL.md" {
		t.Fatalf("improve source_missing location = %q", codes["skill-set.source_missing"])
	}
}

func TestValidateScenarioCleanHasNoFindings(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "scenarios", "demo")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{}`)

	resp, err := (&handler{repoRoot: repo}).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED || len(resp.Msg.GetAssessment().GetFindings()) != 0 {
		t.Fatalf("clean scenario: status %v findings %d", resp.Msg.GetStatus(), len(resp.Msg.GetAssessment().GetFindings()))
	}
}

func TestValidateScenarioHonoursExplicitScenarioPath(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "isolated", "demo")
	writeFile(t, filepath.Join(workspace, ".vrooli", "service.json"), `{}`)

	// The handler's own repository has no such scenario; only the explicit path does.
	resp, err := (&handler{repoRoot: t.TempDir()}).ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", Path: workspace}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status = %v with findings %v, want PASSED from the explicit path", resp.Msg.GetStatus(), resp.Msg.GetAssessment().GetLocal().GetBlockingFindingCodes())
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
