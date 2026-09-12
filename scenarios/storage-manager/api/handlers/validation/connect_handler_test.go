package validation

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"storage-manager/internal/validation"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// stubValidator returns a fixed report so the handler can be exercised without
// running real analyzers.
type stubValidator struct {
	report validation.Report
	err    error
}

func (s stubValidator) ValidateScenario(context.Context, string) (validation.Report, error) {
	return s.report, s.err
}

// loadRealSpec parses the scenario's own .vrooli/maturity.json so the handler
// resolves finding levels through the real catalog.
func loadRealSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("load descriptor maturity: %v", err)
	}
	return spec
}

func TestValidateScenario_CleanAssessment(t *testing.T) {
	h := NewConnectHandler(Deps{
		Validator:    stubValidator{report: validation.Report{Scenario: "demo", Language: "go"}},
		MaturitySpec: loadRealSpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario error = %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status = %v, want PASSED for clean report", resp.Msg.GetStatus())
	}
	a := resp.Msg.GetAssessment()
	if a == nil || a.GetProvider() != "storage-manager" || a.GetPhase() != "storage" {
		t.Fatalf("assessment provider/phase wrong: %+v", a)
	}
	if len(a.GetFindings()) != 0 {
		t.Fatalf("findings = %d, want 0", len(a.GetFindings()))
	}
	var detail structpb.Struct
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
		t.Fatalf("unmarshal native detail: %v", err)
	}
	if detail.Fields["file_persisting"].GetBoolValue() {
		t.Fatal("file_persisting = true, want false for a report without file engine")
	}
}

func TestValidateScenarioPublishesFilePersistenceClassification(t *testing.T) {
	h := NewConnectHandler(Deps{
		Validator:    stubValidator{report: validation.Report{Scenario: "demo", Language: "go", Engines: []validation.Engine{validation.EngineFile}}},
		MaturitySpec: loadRealSpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario error = %v", err)
	}
	var detail structpb.Struct
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
		t.Fatalf("unmarshal native detail: %v", err)
	}
	if !detail.Fields["file_persisting"].GetBoolValue() {
		t.Fatal("file_persisting = false, want true for file engine")
	}
}

func TestValidateScenario_BlockingFindingFails(t *testing.T) {
	h := NewConnectHandler(Deps{
		Validator: stubValidator{report: validation.Report{
			Scenario: "demo",
			Language: "go",
			Findings: []validation.Finding{{
				Code:     "ROUTED_SEAMS_UNWIRED",
				Severity: validation.SeverityError,
				Title:    "Routed isolation seams unwired",
				Message:  "destructive playbooks would hit the real database",
			}},
		}},
		MaturitySpec: loadRealSpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario error = %v", err)
	}
	if resp.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status = %v, want FAILED for ERROR finding", resp.Msg.GetStatus())
	}
	a := resp.Msg.GetAssessment()
	if a.GetLocal().GetCurrentLevel() != "L1" {
		t.Fatalf("current level = %q, want L1 (blocked at L2 by ROUTED_SEAMS_UNWIRED)", a.GetLocal().GetCurrentLevel())
	}
}
