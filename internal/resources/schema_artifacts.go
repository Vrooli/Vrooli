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

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	resourceDefinitionsRelPath = ".vrooli/schemas/resource-definitions.json"
	resourcesSchemaRelPath     = ".vrooli/schemas/resources.schema.json"

	// resourceConfigRef points at a sibling file. resource-definitions.json and
	// resources.schema.json both live in .vrooli/schemas, so a "../" prefix
	// resolves one directory too high and leaves the ref unresolvable — which
	// makes every schema that reaches it, service.schema.json included,
	// uncompilable in a standards-compliant validator.
	resourceConfigRef = "resources.schema.json#/definitions/resourceConfig"
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
	if err := os.MkdirAll(filepath.Dir(report.DefinitionPath), tuning.PermDir); err != nil {
		return ResourceSchemaSyncReport{}, fmt.Errorf("mkdir %s: %w", filepath.Dir(report.DefinitionPath), err)
	}
	if err := os.WriteFile(report.DefinitionPath, docBytes, tuning.PermFile); err != nil {
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
	sharedProperties, err := loadResourceConfigPropertyNames(root)
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
		if closed, ok := schemaMap["additionalProperties"].(bool); ok && !closed {
			allowSharedResourceConfigProperties(schemaMap, sharedProperties)
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
				map[string]any{"$ref": resourceConfigRef},
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

// loadResourceConfigPropertyNames returns the property names declared by
// resources.schema.json#/definitions/resourceConfig — the shared dependency
// governance keys (enabled, required, type, purpose, startup_policy, …) every
// scenario may set on any resource dependency.
//
// A missing or empty definition is an error rather than a silent fallback: the
// composition below is only correct when the generator knows the real key set,
// and a quiet degradation here reproduces exactly the class of invisible schema
// breakage this artifact already suffered.
func loadResourceConfigPropertyNames(root string) ([]string, error) {
	path := filepath.Join(root, filepath.FromSlash(resourcesSchemaRelPath))
	raw, err := os.ReadFile(path) // #nosec G304 -- path is a fixed repository-relative schema artifact.
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", resourcesSchemaRelPath, err)
	}
	var doc struct {
		Definitions struct {
			ResourceConfig struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"resourceConfig"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", resourcesSchemaRelPath, err)
	}
	if len(doc.Definitions.ResourceConfig.Properties) == 0 {
		return nil, fmt.Errorf("%s declares no resourceConfig properties", resourcesSchemaRelPath)
	}
	names := make([]string, 0, len(doc.Definitions.ResourceConfig.Properties))
	for name := range doc.Definitions.ResourceConfig.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// allowSharedResourceConfigProperties repeats the shared resourceConfig property
// names inside a closed resource schema.
//
// Each catalog entry composes resourceConfig with a resource-specific schema
// through allOf, and JSON Schema evaluates every allOf branch against the whole
// instance independently — a branch's additionalProperties never sees the
// sibling branch's properties. A resource schema that closes itself with
// additionalProperties:false therefore rejects the governance keys the other
// branch supplies, which is why most scenario manifests failed validation on
// dependencies.resources.<name>.
//
// The placeholder is an empty schema, so resourceConfig still governs each
// value's shape and genuinely unknown keys stay rejected.
func allowSharedResourceConfigProperties(schemaMap map[string]any, shared []string) {
	properties, ok := schemaMap["properties"].(map[string]any)
	if !ok {
		properties = map[string]any{}
		schemaMap["properties"] = properties
	}
	for _, name := range shared {
		if _, exists := properties[name]; !exists {
			properties[name] = map[string]any{}
		}
	}
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
		if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
			continue
		} else if statErr != nil {
			return nil, fmt.Errorf("inspect resource %s manifest: %w", name, statErr)
		}
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
		if _, err := os.Stat(filepath.Join(root, "resources", entry.Name(), "resource.json")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("inspect resource %s manifest: %w", entry.Name(), err)
		}
		resourceSet[entry.Name()] = struct{}{}
		resourceNames = append(resourceNames, entry.Name())
	}
	slices.Sort(resourceNames)

	scenarioRoot := filepath.Join(root, repocontractmeta.ScenarioDir)
	scenarioEntries, err := os.ReadDir(scenarioRoot)
	if err != nil {
		return nil, fmt.Errorf("read scenarios dir: %w", err)
	}
	var missing []ScenarioResourceReference
	for _, entry := range scenarioEntries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(scenarioRoot, entry.Name(), repocontractmeta.ProjectConfigDir, "service.json")
		manifest, err := scenario.LoadServiceManifest(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read scenario manifest %s: %w", manifestPath, err)
		}
		for resourceName := range manifest.Dependencies.Resources {
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
