package capabilities

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities"
	internalcapabilities "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/capabilities"
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
