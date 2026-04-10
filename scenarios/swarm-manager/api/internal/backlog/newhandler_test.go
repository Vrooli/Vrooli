package backlog

import (
	"testing"

	"swarm-manager/internal/pathutil"
)

func TestNewHandler_DefaultRoot(t *testing.T) {
	h := NewHandler("")
	expected := pathutil.ResolveScenarioRoot("swarm-manager")
	if h.rootDir != expected {
		t.Fatalf("expected rootDir %q, got %q", expected, h.rootDir)
	}
}
