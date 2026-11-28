package monetization

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStripeOutcomePhaseWithoutExecutionIsNotEvaluated(t *testing.T) {
	handler := Handler{}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "landing-page-business-suite", Phase: "stripe-test-outcomes", IncludeExecution: false,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	if got := response.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED {
		t.Fatalf("status = %s, want NOT_EVALUATED", got)
	}
	if response.Msg.GetNativeDetail() == nil {
		t.Fatal("phase omitted native detail for the not-evaluated report")
	}
	assessment := response.Msg.GetAssessment()
	if assessment.GetPhase() != "stripe-test-outcomes" || assessment.GetLocal().GetCurrentLevel() != "L0" || assessment.GetLocal().GetClean() {
		t.Fatalf("assessment = %+v, want stripe phase at L0 with a blocking finding", assessment)
	}
}

func TestStripeOutcomeAssessmentOnlyAdvancesObservedLevels(t *testing.T) {
	assessment := buildStripeOutcomeAssessment("landing-page-business-suite", &StripeOutcomeReport{
		Catalog:  CatalogReplay{Status: "passed", DonationPrice: "verified"},
		Outcomes: []OutcomeObservation{{Name: "real_webhook_delivery", Status: "passed"}},
	})
	if assessment.GetPhase() != "stripe-test-outcomes" || assessment.GetLocal().GetCurrentLevel() != "L2" || assessment.GetLocal().GetClean() {
		t.Fatalf("assessment = %+v, want observed L2 with remaining lifecycle blocker", assessment)
	}
}

func TestStripeOutcomePhaseReportsExecutorEvidence(t *testing.T) {
	want := &StripeOutcomeReport{
		Status: "not_evaluated", HostEnvironment: "local", StripeMode: "test", LocalStandIn: true,
		Catalog:  CatalogReplay{Status: "passed", VerifiedPrices: 11, DonationPrice: "verified"},
		Outcomes: []OutcomeObservation{{Name: "checkout_success", Status: "unknown"}},
	}
	handler := Handler{outcomeRunner: func(context.Context) (*StripeOutcomeReport, error) { return want, nil }}
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "landing-page-business-suite", Phase: "stripe-test-outcomes", IncludeExecution: true,
	}))
	if err != nil {
		t.Fatalf("ValidateScenario() error = %v", err)
	}
	if got := response.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED {
		t.Fatalf("status = %s, want NOT_EVALUATED while outcomes are unknown", got)
	}
	var decoded structpb.Struct
	if err := response.Msg.GetNativeDetail().UnmarshalTo(&decoded); err != nil {
		t.Fatalf("unmarshal native detail: %v", err)
	}
	stripe := decoded.AsMap()["stripe_test_outcomes"].(map[string]any)
	catalog := stripe["catalog"].(map[string]any)
	if got := catalog["verified_prices"]; got != float64(11) {
		t.Fatalf("verified_prices = %v, want 11", got)
	}
	if got := stripe["local_stand_in"]; got != true {
		t.Fatalf("local_stand_in = %v, want true", got)
	}
}

func TestDescribeStripeOutcomePhaseDeclaresExecutionSupport(t *testing.T) {
	response, err := (Handler{}).DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{Phase: "stripe-test-outcomes"}))
	if err != nil {
		t.Fatalf("DescribeProvider() error = %v", err)
	}
	if response.Msg.GetPhase() != "stripe-test-outcomes" || !response.Msg.GetCapabilities().GetSupportsExecution() {
		t.Fatalf("phase capabilities = %+v, want stripe-test-outcomes with execution", response.Msg)
	}
}
