package adapters

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "adapters" {
		t.Fatalf("module name = %q, want adapters", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected adapters endpoints")
	}
}
