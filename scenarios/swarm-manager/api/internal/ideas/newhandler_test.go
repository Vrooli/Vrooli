package ideas

import (
	"path/filepath"
	"testing"
)

func TestNewHandler_DefaultDir(t *testing.T) {
	h := NewHandler("")
	expected := filepath.Join("scenarios", "swarm-manager", "ideas")
	if h.ideasDir != expected {
		t.Fatalf("expected ideasDir %q, got %q", expected, h.ideasDir)
	}
}
