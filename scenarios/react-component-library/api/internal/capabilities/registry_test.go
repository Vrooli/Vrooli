package capabilities

import "testing"

func TestKnownRegistryDefinitionsAreValid(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Validate(); err != nil {
		t.Fatalf("capability registry should validate: %v", err)
	}
	if len(Known) != 0 {
		t.Fatalf("unexpected capability definitions: %#v", Known)
	}
}
