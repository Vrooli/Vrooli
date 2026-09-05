package dochealth

import (
	"os"
	"path/filepath"
	"testing"
)

// Every scenario manifest in the repository uses the v2 sections shape. Before
// this was parsed, the declared set came back empty for all of them, so
// manifest_missing_doc could never fire and --require-all-docs-registered
// reported every document as an orphan.
func TestCheckManifestCoverageReadsV2Sections(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs", "concepts")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	present := filepath.Join(docsDir, "ARCHITECTURE.md")
	if err := os.WriteFile(present, []byte("# Architecture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{
	  "version": "2.0.0",
	  "sections": [
	    {"id": "concepts", "documents": [
	      {"path": "concepts/ARCHITECTURE.md"},
	      {"path": "concepts/MISSING.md"}
	    ]}
	  ]
	}`
	if err := os.WriteFile(filepath.Join(root, "docs", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, coverage, err := checkManifestCoverage(root, filepath.Join("docs", "manifest.json"), false, []string{present})
	if err != nil {
		t.Fatalf("checkManifestCoverage returned error: %v", err)
	}
	if coverage.InManifest != 1 {
		t.Errorf("InManifest = %d, want 1 (the declared file that exists)", coverage.InManifest)
	}
	if len(coverage.MissingDocs) != 1 || coverage.MissingDocs[0] != filepath.Join("docs", "concepts", "MISSING.md") {
		t.Errorf("MissingDocs = %#v, want the one declared-but-absent doc", coverage.MissingDocs)
	}
	if coverage.NotInManifest != 0 {
		t.Errorf("NotInManifest = %d, want 0; the present file is declared", coverage.NotInManifest)
	}
	var sawMissing bool
	for _, finding := range findings {
		if finding.Code == "manifest_missing_doc" {
			sawMissing = true
		}
	}
	if !sawMissing {
		t.Error("expected a manifest_missing_doc finding for the declared-but-absent doc")
	}
}

// A parent-escaping path such as ../README.md resolves to the scenario root,
// not to docs/../README.md, so it must not be reported as missing.
func TestCheckManifestCoverageResolvesParentEscapingPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"sections":[{"documents":[{"path":"../README.md"}]}]}`
	if err := os.WriteFile(filepath.Join(root, "docs", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	_, coverage, err := checkManifestCoverage(root, filepath.Join("docs", "manifest.json"), false, []string{readme})
	if err != nil {
		t.Fatalf("checkManifestCoverage returned error: %v", err)
	}
	if len(coverage.MissingDocs) != 0 {
		t.Errorf("MissingDocs = %#v, want none; ../README.md resolves to the scenario root", coverage.MissingDocs)
	}
	if coverage.InManifest != 1 {
		t.Errorf("InManifest = %d, want 1", coverage.InManifest)
	}
}
