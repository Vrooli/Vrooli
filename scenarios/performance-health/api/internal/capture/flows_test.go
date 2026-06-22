package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:PH-CAPTURE-005] FileFlowResolver reads <repo>/scenarios/<scn>/bas/flows/<slug>.json.
func TestFileFlowResolverReadsFlow(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "demo", "bas", "flows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"nodes":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "scroll-list.json"), want, 0o644); err != nil {
		t.Fatal(err)
	}
	r := &FileFlowResolver{RepoRoot: root}
	got, err := r.Resolve("demo", "scroll-list")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

// [REQ:PH-CAPTURE-005] A missing flow file is a typed error (FAILED audit upstream).
func TestFileFlowResolverMissingFile(t *testing.T) {
	r := &FileFlowResolver{RepoRoot: t.TempDir()}
	if _, err := r.Resolve("demo", "nope"); err == nil {
		t.Fatal("expected error for a missing flow file")
	}
}

// A slug that could traverse the filesystem is rejected before any read.
func TestFileFlowResolverRejectsUnsafeSlug(t *testing.T) {
	r := &FileFlowResolver{RepoRoot: t.TempDir()}
	for _, bad := range []string{"../secret", "a/b", "Foo", ""} {
		if _, err := r.Resolve("demo", bad); err == nil {
			t.Fatalf("expected rejection of unsafe slug %q", bad)
		}
	}
}

// [REQ:PH-CAPTURE-006] A perf-capture flow must be assertion-free; an ASSERT
// node belongs in bas/cases/**, not a perf trace.
func TestValidatePerfFlowRejectsAsserts(t *testing.T) {
	withAssert := []byte(`{"nodes":[{"id":"a","action":{"type":"ACTION_TYPE_ASSERT","assert":{}}}]}`)
	if err := ValidatePerfFlow(withAssert); err == nil {
		t.Fatal("expected an assertion-bearing flow to be rejected")
	}
	clean := []byte(`{"nodes":[{"id":"s","action":{"type":"ACTION_TYPE_SCROLL","scroll":{"delta_y":100}}}]}`)
	if err := ValidatePerfFlow(clean); err != nil {
		t.Fatalf("assertion-free flow should validate: %v", err)
	}
}

// The template's seeded perf example flow follows the convention (intent marker,
// assertion-free) so detemplated scenarios inherit a valid starting point.
func TestTemplatePerfExampleFlowIsConformant(t *testing.T) {
	// internal/capture → api → performance-health → scenarios → repo root.
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	path := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "bas", "flows", "perf-example-scroll.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("template perf example not found (%v)", err)
	}
	if err := ValidatePerfFlow(data); err != nil {
		t.Fatalf("template perf example must be assertion-free: %v", err)
	}
	if !strings.Contains(string(data), `"intent": "performance"`) {
		t.Fatal("template perf example must carry metadata.labels.intent:performance")
	}
}
