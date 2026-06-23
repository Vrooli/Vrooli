package resolver

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "resolver" {
		t.Fatalf("module name = %q, want resolver", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected resolver endpoints")
	}
}
