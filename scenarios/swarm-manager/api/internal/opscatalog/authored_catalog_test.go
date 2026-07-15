package opscatalog

import "testing"

// scenarioRoot is the swarm-manager scenario root relative to this package dir,
// where the authored operation-contracts/, bindings/, and policy/ live.
const scenarioRoot = "../../.."

// TestAuthoredCatalogLoads proves the shipped catalog (bindings + transition
// policies) loads and validates as authored. It is the durable gate for policy
// and binding edits: a route that names an unregistered action, an invalid state,
// or a malformed binding fails Load here rather than only at runtime (where the
// runner silently disables itself).
func TestAuthoredCatalogLoads(t *testing.T) {
	cat, err := Load(scenarioRoot)
	if err != nil {
		t.Fatalf("load authored catalog: %v", err)
	}
	// The backlog-item and initiative transition policies must both be present and
	// valid (they route the review completion handlers this phase added).
	for _, id := range []string{"backlog-item-default", "initiative-default"} {
		if _, ok := cat.Policy(id); !ok {
			t.Fatalf("authored catalog missing policy %q", id)
		}
	}
}
