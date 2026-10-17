package validation

import "testing"

func TestModuleExposesValidationEndpoints(t *testing.T) {
	mod := Module(nil, t.TempDir(), nil)
	if mod.Name != "validation" {
		t.Fatalf("module name = %q", mod.Name)
	}
	if len(mod.Endpoints) != len(Endpoints) {
		t.Fatalf("endpoints = %d, want %d", len(mod.Endpoints), len(Endpoints))
	}
	for _, endpoint := range mod.Endpoints {
		if endpoint.Path == "" || endpoint.Method != "POST" {
			t.Fatalf("invalid endpoint descriptor: %+v", endpoint)
		}
	}
}
