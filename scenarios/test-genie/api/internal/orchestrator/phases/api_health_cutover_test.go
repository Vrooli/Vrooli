package phases

import (
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func TestAPIHealthCutoverUsesSharedValidationProvider(t *testing.T) {
	catalog := NewDefaultCatalog(DefaultTimeout)

	apiSpec, ok := catalog.Lookup(API.String())
	if !ok {
		t.Fatal("api phase missing from default catalog")
	}
	if apiSpec.Delegated == nil {
		t.Fatal("api phase must be provider-delegated")
	}
	if apiSpec.Delegated.ProviderScenario != "api-health" {
		t.Fatalf("api phase provider = %q, want api-health", apiSpec.Delegated.ProviderScenario)
	}
	if apiSpec.Delegated.Client != nil {
		t.Fatal("api phase must use the shared ScenarioValidationService transport")
	}
	if apiSpec.FindingSource != architecturev1.FindingSource_FINDING_SOURCE_STANDARDS {
		t.Fatalf("api phase finding source = %v, want standards finding source until a dedicated API source exists", apiSpec.FindingSource)
	}
	if !strings.Contains(apiSpec.Description, "API readiness") {
		t.Fatalf("api phase description should advertise API readiness ownership, got %q", apiSpec.Description)
	}

	apiOrder, ok := catalog.Order(API)
	if !ok {
		t.Fatal("api phase missing order")
	}
	architectureOrder, ok := catalog.Order(Architecture)
	if !ok {
		t.Fatal("architecture phase missing order")
	}
	if apiOrder >= architectureOrder {
		t.Fatalf("api phase order = %d, architecture order = %d; API readiness should run before architecture review", apiOrder, architectureOrder)
	}
}

func TestCuratedPresetsIncludeAPIHealth(t *testing.T) {
	presets := DefaultPresets()
	for _, preset := range []Preset{PresetSmoke, PresetArchitectureAudit, PresetComprehensive} {
		names, ok := presets[preset.String()]
		if !ok {
			t.Fatalf("preset %q missing from DefaultPresets", preset)
		}
		if !containsPhase(names, API.String()) {
			t.Fatalf("preset %q must include api phase after API Health cutover, got %v", preset, names)
		}
	}
}

func containsPhase(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
