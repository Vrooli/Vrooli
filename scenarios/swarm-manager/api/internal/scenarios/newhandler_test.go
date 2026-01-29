package scenarios

import "testing"

func TestNewHandler_DefaultDir(t *testing.T) {
	h := NewHandler("")
	if h.scenariosDir != "scenarios" {
		t.Fatalf("expected scenariosDir %q, got %q", "scenarios", h.scenariosDir)
	}
	if h.source == nil {
		t.Fatal("expected scenario source to be initialized")
	}
	if h.lifecycle == nil {
		t.Fatal("expected scenario lifecycle to be initialized")
	}
	if h.completeness == nil {
		t.Fatal("expected completeness source to be initialized")
	}
}
