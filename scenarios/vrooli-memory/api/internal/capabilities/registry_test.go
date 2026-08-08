package capabilities

import (
	"context"
	"testing"
)

func TestKnownDependenciesExposeRecoveryContract(t *testing.T) {
	want := map[string]bool{"source-ledger": true, "vrooli-events": true, "swarm-manager": true}
	if len(Known) != len(want) {
		t.Fatalf("Known = %#v", Known)
	}
	for _, definition := range Known {
		if !want[definition.DependencySlug] || definition.Description == "" || definition.ActionKind == "" || definition.OperatorCommand == "" {
			t.Fatalf("incomplete dependency definition = %#v", definition)
		}
	}
	if status, _ := (ScenarioChecker{Slug: "swarm-manager"}).Check(context.Background()); status != "unknown" {
		t.Fatalf("status = %q, want unknown until control-plane check", status)
	}
}
