package phases

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func admissionWithRoute() *domain.RunAdmission {
	return &domain.RunAdmission{
		RequestedRoleRef: "code.flatrate",
		EffectiveRunner:  "opencode",
		EffectiveModel:   "opencode-go/deepseek-v4.1-flash",
		EffectiveEffort:  "medium",
		RuntimeVersion:   "opencode-go/1.2.3",
	}
}

func TestHandleResultCapturesQualificationReceiptFromLiveEvidence(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)
	runID := uuid.New()
	run := &domain.Run{
		ID:             runID,
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		IdempotencyKey: "op-123",
		ResolvedConfig: &domain.RunConfig{Admission: admissionWithRoute()},
	}
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Metrics:  runner.ExecutionMetrics{TokensInput: 120, TokensOutput: 45},
	}

	HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{Clock: func() time.Time { return now }}, Run: run, Result: result,
	})

	receipt := run.ResolvedConfig.Admission.Receipt
	if receipt == nil {
		t.Fatal("expected a qualification receipt to be captured on the terminal seam")
	}
	if receipt.Route != "code.flatrate" || receipt.RuntimeVersion != "opencode-go/1.2.3" || !receipt.AcceptedOutput {
		t.Fatalf("receipt did not bind route/runtime/accepted output: %+v", receipt)
	}
	if receipt.RunID != runID.String() || receipt.OperationID != "op-123" {
		t.Fatalf("receipt did not bind operation identity: %+v", receipt)
	}
	if receipt.Usage.State != domain.QualificationUsageMeasured ||
		receipt.Usage.InputTokens != 120 || receipt.Usage.OutputTokens != 45 {
		t.Fatalf("provider-reported usage must be measured: %+v", receipt.Usage)
	}
	if len(receipt.ProviderAcknowledgment) != 0 {
		t.Fatalf("provider acknowledgment must stay unsourced rather than inferred: %+v", receipt.ProviderAcknowledgment)
	}
	if !receipt.CapturedAt.Equal(now) {
		t.Fatalf("capturedAt = %s, want %s", receipt.CapturedAt, now)
	}
	req := domain.DependentDelegationRequest{
		Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash", Effort: "medium",
	}
	if err := domain.AdmitDependentDelegation(receipt, req); err != nil {
		t.Fatalf("a captured successful receipt must admit the matching identity: %v", err)
	}
}

func TestHandleResultCapturesProfileRouteWhenNotRequested(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)
	run := &domain.Run{
		ID:             uuid.New(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		IdempotencyKey: "op-profile",
		ResolvedConfig: &domain.RunConfig{
			RoleRef: "code.flatrate",
			Admission: &domain.RunAdmission{
				EffectiveRunner: "opencode",
				EffectiveModel:  "opencode-go/deepseek-v4.1-flash",
				RuntimeVersion:  "opencode-go/1.2.3",
			},
		},
	}

	HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{Clock: func() time.Time { return now }},
		Run:  run,
		Result: &runner.ExecuteResult{
			Success:  true,
			ExitCode: 0,
			Metrics:  runner.ExecutionMetrics{TokensInput: 10, TokensOutput: 5},
		},
	})

	receipt := run.ResolvedConfig.Admission.Receipt
	if receipt == nil {
		t.Fatal("expected a receipt for a profile-based run")
	}
	if receipt.Route != "code.flatrate" {
		t.Fatalf("profile-based run must key the receipt to the resolved role, got %q", receipt.Route)
	}
	req := domain.DependentDelegationRequest{Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash"}
	if err := domain.AdmitDependentDelegation(receipt, req); err != nil {
		t.Fatalf("a profile-based successful receipt must admit the matching identity: %v", err)
	}
}

func TestHandleResultCapturesUnknownUsageAndFailureKeepsGateClosed(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)
	run := &domain.Run{
		ID:             uuid.New(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		IdempotencyKey: "op-failed",
		ResolvedConfig: &domain.RunConfig{Admission: admissionWithRoute()},
	}

	HandleResult(context.Background(), HandleResultInput{
		Deps:   Deps{Clock: func() time.Time { return now }},
		Run:    run,
		Result: &runner.ExecuteResult{ExitCode: 1, ErrorMessage: "runner failed"},
	})

	receipt := run.ResolvedConfig.Admission.Receipt
	if receipt == nil {
		t.Fatal("expected the failed bounded run to retain a truthful receipt")
	}
	if receipt.AcceptedOutput {
		t.Fatal("a failed run must not record accepted output")
	}
	if receipt.Usage.State != domain.QualificationUsageUnknownReserved || !receipt.Usage.ReservedUnknown {
		t.Fatalf("unreported usage must stay unknown and reserved: %+v", receipt.Usage)
	}
	req := domain.DependentDelegationRequest{
		Runner: "opencode", Model: "opencode-go/deepseek-v4.1-flash", Effort: "medium",
	}
	if err := domain.AdmitDependentDelegation(receipt, req); err == nil {
		t.Fatal("a failed run must not open the dependent-delegation gate")
	}
}

func TestHandleResultWithoutAdmissionCapturesNothing(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)
	run := &domain.Run{
		ID:             uuid.New(),
		RunMode:        domain.RunModeInPlace,
		Status:         domain.RunStatusRunning,
		ResolvedConfig: &domain.RunConfig{},
	}

	HandleResult(context.Background(), HandleResultInput{
		Deps:   Deps{Clock: func() time.Time { return now }},
		Run:    run,
		Result: &runner.ExecuteResult{Success: true},
	})

	if run.ResolvedConfig.Admission != nil {
		t.Fatalf("a run without an admission record must not gain one: %+v", run.ResolvedConfig.Admission)
	}
}
