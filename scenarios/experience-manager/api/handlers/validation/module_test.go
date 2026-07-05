package validation

import "testing"

func TestModuleExposesPhaseOneEndpoints(t *testing.T) {
	mod := Module(nil, t.TempDir(), nil)
	if mod.Name != "validation" {
		t.Fatalf("module name = %q", mod.Name)
	}
	if len(mod.Endpoints) != 4 {
		t.Fatalf("endpoints = %d", len(mod.Endpoints))
	}
	for _, endpoint := range mod.Endpoints {
		if endpoint.Path == "" || endpoint.Method != "POST" {
			t.Fatalf("invalid endpoint descriptor: %+v", endpoint)
		}
	}
}
