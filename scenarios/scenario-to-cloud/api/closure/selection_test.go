package closure

import (
	"context"
	"testing"

	"scenario-to-cloud/domain"

	"google.golang.org/protobuf/proto"
)

// TestToSelectionParity proves the same closure maps to identical Selection
// bytes and that the Selection and the closure cover exactly the same
// configurable components (P05-A04 groundwork). [REQ:STC-P0-023]
func TestToSelectionParity(t *testing.T) {
	inputs := fixtureInputs(t, "basic", "app")
	inputs.Overrides.OperatingMode = map[string]string{"store": "attach-only", "not-included": "managed-private"}
	closure, err := Resolve(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	first := ToSelection(closure, "vps-1", inputs.Overrides)
	second := ToSelection(closure, "vps-1", inputs.Overrides)
	options := proto.MarshalOptions{Deterministic: true}
	a, err := options.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := options.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("identical closures must produce identical Selection bytes")
	}
	if first.SchemaVersion != SelectionSchemaVersion || first.Target != "vps-1" || first.Apply {
		t.Fatalf("selection envelope: %+v", first)
	}

	expect := map[domain.ClosureComponentKind]map[string]bool{
		domain.ClosureKindScenario:             {},
		domain.ClosureKindResource:             {},
		domain.ClosureKindTool:                 {},
		domain.ClosureKindSafeguard:            {},
		domain.ClosureKindCredentialDescriptor: {},
	}
	for _, component := range closure.Components {
		set, ok := expect[component.Kind]
		if !ok {
			continue
		}
		if component.Kind == domain.ClosureKindResource && !component.OptionalSelected {
			continue
		}
		set[component.ID] = true
	}
	check := func(kind domain.ClosureComponentKind, got []string) {
		t.Helper()
		want := expect[kind]
		if len(got) != len(want) {
			t.Fatalf("%s: selection %v vs closure %v", kind, got, want)
		}
		for _, id := range got {
			if !want[id] {
				t.Fatalf("%s: selection carries %q which the closure does not", kind, id)
			}
		}
	}
	check(domain.ClosureKindScenario, first.Scenarios)
	check(domain.ClosureKindResource, first.OptionalResources)
	check(domain.ClosureKindTool, first.HostTools)
	check(domain.ClosureKindSafeguard, first.HostSafeguards)
	check(domain.ClosureKindCredentialDescriptor, first.CredentialAddresses)

	if len(first.Scenarios) != 3 || first.Scenarios[0] != "app" {
		t.Fatalf("scenarios must be sorted and complete: %v", first.Scenarios)
	}
	// Required resources are derived by setup from the scenario manifests;
	// only operator-selected optional resources ride in optional_resources.
	if len(first.OptionalResources) != 0 {
		t.Fatalf("no optional resource was selected in the fixture: %v", first.OptionalResources)
	}
	if first.TransientHeadroomReserveBytes != closure.Capacity.TransientUpdateHeadroomBytes {
		t.Fatalf("headroom must flow into the selection")
	}
	if first.OperatingMode["store"] != "attach-only" || len(first.OperatingMode) != 1 {
		t.Fatalf("operating modes must be limited to included resources: %v", first.OperatingMode)
	}
	for _, field := range []string{"scenarios", "host_tools", "host_safeguards", "credential_addresses", "operating_mode", "transient_headroom_reserve_bytes"} {
		if first.FieldPresence[field] == 0 {
			t.Fatalf("field presence must be explicit for %s: %v", field, first.FieldPresence)
		}
	}
	if _, ok := first.FieldPresence["optional_resources"]; ok {
		t.Fatalf("an empty field must not claim presence")
	}
}

// TestManifestDependenciesProjection proves the manifest dependency and
// bundle sections are the closure's projection and carry its digest.
// [REQ:STC-P0-021]
func TestManifestDependenciesProjection(t *testing.T) {
	closure, err := Resolve(context.Background(), fixtureInputs(t, "basic", "app"))
	if err != nil {
		t.Fatal(err)
	}
	deps, bundle := ManifestDependencies(closure)
	if deps.ClosureDigest != closure.Digest {
		t.Fatalf("closure_digest = %q", deps.ClosureDigest)
	}
	wantScenarios := []string{"app", "optional-helper", "records-service"}
	wantResources := []string{"cache", "store"}
	if len(deps.Scenarios) != len(wantScenarios) || len(bundle.Scenarios) != len(wantScenarios) {
		t.Fatalf("scenarios = %v / %v", deps.Scenarios, bundle.Scenarios)
	}
	for i := range wantScenarios {
		if deps.Scenarios[i] != wantScenarios[i] || bundle.Scenarios[i] != wantScenarios[i] {
			t.Fatalf("scenarios = %v / %v", deps.Scenarios, bundle.Scenarios)
		}
	}
	for i := range wantResources {
		if deps.Resources[i] != wantResources[i] || bundle.Resources[i] != wantResources[i] {
			t.Fatalf("resources = %v / %v", deps.Resources, bundle.Resources)
		}
	}
}
