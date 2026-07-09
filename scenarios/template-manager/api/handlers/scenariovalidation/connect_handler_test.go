package scenariovalidation

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatevalidation"

	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeValidator struct {
	report templatevalidation.Report
}

func (f fakeValidator) ValidateScenario(context.Context, string, string) (templatevalidation.Report, error) {
	return f.report, nil
}

type fakeFixer struct{}

func (fakeFixer) Preview(string, []string) ([]autofix.Candidate, error) {
	return []autofix.Candidate{{RuleID: templatevalidation.CodeProvenanceMissing, FilePath: ".vrooli/service.json", Description: "stamp"}}, nil
}
func (fakeFixer) Apply(string, []string) ([]autofix.Candidate, error) { return nil, nil }

func TestValidateScenarioBuildsSharedAssessmentIdentity(t *testing.T) {
	handler := NewConnectHandler(Deps{
		Validator: fakeValidator{report: templatevalidation.Report{
			Scenario: "legacy",
			RootPath: t.TempDir(),
			Findings: []templatevalidation.Finding{{
				Code:     templatevalidation.CodeProvenanceMissing,
				Severity: templatevalidation.SeverityError,
				Autofix:  true,
			}},
		}},
		MaturitySpec: templatevalidation.MaturitySpec(),
	})
	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "legacy"}))
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	if err := assessment.RequireIdentity(templatevalidation.Provider, templatevalidation.Phase, resp.Msg.GetAssessment()); err != nil {
		t.Fatalf("assessment identity: %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status = %v, want failed", resp.Msg.GetStatus())
	}
	if resp.Msg.GetAssessment().GetAutofixableCount() != 1 {
		t.Fatalf("autofixable count = %d, want 1", resp.Msg.GetAssessment().GetAutofixableCount())
	}
}

func TestPreviewFixReturnsCandidates(t *testing.T) {
	handler := NewConnectHandler(Deps{
		Validator:    fakeValidator{report: templatevalidation.Report{Scenario: "legacy", RootPath: t.TempDir()}},
		Fixers:       fakeFixer{},
		MaturitySpec: templatevalidation.MaturitySpec(),
	})
	resp, err := handler.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "legacy"}))
	if err != nil {
		t.Fatalf("PreviewFix() error = %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatal("preview should not be applied")
	}
	if len(resp.Msg.GetCandidates()) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(resp.Msg.GetCandidates()))
	}
}
