package resources

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestSyncAndValidateSchemaArtifacts(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourcesSchema(t, root)
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceDisplayName("PostgreSQL"),
		testresource.WithResourceDescription("Database"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
		}),
		testresource.WithResourceDependencySchema(json.RawMessage(`{
  "type": "object",
  "properties": {
    "database": {
      "type": "string"
    }
  },
  "additionalProperties": false
}`)),
	))
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true},
			},
		}),
	))

	syncReport, err := SyncSchemaArtifacts(root)
	if err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}
	if !syncReport.Passed {
		t.Fatalf("sync report = %+v", syncReport)
	}

	validateReport, err := ValidateSchemaArtifacts(root)
	if err != nil {
		t.Fatalf("ValidateSchemaArtifacts: %v", err)
	}
	if !validateReport.Passed {
		t.Fatalf("validate report = %+v", validateReport)
	}
	if syncReport.DefinitionPath == "" {
		t.Fatalf("definition path missing: %+v", syncReport)
	}
	if len(syncReport.WrittenPaths) != 1 || syncReport.WrittenPaths[0] != syncReport.DefinitionPath {
		t.Fatalf("written paths = %+v", syncReport.WrittenPaths)
	}
	if len(validateReport.ArtifactIssues) != 0 {
		t.Fatalf("unexpected artifact issues = %+v", validateReport.ArtifactIssues)
	}

	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "schemas", "resource-definitions.json"))
	if err != nil {
		t.Fatalf("read resource-definitions: %v", err)
	}
	var payload struct {
		Definitions struct {
			ResourceSchemas map[string]map[string]any `json:"resourceSchemas"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal resource-definitions: %v", err)
	}
	if _, ok := payload.Definitions.ResourceSchemas["postgres"]; !ok {
		t.Fatalf("postgres schema missing: %#v", payload.Definitions.ResourceSchemas)
	}
}

// TestGeneratedCatalogAcceptsSharedDependencyKeys is the regression test for two
// defects that together made every scenario manifest unvalidatable.
//
// First, the catalog emitted "../resources.schema.json#/…", which resolves one
// directory above where that file lives, so no standards-compliant validator
// could compile any schema reaching the catalog — service.schema.json included.
//
// Second, each catalog entry composes resourceConfig with a resource-specific
// schema through allOf. JSON Schema evaluates allOf branches independently, so a
// resource schema closed with additionalProperties:false rejected the shared
// governance keys the other branch supplies. 162 of the repository's 405
// manifest violations came from that single composition mistake.
//
// The fixture's postgres dependency_schema is closed, so this exercises both.
func TestGeneratedCatalogAcceptsSharedDependencyKeys(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourcesSchema(t, root)
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{Image: "postgres:16-alpine"}),
		testresource.WithResourceDependencySchema(json.RawMessage(`{
  "type": "object",
  "properties": {
    "database": {
      "type": "string"
    }
  },
  "additionalProperties": false
}`)),
	))
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true},
			},
		}),
	))
	if _, err := SyncSchemaArtifacts(root); err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}

	schemaDir := filepath.Join(root, ".vrooli", "schemas")
	catalogBytes, err := os.ReadFile(filepath.Join(schemaDir, "resource-definitions.json"))
	if err != nil {
		t.Fatalf("read resource-definitions: %v", err)
	}
	if strings.Contains(string(catalogBytes), "../resources.schema.json") {
		t.Fatalf("catalog emitted a parent-relative ref; it must point at the sibling schema")
	}

	// Register each schema only under its canonical id. An extra alias that
	// swallows a parent-relative ref is exactly what hid the broken ref before.
	compiler := jsonschema.NewCompiler()
	for _, name := range []string{"resource-definitions.json", "resources.schema.json"} {
		data, readErr := os.ReadFile(filepath.Join(schemaDir, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		if addErr := compiler.AddResource("https://vrooli.com/schemas/"+name, bytes.NewReader(data)); addErr != nil {
			t.Fatalf("add %s: %v", name, addErr)
		}
	}
	catalog, err := compiler.Compile("https://vrooli.com/schemas/resource-definitions.json#/resourceCatalog")
	if err != nil {
		t.Fatalf("compile catalog: %v", err)
	}

	// A real dependency declaration: governance keys from resourceConfig plus a
	// resource-specific key from the closed per-resource schema.
	valid := map[string]any{
		"postgres": map[string]any{
			"enabled":  true,
			"required": true,
			"type":     "postgres",
			"purpose":  "primary store",
			"database": "alpha",
		},
	}
	if err := catalog.Validate(valid); err != nil {
		t.Fatalf("catalog rejected a valid dependency declaration: %v", err)
	}

	// Strictness must survive the fix: an unknown key is still a violation.
	unknown := map[string]any{
		"postgres": map[string]any{"enabled": true, "definitely_not_a_key": true},
	}
	if err := catalog.Validate(unknown); err == nil {
		t.Fatalf("catalog accepted an unknown resource key; per-resource schemas must stay closed")
	}
}

func TestValidateSchemaArtifactsDetectsMissingScenarioResourceReferences(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourcesSchema(t, root)
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceDisplayName("PostgreSQL"),
		testresource.WithResourceDescription("Database"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
		}),
	))
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"n8n": {Enabled: true},
			},
		}),
	))

	report, err := SyncSchemaArtifacts(root)
	if err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}
	if report.Passed {
		t.Fatalf("expected failed sync report, got %+v", report)
	}
	if len(report.MissingReferences) != 1 || report.MissingReferences[0].Resource != "n8n" {
		t.Fatalf("missing refs = %+v", report.MissingReferences)
	}

	validateReport, err := ValidateSchemaArtifacts(root)
	if err != nil {
		t.Fatalf("ValidateSchemaArtifacts: %v", err)
	}
	if validateReport.Passed {
		t.Fatalf("expected failed validation, got %+v", validateReport)
	}
}

func TestFindMissingScenarioResourceReferencesToleratesUnknownManifestFields(t *testing.T) {
	root := t.TempDir()
	testkitgo.WriteFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres"}`)
	testkitgo.WriteFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{
  "dependencies": {"resources": {
    "postgres": {"enabled": true, "future_dependency_field": true},
    "n8n": {"enabled": true}
  }},
  "future_manifest_field": {"version": 2}
}`)

	missing, err := findMissingScenarioResourceReferences(root)
	if err != nil {
		t.Fatalf("findMissingScenarioResourceReferences: %v", err)
	}
	if len(missing) != 1 || missing[0].Scenario != "alpha" || missing[0].Resource != "n8n" {
		t.Fatalf("missing references = %+v", missing)
	}
}

func TestValidateSchemaArtifactsDetectsStaleFiles(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourcesSchema(t, root)
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("external-cli"),
		testresource.WithResourceDriver("external-cli"),
		testresource.WithResourceDisplayName("PostgreSQL"),
		testresource.WithResourceDescription("Database"),
		testresource.WithResourceRuntime(manifestpkg.ResourceRuntime{
			Image: "postgres:16-alpine",
		}),
	))
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"postgres": {Enabled: true},
			},
		}),
	))
	if _, err := SyncSchemaArtifacts(root); err != nil {
		t.Fatalf("SyncSchemaArtifacts: %v", err)
	}
	definitionPath := filepath.Join(root, ".vrooli", "schemas", "resource-definitions.json")
	if err := os.WriteFile(definitionPath, []byte("{\"stale\":true}\n"), 0o644); err != nil {
		t.Fatalf("write stale definitions: %v", err)
	}
	report, err := ValidateSchemaArtifacts(root)
	if err != nil {
		t.Fatalf("ValidateSchemaArtifacts: %v", err)
	}
	if report.Passed {
		t.Fatalf("expected stale artifact failure, got %+v", report)
	}
	if len(report.ArtifactIssues) == 0 || !strings.Contains(report.ArtifactIssues[0].Message, "stale") {
		t.Fatalf("artifact issues = %+v", report.ArtifactIssues)
	}
}
