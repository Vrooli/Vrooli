package validation

import (
	"context"
	"errors"
	"log"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/maturity-go/assessment"
	"security-health/internal/validation"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type stubValidator struct {
	report validation.Report
	err    error
}

func (s stubValidator) ValidateScenario(context.Context, string) (validation.Report, error) {
	return s.report, s.err
}

func TestValidateScenario_MapsReport(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: stubValidator{report: validation.Report{
		Scenario: "demo",
		Passed:   false,
		Findings: []validation.Finding{{
			RuleID:      "gitleaks.aws",
			Severity:    validation.SeverityError,
			Title:       "secret",
			Remediation: "rotate",
			FilePath:    "leak.go:2",
			Scanner:     "gitleaks",
		}},
		Summary:         validation.Summary{Errors: 1},
		SkippedScanners: []string{"osv-scanner"},
	}}, MaturitySpec: testMaturitySpec()})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" || msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Errorf("unexpected scenario/status: %v / %v", msg.GetScenario(), msg.GetStatus())
	}
	ma := msg.GetAssessment()
	if ma.GetProvider() != "security-health" || ma.GetPhase() != "security" {
		t.Fatalf("assessment identity wrong: %+v", ma)
	}
	if len(ma.GetFindings()) != 1 || ma.GetFindings()[0].GetSeverity() != "FINDING_SEVERITY_ERROR" {
		t.Fatalf("finding mapping wrong: %+v", ma.GetFindings())
	}
	if ma.GetFindings()[0].GetCode() != "gitleaks.aws" || ma.GetFindings()[0].GetLocation() != "leak.go:2" {
		t.Errorf("field mapping wrong: %+v", ma.GetFindings()[0])
	}
	if ma.GetFindingsBySeverity()["FINDING_SEVERITY_ERROR"] != 1 {
		t.Errorf("summary errors = %d, want 1", ma.GetFindingsBySeverity()["FINDING_SEVERITY_ERROR"])
	}
	if ma.GetLocal().GetCurrentLevel() != "L1" || ma.GetLocal().GetNextLevel() != "L2" {
		t.Fatalf("assessment local maturity wrong: %+v", ma.GetLocal())
	}
	if got := ma.GetFindingsByGlobalImpact()["safety_blocker"]; got != 1 {
		t.Fatalf("global impact count = %d, want 1", got)
	}
	if ma.GetFindings()[0].GetMaturity().GetGlobalImpact() != commonv1.GlobalImpact_GLOBAL_IMPACT_SAFETY_BLOCKER {
		t.Fatalf("finding maturity impact wrong: %+v", ma.GetFindings()[0].GetMaturity())
	}
}

func TestValidateScenario_ErrorIsInvalidArgument(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: stubValidator{err: errors.New("nope")}})
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "x"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestValidateScenario_NoValidatorUnimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "x"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("want Unimplemented, got %v", connect.CodeOf(err))
	}
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger:       log.New(log.Writer(), "", 0),
		Validator:    stubValidator{report: validation.Report{Scenario: "security-health", Passed: true}},
		MaturitySpec: testMaturitySpec(),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "security-health"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}

func testMaturitySpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "security-health",
		Phase:    "security",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "target readable"},
			{ID: "L1", Name: "substrates classified"},
			{ID: "L2", Name: "safety blockers absent"},
		},
		Findings: map[string]assessment.FindingMapping{},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L2",
			GlobalImpact:     assessment.ImpactSafetyBlocker,
			Dimension:        "security",
			SeverityDefault:  "SEVERITY_ERROR",
		},
	}
}
