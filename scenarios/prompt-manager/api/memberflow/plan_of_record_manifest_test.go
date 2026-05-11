package memberflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPlanOfRecordManifestRejectsUnknownFields(t *testing.T) {
	repoRoot := t.TempDir()
	writePORFile(t, repoRoot, "docs/team-a/manifest.json", `{
  "version": "1.0.0",
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [],
  "optionalFolder": ["typo/"]
}`)

	_, err := LoadPlanOfRecordManifest(filepath.Join(repoRoot, "docs", "team-a", "manifest.json"))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestLoadPlanOfRecordManifestAcceptsOptionalFolders(t *testing.T) {
	repoRoot := t.TempDir()
	writePORFile(t, repoRoot, "docs/team-a/manifest.json", `{
  "$schema": "../agent-system/team-plan-of-record.schema.json",
  "version": "1.0.0",
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [{
    "id": "taxonomies",
    "path": "taxonomies/",
    "packageType": "taxonomy",
    "requiredFiles": ["README.md", "taxonomy.json"],
    "optionalFolders": ["schemas/"],
    "packages": [{
      "id": "signal",
      "path": "signal/",
      "optionalFolders": ["examples/"]
    }]
  }]
}`)

	manifest, err := LoadPlanOfRecordManifest(filepath.Join(repoRoot, "docs", "team-a", "manifest.json"))
	if err != nil {
		t.Fatalf("LoadPlanOfRecordManifest: %v", err)
	}
	if got := manifest.Sections[0].OptionalFolders; len(got) != 1 || got[0] != "schemas/" {
		t.Fatalf("section optionalFolders not decoded: %#v", got)
	}
	if got := manifest.Sections[0].Packages[0].OptionalFolders; len(got) != 1 || got[0] != "examples/" {
		t.Fatalf("package optionalFolders not decoded: %#v", got)
	}
}

func TestLoadResolvedPlanOfRecordManifestMergesBaseContract(t *testing.T) {
	repoRoot := t.TempDir()
	writePORBase(t, repoRoot)
	writePORFile(t, repoRoot, "docs/team-a/manifest.json", `{
  "version": "1.0.0",
  "contract": {
    "kind": "team-plan-of-record",
    "schema": "team-plan-of-record/v1",
    "base": "docs/agent-system/team-plan-of-record.manifest.json",
    "team": "team-a"
  },
  "sections": [
    {
      "id": "entrypoint",
      "path": ".",
      "documents": [{
        "path": "README.md",
        "required": true,
        "validation": {"requiredHeadings": ["Custom Start"]}
      }]
    },
    {
      "id": "taxonomies",
      "path": "taxonomies/",
      "packages": [{"id": "signal", "path": "signal/"}]
    }
  ]
}`)

	manifest, err := LoadResolvedPlanOfRecordManifest(repoRoot, filepath.Join(repoRoot, "docs", "team-a", "manifest.json"))
	if err != nil {
		t.Fatalf("LoadResolvedPlanOfRecordManifest: %v", err)
	}

	entrypoint := requirePORSection(t, manifest, "entrypoint")
	if got := entrypoint.Documents[0].Validation.RequiredHeadings; len(got) != 1 || got[0] != "Custom Start" {
		t.Fatalf("child document validation should override base headings, got %#v", got)
	}
	governance := requirePORSection(t, manifest, "governance")
	if !governance.Required || len(governance.Documents) != 3 {
		t.Fatalf("base governance section not preserved: %#v", governance)
	}
	taxonomies := requirePORSection(t, manifest, "taxonomies")
	if taxonomies.PackageType != "taxonomy" {
		t.Fatalf("base packageType not inherited: %#v", taxonomies)
	}
	if len(taxonomies.Packages) != 1 || len(taxonomies.Packages[0].RequiredFiles) != 0 {
		t.Fatalf("child package should be present without forced copy of section defaults: %#v", taxonomies.Packages)
	}
}

func TestValidateAllPlanOfRecordsDoesNotRequireOperatingModelDiscovery(t *testing.T) {
	repoRoot := t.TempDir()
	writePORBase(t, repoRoot)
	writePORFile(t, repoRoot, "docs/team-a/manifest.json", `{
  "version": "1.0.0",
  "contract": {
    "kind": "team-plan-of-record",
    "schema": "team-plan-of-record/v1",
    "base": "docs/agent-system/team-plan-of-record.manifest.json",
    "team": "team-a"
  },
  "sections": [{
    "id": "entrypoint",
    "path": ".",
    "documents": [{
      "path": "README.md",
      "required": true,
      "validation": {"requiredHeadings": ["Start here for agents"]}
    }]
  }]
}`)
	writePORFile(t, repoRoot, "docs/team-a/README.md", "# Team A\n")

	findings := ValidateAllPlanOfRecords(repoRoot)
	assertPORFinding(t, findings, "por_required_heading_missing")
	assertPORFinding(t, findings, "por_required_section_missing")
}

func TestLoadResolvedPlanOfRecordManifestRejectsBaseCycle(t *testing.T) {
	repoRoot := t.TempDir()
	writePORFile(t, repoRoot, "docs/a/manifest.json", `{
  "version": "1.0.0",
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "base": "docs/b/manifest.json", "team": "a"},
  "sections": []
}`)
	writePORFile(t, repoRoot, "docs/b/manifest.json", `{
  "version": "1.0.0",
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "base": "docs/a/manifest.json", "team": "b"},
  "sections": []
}`)

	_, err := LoadResolvedPlanOfRecordManifest(repoRoot, filepath.Join(repoRoot, "docs", "a", "manifest.json"))
	if err == nil || !strings.Contains(err.Error(), "base cycle") {
		t.Fatalf("expected base cycle error, got %v", err)
	}
}

func TestTeamPlanOfRecordSchemaJSONIsValid(t *testing.T) {
	repoRoot := findRepoRootForPORTest(t)
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "agent-system", "team-plan-of-record.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema must be valid JSON: %v", err)
	}
	if schema["$schema"] == "" || schema["$id"] == "" {
		t.Fatalf("schema missing $schema or $id: %#v", schema)
	}
}

