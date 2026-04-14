package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const (
	resourceDefinitionsRelPath = ".vrooli/schemas/resource-definitions.json"
)

type SchemaArtifactIssue struct {
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type ScenarioResourceReference struct {
	Scenario     string `json:"scenario"`
	Resource     string `json:"resource"`
	ManifestPath string `json:"manifest_path"`
}

type ResourceSchemaValidationReport struct {
	Passed            bool                        `json:"passed"`
	ResourceCount     int                         `json:"resource_count"`
	DefinitionPath    string                      `json:"definition_path"`
	ArtifactIssues    []SchemaArtifactIssue       `json:"artifact_issues,omitempty"`
	MissingReferences []ScenarioResourceReference `json:"missing_references,omitempty"`
}

type ResourceSchemaSyncReport struct {
	Passed            bool                        `json:"passed"`
	ResourceCount     int                         `json:"resource_count"`
	DefinitionPath    string                      `json:"definition_path"`
	WrittenPaths      []string                    `json:"written_paths,omitempty"`
	MissingReferences []ScenarioResourceReference `json:"missing_references,omitempty"`
}

type resourceSchemaDocument struct {
	Schema      string         `json:"$schema"`
	ID          string         `json:"$id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Definitions map[string]any `json:"definitions"`
	Catalog     map[string]any `json:"resourceCatalog"`
}

func (c *Controller) ValidateSchemaArtifacts() (ResourceSchemaValidationReport, error) {
	return ValidateSchemaArtifacts(c.Root)
}

func (c *Controller) SyncSchemaArtifacts() (ResourceSchemaSyncReport, error) {
	return SyncSchemaArtifacts(c.Root)
}

func ValidateSchemaArtifacts(root string) (ResourceSchemaValidationReport, error) {
	docBytes, resourceCount, err := buildSchemaArtifacts(root)
	if err != nil {
		return ResourceSchemaValidationReport{}, err
	}
	report := ResourceSchemaValidationReport{
		Passed:         true,
		ResourceCount:  resourceCount,
		DefinitionPath: filepath.Join(root, filepath.FromSlash(resourceDefinitionsRelPath)),
	}
	got, err := os.ReadFile(report.DefinitionPath)
	if err != nil {
		report.Passed = false
		report.ArtifactIssues = append(report.ArtifactIssues, SchemaArtifactIssue{
			Path:    report.DefinitionPath,
			Message: fmt.Sprintf("read artifact: %v", err),
		})
	} else if !bytes.Equal(got, docBytes) {
		report.Passed = false
		report.ArtifactIssues = append(report.ArtifactIssues, SchemaArtifactIssue{
			Path:    report.DefinitionPath,
			Message: "artifact is stale; run `vrooli resource schema sync`",
		})
	}
	missing, err := findMissingScenarioResourceReferences(root)
	if err != nil {
		return ResourceSchemaValidationReport{}, err
	}
	if len(missing) > 0 {
		report.Passed = false
		report.MissingReferences = missing
	}
	return report, nil
}

func SyncSchemaArtifacts(root string) (ResourceSchemaSyncReport, error) {
	docBytes, resourceCount, err := buildSchemaArtifacts(root)
	if err != nil {
		return ResourceSchemaSyncReport{}, err
	}
	report := ResourceSchemaSyncReport{
		Passed:         true,
		ResourceCount:  resourceCount,
		DefinitionPath: filepath.Join(root, filepath.FromSlash(resourceDefinitionsRelPath)),
	}
	if err := os.MkdirAll(filepath.Dir(report.DefinitionPath), 0o755); err != nil {
		return ResourceSchemaSyncReport{}, fmt.Errorf("mkdir %s: %w", filepath.Dir(report.DefinitionPath), err)
	}
	if err := os.WriteFile(report.DefinitionPath, docBytes, 0o644); err != nil {
		return ResourceSchemaSyncReport{}, fmt.Errorf("write %s: %w", report.DefinitionPath, err)
	}
	report.WrittenPaths = append(report.WrittenPaths, report.DefinitionPath)
	missing, err := findMissingScenarioResourceReferences(root)
	if err != nil {
		return ResourceSchemaSyncReport{}, err
	}
	if len(missing) > 0 {
		report.Passed = false
		report.MissingReferences = missing
	}
	return report, nil
}

func buildSchemaArtifacts(root string) ([]byte, int, error) {
	manifests, err := loadSchemaArtifactManifests(root)
	if err != nil {
		return nil, 0, err
	}
	definitions := make(map[string]any, len(manifests))
	catalogProperties := make(map[string]any, len(manifests))

	for _, item := range manifests {
		schemaMap := map[string]any{}
		if len(item.Manifest.DependencySchema) > 0 {
			if err := json.Unmarshal(item.Manifest.DependencySchema, &schemaMap); err != nil {
				return nil, 0, fmt.Errorf("parse dependency_schema for %s: %w", item.Name, err)
			}
		}
		if schemaMap == nil {
			schemaMap = map[string]any{}
		}
		if _, ok := schemaMap["type"]; !ok {
			schemaMap["type"] = "object"
		}
		if _, ok := schemaMap["properties"]; !ok {
			schemaMap["properties"] = map[string]any{}
		}
		if _, ok := schemaMap["additionalProperties"]; !ok {
			schemaMap["additionalProperties"] = true
		}
		if title, _ := schemaMap["title"].(string); strings.TrimSpace(title) == "" {
			schemaMap["title"] = firstNonEmpty(item.Manifest.DisplayName, item.Manifest.Name, item.Name)
		}
		if description, _ := schemaMap["description"].(string); strings.TrimSpace(description) == "" {
			schemaMap["description"] = item.Manifest.Description
		}
		schemaMap["resourceName"] = item.Name
		definitions[item.Name] = schemaMap
		catalogProperties[item.Name] = map[string]any{
			"allOf": []any{
				map[string]any{"$ref": "../resources.schema.json#/definitions/resourceConfig"},
				map[string]any{"$ref": "#/definitions/resourceSchemas/" + item.Name},
			},
		}
	}

	definitionDoc := resourceSchemaDocument{
		Schema:      "http://json-schema.org/draft-07/schema#",
		ID:          "https://vrooli.com/schemas/resource-definitions.json",
		Title:       "Resource Definitions for IDE Support",
		Description: "Auto-generated aggregated schemas for all available resources",
		Definitions: map[string]any{
			"resourceSchemas": definitions,
		},
		Catalog: map[string]any{
			"type":       "object",
			"properties": catalogProperties,
		},
	}
	definitionBytes, err := json.MarshalIndent(definitionDoc, "", "  ")
	if err != nil {
		return nil, 0, fmt.Errorf("marshal resource definitions: %w", err)
	}
	definitionBytes = append(definitionBytes, '\n')
	return definitionBytes, len(manifests), nil
}

type loadedSchemaArtifactManifest struct {
	Name     string
	Manifest ResourceManifest
}

func loadSchemaArtifactManifests(root string) ([]loadedSchemaArtifactManifest, error) {
	resourceRoot := filepath.Join(root, "resources")
	entries, err := os.ReadDir(resourceRoot)
	if err != nil {
		return nil, fmt.Errorf("read resources dir: %w", err)
	}
	items := make([]loadedSchemaArtifactManifest, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		manifestPath := filepath.Join(resourceRoot, name, "resource.json")
		manifest, err := manifestpkg.Load(manifestPath)
		if err != nil {
			return nil, err
		}
		items = append(items, loadedSchemaArtifactManifest{Name: name, Manifest: manifest})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func findMissingScenarioResourceReferences(root string) ([]ScenarioResourceReference, error) {
	resourceEntries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		return nil, fmt.Errorf("read resources dir: %w", err)
	}
	resourceNames := make([]string, 0, len(resourceEntries))
	resourceSet := make(map[string]struct{}, len(resourceEntries))
	for _, entry := range resourceEntries {
		if !entry.IsDir() {
			continue
		}
		resourceSet[entry.Name()] = struct{}{}
		resourceNames = append(resourceNames, entry.Name())
	}
	slices.Sort(resourceNames)

	scenarioRoot := filepath.Join(root, "scenarios")
	scenarioEntries, err := os.ReadDir(scenarioRoot)
	if err != nil {
		return nil, fmt.Errorf("read scenarios dir: %w", err)
	}
	var missing []ScenarioResourceReference
	for _, entry := range scenarioEntries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(scenarioRoot, entry.Name(), ".vrooli", "service.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read scenario manifest %s: %w", manifestPath, err)
		}
		var payload struct {
			Dependencies struct {
				Resources map[string]json.RawMessage `json:"resources"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("parse scenario manifest %s: %w", manifestPath, err)
		}
		for resourceName := range payload.Dependencies.Resources {
			if _, ok := resourceSet[resourceName]; ok {
				continue
			}
			missing = append(missing, ScenarioResourceReference{
				Scenario:     entry.Name(),
				Resource:     resourceName,
				ManifestPath: manifestPath,
			})
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Scenario == missing[j].Scenario {
			return missing[i].Resource < missing[j].Resource
		}
		return missing[i].Scenario < missing[j].Scenario
	})
	return missing, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
