package operatorcapability

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

type fixtureProvider struct{}

func (fixtureProvider) Descriptor() Descriptor {
	return Descriptor{
		Version: ContractVersion, ID: "fixture/export", Owner: "fixture-owner", Scope: "fixture host", Purpose: "exercise typed operator export", Sensitivity: SensitivitySecret, Disposition: DispositionConfigurable,
		Provenance: PermissionProvenance{Requester: "fixture operator", Scope: "fixture host", GrantSource: "explicit fixture consent", RevocationLimit: "fixture owner can revoke the export"}, Lifecycle: Lifecycle{Preview: true, Apply: true, Verify: true, Revoke: true, Recover: true, Recovery: "retry the fixture export"}, Title: "Export fixture",
		Inputs: []InputDescriptor{
			{ID: "sink", Kind: KindPath, Label: "Destination", Required: true, CompanionCredentials: []string{"fixture/access-key-id"}},
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

func TestStableIdempotencyKeyIncludesTarget(t *testing.T) {
	inputs := rawInputs(map[string]any{"sink": "/media/recovery"})
	local := StableIdempotencyKey("fixture/export", inputs, "local")
	remote := StableIdempotencyKey("fixture/export", inputs, "node-7")
	if local == remote {
		t.Fatal("different targets received the same idempotency key")
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
	if len(requests[0].CompanionCredentials) != 1 || requests[0].CompanionCredentials[0] != "fixture/access-key-id" {
		t.Fatalf("companion credentials = %#v, want the declared credential dependency", requests[0].CompanionCredentials)
	}
}

type otherFixtureProvider struct{}

func (otherFixtureProvider) Descriptor() Descriptor {
	return Descriptor{Version: ContractVersion, ID: "fixture/other", Owner: "other-owner", Scope: "fixture host", Purpose: "exercise a second provider", Sensitivity: SensitivityPublic, Disposition: DispositionProtected, Provenance: PermissionProvenance{Requester: "fixture operator", Scope: "fixture host", GrantSource: "fixture owner evidence", RevocationLimit: "not applicable"}, Lifecycle: Lifecycle{Preview: true, Verify: true}, Title: "Other fixture", Policy: Policy{Idempotent: true}, Evidence: EvidenceContract{SecretFree: true}}
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

func TestDescriptorRejectsUnsafeURLsAndOversizedInputs(t *testing.T) {
	descriptor := fixtureProvider{}.Descriptor()
	descriptor.ReferenceURL = "javascript:alert(1)"
	if err := descriptor.Validate(); err == nil {
		t.Fatal("unsafe reference URL was accepted")
	}
	descriptor = fixtureProvider{}.Descriptor()
	oversized := strings.Repeat("x", maxInputBytes+1)
	_, err := descriptor.ValidateInputs(rawInputs(map[string]any{"sink": oversized, "passphrase": "secret", "confirm": true}))
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized input error = %v", err)
	}
	if _, err := descriptor.ValidateInputs(map[string]json.RawMessage{"sink": []byte(`{"nested":true}`), "passphrase": []byte(`"secret"`), "confirm": []byte(`true`)}); err == nil {
		t.Fatal("recursive object input was accepted for a typed path")
	}
	descriptor = fixtureProvider{}.Descriptor()
	descriptor.Inputs[0].Candidates = []Candidate{{ID: "fixture", Label: "Fixture", Metadata: map[string]string{"command": "rm -rf /"}}}
	if err := descriptor.Validate(); err == nil || !strings.Contains(err.Error(), "forbidden execution") {
		t.Fatalf("execution metadata error = %v", err)
	}
}

func TestDescriptorRejectsAggregateResourceAmplification(t *testing.T) {
	base := fixtureProvider{}.Descriptor()
	cases := map[string]func(*Descriptor){
		"inputs": func(d *Descriptor) {
			d.Inputs = make([]InputDescriptor, maxInputs+1)
			for i := range d.Inputs {
				d.Inputs[i] = InputDescriptor{ID: fmt.Sprintf("input-%d", i), Kind: KindBoolean, Label: "Input"}
			}
		},
		"prerequisites":   func(d *Descriptor) { d.Prerequisites = make([]string, maxOptions+1) },
		"protected roots": func(d *Descriptor) { d.Policy.ProtectedRoots = make([]string, maxOptions+1) },
		"evidence fields": func(d *Descriptor) { d.Evidence.RequiredFields = make([]string, maxOptions+1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			descriptor := base
			mutate(&descriptor)
			if err := descriptor.Validate(); err == nil {
				t.Fatal("oversized descriptor was accepted")
			}
		})
	}

	raw := make(map[string]json.RawMessage, maxInputs+1)
	for i := 0; i <= maxInputs; i++ {
		raw[fmt.Sprintf("unknown-%d", i)] = json.RawMessage(`true`)
	}
	if _, err := base.ValidateInputs(raw); err == nil {
		t.Fatal("oversized raw input set was accepted")
	}

	if err := (ActionRequest{CapabilityID: base.ID, IdempotencyKey: strings.Repeat("x", maxActionTextBytes+1)}).Validate(); err == nil {
		t.Fatal("oversized idempotency key was accepted")
	}
}

func TestBuildInventoryIsStableAndSecretFree(t *testing.T) {
	first := fixtureProvider{}.Descriptor()
	second := otherFixtureProvider{}.Descriptor()
	inventory, err := BuildInventory([]Status{{Descriptor: second, State: StateReady}, {Descriptor: first, State: StateNeedsInput, Remediation: "enter the secret"}})
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Version != ContractVersion || len(inventory.Entries) != 2 || inventory.Entries[0].Descriptor.ID != "fixture/export" {
		t.Fatalf("inventory = %+v", inventory)
	}
	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "never-persist-this") {
		t.Fatal("inventory contains secret input data")
	}
}

func TestOptionalDeclineRemainsDurableAcrossReload(t *testing.T) {
	oldPath := queuePathFn
	path := t.TempDir() + "/operator-input.json"
	queuePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { queuePathFn = oldPath })
	if err := Replace([]Request{{ID: "fixture:optional", Kind: KindBoolean, Title: "Optional fixture", Declinable: true, Decision: DecisionPending}}); err != nil {
		t.Fatal(err)
	}
	called := false
	if _, err := ResolveWith([]Answer{{RequestID: "fixture:optional", Declined: true}}, func(map[string]string) error { called = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("declined optional input invoked the owner")
	}
	queue, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(queue.Requests) != 1 || queue.Requests[0].Decision != DecisionDeclined {
		t.Fatalf("durable decline = %+v", queue.Requests)
	}
}
