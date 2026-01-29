package backlog

import (
	"path/filepath"
	"testing"
)

func TestNewHandler_DefaultRoot(t *testing.T) {
	h := NewHandler("")
	expected := filepath.Join("scenarios", "swarm-manager")
	if h.rootDir != expected {
		t.Fatalf("expected rootDir %q, got %q", expected, h.rootDir)
	}
}
