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

func TestImplicitBasenamesDoNotCollideOrOverrideExplicitIdentifiers(t *testing.T) {
	docs := []Document{
		{Path: "../README.md", DocType: "readme", Title: "Scenario overview"},
		{Path: "README.md", DocType: "docs-index", Title: "Documentation hub"},
		{Path: "../experience/README.md", DocType: "experience-contract", Title: "Experience contract"},
		{Path: "a/DETAIL.md", DocType: "first", Title: "First detail"},
		{Path: "b/DETAIL.md", DocType: "second", Title: "Second detail"},
		{Path: "UNIQUE.md", DocType: "unique-type", Title: "Unique title"},
		{Path: "other.md", DocType: "explicit", Title: "Explicit title", Aliases: []string{"unique"}},
	}
	for _, reverse := range []bool{false, true} {
		if reverse {
			for i, j := 0, len(docs)-1; i < j; i, j = i+1, j-1 {
				docs[i], docs[j] = docs[j], docs[i]
			}
		}
		contract, findings := Resolve(&Manifest{Sections: []Section{{Documents: docs}}}, "manifest.json")
		for _, finding := range findings {
			if finding.Code == "duplicate_identifier" {
				t.Fatal(finding)
			}
		}
		for key, want := range map[string]string{"readme": "readme", "experience-contract": "experience-contract", "unique": "explicit", "other": "explicit"} {
			got, ok := contract.ResolveIdentifier(key)
			if !ok || got.DocType != want {
				t.Errorf("%s resolved to %+v", key, got)
			}
		}
		if _, ok := contract.ResolveIdentifier("detail"); ok {
			t.Fatal("ambiguous implicit basename resolved arbitrarily")
		}
		if got, ok := contract.ResolvePath("experience/README.md"); !ok || got.DocType != "experience-contract" {
			t.Fatalf("canonical path lost: %+v", got)
		}
	}
}

func TestExplicitIdentifierCollisionsRemainErrors(t *testing.T) {
	for _, duplicate := range []Document{
		{Path: "b.md", DocType: "first", Title: "Second"},
		{Path: "b.md", DocType: "second", Title: "First"},
		{Path: "b.md", DocType: "second", Title: "Second", Aliases: []string{"first"}},
	} {
		_, findings := Resolve(&Manifest{Sections: []Section{{Documents: []Document{{Path: "a.md", DocType: "first", Title: "First"}, duplicate}}}}, "manifest.json")
		found := false
		for _, finding := range findings {
			found = found || finding.Code == "duplicate_identifier"
		}
		if !found {
			t.Fatalf("explicit collision accepted: %+v", duplicate)
		}
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
