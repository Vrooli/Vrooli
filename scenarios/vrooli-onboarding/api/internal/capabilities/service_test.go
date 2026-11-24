package capabilities

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

type cleanupFailExecutor struct {
	applyCalls  int
	removeCalls int
	removeErr   error
}

func (e *cleanupFailExecutor) DiscoverCapabilities(context.Context) ([]operatorcapability.Status, error) {
	return nil, nil
}

func (e *cleanupFailExecutor) PreviewCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Preview, error) {
	return operatorcapability.Preview{}, nil
}

func (e *cleanupFailExecutor) ApplyCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	e.applyCalls++
	return operatorcapability.Result{
		CapabilityID: "fixture/action", State: operatorcapability.StateReady,
		Outcome: "applied", Retryable: true,
		Mutations: []operatorcapability.Mutation{{ID: "install", Summary: "Synthetic install", Reversible: true}},
	}, nil
}

func (e *cleanupFailExecutor) VerifyCapability(context.Context, operatorcapability.VerificationRequest) ([]operatorcapability.EvidenceReference, error) {
	return nil, nil
}

func (e *cleanupFailExecutor) RemoveCapability(string) error {
	e.removeCalls++
	return e.removeErr
}

func TestApplyReturnsDegradedSuccessWhenOnlyMetadataCleanupFails(t *testing.T) {
	executor := &cleanupFailExecutor{removeErr: errors.New("synthetic metadata store unavailable")}
	service := Service{Executor: executor}

	result, err := service.Apply(context.Background(), operatorcapability.ActionRequest{
		CapabilityID: "fixture/action", Confirm: true,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v, want explicit degraded result", err)
	}
	if executor.applyCalls != 1 || executor.removeCalls != 1 {
		t.Fatalf("calls: apply=%d remove=%d, want exactly one of each", executor.applyCalls, executor.removeCalls)
	}
	if result.State != operatorcapability.StateDegraded || result.Outcome != "applied_with_metadata_cleanup_pending" {
		t.Fatalf("result state/outcome = %q/%q, want degraded cleanup-pending success", result.State, result.Outcome)
	}
	if result.ErrorCode != "operator_metadata_cleanup_pending" || result.Retryable {
		t.Fatalf("result error/retry = %q/%v, want non-retryable cleanup signal", result.ErrorCode, result.Retryable)
	}
	if len(result.Mutations) != 1 || result.Mutations[0].ID != "install" {
		t.Fatalf("successful mutation evidence was lost: %+v", result.Mutations)
	}
	if result.Remediation == "" {
		t.Fatal("degraded success needs operator remediation")
	}
}
