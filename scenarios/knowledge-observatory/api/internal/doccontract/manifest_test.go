package doccontract

import (
	"path/filepath"
	"testing"
)

func TestReactViteManifestAliasesAndAppendLogs(t *testing.T) {
	root := repoRootForTest(t)
	manifest, err := LoadManifest(filepath.Join(root, "templates", "scenarios", "react-vite", "docs", "manifest.json"))
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	contract, findings := Resolve(manifest, "manifest.json")
	if err := ErrorFromFindings(findings); err != nil {
		t.Fatalf("contract findings: %v", err)
	}
	doc, ok := contract.ResolveIdentifier("flow")
	if !ok || doc.DocType != "flows" {
		t.Fatalf("flow alias resolved to %#v", doc)
	}
	progress, ok := contract.ResolveIdentifier("progress")
	if !ok || progress.Operations.AppendLog == nil || !progress.Operations.AppendLog.Retention.SupportsReset {
		t.Fatalf("progress append log not declared: %#v", progress)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := LoadManifest(filepath.Join(dir, "templates", "scenarios", "react-vite", "docs", "manifest.json")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("repo root not found")
		}
		dir = next
	}
}
