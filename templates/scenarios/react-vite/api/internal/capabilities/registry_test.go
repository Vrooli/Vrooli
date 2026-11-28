package capabilities

import "testing"

// TestKnownDefinitionsAreComplete guards the declared capability surface: an
// entry missing an id, a dependency slug, or an operator action is not
// actionable by the console or by an agent, and the omission is silent.
func TestKnownDefinitionsAreComplete(t *testing.T) {
	if len(Known) == 0 {
		t.Fatal("Known is empty; the capability surface declares nothing")
	}
	seen := map[string]bool{}
	for _, def := range Known {
		if def.ID == "" {
			t.Error("capability definition has no ID")
			continue
		}
		if seen[def.ID] {
			t.Errorf("capability %q is declared more than once", def.ID)
		}
		seen[def.ID] = true
		if def.Name == "" || def.Description == "" {
			t.Errorf("capability %q must carry a name and a description", def.ID)
		}
		if def.DependencySlug == "" {
			t.Errorf("capability %q must name the dependency it needs", def.ID)
		}
		if def.OperatorCommand == "" || def.ActionLabel == "" {
			t.Errorf("capability %q must offer an operator action; an unavailable capability with no remedy is a dead end", def.ID)
		}
	}
}

// TestNewRegistryCoversEveryDeclaredCapability validates the real registry
// wiring, including each declared scenario target.
func TestNewRegistryCoversEveryDeclaredCapability(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if err := registry.Validate(); err != nil {
		t.Fatalf("NewRegistry wiring is invalid: %v", err)
	}
}
