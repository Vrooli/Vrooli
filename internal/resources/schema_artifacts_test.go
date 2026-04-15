package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	testresource "github.com/vrooli/vrooli/packages/testkit-go/resourcefixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestSyncAndValidateSchemaArtifacts(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDriver("docker-service"),
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

func TestValidateSchemaArtifactsDetectsMissingScenarioResourceReferences(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDriver("docker-service"),
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

func TestValidateSchemaArtifactsDetectsStaleFiles(t *testing.T) {
	root := t.TempDir()
	testresource.WriteResourceManifest(t, root, "postgres", testresource.ResourceManifest(
		"postgres",
		testresource.WithResourceTemplate("docker-service"),
		testresource.WithResourceDriver("docker-service"),
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
