package capabilities

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	internalcapabilities "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/capabilities"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestActionRequestBindsRPCTargetWhenProviderTargetIsOmitted(t *testing.T) {
	request, err := actionRequest("node-7", &capabilitiesv1.ActionRequest{CapabilityId: "fixture/action"})
	if err != nil {
		t.Fatal(err)
	}
	if request.TargetID != "node-7" {
		t.Fatalf("target_id = %q, want node-7", request.TargetID)
	}
	if request.IdempotencyKey == "" {
		t.Fatal("target-bound action did not receive an idempotency key")
	}
}

func TestDescriptorProjectionPreservesCompanionCredentials(t *testing.T) {
	descriptor := operatorcapability.Descriptor{
		Inputs: []operatorcapability.InputDescriptor{{
			ID: "access-key", Kind: operatorcapability.KindSecret, Label: "Access key",
			CompanionCredentials: []string{"delivery/access-key-id"},
		}},
	}
	projected := descriptorToProto(descriptor)
	if got := projected.GetInputs()[0].GetCompanionCredentials(); len(got) != 1 || got[0] != "delivery/access-key-id" {
		t.Fatalf("companion credentials = %#v, want the provider-declared companion", got)
	}
}

func TestActionRequestRejectsConflictingRPCAndProviderTargets(t *testing.T) {
	_, err := actionRequest("node-7", &capabilitiesv1.ActionRequest{CapabilityId: "fixture/action", TargetId: "node-8"})
	if err == nil {
		t.Fatal("conflicting targets were accepted")
	}
}

func TestVerificationRequestBindsRPCTargetWhenNestedTargetIsOmitted(t *testing.T) {
	request, err := verificationRequest("node-7", &capabilitiesv1.VerificationRequest{CapabilityId: "fixture/action", Operation: "readiness"})
	if err != nil {
		t.Fatal(err)
	}
	if request.TargetID != "node-7" {
		t.Fatalf("target_id = %q, want node-7", request.TargetID)
	}
}

type typedVerificationExecutor struct{}

func (typedVerificationExecutor) DiscoverCapabilities(context.Context) ([]operatorcapability.Status, error) {
	return nil, nil
}

func (typedVerificationExecutor) PreviewCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Preview, error) {
	return operatorcapability.Preview{}, nil
}

func (typedVerificationExecutor) ApplyCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	return operatorcapability.Result{}, nil
}

func (typedVerificationExecutor) VerifyCapability(context.Context, operatorcapability.VerificationRequest) ([]operatorcapability.EvidenceReference, error) {
	return nil, &operatorcapability.VerificationError{Code: "provider_rate_limited", Retryable: true, RetryAfter: 5 * time.Second, NextAction: "retry-after-provider-backoff"}
}

func (typedVerificationExecutor) RemoveCapability(string) error { return nil }

