package operatorcapability

import (
	"context"
	"testing"
	"time"
)

// These providers are deliberately test-only. They exercise the public
// descriptor contract used by the generic onboarding and CLI renderers without
// granting a fixture the ability to run a host command.
type descriptorFixtureProvider struct {
	descriptor Descriptor
}

func (p descriptorFixtureProvider) Descriptor() Descriptor { return p.descriptor }

func (p descriptorFixtureProvider) Discover(context.Context) (Status, error) {
	missing := make([]string, 0)
	for _, input := range p.descriptor.Inputs {
		if input.Required {
			missing = append(missing, input.ID)
		}
	}
	state := StateReady
	if len(missing) > 0 {
		state = StateNeedsInput
	}
	return Status{Descriptor: p.descriptor, State: state, MissingInputs: missing, UpdatedAt: time.Now().UTC()}, nil
}

func (p descriptorFixtureProvider) Preview(context.Context, InputSet) (Preview, error) {
	return Preview{CapabilityID: p.descriptor.ID, PlanID: p.descriptor.ID, State: StateReadyToPreview, Mutations: []Mutation{{ID: p.descriptor.ID + ":preview", Summary: "record fixture metadata", Reversible: true}}}, nil
}

func (p descriptorFixtureProvider) Apply(context.Context, InputSet) (Result, error) {
	return Result{CapabilityID: p.descriptor.ID, State: StateReady, Outcome: "fixture_verified", Evidence: []EvidenceReference{{Kind: "fixture", ArtifactIdentity: p.descriptor.ID + "/evidence", ObservedAt: time.Now().UTC(), Verified: true}}}, nil
}

func fixtureDescriptor(id, title string, sensitivity Sensitivity, inputs []InputDescriptor) Descriptor {
	return Descriptor{
		Version: ContractVersion, ID: id, Owner: "fixture.owner", Scope: "fixture host", Purpose: title,
		Sensitivity: sensitivity, Disposition: DispositionConfigurable,
		Provenance: PermissionProvenance{Requester: "fixture operator", Scope: "fixture host", GrantSource: "fixture consent", RevocationLimit: "fixture owner can revoke the fixture decision"},
		Lifecycle:  Lifecycle{Preview: true, Apply: true, Verify: true, Revoke: true, Recover: true, Recovery: "retry the fixture operation"},
		Title:      title, Inputs: inputs, Policy: Policy{RequiresConfirmation: true, Idempotent: true, Retryable: true}, Evidence: EvidenceContract{Kinds: []string{"fixture"}, RequiredFields: []string{"artifact_identity", "observed_at", "verified"}, SecretFree: true},
	}
}

func fixtureProviders() []Provider {
	return []Provider{
		descriptorFixtureProvider{descriptor: fixtureDescriptor("fixture/secret", "Secret input fixture", SensitivitySecret, []InputDescriptor{{ID: "secret", Kind: KindSecret, Label: "Secret", Required: true}, {ID: "confirm", Kind: KindConfirmation, Label: "Confirm", Required: true}})},
		descriptorFixtureProvider{descriptor: fixtureDescriptor("fixture/safeguard", "Safeguard consent fixture", SensitivityOperator, []InputDescriptor{{ID: "enabled", Kind: KindBoolean, Label: "Enable safeguard", Declinable: true}, {ID: "confirm", Kind: KindConfirmation, Label: "Confirm", Required: true}})},
		descriptorFixtureProvider{descriptor: fixtureDescriptor("fixture/platform-permission", "Platform permission fixture", SensitivityOperator, []InputDescriptor{{ID: "permission", Kind: KindEnum, Label: "Permission", Required: true, Options: []string{"notifications", "filesystem"}}, {ID: "confirm", Kind: KindConfirmation, Label: "Confirm", Required: true}})},
	}
}

func TestFixtureProvidersConformToGenericDescriptorSurface(t *testing.T) {
	registry, err := NewRegistry(fixtureProviders()...)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := registry.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Entries) != 3 || inventory.Entries[0].Descriptor.ID != "fixture/platform-permission" {
		t.Fatalf("fixture inventory = %+v", inventory)
	}
	for _, entry := range inventory.Entries {
		if entry.Descriptor.Owner == "" || entry.Descriptor.Provenance.GrantSource == "" || entry.Descriptor.Lifecycle.Preview == false {
			t.Fatalf("fixture metadata incomplete: %+v", entry)
		}
	}
}
