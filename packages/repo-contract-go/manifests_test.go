package repocontract

import (
	"path/filepath"
	"testing"
)

func TestScenarioDocsManifestPath(t *testing.T) {
	root := fixtureRoot(t)
	got, err := ScenarioDocsManifestPath(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioDocsManifestPath() error = %v", err)
	}
	want := filepath.Join(root, "scenarios", "test-genie", "docs", "manifest.json")
	if got != want {
		t.Fatalf("ScenarioDocsManifestPath() = %q, want %q", got, want)
	}
}

func TestScenarioCLIManifestPath(t *testing.T) {
	root := fixtureRoot(t)
	got, err := ScenarioCLIManifestPath(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioCLIManifestPath() error = %v", err)
	}
	want := filepath.Join(root, "scenarios", "test-genie", "cli", "manifest.json")
	if got != want {
		t.Fatalf("ScenarioCLIManifestPath() = %q, want %q", got, want)
	}
}

func TestScenarioServiceManifestPath(t *testing.T) {
	root := fixtureRoot(t)
	got, err := ScenarioServiceManifestPath(root, "test-genie")
	if err != nil {
		t.Fatalf("ScenarioServiceManifestPath() error = %v", err)
	}
	want := filepath.Join(root, filepath.FromSlash("scenarios/test-genie/.vrooli/service.json"))
	if got != want {
		t.Fatalf("ScenarioServiceManifestPath() = %q, want %q", got, want)
	}
}

func TestScenarioManifestPathFallback(t *testing.T) {
	// No fixture: LoadDefault fails, helpers fall back to canonical defaults.
	docs, err := ScenarioDocsManifestPath("/nonexistent/repo", "demo")
	if err != nil {
		t.Fatalf("ScenarioDocsManifestPath() error = %v", err)
	}
	if want := filepath.Join("/nonexistent/repo", "scenarios", "demo", "docs", "manifest.json"); docs != want {
		t.Fatalf("docs fallback = %q, want %q", docs, want)
	}
	cli, err := ScenarioCLIManifestPath("/nonexistent/repo", "demo")
	if err != nil {
		t.Fatalf("ScenarioCLIManifestPath() error = %v", err)
	}
	if want := filepath.Join("/nonexistent/repo", "scenarios", "demo", "cli", "manifest.json"); cli != want {
		t.Fatalf("cli fallback = %q, want %q", cli, want)
	}
	service, err := ScenarioServiceManifestPath("/nonexistent/repo", "demo")
	if err != nil {
		t.Fatalf("ScenarioServiceManifestPath() error = %v", err)
	}
	if want := filepath.Join("/nonexistent/repo", filepath.FromSlash("scenarios/demo/.vrooli/service.json")); service != want {
		t.Fatalf("service fallback = %q, want %q", service, want)
	}
}

func TestScenarioManifestPathRejectsEmpty(t *testing.T) {
	if _, err := ScenarioDocsManifestPath("/repo", ""); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
	if _, err := ScenarioCLIManifestPath("/repo", ""); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
	if _, err := ScenarioServiceManifestPath("/repo", ""); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
}

func TestScenarioManifestRel(t *testing.T) {
	root := fixtureRoot(t)
	docs, err := ScenarioDocsManifestRel(root)
	if err != nil {
		t.Fatalf("ScenarioDocsManifestRel() error = %v", err)
	}
	if docs != "docs/manifest.json" {
		t.Fatalf("ScenarioDocsManifestRel() = %q", docs)
	}
	cli, err := ScenarioCLIManifestRel(root)
	if err != nil {
		t.Fatalf("ScenarioCLIManifestRel() error = %v", err)
	}
	if cli != "cli/manifest.json" {
		t.Fatalf("ScenarioCLIManifestRel() = %q", cli)
	}
	service, err := ScenarioServiceManifestRel(root)
	if err != nil {
		t.Fatalf("ScenarioServiceManifestRel() error = %v", err)
	}
	if service != ".vrooli/service.json" {
		t.Fatalf("ScenarioServiceManifestRel() = %q", service)
	}
}

func TestScenarioManifestRelFallback(t *testing.T) {
	docs, err := ScenarioDocsManifestRel("/nonexistent/repo")
	if err != nil {
		t.Fatalf("ScenarioDocsManifestRel() error = %v", err)
	}
	if docs != "docs/manifest.json" {
		t.Fatalf("docs fallback = %q", docs)
	}
	cli, err := ScenarioCLIManifestRel("/nonexistent/repo")
	if err != nil {
		t.Fatalf("ScenarioCLIManifestRel() error = %v", err)
	}
	if cli != "cli/manifest.json" {
		t.Fatalf("cli fallback = %q", cli)
	}
}

func TestSchemaPath(t *testing.T) {
	root := fixtureRoot(t)
	got, err := SchemaPath(root, "cli-manifest.schema.json")
	if err != nil {
		t.Fatalf("SchemaPath() error = %v", err)
	}
	want := filepath.Join(root, ".vrooli", "schemas", "cli-manifest.schema.json")
	if got != want {
		t.Fatalf("SchemaPath() = %q, want %q", got, want)
	}
}

func TestSchemaPathFallback(t *testing.T) {
	got, err := SchemaPath("/nonexistent/repo", "scenario-docs-manifest.schema.json")
	if err != nil {
		t.Fatalf("SchemaPath() error = %v", err)
	}
	want := filepath.Join("/nonexistent/repo", ".vrooli", "schemas", "scenario-docs-manifest.schema.json")
	if got != want {
		t.Fatalf("SchemaPath() = %q, want %q", got, want)
	}
}

func TestSchemaPathRejectsBadName(t *testing.T) {
	cases := []string{"", "../escape.json", "sub/dir.json", `back\slash.json`}
	for _, name := range cases {
		if _, err := SchemaPath("/repo", name); err == nil {
			t.Errorf("expected error for schema name %q", name)
		}
	}
}
