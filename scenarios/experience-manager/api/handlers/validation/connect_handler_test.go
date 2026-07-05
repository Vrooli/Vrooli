package validation

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"experience-manager/internal/spec"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeEngine struct {
	report spec.Report
	err    error
}

func (e fakeEngine) ValidateScenario(context.Context, string, string) (spec.Report, error) {
	return e.report, e.err
}

func TestSharedValidateScenarioUsesParserReport(t *testing.T) {
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
	}}})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" {
		t.Fatalf("scenario = %q", resp.Msg.GetScenario())
	}
	if resp.Msg.GetAssessment() == nil {
		t.Fatal("expected maturity assessment")
	}
}

func TestSharedValidateScenarioHonorsExperienceGate(t *testing.T) {
	report := spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []spec.Finding{{
			Code:      spec.CodeSchemaInvalid,
			Severity:  spec.SeverityError,
			Message:   "bad schema",
			Locations: []string{"experience/index.json"},
		}},
	}
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: report}})

	t.Setenv("EXPERIENCE_ALIGNMENT_GATE", "")
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario advisory: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("advisory status = %s, want PASSED", got)
	}

	t.Setenv("EXPERIENCE_ALIGNMENT_GATE", "strict")
	resp, err = h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario strict: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("strict status = %s, want FAILED", got)
	}
}

func TestNativeValidateScenarioReturnsStatusFromFindings(t *testing.T) {
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []spec.Finding{{
			Code:      spec.CodeSchemaInvalid,
			Severity:  spec.SeverityError,
			Message:   "bad schema",
			Locations: []string{"experience/index.json"},
		}},
	}}})
	resp, err := h.validateNative(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("validateNative: %v", err)
	}
	if resp.GetStatus() != "FAILED" {
		t.Fatalf("status = %q, want FAILED", resp.GetStatus())
	}
	if resp.GetReport().GetFindings()[0].GetCode() != spec.CodeSchemaInvalid {
		t.Fatalf("finding = %+v", resp.GetReport().GetFindings()[0])
	}
}

func TestFixRPCsReturnHonestUnimplemented(t *testing.T) {
	h := NewConnectHandler(Deps{})
	for name, call := range map[string]func() error{
		"preview": func() error {
			_, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{}))
			return err
		},
		"apply": func() error {
			_, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{}))
			return err
		},
	} {
		if err := call(); connect.CodeOf(err) != connect.CodeUnimplemented {
			t.Fatalf("%s code = %s, err=%v", name, connect.CodeOf(err), err)
		}
	}
}
