package validationstub

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestValidationStubKeepsProviderIdentityAndContract(t *testing.T) {
	h := &handler{provider: "test-provider"}
	got, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "example"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetScenario() != "example" || got.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("response identity/status = %q/%v", got.Msg.GetScenario(), got.Msg.GetStatus())
	}
	assessment := got.Msg.GetAssessment()
	if assessment.GetScenario() != "example" || assessment.GetProvider() != "test-provider" || assessment.GetPhase() != "test-provider" || assessment.GetLocal().GetCurrentLevel() != "L2" {
		t.Fatalf("assessment = %#v", assessment)
	}
	if got.Msg.GetMetrics().GetStartedAt() == nil || got.Msg.GetMetrics().GetCompletedAt() == nil {
		t.Fatal("metrics timestamps must be populated")
	}

	target, err := h.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{Target: &commonv1.ValidationTarget{Id: "target-id"}}))
	if err != nil {
		t.Fatal(err)
	}
	if target.Msg.GetTarget().GetId() != "target-id" || target.Msg.GetAssessment().GetScenario() != "target-id" {
		t.Fatalf("target response failed to preserve target identity: %#v", target.Msg)
	}
	if target.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED || target.Msg.GetMetrics().GetCompletedAt() == nil {
		t.Fatalf("target response contract incomplete: %#v", target.Msg)
	}
}
