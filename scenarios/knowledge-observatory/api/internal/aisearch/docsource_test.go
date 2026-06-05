package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// buildFixtureRepo lays out a small repo with a 1.2.0 root manifest and a v2
// scenario manifest, plus an un-manifested scenario, a missing referenced doc,
// a non-markdown referenced artifact, and a pruned node_modules manifest.
func buildFixtureRepo(t *testing.T) (repoRoot, scenariosRoot string) {
	t.Helper()
	repoRoot = t.TempDir()
	scenariosRoot = filepath.Join(repoRoot, "scenarios")

	// Root project manifest (older 1.2.0 nav schema: no contract/docType).
	writeFile(t, filepath.Join(repoRoot, "docs", "manifest.json"), `{
      "version": "1.2.0",
      "title": "Vrooli Documentation",
      "sections": [
        {"id": "start", "title": "Getting Started", "documents": [
          {"path": "QUICKSTART.md", "title": "Quick Start", "description": "Set up the platform", "audience": ["users"]},
          {"path": "concepts/ARCH.md", "title": "Architecture"}
        ]}
      ]
    }`)
	writeFile(t, filepath.Join(repoRoot, "docs", "QUICKSTART.md"), "# Quick Start\n\nGet going fast.")
	writeFile(t, filepath.Join(repoRoot, "docs", "concepts", "ARCH.md"), "# Architecture\n\nHow it fits together.")

	// Scenario foo: v2 manifest referencing ../README.md, a guide, a missing
	// doc, and a non-markdown schema.
	writeFile(t, filepath.Join(scenariosRoot, "foo", "docs", "manifest.json"), `{
      "version": "2.0.0",
      "contract": {"kind": "scenario-docs", "schema": "scenario-docs-manifest/v2", "maturityValues": ["active"], "stages": ["generated"]},
      "sections": [
        {"id": "g", "title": "Getting Started", "documents": [
          {"path": "../README.md", "docType": "readme", "title": "README", "description": "Top-level summary", "audience": ["users","agents"], "canonicalFor": ["entrypoint"], "maturity": "active", "requiredBy": ["generated"], "completion": "required"},
          {"path": "guides/SETUP.md", "docType": "guide", "title": "Setup", "maturity": "active", "requiredBy": ["generated"], "completion": "required"},
          {"path": "MISSING.md", "docType": "guide", "title": "Missing", "maturity": "active", "requiredBy": ["generated"], "completion": "required"},
          {"path": "schema.json", "docType": "reference", "title": "Schema", "maturity": "active", "requiredBy": ["generated"], "completion": "required"}
        ]}
      ]
    }`)
	writeFile(t, filepath.Join(scenariosRoot, "foo", "README.md"), "# Foo\n\nThe foo scenario.")
	writeFile(t, filepath.Join(scenariosRoot, "foo", "docs", "guides", "SETUP.md"), "# Setup\n\nHow to set up foo.")
	writeFile(t, filepath.Join(scenariosRoot, "foo", "docs", "schema.json"), `{"not":"prose"}`)

	// An un-manifested doc under foo/docs: the filesystem sweep must index it
	// even though no manifest entry references it.
	writeFile(t, filepath.Join(scenariosRoot, "foo", "docs", "operations", "RUNBOOK.md"), "# Runbook\n\nOperate foo in production.")

	// An internal status tracker the sweep must NOT index (not documentation).
	writeFile(t, filepath.Join(scenariosRoot, "foo", "docs", "internal", "PROGRESS.md"), "# Progress\n\nWIP notes.")

	// Scenario bar: no manifest at all — supplement must still index its README.
	writeFile(t, filepath.Join(scenariosRoot, "bar", "README.md"), "# Bar\n\nThe bar scenario.")

	// A pruned directory that must never be discovered.
	writeFile(t, filepath.Join(scenariosRoot, "foo", "node_modules", "pkg", "docs", "manifest.json"), `{"sections":[]}`)

	return repoRoot, scenariosRoot
}

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func loadByID(docs []pkg.SourceDoc) map[string]pkg.SourceDoc {
	out := make(map[string]pkg.SourceDoc, len(docs))
	for _, d := range docs {
		out[d.ID] = d
	}
	return out
}

