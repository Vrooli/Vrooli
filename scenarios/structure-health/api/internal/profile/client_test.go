package profile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

type fakeLocator struct{ root string }

func (l fakeLocator) Locate(_ context.Context, scenario, _ string) (string, string, string, error) {
	return scenario, "scenario", l.root, nil
}

func TestFromCodeFactsExcludesMissingSurfaces(t *testing.T) {
	report := &factsv1.CodeFactsReport{
		Surfaces: []*factsv1.Surface{
			{Id: "api", Kind: factsv1.SurfaceKind_SURFACE_KIND_API, Status: factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN, Path: "api"},
			{Id: "runtime", Kind: factsv1.SurfaceKind_SURFACE_KIND_RUNTIME, Status: factsv1.SurfaceStatus_SURFACE_STATUS_MISSING, Path: "runtime"},
		},
	}

	facts := fromCodeFacts(report, "demo", "scenario", t.TempDir())
	if len(facts.Surfaces) != 1 {
		t.Fatalf("surface count = %d, want 1: %+v", len(facts.Surfaces), facts.Surfaces)
	}
	if facts.Surfaces[0].ID != "api" {
		t.Fatalf("only known API surface should remain: %+v", facts.Surfaces)
	}
}

type erroringResolver struct{}

func (erroringResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("code-facts down")
}

// [REQ:SH-GT-001]
func TestDescribeFallsBackToFilesystemWhenCodeFactsDown(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"api", "ui"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module demo/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "package.json"), []byte(`{"dependencies":{"react":"18","vite":"5"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := CodeFactsClient{Locator: fakeLocator{root: root}, Resolver: erroringResolver{}}
	facts, err := c.Describe(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if facts.DegradedReason == "" {
		t.Fatal("degraded reason must be set when code-facts is down")
	}
	if len(facts.Surfaces) != 2 {
		t.Fatalf("want 2 fallback surfaces, got %d: %+v", len(facts.Surfaces), facts.Surfaces)
	}
	p := Derive(facts)
	if p.ID != DefaultProfileID {
		t.Fatalf("fallback profile id = %q, want %q", p.ID, DefaultProfileID)
	}
}
