package resources

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const resourceTemplateBasePath = "templates/resources"

var (
	unresolvedResourceTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)
	canonicalResourceTemplateNames    = []string{
		"cloud-api",
		"compose-service",
		"desktop-app",
		"docker-service",
		"external-cli",
		"manual-resource",
	}
)

var resourceTemplateRequiredFiles = []string{
	"README.md",
	"resource.json",
	"config/defaults.json",
	"config/schema.json",
	"test/smoke.json",
	"test/integration.json",
	"docs/OPERATIONS.md",
}

type ResourceTemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type ResourceTemplateManifest struct {
	Name                 string                         `json:"name,omitempty"`
	DisplayName          string                         `json:"displayName,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Driver               string                         `json:"driver,omitempty"`
	RequiredVars         map[string]ResourceTemplateVar `json:"requiredVars,omitempty"`
	OptionalVars         map[string]ResourceTemplateVar `json:"optionalVars,omitempty"`
	Docs                 map[string]string              `json:"docs,omitempty"`
	PlatformExpectations []string                       `json:"platformExpectations,omitempty"`
	Transitional         bool                           `json:"transitional,omitempty"`
}

type ResourceTemplateInfo struct {
	Name     string                   `json:"name"`
	Path     string                   `json:"path"`
	Manifest ResourceTemplateManifest `json:"manifest"`
}

type ResourceTemplateSummary struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	Driver       string `json:"driver"`
	Transitional bool   `json:"transitional"`
	Description  string `json:"description"`
}

type ResourceTemplateValidationReport struct {
	Templates []ResourceTemplateSummary `json:"templates"`
	Count     int                       `json:"count"`
}

type ResourceTemplateGenerateRequest struct {
	TemplateName  string
	BlueprintName string
	Destination   string
	Force         bool
	DryRun        bool
	Values        map[string]string
}

type ResourceTemplateGenerateReport struct {
	Template      ResourceTemplateSummary `json:"template"`
	BlueprintName string                  `json:"blueprint_name,omitempty"`
	Destination   string                  `json:"destination"`
	Values        map[string]string       `json:"values"`
	Files         []string                `json:"files"`
	DryRun        bool                    `json:"dry_run"`
}

func AllowedSuggestedTemplates() []string {
	return copyStringSlice(canonicalResourceTemplateNames)
}

func (c *Controller) ListResourceTemplates() ([]ResourceTemplateInfo, error) {
	baseDir := filepath.Join(c.Root, filepath.FromSlash(resourceTemplateBasePath))
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("read resource templates %s: %w", baseDir, err)
	}

	templates := make([]ResourceTemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := c.ResourceTemplate(entry.Name())
		if err != nil {
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func (c *Controller) ResourceTemplate(name string) (ResourceTemplateInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResourceTemplateInfo{}, fmt.Errorf("resource template name cannot be empty")
	}

	templateDir := filepath.Join(c.Root, filepath.FromSlash(resourceTemplateBasePath), name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ResourceTemplateInfo{}, fmt.Errorf("resource template %q not found", name)
		}
		return ResourceTemplateInfo{}, fmt.Errorf("stat resource template %s: %w", templateDir, err)
	}
	if !info.IsDir() {
		return ResourceTemplateInfo{}, fmt.Errorf("resource template path is not a directory: %s", templateDir)
	}

	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("read resource template manifest %s: %w", name, err)
	}
	var manifest ResourceTemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("parse resource template manifest %s: %w", name, err)
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if err := validateResourceTemplateManifest(manifest); err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("validate resource template %s: %w", name, err)
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]ResourceTemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]ResourceTemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
	if err := c.validateResourceTemplateAssets(templateDir, manifest); err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("validate resource template assets %s: %w", name, err)
	}

	return ResourceTemplateInfo{
		Name:     name,
		Path:     templateDir,
		Manifest: manifest,
	}, nil
}

func (c *Controller) ValidateResourceTemplates() (ResourceTemplateValidationReport, error) {
	templates, err := c.ListResourceTemplates()
	if err != nil {
		return ResourceTemplateValidationReport{}, err
	}

	seen := make(map[string]struct{}, len(templates))
	report := ResourceTemplateValidationReport{
		Templates: make([]ResourceTemplateSummary, 0, len(templates)),
		Count:     len(templates),
	}
	for _, item := range templates {
		seen[item.Name] = struct{}{}
		report.Templates = append(report.Templates, resourceTemplateSummary(item))
	}
	for _, required := range canonicalResourceTemplateNames {
		if _, ok := seen[required]; !ok {
			return ResourceTemplateValidationReport{}, fmt.Errorf("missing canonical resource template %q", required)
		}
	}
	return report, nil
}

func (c *Controller) GenerateResourceTemplate(req ResourceTemplateGenerateRequest) (ResourceTemplateGenerateReport, error) {
	info, blueprint, err := c.resolveTemplateGenerationInputs(req)
	if err != nil {
		return ResourceTemplateGenerateReport{}, err
	}

	values := copyStringMap(req.Values)
	if values == nil {
		values = map[string]string{}
	}
	seedResourceTemplateValues(values, info, blueprint)
	applyResourceTemplateDefaults(values, info.Manifest)

	missing := missingResourceTemplateVars(values, info.Manifest.RequiredVars)
	if len(missing) > 0 {
		return ResourceTemplateGenerateReport{}, fmt.Errorf("missing required values: %s", strings.Join(missing, ", "))
	}

	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		destination = filepath.Join(c.Root, "resources", values["RESOURCE_NAME"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(c.Root, filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)

	files, err := previewResourceTemplateFiles(info.Path, destination, values)
	if err != nil {
		return ResourceTemplateGenerateReport{}, err
	}

	report := ResourceTemplateGenerateReport{
		Template:      resourceTemplateSummary(info),
		Destination:   destination,
		Values:        copyStringMap(values),
		Files:         files,
		DryRun:        req.DryRun,
		BlueprintName: "",
	}
	if blueprint != nil {
		report.BlueprintName = blueprint.Name
	}

	if req.DryRun {
		return report, nil
	}

	if stat, err := os.Stat(destination); err == nil && stat != nil {
		if !req.Force {
			return ResourceTemplateGenerateReport{}, fmt.Errorf("destination already exists: %s (use --force to overwrite)", destination)
		}
		if err := os.RemoveAll(destination); err != nil {
			return ResourceTemplateGenerateReport{}, fmt.Errorf("remove destination %s: %w", destination, err)
		}
	}

	if err := copyResourceTemplate(info.Path, destination, values); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	if err := verifyRenderedResourceTemplate(destination); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	if err := verifyGeneratedResourceManifest(destination); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	return report, nil
}

func (c *Controller) ResolveTemplateGenerationRequest(req ResourceTemplateGenerateRequest) (ResourceTemplateInfo, error) {
	info, _, err := c.resolveTemplateGenerationInputs(req)
	if err != nil {
		return ResourceTemplateInfo{}, err
	}
	return info, nil
}

func (c *Controller) resolveTemplateGenerationInputs(req ResourceTemplateGenerateRequest) (ResourceTemplateInfo, *Blueprint, error) {
	templateName := strings.TrimSpace(req.TemplateName)
	blueprintName := strings.TrimSpace(req.BlueprintName)
	var blueprint *Blueprint
	if blueprintName != "" {
		item, err := c.Blueprint(blueprintName)
		if err != nil {
			return ResourceTemplateInfo{}, nil, err
		}
		blueprint = &item
		if templateName == "" {
			templateName = item.SuggestedTemplate
		} else if templateName != item.SuggestedTemplate {
			return ResourceTemplateInfo{}, nil, fmt.Errorf("template %q does not match blueprint %q suggested template %q", templateName, item.Name, item.SuggestedTemplate)
		}
	}
	if templateName == "" {
		return ResourceTemplateInfo{}, nil, errors.New("resource template generate requires a template name or --from-blueprint")
	}
	info, err := c.ResourceTemplate(templateName)
	if err != nil {
		return ResourceTemplateInfo{}, nil, err
	}
	return info, blueprint, nil
}

func validateResourceTemplateManifest(manifest ResourceTemplateManifest) error {
	if !isAllowedValue(manifest.Name, AllowedSuggestedTemplates()) {
		return fmt.Errorf("name %q is invalid", manifest.Name)
	}
	if strings.TrimSpace(manifest.DisplayName) == "" {
		return fmt.Errorf("displayName is required")
	}
	if strings.TrimSpace(manifest.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if !isAllowedValue(strings.TrimSpace(manifest.Driver), []string{"docker-service", "compose-service", "external-cli", "cloud-api", "desktop-app", "manual"}) {
		return fmt.Errorf("driver %q is invalid", manifest.Driver)
	}
	for key, variable := range manifest.RequiredVars {
		if err := validateResourceTemplateVar(key, variable); err != nil {
			return err
		}
	}
	for key, variable := range manifest.OptionalVars {
		if err := validateResourceTemplateVar(key, variable); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) validateResourceTemplateAssets(templateDir string, manifest ResourceTemplateManifest) error {
	for _, relPath := range resourceTemplateRequiredFiles {
		path := filepath.Join(templateDir, filepath.FromSlash(relPath))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("missing required file %s", relPath)
			}
			return fmt.Errorf("stat %s: %w", relPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("required file %s is a directory", relPath)
		}
	}

	docKeys := make([]string, 0, len(manifest.Docs))
	for key := range manifest.Docs {
		docKeys = append(docKeys, key)
	}
	sort.Strings(docKeys)
	for _, key := range docKeys {
		relPath := strings.TrimSpace(manifest.Docs[key])
		if relPath == "" {
			return fmt.Errorf("docs entry %q cannot be empty", key)
		}
		path := filepath.Join(c.Root, filepath.FromSlash(relPath))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("docs entry %q points to missing path %s", key, relPath)
			}
			return fmt.Errorf("stat docs entry %q (%s): %w", key, relPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("docs entry %q points to a directory: %s", key, relPath)
		}
	}
	return nil
}

func validateResourceTemplateVar(key string, variable ResourceTemplateVar) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("template variable keys cannot be empty")
	}
	if strings.TrimSpace(variable.Flag) == "" {
		return fmt.Errorf("template variable %s must define a flag", key)
	}
	return nil
}

func resourceTemplateSummary(info ResourceTemplateInfo) ResourceTemplateSummary {
	return ResourceTemplateSummary{
		Name:         info.Name,
		DisplayName:  info.Manifest.DisplayName,
		Driver:       info.Manifest.Driver,
		Transitional: info.Manifest.Transitional,
		Description:  info.Manifest.Description,
	}
}

func seedResourceTemplateValues(values map[string]string, info ResourceTemplateInfo, blueprint *Blueprint) {
	values["RESOURCE_TEMPLATE"] = info.Name
	values["RESOURCE_DRIVER"] = info.Manifest.Driver
	values["CURRENT_DATE"] = time.Now().UTC().Format("2006-01-02")

	if blueprint == nil {
		return
	}
	setIfBlank(values, "RESOURCE_NAME", blueprint.Name)
	setIfBlank(values, "RESOURCE_DISPLAY_NAME", blueprint.DisplayName)
	setIfBlank(values, "RESOURCE_DESCRIPTION", blueprint.Summary)
	setIfBlank(values, "RESOURCE_CATEGORY", blueprint.Category)
	setIfBlank(values, "RESOURCE_PORTABILITY_TIER", blueprint.PlatformSupport.PortabilityTier)
	setIfBlank(values, "RESOURCE_BLUEPRINT_NAME", blueprint.Name)
	setIfBlank(values, "RESOURCE_BLUEPRINT_STATUS", blueprint.Status)
	setIfBlank(values, "RESOURCE_BLUEPRINT_INTEGRATION_KIND", blueprint.IntegrationKind)
}

func applyResourceTemplateDefaults(values map[string]string, manifest ResourceTemplateManifest) {
	optionalKeys := make([]string, 0, len(manifest.OptionalVars))
	for key := range manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) != "" {
			continue
		}
		values[key] = renderResourceTemplateString(manifest.OptionalVars[key].Default, values)
	}

	if strings.TrimSpace(values["RESOURCE_NAME"]) != "" {
		setIfBlank(values, "RESOURCE_ID", values["RESOURCE_NAME"])
	}
	if strings.TrimSpace(values["RESOURCE_DISPLAY_NAME"]) == "" && strings.TrimSpace(values["RESOURCE_NAME"]) != "" {
		values["RESOURCE_DISPLAY_NAME"] = kebabToDisplayName(values["RESOURCE_NAME"])
	}
	if strings.TrimSpace(values["RESOURCE_DESCRIPTION"]) == "" {
		values["RESOURCE_DESCRIPTION"] = "Scaffolded resource generated from the " + manifest.Name + " template."
	}
	if strings.TrimSpace(values["RESOURCE_CATEGORY"]) == "" {
		values["RESOURCE_CATEGORY"] = "operations"
	}
	if strings.TrimSpace(values["RESOURCE_PORTABILITY_TIER"]) == "" {
		values["RESOURCE_PORTABILITY_TIER"] = "partial"
	}
}

func missingResourceTemplateVars(values map[string]string, required map[string]ResourceTemplateVar) []string {
	missing := make([]string, 0)
	keys := make([]string, 0, len(required))
	for key := range required {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.TrimSpace(values[key]) != "" {
			continue
		}
		name := "--" + required[key].Flag
		if required[key].Flag == "" {
			name = key
		}
		missing = append(missing, name)
	}
	return missing
}

func previewResourceTemplateFiles(templateDir, destination string, values map[string]string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if relPath == "template.json" || filepath.Base(path) == ".DS_Store" {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		renderedRel := renderResourceTemplateString(relPath, values)
		files = append(files, filepath.Join(destination, renderedRel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func copyResourceTemplate(templateDir, destination string, values map[string]string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if relPath == "template.json" || filepath.Base(path) == ".DS_Store" {
			return nil
		}

		renderedRel := renderResourceTemplateString(relPath, values)
		targetPath := filepath.Join(destination, renderedRel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTemplateTextFile(data) {
			data = []byte(renderResourceTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func verifyRenderedResourceTemplate(destination string) error {
	var unresolved []string
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if unresolvedResourceTemplatePattern.MatchString(path) {
			unresolved = append(unresolved, path)
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !looksLikeTemplateTextFile(data) {
			return nil
		}
		if unresolvedResourceTemplatePattern.Match(data) {
			unresolved = append(unresolved, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(unresolved) == 0 {
		return nil
	}
	sort.Strings(unresolved)
	return fmt.Errorf("unresolved placeholders remain in: %s", strings.Join(unresolved, ", "))
}

func verifyGeneratedResourceManifest(destination string) error {
	path := filepath.Join(destination, "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read generated resource manifest: %w", err)
	}
	var manifest ResourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse generated resource manifest: %w", err)
	}
	if err := validateResourceManifest(manifest); err != nil {
		return fmt.Errorf("validate generated resource manifest: %w", err)
	}
	return nil
}

func looksLikeTemplateTextFile(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

func renderResourceTemplateString(value string, values map[string]string) string {
	if value == "" {
		return value
	}
	rendered := value
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", values[key])
	}
	return rendered
}

func setIfBlank(values map[string]string, key, value string) {
	if strings.TrimSpace(values[key]) == "" {
		values[key] = value
	}
}

func kebabToDisplayName(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func copyStringSlice(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}
