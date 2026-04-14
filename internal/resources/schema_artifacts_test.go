package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncAndValidateSchemaArtifacts(t *testing.T) {
	root := t.TempDir()
	writeSchemaArtifactResource(t, root, "postgres", map[string]any{
		"$schema":          "../../.vrooli/schemas/resource.schema.json",
		"name":             "postgres",
		"display_name":     "PostgreSQL",
		"description":      "Database",
		"template":         "docker-service",
		"driver":           "docker-service",
		"portability_tier": "full",
		"runtime": map[string]any{
			"image": "postgres:16-alpine",
		},
		"dependency_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"database": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
	})
	writeScenarioManifest(t, root, "alpha", map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				"postgres": map[string]any{"enabled": true},
			},
		},
	})

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
	writeSchemaArtifactResource(t, root, "postgres", map[string]any{
		"$schema":          "../../.vrooli/schemas/resource.schema.json",
		"name":             "postgres",
		"display_name":     "PostgreSQL",
		"description":      "Database",
		"template":         "docker-service",
		"driver":           "docker-service",
		"portability_tier": "full",
		"runtime": map[string]any{
			"image": "postgres:16-alpine",
		},
	})
	writeScenarioManifest(t, root, "alpha", map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				"n8n": map[string]any{"enabled": true},
			},
		},
	})

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
	writeSchemaArtifactResource(t, root, "postgres", map[string]any{
		"$schema":          "../../.vrooli/schemas/resource.schema.json",
		"name":             "postgres",
		"display_name":     "PostgreSQL",
		"description":      "Database",
		"template":         "docker-service",
		"driver":           "docker-service",
		"portability_tier": "full",
		"runtime": map[string]any{
			"image": "postgres:16-alpine",
		},
	})
	writeScenarioManifest(t, root, "alpha", map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				"postgres": map[string]any{"enabled": true},
			},
		},
	})
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

func writeSchemaArtifactResource(t *testing.T, root, name string, payload map[string]any) {
	t.Helper()
	path := filepath.Join(root, "resources", name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir resource: %v", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal resource payload: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(path, "resource.json"), data, 0o644); err != nil {
		t.Fatalf("write resource.json: %v", err)
	}
}

func writeScenarioManifest(t *testing.T, root, name string, payload map[string]any) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir scenario: %v", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal scenario payload: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(path, "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}
