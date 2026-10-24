package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:KO-KB-004] Targeted verification must actually check links and use the repository reference root.
func TestKnowledgeVerificationPathScope(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	for _, p := range []string{scenarios, filepath.Join(root, "docs", "nested"), filepath.Join(root, "packages", "example")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{"docs/a.md": "# A\n[Broken](missing.md)\npath:packages/example/code.go\n", "docs/nested/b.md": "# B\n", "packages/example/code.go": "package example\n"}
	for p, body := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.DocHealth(context.Background(), "", DocHealthOptions{Scope: "path", Path: "docs", Checks: []string{"links", "refs"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Counts.BrokenLinks != 1 {
		t.Fatalf("links-only selection did not check links: %+v", out.Counts)
	}
	if out.Counts.MarkedRefsBroken != 0 {
		t.Fatalf("project reference incorrectly scoped: %+v", out.ReferenceFindings)
	}
	if out.TotalDocs != 2 {
		t.Fatalf("recursive corpus count = %d; want 2", out.TotalDocs)
	}
}

func TestKnowledgeVerificationExactPathDoesNotWidenToScenario(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	target := filepath.Join(scenarios, "demo")
	for _, p := range []string{filepath.Join(target, "docs", "selected"), filepath.Join(target, "api")} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for p, body := range map[string]string{"docs/broken.md": "[Broken](missing.md)", "docs/selected/clean.md": "# Clean\npath:api/code.go\n", "api/code.go": "package example\n"} {
		if err := os.WriteFile(filepath.Join(target, filepath.FromSlash(p)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.DocHealth(context.Background(), "", DocHealthOptions{Scope: "path-exact", Path: "scenarios/demo/docs/selected", Checks: []string{"links", "refs"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalDocs != 1 || out.Counts.BrokenLinks != 0 || out.Counts.MarkedRefsBroken != 0 {
		t.Fatalf("scope widened or local owner references changed: %+v", out)
	}
}
