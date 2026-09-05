package capabilities

import (
	"context"
	"testing"
)

func TestKnownDependenciesDoNotDuplicateManifestDependencies(t *testing.T) {
	if len(Known) != 0 {
		t.Fatalf("manifest dependencies must not be repeated in Known: %#v", Known)
	}
	if status, _ := (ScenarioChecker{Slug: "swarm-manager"}).Check(context.Background()); status != "unknown" {
		t.Fatalf("status = %q, want unknown until control-plane check", status)
	}
}
