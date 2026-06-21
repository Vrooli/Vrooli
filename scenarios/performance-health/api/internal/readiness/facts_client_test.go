package readiness

import (
	"context"
	"errors"
	"testing"
)

type erroringResolver struct{}

func (erroringResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return "", errors.New("code-facts not running")
}

// [REQ:PH-TIER-002] When Code Facts is unavailable, the client falls back to a
// filesystem scan: surfaces from the tree, framework from package.json deps, and
// a recorded DegradedReason. React is never assumed — vite-only stays "vite".
func TestCodeFactsClientFilesystemFallback(t *testing.T) {
	root := writeBareReactVite(t)
	client := &CodeFactsClient{Resolver: erroringResolver{}}
	facts, err := client.Describe(context.Background(), "bare", root)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if facts.UIFramework != "react-vite" {
		t.Fatalf("framework = %q, want react-vite", facts.UIFramework)
	}
	if !hasUI(facts.Surfaces) {
		t.Fatalf("expected a ui surface from the filesystem scan, got %v", facts.Surfaces)
	}
	if facts.RootPath == "" {
		t.Fatal("RootPath should be populated for downstream infra inspection")
	}
	if facts.DegradedReason == "" {
		t.Fatal("a fallback should record a DegradedReason")
	}
}

func TestFrameworkFromFilesystemNeverAssumesReact(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, root+"/ui")
	mustWrite(t, root+"/ui/package.json", `{"devDependencies":{"vite":"^6.0.0"}}`)
	if got := frameworkFromFilesystem(root); got != "vite" {
		t.Fatalf("vite-only package.json framework = %q, want vite", got)
	}
}

func TestDescribeRequiresTarget(t *testing.T) {
	client := &CodeFactsClient{Resolver: erroringResolver{}}
	if _, err := client.Describe(context.Background(), "", ""); err == nil {
		t.Fatal("expected error when neither scenario nor path is provided")
	}
}
