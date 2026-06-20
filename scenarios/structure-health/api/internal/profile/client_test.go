package profile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeLocator struct{ root string }

func (l fakeLocator) Locate(_ context.Context, scenario, _ string) (string, string, string, error) {
	return scenario, "scenario", l.root, nil
}

type erroringResolver struct{}

func (erroringResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("code-facts down")
}

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