func TestDocSourceDiscoversManifestsAndSupplements(t *testing.T) {
	_, scenariosRoot := buildFixtureRepo(t)
	src, err := NewDocSource(scenariosRoot)
	if err != nil {
		t.Fatalf("NewDocSource: %v", err)
	}
	docs, err := src.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	byID := loadByID(docs)

	want := []string{
		"docs/QUICKSTART.md",
		"docs/concepts/ARCH.md",
		"scenarios/foo/README.md",
		"scenarios/foo/docs/guides/SETUP.md",
		"scenarios/bar/README.md",
	}
	for _, id := range want {
		if _, ok := byID[id]; !ok {
			t.Errorf("missing expected doc %q", id)
		}
	}

	// Missing referenced doc and non-markdown artifact are skipped.
	if _, ok := byID["scenarios/foo/docs/MISSING.md"]; ok {
		t.Error("MISSING.md should be skipped")
	}
	if _, ok := byID["scenarios/foo/docs/schema.json"]; ok {
		t.Error("non-markdown schema.json should be skipped")
	}
	// Pruned node_modules manifest must not contribute anything.
	for id := range byID {
		if filepath.Base(filepath.Dir(id)) == "node_modules" {
			t.Errorf("pruned dir leaked into index: %q", id)
		}
	}
}

func TestDocSourceSweepsUnmanifestedDocs(t *testing.T) {
	_, scenariosRoot := buildFixtureRepo(t)
	src, _ := NewDocSource(scenariosRoot)
	docs, _ := src.LoadAll(context.Background())
	byID := loadByID(docs)

	// The un-manifested doc under foo/docs is indexed by the sweep, with
	// inferred metadata and the sweep origin.
	runbook, ok := byID["scenarios/foo/docs/operations/RUNBOOK.md"]
	if !ok {
		t.Fatal("filesystem sweep missed un-manifested doc scenarios/foo/docs/operations/RUNBOOK.md")
	}
	if runbook.Meta[MetaSource] != SourceSweep {
		t.Errorf("swept doc source = %v, want %q", runbook.Meta[MetaSource], SourceSweep)
	}
	if runbook.Meta[MetaScope] != ScopeScenario || runbook.Meta[MetaScenario] != "foo" {
		t.Errorf("swept doc scope/scenario = %v/%v", runbook.Meta[MetaScope], runbook.Meta[MetaScenario])
	}
	if runbook.Meta[MetaTitle] != "Runbook" {
		t.Errorf("swept doc title = %v, want inferred first heading", runbook.Meta[MetaTitle])
	}
	// Ancestor path prefixes are stored for server-side path scope.
	prefixes, _ := runbook.Meta[MetaPathPrefixes].([]string)
	if !containsStr(prefixes, "scenarios/foo/docs/operations") || !containsStr(prefixes, "scenarios/foo") {
		t.Errorf("swept doc path_prefixes = %v, want ancestor dirs", prefixes)
	}

	// Status-tracker files are excluded from the sweep.
	if _, ok := byID["scenarios/foo/docs/internal/PROGRESS.md"]; ok {
		t.Error("sweep indexed a PROGRESS.md status tracker; it should be excluded")
	}

	// A manifest-listed doc still wins its richer origin (dedup favors manifest).
	if setup := byID["scenarios/foo/docs/guides/SETUP.md"]; setup.Meta[MetaSource] != SourceManifest {
		t.Errorf("manifest doc source = %v, want manifest (sweep must not override)", setup.Meta[MetaSource])
	}
}

