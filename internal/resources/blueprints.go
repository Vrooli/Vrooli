package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const blueprintDirPath = ".vrooli/resource-blueprints"

type Blueprint struct {
	Schema              string                   `json:"$schema,omitempty"`
	Name                string                   `json:"name"`
	DisplayName         string                   `json:"display_name"`
	Category            string                   `json:"category"`
	Summary             string                   `json:"summary"`
	WhyItMatters        string                   `json:"why_it_matters"`
	WhenToUse           []string                 `json:"when_to_use"`
	ExampleScenarios    []string                 `json:"example_scenarios,omitempty"`
	IntegrationKind     string                   `json:"integration_kind"`
	PlatformSupport     BlueprintPlatformSupport `json:"platform_support"`
	Prerequisites       []string                 `json:"prerequisites,omitempty"`
	Dependencies        []string                 `json:"dependencies,omitempty"`
	SuggestedTemplate   string                   `json:"suggested_template"`
	ImplementationNotes []string                 `json:"implementation_notes"`
	OperationalNotes    []string                 `json:"operational_notes"`
	Risks               []string                 `json:"risks"`
	Status              string                   `json:"status"`
	ReplacementFor      []string                 `json:"replacement_for,omitempty"`
	References          []BlueprintReference     `json:"references"`
	LastReviewed        string                   `json:"last_reviewed"`
}

type BlueprintPlatformSupport struct {
	PortabilityTier string `json:"portability_tier"`
	Notes           string `json:"notes"`
	Linux           string `json:"linux"`
	MacOS           string `json:"macos"`
	Windows         string `json:"windows"`
}

type BlueprintReference struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type BlueprintSummary struct {
	Name              string `json:"name"`
	DisplayName       string `json:"display_name"`
	Category          string `json:"category"`
	Status            string `json:"status"`
	IntegrationKind   string `json:"integration_kind"`
	SuggestedTemplate string `json:"suggested_template"`
	LastReviewed      string `json:"last_reviewed"`
	Summary           string `json:"summary"`
}

type BlueprintValidationReport struct {
	Blueprints []BlueprintSummary `json:"blueprints"`
	Count      int                `json:"count"`
}

var allowedBlueprintTemplateRules = map[string][]string{
	"docker-service":  {"docker-service"},
	"compose-service": {"compose-service"},
	"external-cli":    {"external-cli"},
	"cloud-api":       {"cloud-api"},
	"desktop-app":     {"desktop-app"},
	"manual":          {"manual-resource"},
	"hardware":        {"manual-resource"},
	"library":         {"manual-resource", "external-cli"},
}

func (c *Controller) ListBlueprints() ([]Blueprint, error) {
	blueprints, err := c.loadBlueprints()
	if err != nil {
		return nil, err
	}
	items := make([]Blueprint, 0, len(blueprints))
	for _, item := range blueprints {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (c *Controller) Blueprint(name string) (Blueprint, error) {
	blueprints, err := c.loadBlueprints()
	if err != nil {
		return Blueprint{}, err
	}
	item, ok := blueprints[strings.TrimSpace(name)]
	if !ok {
		return Blueprint{}, fmt.Errorf("resource blueprint %q not found", name)
	}
	return item, nil
}

func (c *Controller) SearchBlueprints(query string) ([]Blueprint, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("blueprint search query cannot be empty")
	}
	items, err := c.ListBlueprints()
	if err != nil {
		return nil, err
	}
	matches := make([]Blueprint, 0)
	for _, item := range items {
		haystack := []string{
			item.Name,
			item.DisplayName,
			item.Category,
			item.Summary,
			item.WhyItMatters,
			item.IntegrationKind,
			item.SuggestedTemplate,
		}
		for _, text := range haystack {
			if strings.Contains(strings.ToLower(text), query) {
				matches = append(matches, item)
				break
			}
		}
	}
	return matches, nil
}

func (c *Controller) ValidateBlueprints() (BlueprintValidationReport, error) {
	items, err := c.ListBlueprints()
	if err != nil {
		return BlueprintValidationReport{}, err
	}
	report := BlueprintValidationReport{
		Blueprints: make([]BlueprintSummary, 0, len(items)),
		Count:      len(items),
	}
	for _, item := range items {
		report.Blueprints = append(report.Blueprints, blueprintSummary(item))
	}
	return report, nil
}

func (c *Controller) loadBlueprints() (map[string]Blueprint, error) {
	path := filepath.Join(c.Root, filepath.FromSlash(blueprintDirPath))
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Blueprint{}, nil
		}
		return nil, fmt.Errorf("read blueprint directory %s: %w", path, err)
	}

	items := make(map[string]Blueprint, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		fullPath := filepath.Join(path, entry.Name())
		item, err := loadBlueprintFile(fullPath)
		if err != nil {
			return nil, err
		}
		expectedName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if item.Name != expectedName {
			return nil, fmt.Errorf("blueprint %s: filename %q does not match name %q", fullPath, expectedName, item.Name)
		}
		if _, exists := items[item.Name]; exists {
			return nil, fmt.Errorf("duplicate resource blueprint %q", item.Name)
		}
		items[item.Name] = item
	}
	return items, nil
}

