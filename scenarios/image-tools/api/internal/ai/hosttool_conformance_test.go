package ai

import (
	"encoding/json"
	"os"
	"testing"
)

// serviceManifestPath locates the scenario service.json from the package
// directory (api/internal/ai → ../../../.vrooli/service.json); see
// packageRelative for why runtime.Caller is not used.
func serviceManifestPath(t *testing.T) string {
	t.Helper()
	return packageRelative(t, "..", "..", "..", ".vrooli", "service.json")
}

func serviceHostToolNames(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(serviceManifestPath(t))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}
	var manifest struct {
		HostTools []struct {
			Name string `json:"name"`
		} `json:"hostTools"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse service.json: %v", err)
	}
	names := make(map[string]bool, len(manifest.HostTools))
	for _, ht := range manifest.HostTools {
		names[ht.Name] = true
	}
	return names
}

// TestProviderHostToolsDeclaredInServiceManifest is the scenario-side drift
// ratchet: every backend provider's host tool must be declared in service.json
// hostTools, so the platform is asked to provision exactly what the providers
// need. Adding a provider that needs a new host tool without declaring it is a
// red build.
func TestProviderHostToolsDeclaredInServiceManifest(t *testing.T) {
	declared := serviceHostToolNames(t)
	for _, binding := range HostToolBindings() {
		if binding.HostTool == "" {
			t.Errorf("provider %q has an empty hostTool binding", binding.Provider)
			continue
		}
		if !declared[binding.HostTool] {
			t.Errorf("provider %q needs host tool %q, but it is not declared in service.json hostTools", binding.Provider, binding.HostTool)
		}
	}
}
