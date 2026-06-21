package validation

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"performance-health/internal/autofix"
	"performance-health/internal/readiness"

	"github.com/vrooli/maturity-go/assessment"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// [REQ:PH-VALIDATION-002] The shared ScenarioValidationService delegates to the
// readiness engine, packs the native readiness response into native_detail, and
// returns a valid shared status.
func TestSharedValidateScenarioPacksNativeDetail(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot(t))
	if err != nil {
		t.Skipf("maturity spec unavailable in this checkout: %v", err)
	}
	h := NewHandlerWithDeps(Deps{
		Readiness:    readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}),
		Autofix:      autofix.NewService(),
		MaturitySpec: spec,
	})
	shared := NewSharedHandler(h)
	resp, err := shared.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		t.Fatal("shared response must carry a concrete status")
	}
	if resp.Msg.GetNativeDetail() == nil {
		t.Fatal("shared response must pack the native readiness detail")
	}
	native := &readinessv1.ValidateReadinessResponse{}
	if err := resp.Msg.GetNativeDetail().UnmarshalTo(native); err != nil {
		t.Fatalf("native_detail should unmarshal to ValidateReadinessResponse: %v", err)
	}
	if native.GetTier() != readinessv1.CaptureTier_CAPTURE_TIER_1 {
		t.Fatalf("native detail tier = %v, want Tier1", native.GetTier())
	}
}

// scenarioRoot returns the repo root containing scenarios/performance-health so
// the maturity spec loads in-checkout.
func scenarioRoot(t *testing.T) string {
	t.Helper()
	// handlers/validation -> api -> performance-health
	return "../.."
}
