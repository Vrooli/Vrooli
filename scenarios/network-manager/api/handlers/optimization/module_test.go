package optimization

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "optimization" {
		t.Fatalf("module name = %q, want optimization", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected optimization endpoints")
	}
}
