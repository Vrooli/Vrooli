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

	err := service.Resolve(context.Background(), "old-revision", []operatorcapability.Answer{{RequestID: "input", Value: "value"}})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("Resolve() error = %v, want revision conflict", err)
	}
	if called {
		t.Fatal("stale answers reached the capability applier")
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