func TestValidateAllPlanOfRecordsRepoBaseline(t *testing.T) {
	repoRoot := findRepoRootForPORTest(t)
	findings := ValidateAllPlanOfRecords(repoRoot)
	var errors []OperatingGraphFinding
	for _, finding := range findings {
		if finding.Severity == string(SeverityError) {
			errors = append(errors, finding)
		}
	}
	if len(errors) > 0 {
		t.Fatalf("expected migrated plan-of-record baseline to have no errors, got %#v", errors)
	}
}

func writePORBase(t *testing.T, repoRoot string) {
	t.Helper()
	writePORFile(t, repoRoot, "docs/agent-system/team-plan-of-record.manifest.json", `{
  "version": "1.0.0",
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1"},
  "sections": [
    {
      "id": "entrypoint",
      "path": ".",
      "required": true,
      "documents": [{"path": "README.md", "required": true}]
    },
    {
      "id": "operating",
      "path": "operating/",
      "required": true,
      "documents": [{"path": "OPERATING_MODEL.md", "required": true}]
    },
    {
      "id": "taxonomies",
      "path": "taxonomies/",
      "packageType": "taxonomy",
      "requiredFiles": ["README.md", "taxonomy.json"],
      "optionalFolders": ["schemas/"]
    },
    {
      "id": "governance",
      "path": "governance/",
      "required": true,
      "documents": [
        {"path": "editing.md", "required": true},
        {"path": "adoption-validation.md", "required": true},
        {"path": "changelog.md", "required": true}
      ]
    }
  ]
}`)
}

func writePORFile(t *testing.T, repoRoot, rel, body string) {
	t.Helper()
	path := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func requirePORSection(t *testing.T, manifest PlanOfRecordManifest, id string) PlanOfRecordSection {
	t.Helper()
	for _, section := range manifest.Sections {
		if section.ID == id {
			return section
		}
	}
	t.Fatalf("missing section %q in %#v", id, manifest.Sections)
	return PlanOfRecordSection{}
}

func assertPORFinding(t *testing.T, findings []OperatingGraphFinding, rule string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Rule == rule {
			return
		}
	}
	t.Fatalf("expected finding %q in %#v", rule, findings)
}

func findRepoRootForPORTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "docs", "agent-system", "team-plan-of-record.manifest.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("repo root not found")
		}
		dir = parent
	}
}
