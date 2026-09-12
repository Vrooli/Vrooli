package execution

import (
	"testing"

	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/transitions"
)

func testTransitionRegistry(t *testing.T) transitions.Registry {
	t.Helper()
	registry, err := transitions.LoadDir(pathutil.ResolveScenarioRoot("swarm-manager") + "/.vrooli/swarm-transitions")
	if err != nil {
		t.Fatalf("load transition registry: %v", err)
	}
	return registry
}
