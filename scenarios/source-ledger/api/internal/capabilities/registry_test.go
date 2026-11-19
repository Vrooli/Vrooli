package capabilities

import (
	"reflect"
	"sort"
	"testing"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

func TestKnownMatchesManifestScenarioDependencies(t *testing.T) {
	manifestDefs, err := capabilityregistry.ProjectManifest("../../../.vrooli/service.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, def := range manifestDefs {
		if def.DependencyKind == capabilityregistry.DependencyScenario {
			want = append(want, def.DependencySlug)
		}
	}
	got := make([]string, 0, len(Known))
	for _, def := range Known {
		if def.DependencyKind == capabilityregistry.DependencyScenario {
			got = append(got, def.DependencySlug)
		}
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capability scenario slugs = %v, manifest scenario slugs = %v", got, want)
	}
}

func TestNewRegistryValidatesScenarioTargets(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if err := registry.Validate(); err != nil {
		t.Fatalf("NewRegistry checker wiring is invalid: %v", err)
	}
}