func TestDocSourceMetadataAndScope(t *testing.T) {
	_, scenariosRoot := buildFixtureRepo(t)
	src, _ := NewDocSource(scenariosRoot)
	docs, _ := src.LoadAll(context.Background())
	byID := loadByID(docs)

	// Project doc: scope=project, no scenario.
	qs := byID["docs/QUICKSTART.md"]
	if qs.Meta[MetaScope] != ScopeProject || qs.Meta[MetaScenario] != "" {
		t.Errorf("QUICKSTART scope/scenario = %v/%v", qs.Meta[MetaScope], qs.Meta[MetaScenario])
	}
	if qs.Meta[MetaTitle] != "Quick Start" {
		t.Errorf("QUICKSTART title = %v", qs.Meta[MetaTitle])
	}

	// Manifest-listed README carries v2 metadata and is sourced from the manifest
	// (not the supplement), with scope=scenario/scenario=foo.
	readme := byID["scenarios/foo/README.md"]
	if readme.Meta[MetaScope] != ScopeScenario || readme.Meta[MetaScenario] != "foo" {
		t.Errorf("foo README scope/scenario = %v/%v", readme.Meta[MetaScope], readme.Meta[MetaScenario])
	}
	if readme.Meta[MetaSource] != SourceManifest {
		t.Errorf("foo README source = %v, want manifest (dedup favors manifest)", readme.Meta[MetaSource])
	}
	if readme.Meta[MetaDocType] != "readme" || readme.Meta[MetaMaturity] != "active" {
		t.Errorf("foo README docType/maturity = %v/%v", readme.Meta[MetaDocType], readme.Meta[MetaMaturity])
	}
	if aud, _ := readme.Meta[MetaAudience].([]string); len(aud) != 2 {
		t.Errorf("foo README audience = %v", readme.Meta[MetaAudience])
	}

	// Un-manifested scenario README comes from the supplement.
	bar := byID["scenarios/bar/README.md"]
	if bar.Meta[MetaSource] != SourceReadme || bar.Meta[MetaScenario] != "bar" {
		t.Errorf("bar README source/scenario = %v/%v", bar.Meta[MetaSource], bar.Meta[MetaScenario])
	}

	// Federation contract fields present and repo-relative.
	if readme.Meta[MetaRelativePath] != "scenarios/foo/README.md" {
		t.Errorf("relative_path = %v", readme.Meta[MetaRelativePath])
	}
	if readme.Meta[MetaPath] != "scenarios/foo/README.md" {
		t.Errorf("path = %v", readme.Meta[MetaPath])
	}
}

func TestDocSourceContentHashStableAndSensitive(t *testing.T) {
	_, scenariosRoot := buildFixtureRepo(t)
	src, _ := NewDocSource(scenariosRoot)

	first, _ := src.LoadAll(context.Background())
	h1 := loadByID(first)["docs/QUICKSTART.md"].ContentHash
	if h1 == "" {
		t.Fatal("content hash empty")
	}

	// Re-load without changes: hash stable.
	second, _ := src.LoadAll(context.Background())
	if h2 := loadByID(second)["docs/QUICKSTART.md"].ContentHash; h2 != h1 {
		t.Fatalf("content hash unstable: %q != %q", h2, h1)
	}

	// Change the body: hash must change.
	writeFile(t, filepath.Join(scenariosRoot, "..", "docs", "QUICKSTART.md"), "# Quick Start\n\nNow with more words.")
	third, _ := src.LoadAll(context.Background())
	if h3 := loadByID(third)["docs/QUICKSTART.md"].ContentHash; h3 == h1 {
		t.Fatal("content hash did not change after body edit")
	}
}

func TestNewDocSourceRejectsEmptyRoot(t *testing.T) {
	if _, err := NewDocSource(""); err == nil {
		t.Fatal("expected error for empty scenarios root")
	}
	if _, err := NewDocSource(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing scenarios root")
	}
}

func TestInferHelpers(t *testing.T) {
	if got := titleFromPath("scenarios/foo/docs/internal/ERROR-HANDLING.md"); got != "Error Handling" {
		t.Errorf("titleFromPath = %q", got)
	}
	if got := inferDocType("a/README.md"); got != "readme" {
		t.Errorf("inferDocType README = %q", got)
	}
	if got := inferDocType("a/PRD.md"); got != "prd" {
		t.Errorf("inferDocType PRD = %q", got)
	}
	if got := inferDocType("a/guides/x.md"); got != "doc" {
		t.Errorf("inferDocType other = %q", got)
	}
	if got := firstHeading("\n\nsome text\n# Later\n"); got != "" {
		t.Errorf("firstHeading should ignore headings after prose start? got %q", got)
	}
}
