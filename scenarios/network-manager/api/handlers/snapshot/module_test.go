package snapshot

import "testing"

func TestModuleExposesEndpoints(t *testing.T) {
	m := Module()
	if m.Name != "snapshot" {
		t.Fatalf("module name = %q, want snapshot", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected snapshot endpoints")
	}
}
