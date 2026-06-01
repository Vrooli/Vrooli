package validation

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"security-health/internal/validation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/validation"
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
	}}})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatal(err)
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" || msg.GetPassed() {
		t.Errorf("unexpected scenario/passed: %v / %v", msg.GetScenario(), msg.GetPassed())
	}
	if len(msg.Findings) != 1 || msg.Findings[0].GetSeverity() != validationv1.Severity_SEVERITY_ERROR {
		t.Fatalf("finding mapping wrong: %+v", msg.Findings)
	}
	if msg.Findings[0].GetRuleId() != "gitleaks.aws" || msg.Findings[0].GetFilePath() != "leak.go:2" {
		t.Errorf("field mapping wrong: %+v", msg.Findings[0])
	}
	if msg.GetSummary().GetErrors() != 1 {
		t.Errorf("summary errors = %d, want 1", msg.GetSummary().GetErrors())
	}
	if len(msg.GetSkippedScanners()) != 1 {
		t.Errorf("skipped scanners not propagated: %v", msg.GetSkippedScanners())
	}
}

func TestValidateScenario_ErrorIsInvalidArgument(t *testing.T) {
	h := NewConnectHandler(Deps{Validator: stubValidator{err: errors.New("nope")}})
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "x"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("want InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestValidateScenario_NoValidatorUnimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "x"}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("want Unimplemented, got %v", connect.CodeOf(err))
	}
}
