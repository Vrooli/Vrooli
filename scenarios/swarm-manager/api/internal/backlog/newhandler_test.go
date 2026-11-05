package backlog

import (
	"testing"

	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/runtimepaths"
)

func TestNewHandler_DefaultRoots(t *testing.T) {
	h := NewHandler("", "")
	wantData, err := runtimepaths.DataPath("")
	if err != nil {
		t.Fatalf("DataPath: %v", err)
	}
	if h.dataRoot != wantData {
		t.Fatalf("expected dataRoot %q, got %q", wantData, h.dataRoot)
	}
	wantRepo := pathutil.ResolveScenarioRoot("swarm-manager")
	if h.repoRoot != wantRepo {
		t.Fatalf("expected repoRoot %q, got %q", wantRepo, h.repoRoot)
	}
}
