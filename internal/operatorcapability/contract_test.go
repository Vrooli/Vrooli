package operatorcapability

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type fixtureProvider struct{}

func (fixtureProvider) Descriptor() Descriptor {
	return Descriptor{
		Version: ContractVersion, ID: "fixture/export", Owner: "fixture-owner", Title: "Export fixture",
		Inputs: []InputDescriptor{
			{ID: "sink", Kind: KindPath, Label: "Destination", Required: true},
			{ID: "passphrase", Kind: KindSecret, Label: "Passphrase", Required: true},
			{ID: "interval", Kind: KindDuration, Label: "Refresh interval", Default: "15m"},
			{ID: "confirm", Kind: KindConfirmation, Label: "Confirm", Required: true},
		},
		Policy:   Policy{RequiresConfirmation: true, Idempotent: true, Retryable: true},
		Evidence: EvidenceContract{SecretFree: true, RequiredFields: []string{"checksum"}},
	}
}

func (fixtureProvider) Discover(context.Context) (Status, error) {
	return Status{Descriptor: fixtureProvider{}.Descriptor(), State: StateNeedsInput, UpdatedAt: time.Now().UTC()}, nil
}

func (fixtureProvider) Preview(context.Context, InputSet) (Preview, error) {
	return Preview{CapabilityID: "fixture/export", State: StateReadyToPreview, PlanID: "plan-1", Mutations: []Mutation{{ID: "write", Summary: "Write encrypted artifact", Reversible: true}}}, nil
}

func (fixtureProvider) Apply(context.Context, InputSet) (Result, error) {
	return Result{CapabilityID: "fixture/export", State: StateReady, Outcome: "verified", Evidence: []EvidenceReference{{Kind: "fixture", ArtifactIdentity: "artifact-1", Checksum: "abc", Verified: true, ObservedAt: time.Now().UTC()}}}, nil
}

func rawInputs(values map[string]any) map[string]json.RawMessage {
	raw := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		encoded, _ := json.Marshal(value)
		raw[key] = encoded
	}
	return raw
}

func TestDescriptorValidatesTypedInputsAndRedactsSecretsFromResults(t *testing.T) {
	registry, err := NewRegistry(fixtureProvider{})
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{
		CapabilityID:   "fixture/export",
		IdempotencyKey: "attempt-1",
		Confirm:        true,
		Inputs: rawInputs(map[string]any{
			"sink":       "/media/recovery",
			"passphrase": "never-persist-this",
			"confirm":    true,
		}),
	}
	result, err := registry.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "never-persist-this") {
		t.Fatal("secret input leaked into action result")
	}
	if result.State != StateReady || len(result.Evidence) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestDescriptorRejectsInvalidTypedInputsAndConfirmation(t *testing.T) {
	registry, err := NewRegistry(fixtureProvider{})
	if err != nil {
		t.Fatal(err)
	}
	base := rawInputs(map[string]any{"sink": "/media/recovery", "passphrase": "secret", "confirm": true})
	for name, values := range map[string]map[string]json.RawMessage{
		"missing secret": rawInputs(map[string]any{"sink": "/media/recovery", "confirm": true}),
		"unknown input": func() map[string]json.RawMessage {
			copy := cloneRaw(base)
			copy["extra"] = json.RawMessage(`"x"`)
			return copy
		}(),
		"bad confirmation": rawInputs(map[string]any{"sink": "/media/recovery", "passphrase": "secret", "confirm": false}),
	} {
		_, err := registry.Apply(context.Background(), ActionRequest{CapabilityID: "fixture/export", IdempotencyKey: name, Confirm: true, Inputs: values})
		if err == nil {
			t.Fatalf("%s unexpectedly succeeded", name)
		}
	}
	result, err := registry.Apply(context.Background(), ActionRequest{CapabilityID: "fixture/export", IdempotencyKey: "needs-confirmation", Inputs: base})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorCode != "confirmation_required" || result.State != StateReadyToPreview {
		t.Fatalf("confirmation result = %+v", result)
	}
}

func TestRegistryDiscoversProvidersInStableOrder(t *testing.T) {
	first := fixtureProvider{}
	second := otherFixtureProvider{}
	registry, err := NewRegistry(second, first)
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := registry.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 2 || statuses[0].Descriptor.ID != "fixture/export" || statuses[1].Descriptor.ID != "fixture/other" {
		t.Fatalf("statuses = %+v", statuses)
	}
}

func TestOperatorInputsUseStableCapabilityScopedIDs(t *testing.T) {
	requests, err := (fixtureProvider{}).Descriptor().OperatorInputs()
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 4 || requests[0].ID != "fixture/export:sink" || requests[0].CapabilityID != "fixture/export" {
		t.Fatalf("requests = %+v", requests)
	}
	if requests[1].Kind != "secret" || len(requests[0].Candidates) != 0 {
		t.Fatalf("request metadata = %+v", requests)
	}
}

type otherFixtureProvider struct{}

func (otherFixtureProvider) Descriptor() Descriptor {
	return Descriptor{Version: ContractVersion, ID: "fixture/other", Owner: "other-owner", Title: "Other fixture", Policy: Policy{Idempotent: true}, Evidence: EvidenceContract{SecretFree: true}}
}

func (otherFixtureProvider) Discover(context.Context) (Status, error) {
	return Status{Descriptor: otherFixtureProvider{}.Descriptor(), State: StateReady, UpdatedAt: time.Now().UTC()}, nil
}

func (otherFixtureProvider) Preview(context.Context, InputSet) (Preview, error) {
	return Preview{}, nil
}
func (otherFixtureProvider) Apply(context.Context, InputSet) (Result, error) { return Result{}, nil }

func cloneRaw(input map[string]json.RawMessage) map[string]json.RawMessage {
	output := make(map[string]json.RawMessage, len(input))
	for key, value := range input {
		output[key] = append(json.RawMessage(nil), value...)
	}
	return output
}
