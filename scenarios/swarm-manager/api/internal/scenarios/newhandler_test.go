package scenarios

import "testing"

func TestNewHandler_DefaultDir(t *testing.T) {
	h := NewHandler("")
	if h.scenariosDir != "scenarios" {
		t.Fatalf("expected scenariosDir %q, got %q", "scenarios", h.scenariosDir)
	}
}