func loadBlueprintFile(path string) (Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Blueprint{}, fmt.Errorf("read blueprint %s: %w", path, err)
	}
	var item Blueprint
	if err := json.Unmarshal(data, &item); err != nil {
		return Blueprint{}, fmt.Errorf("parse blueprint %s: %w", path, err)
	}
	if err := validateBlueprint(item); err != nil {
		return Blueprint{}, fmt.Errorf("validate blueprint %s: %w", path, err)
	}
	return item, nil
}

func validateBlueprint(item Blueprint) error {
	if !isKebabCase(item.Name) {
		return fmt.Errorf("name must be lowercase kebab-case")
	}
	if strings.TrimSpace(item.DisplayName) == "" {
		return fmt.Errorf("display_name is required")
	}
	if strings.TrimSpace(item.Category) == "" {
		return fmt.Errorf("category is required")
	}
	if strings.TrimSpace(item.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	if strings.TrimSpace(item.WhyItMatters) == "" {
		return fmt.Errorf("why_it_matters is required")
	}
	if len(item.WhenToUse) == 0 {
		return fmt.Errorf("when_to_use must contain at least one entry")
	}
	if !isAllowedValue(item.IntegrationKind, []string{"docker-service", "compose-service", "external-cli", "cloud-api", "library", "desktop-app", "hardware", "manual"}) {
		return fmt.Errorf("integration_kind %q is invalid", item.IntegrationKind)
	}
	if !isAllowedValue(item.SuggestedTemplate, AllowedSuggestedTemplates()) {
		return fmt.Errorf("suggested_template %q is invalid", item.SuggestedTemplate)
	}
	if err := validateBlueprintSuggestedTemplate(item.IntegrationKind, item.SuggestedTemplate); err != nil {
		return err
	}
	if !isAllowedValue(item.Status, []string{"candidate", "validated", "prioritized"}) {
		return fmt.Errorf("status %q is invalid", item.Status)
	}
	if err := validatePlatformSupport(item.PlatformSupport); err != nil {
		return err
	}
	if len(item.ImplementationNotes) == 0 {
		return fmt.Errorf("implementation_notes must contain at least one entry")
	}
	if len(item.OperationalNotes) == 0 {
		return fmt.Errorf("operational_notes must contain at least one entry")
	}
	if len(item.Risks) == 0 {
		return fmt.Errorf("risks must contain at least one entry")
	}
	if len(item.References) == 0 {
		return fmt.Errorf("references must contain at least one entry")
	}
	for _, entry := range item.References {
		if !isAllowedValue(entry.Kind, []string{"doc", "resource", "scenario", "registry", "website", "note"}) {
			return fmt.Errorf("reference kind %q is invalid", entry.Kind)
		}
		if strings.TrimSpace(entry.Value) == "" {
			return fmt.Errorf("reference value cannot be empty")
		}
	}
	if _, err := time.Parse("2006-01-02", item.LastReviewed); err != nil {
		return fmt.Errorf("last_reviewed must be YYYY-MM-DD: %w", err)
	}
	return nil
}

func validateBlueprintSuggestedTemplate(integrationKind, suggestedTemplate string) error {
	allowed, ok := allowedBlueprintTemplateRules[integrationKind]
	if !ok {
		return fmt.Errorf("integration_kind %q has no template recommendation rule", integrationKind)
	}
	if !isAllowedValue(suggestedTemplate, allowed) {
		return fmt.Errorf(
			"suggested_template %q is invalid for integration_kind %q (allowed: %s)",
			suggestedTemplate,
			integrationKind,
			strings.Join(allowed, ", "),
		)
	}
	return nil
}

func validatePlatformSupport(item BlueprintPlatformSupport) error {
	if !isAllowedValue(item.PortabilityTier, []string{"full", "partial", "platform-specific"}) {
		return fmt.Errorf("platform_support.portability_tier %q is invalid", item.PortabilityTier)
	}
	if strings.TrimSpace(item.Notes) == "" {
		return fmt.Errorf("platform_support.notes is required")
	}
	for field, value := range map[string]string{
		"linux":   item.Linux,
		"macos":   item.MacOS,
		"windows": item.Windows,
	} {
		if !isAllowedValue(value, []string{"supported", "partial", "unsupported", "unknown"}) {
			return fmt.Errorf("platform_support.%s %q is invalid", field, value)
		}
	}
	return nil
}

func blueprintSummary(item Blueprint) BlueprintSummary {
	return BlueprintSummary{
		Name:              item.Name,
		DisplayName:       item.DisplayName,
		Category:          item.Category,
		Status:            item.Status,
		IntegrationKind:   item.IntegrationKind,
		SuggestedTemplate: item.SuggestedTemplate,
		LastReviewed:      item.LastReviewed,
		Summary:           item.Summary,
	}
}

func isAllowedValue(value string, allowed []string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func isKebabCase(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
			if i == 0 || i == len(value)-1 {
				return false
			}
		default:
			return false
		}
	}
	return !strings.Contains(value, "--")
}
