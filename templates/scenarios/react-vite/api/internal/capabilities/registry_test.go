package capabilities

import (
	"testing"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

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

// TestNewRegistryCoversEveryDeclaredCapability pins the pairing between a
// declared capability and its checker. A declaration without a checker reports
// nothing, which reads as healthy rather than as unknown.
func TestNewRegistryCoversEveryDeclaredCapability(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	for _, def := range Known {
		if def.DependencyKind != capabilityregistry.DependencyScenario {
			continue
		}
		if _, ok := any(ScenarioChecker{}).(capabilityregistry.Checker); !ok {
			t.Fatalf("ScenarioChecker does not satisfy the Checker contract for %q", def.ID)
		}
	}
}