func TestVerifyCapabilityProjectsTypedProviderFailure(t *testing.T) {
	handler := NewConnectHandler(internalcapabilities.Service{Executor: typedVerificationExecutor{}})
	response, err := handler.VerifyCapability(context.Background(), connect.NewRequest(&capabilitiesv1.VerifyCapabilityRequest{
		Target: "local",
		Verification: &capabilitiesv1.VerificationRequest{
			CapabilityId: "fixture/provider",
			TargetId:     "local",
			Operation:    "readiness-check",
			EffectClass:  "read_only",
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetErrorCode() != "provider_rate_limited" || !response.Msg.GetRetryable() {
		t.Fatalf("typed failure = %+v", response.Msg)
	}
	if response.Msg.GetRetryAfterSeconds() != 5 || response.Msg.GetNextAction() != "retry-after-provider-backoff" || response.Msg.GetOutcome() != "verification_failed" {
		t.Fatalf("typed failure remediation = %+v", response.Msg)
	}
}

type applyCleanupFailExecutor struct {
	applyCalls  int
	removeCalls int
}

func (e *applyCleanupFailExecutor) DiscoverCapabilities(context.Context) ([]operatorcapability.Status, error) {
	return nil, nil
}

func (e *applyCleanupFailExecutor) PreviewCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Preview, error) {
	return operatorcapability.Preview{}, nil
}

func (e *applyCleanupFailExecutor) ApplyCapability(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	e.applyCalls++
	return operatorcapability.Result{
		CapabilityID: "fixture/action", State: operatorcapability.StateReady,
		Outcome: "applied", Retryable: true,
		Mutations: []operatorcapability.Mutation{{ID: "install", Summary: "Synthetic install", Reversible: true}},
	}, nil
}

func (e *applyCleanupFailExecutor) VerifyCapability(context.Context, operatorcapability.VerificationRequest) ([]operatorcapability.EvidenceReference, error) {
	return nil, nil
}

func (e *applyCleanupFailExecutor) RemoveCapability(string) error {
	e.removeCalls++
	return errors.New("synthetic metadata cleanup failure")
}

func TestApplyCapabilityReturnsSuccessSignalWhenMetadataCleanupFails(t *testing.T) {
	executor := &applyCleanupFailExecutor{}
	handler := NewConnectHandler(internalcapabilities.Service{Executor: executor})
	response, err := handler.ApplyCapability(context.Background(), connect.NewRequest(&capabilitiesv1.ApplyCapabilityRequest{
		Target: "local",
		Action: &capabilitiesv1.ActionRequest{CapabilityId: "fixture/action", Confirm: true},
	}))
	if err != nil {
		t.Fatalf("ApplyCapability() error = %v, want successful degraded response", err)
	}
	if executor.applyCalls != 1 || executor.removeCalls != 1 {
		t.Fatalf("calls: apply=%d remove=%d, want one apply and one metadata cleanup", executor.applyCalls, executor.removeCalls)
	}
	if response.Msg.GetState() != capabilitiesv1.CapabilityState_CAPABILITY_STATE_DEGRADED || response.Msg.GetOutcome() != "applied_with_metadata_cleanup_pending" {
		t.Fatalf("response state/outcome = %s/%q", response.Msg.GetState(), response.Msg.GetOutcome())
	}
	if response.Msg.GetErrorCode() != "operator_metadata_cleanup_pending" || response.Msg.GetRetryable() {
		t.Fatalf("response cleanup signal = code %q, retryable %v", response.Msg.GetErrorCode(), response.Msg.GetRetryable())
	}
	if len(response.Msg.GetMutations()) != 1 || response.Msg.GetMutations()[0].GetId() != "install" {
		t.Fatalf("successful mutation information was lost: %+v", response.Msg.GetMutations())
	}
	if response.Msg.GetRemediation() == "" {
		t.Fatal("response needs remediation for pending metadata cleanup")
	}
}

func TestServicePrincipalGrantsRequireIndependentHumanEvenWithHumanActorBehindAgent(t *testing.T) {
	for _, capabilityID := range []string{"compute-manager-notification-hub-principals", "compute-manager-adoption-first-touch", "compute-manager-adoption-first-touch-grant", "compute-manager-money-ledger-invoice-writer"} {
		t.Run(capabilityID, func(t *testing.T) {
			executor := &applyCleanupFailExecutor{}
			handler := NewConnectHandler(internalcapabilities.Service{Executor: executor})
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{
				Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: identity.SourceScenarioAuthenticator,
			})
			ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "agent-run"})
			target := "local"
			var inputs *structpb.Struct
			if capabilityID == "compute-manager-adoption-first-touch" {
				var err error
				inputs, err = structpb.NewStruct(map[string]any{"instance_id": "adopted-vps:fixture-instance"})
				if err != nil {
					t.Fatal(err)
				}
			}
			_, applyErr := handler.ApplyCapability(ctx, connect.NewRequest(&capabilitiesv1.ApplyCapabilityRequest{
				Target: target,
				Action: &capabilitiesv1.ActionRequest{CapabilityId: capabilityID, IdempotencyKey: "independent-human-test-key", Inputs: inputs, Confirm: true},
			}))
			if connect.CodeOf(applyErr) != connect.CodePermissionDenied {
				t.Fatalf("delegated human application error = %v, want permission denied", applyErr)
			}
			if executor.applyCalls != 0 || executor.removeCalls != 0 {
				t.Fatalf("agent-attributed request reached provider: apply=%d cleanup=%d", executor.applyCalls, executor.removeCalls)
			}
		})
	}
}
