package privacy

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "privacy" {
		t.Fatalf("module name = %q, want privacy", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected privacy endpoints")
	}
}
