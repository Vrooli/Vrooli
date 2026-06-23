package policy

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "policy" {
		t.Fatalf("module name = %q, want policy", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected policy endpoints")
	}
}
