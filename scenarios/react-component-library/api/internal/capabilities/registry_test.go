package capabilities

import "testing"

func TestKnownRegistryDefinitionsAreValid(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Validate(); err != nil {
		t.Fatalf("capability registry should validate: %v", err)
	}
	if len(Known) != 4 {
		t.Fatalf("known capability definitions = %d, want 4", len(Known))
	}
	for _, id := range []string{"agent-manager", "experience-manager", "ui-health", "typescript-code-graph"} {
		found := false
		for _, definition := range Known {
			if definition.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing declared capability %q", id)
		}
	}
}
