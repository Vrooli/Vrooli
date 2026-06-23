package homeintegration

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "home_integration" {
		t.Fatalf("module name = %q, want home_integration", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected home integration endpoints")
	}
}
