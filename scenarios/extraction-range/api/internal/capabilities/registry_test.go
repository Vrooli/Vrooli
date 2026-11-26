package capabilities

import (
	"testing"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

func TestNewRegistryProjectsOwningServiceManifest(t *testing.T) {
	manifest, err := capabilityregistry.FindServiceManifest()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := capabilityregistry.ProjectManifest(manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := NewRegistry()
	if got == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	actual := got.Definitions()
	if len(actual) != len(expected) {
		t.Fatalf("manifest dependencies = %d, registry definitions = %d", len(expected), len(actual))
	}
	for i := range expected {
		if actual[i].ID != expected[i].ID || actual[i].DependencyKind != expected[i].DependencyKind || actual[i].Required != expected[i].Required {
			t.Errorf("definition[%d] = %+v, want manifest projection %+v", i, actual[i], expected[i])
		}
		if actual[i].Origin != "manifest" {
			t.Errorf("definition %q origin = %q, want manifest", actual[i].ID, actual[i].Origin)
		}
	}
}
