package validation

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"performance-health/internal/autofix"
	"performance-health/internal/readiness"

	"github.com/vrooli/maturity-go/assessment"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
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

type fakeWorkloads struct {
	readings []*sweepv1.WorkloadReading
	err      error
}

func (f fakeWorkloads) ReadAll(context.Context, string) ([]*sweepv1.WorkloadReading, error) {
	return f.readings, f.err
}

func TestWorkloadGateKeepsEvidenceAndCannotPassUnknownOrFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		outcome sweepv1.WorkloadOutcome
		within  bool
		readErr error
		want    scenariovalidationv1.ValidationStatus
	}{
		{"qualified", sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED, true, nil, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED},
		{"breach", sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_MEASURED, false, nil, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED},
		{"failed", sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_FAILED, false, nil, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED},
		{"stale", sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNAVAILABLE, false, nil, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED},
		{"read-error", sweepv1.WorkloadOutcome_WORKLOAD_OUTCOME_UNSPECIFIED, false, errors.New("owner unavailable"), scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := fakeWorkloads{readings: []*sweepv1.WorkloadReading{{Scenario: "demo", Workload: "capture", OperationId: "retained-owner-operation", Outcome: tc.outcome, WithinBudget: tc.within, Reason: tc.name, ReceiptPath: "owner/receipt.json"}}, err: tc.readErr}
			// No maturity spec: loss of optional classification metadata must not
			// turn an actual failed workload or missing receipt into a pass.
			h := NewHandlerWithDeps(Deps{Readiness: readiness.NewService(fakeFacts{facts: readiness.Facts{Scenario: "demo", Surfaces: []string{"ui"}, UIFramework: "react"}}), Workloads: reader})
			native, err := h.ValidateReadiness(t.Context(), connect.NewRequest(&readinessv1.ValidateReadinessRequest{Scenario: "demo"}))
			if err != nil || native.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
				t.Fatalf("native without classification metadata: %v, %v", native, err)
			}
			// The shared provider additionally requires a maturity assessment.
			h.spec, err = assessment.ParseSpec([]byte(budgetGateSpec))
			if err != nil {
				t.Fatal(err)
			}
			r, err := NewSharedHandler(h).ValidateScenario(t.Context(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
			if err != nil {
				t.Fatal(err)
			}
			if r.Msg.GetStatus() != tc.want {
				t.Fatalf("status=%v want=%v", r.Msg.GetStatus(), tc.want)
			}
			if tc.readErr == nil {
				var detail readinessv1.ValidateReadinessResponse
				if err := r.Msg.GetNativeDetail().UnmarshalTo(&detail); err != nil {
					t.Fatal(err)
				}
				if len(detail.Workloads) != 1 || detail.Workloads[0].OperationId != "retained-owner-operation" {
					t.Fatal("gate omitted owner receipt provenance")
				}
			}
		})
	}
}

// scenarioRoot returns the repo root containing scenarios/performance-health so
// the maturity spec loads in-checkout.
func scenarioRoot(t *testing.T) string {
	t.Helper()
	// handlers/validation -> api -> performance-health
	return "../.."
}
