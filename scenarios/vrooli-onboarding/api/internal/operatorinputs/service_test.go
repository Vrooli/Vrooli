package operatorinputs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

func TestResolveRejectsStaleConfigurationRevisionBeforeApplyingAnswers(t *testing.T) {
	called := false
	service := Service{
		CurrentRevision: func(context.Context) (string, error) { return "new-revision", nil },
		Applier: capabilityApplierFunc(func(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error) {
			called = true
			return operatorcapability.Result{}, nil
		}),
	}

	err := service.Resolve(context.Background(), "local", "old-revision", []operatorcapability.Answer{{RequestID: "input", Value: "value"}})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("Resolve() error = %v, want revision conflict", err)
	}
	if called {
		t.Fatal("stale answers reached the capability applier")
	}
}

func TestCapabilityRequestBindsSelectedTargetIntoActionAndRetryKey(t *testing.T) {
	queue := operatorcapability.Pending{Requests: []operatorcapability.Request{{
		ID: "endpoint", CapabilityID: "fixture/endpoint", InputID: "endpoint", Kind: operatorcapability.KindChoice,
	}}}
	answers := map[string]string{"endpoint": "https://provider.example"}

	local, err := BuildCapabilityRequests(queue, answers, " local ")
	if err != nil {
		t.Fatal(err)
	}
	remote, err := BuildCapabilityRequests(queue, answers, "node-7")
	if err != nil {
		t.Fatal(err)
	}
	if len(local) != 1 || len(remote) != 1 {
		t.Fatalf("action counts = local %d, remote %d, want one each", len(local), len(remote))
	}
	if local[0].TargetID != "local" || remote[0].TargetID != "node-7" {
		t.Fatalf("action targets = local %q, remote %q", local[0].TargetID, remote[0].TargetID)
	}
	if local[0].IdempotencyKey != operatorcapability.StableIdempotencyKey(local[0].CapabilityID, local[0].Inputs, "local") {
		t.Fatalf("local key %q does not include its target", local[0].IdempotencyKey)
	}
	if local[0].IdempotencyKey == remote[0].IdempotencyKey {
		t.Fatal("the same capability input on different targets shared an idempotency key")
	}
}

func TestResolveRejectsMissingTargetBeforeReadingOrApplying(t *testing.T) {
	readRevision := false
	apply := false
	service := Service{
		CurrentRevision: func(context.Context) (string, error) { readRevision = true; return "revision-1", nil },
		Applier: capabilityApplierFunc(func(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error) {
			apply = true
			return operatorcapability.Result{}, nil
		}),
	}
	err := service.Resolve(context.Background(), " ", "revision-1", []operatorcapability.Answer{{RequestID: "input", Value: "value"}})
	if !errors.Is(err, ErrInvalidAnswer) {
		t.Fatalf("Resolve() error = %v, want invalid target", err)
	}
	if readRevision || apply {
		t.Fatalf("missing target reached revision/provider: revision=%t apply=%t", readRevision, apply)
	}
}

type capabilityApplierFunc func(context.Context, operatorcapability.ActionRequest) (operatorcapability.Result, error)

func (f capabilityApplierFunc) ApplyCapability(ctx context.Context, request operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	return f(ctx, request)
}

func TestClearCapabilityRequestsZeroesInputBytesBeforeRelease(t *testing.T) {
	secret := json.RawMessage([]byte(`"ephemeral-secret"`))
	requests := []operatorcapability.ActionRequest{{
		CapabilityID: "fixture/secret",
		Inputs:       map[string]json.RawMessage{"token": secret},
	}}

	ClearCapabilityRequests(requests)

	for index, value := range secret {
		if value != 0 {
			t.Fatalf("secret byte %d = %d after clear; expected zero", index, value)
		}
	}
	if len(requests[0].Inputs) != 0 {
		t.Fatalf("inputs = %v after clear; expected released map", requests[0].Inputs)
	}
}
