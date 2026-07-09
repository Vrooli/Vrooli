package templateengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	resourceenv "github.com/vrooli/vrooli/internal/resources/env"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const (
	resourceTemplateBasePath = "templates/resources"
	blueprintDirPath         = ".vrooli/resources/blueprints"
)

var (
	canonicalResourceTemplateNames = []string{
		"cloud-api",
		"compose-service",
		"desktop-app",
		"docker-service",
		"external-cli",
		"manual-resource",
		"native-cli",
	}
	resourceTemplateRequiredFiles = []string{
		".gitignore",
		"Makefile",
		"README.md",
		"resource.json",
		"cli/go.mod",
		"cli/.golangci.yml",
		"cli/install.sh",
		"cli/install.ps1",
		"cli/main.go",
		"cli/main_test.go",
		"docs/OPERATIONS.md",
	}
)

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

type resourceBlueprint struct {
	Name              string `json:"name"`
	DisplayName       string `json:"display_name"`
	Category          string `json:"category"`
	Summary           string `json:"summary"`
	IntegrationKind   string `json:"integration_kind"`
	SuggestedTemplate string `json:"suggested_template"`
	Status            string `json:"status"`
	PlatformSupport   struct {
		PortabilityTier string `json:"portability_tier"`
	} `json:"platform_support"`
}

func (e *Engine) ListResourceTemplates(_ context.Context) ([]ResourceTemplateInfo, error) {
	return listResourceTemplates(e.root)
}

func (e *Engine) ShowResourceTemplate(_ context.Context, name string) (ResourceTemplateInfo, error) {
	return loadResourceTemplate(e.root, name)
}

func (e *Engine) ValidateResourceTemplates(_ context.Context) (ResourceTemplateValidationReport, error) {
	return validateResourceTemplates(e.root)
}

func (e *Engine) GenerateResourceTemplate(_ context.Context, req ResourceTemplateGenerateRequest) (ResourceTemplateGenerateReport, error) {
	return generateResourceTemplate(e.root, req)
}

func listResourceTemplates(root string) ([]ResourceTemplateInfo, error) {
	baseDir := filepath.Join(root, filepath.FromSlash(resourceTemplateBasePath))
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("read resource templates %s: %w", baseDir, err)
	}
	templates := make([]ResourceTemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := loadResourceTemplate(root, entry.Name())
		if err != nil {
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadResourceTemplate(root, name string) (ResourceTemplateInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResourceTemplateInfo{}, errors.New("resource template name cannot be empty")
	}
	templateDir := filepath.Join(root, filepath.FromSlash(resourceTemplateBasePath), name)
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
	normalizeResourceTemplateManifest(&manifest)
	if err := validateResourceTemplateManifest(manifest); err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("validate resource template %s: %w", name, err)
	}
	if err := validateResourceTemplateAssets(root, templateDir, manifest); err != nil {
		return ResourceTemplateInfo{}, fmt.Errorf("validate resource template assets %s: %w", name, err)
	}
	return ResourceTemplateInfo{Name: name, Path: templateDir, Manifest: manifest}, nil
}

func validateResourceTemplates(root string) (ResourceTemplateValidationReport, error) {
	templates, err := listResourceTemplates(root)
	if err != nil {
		return ResourceTemplateValidationReport{}, err
	}
	seen := make(map[string]struct{}, len(templates))
	report := ResourceTemplateValidationReport{
		Templates: make([]ResourceTemplateSummary, 0, len(templates)),
		Count:     len(templates),
	}
	for _, item := range templates {
		if err := validateResourceTemplateGoModuleSource(item); err != nil {
			return ResourceTemplateValidationReport{}, fmt.Errorf("validate resource template %s go.mod source: %w", item.Name, err)
		}
		if err := validateGeneratedResourceTemplate(root, item); err != nil {
			return ResourceTemplateValidationReport{}, fmt.Errorf("validate generated resource template %s: %w", item.Name, err)
		}
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

func generateResourceTemplate(root string, req ResourceTemplateGenerateRequest) (ResourceTemplateGenerateReport, error) {
	info, blueprint, err := resolveResourceTemplateGenerationInputs(root, req)
	if err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	values := copyStringMap(req.Values)
	if values == nil {
		values = map[string]string{}
	}
	seedResourceTemplateValues(values, info, blueprint)
	applyResourceTemplateDefaults(values, info.Manifest)
	if missing := missingResourceTemplateVars(values, info.Manifest.RequiredVars); len(missing) > 0 {
		return ResourceTemplateGenerateReport{}, fmt.Errorf("missing required values: %s", strings.Join(missing, ", "))
	}
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		destination = filepath.Join(root, "resources", values["RESOURCE_NAME"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(root, filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)
	if err := populateResourceTemplatePathValues(root, destination, values); err != nil {
		return ResourceTemplateGenerateReport{}, fmt.Errorf("resolve resource template path placeholders: %w", err)
	}
	files, err := previewResourceTemplateFiles(info.Path, destination, values)
	if err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	report := ResourceTemplateGenerateReport{
		Template:    resourceTemplateSummary(info),
		Destination: destination,
		Values:      copyStringMap(values),
		Files:       files,
		DryRun:      req.DryRun,
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
	if err := verifyTemplate(destination); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	if err := verifyGeneratedResourceManifest(destination); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	if err := verifyGeneratedResourceGoModules(destination); err != nil {
		return ResourceTemplateGenerateReport{}, err
	}
	return report, nil
}

func resolveResourceTemplateGenerationInputs(root string, req ResourceTemplateGenerateRequest) (ResourceTemplateInfo, *resourceBlueprint, error) {
	templateName := strings.TrimSpace(req.TemplateName)
	blueprintName := strings.TrimSpace(req.BlueprintName)
	var blueprint *resourceBlueprint
	if blueprintName != "" {
		item, err := loadResourceBlueprint(root, blueprintName)
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
	info, err := loadResourceTemplate(root, templateName)
	if err != nil {
		return ResourceTemplateInfo{}, nil, err
	}
	return info, blueprint, nil
}

func loadResourceBlueprint(root, name string) (resourceBlueprint, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return resourceBlueprint{}, errors.New("--from-blueprint requires a value")
	}
	path := filepath.Join(root, filepath.FromSlash(blueprintDirPath), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return resourceBlueprint{}, fmt.Errorf("resource blueprint %q not found", name)
		}
		return resourceBlueprint{}, fmt.Errorf("read resource blueprint %s: %w", path, err)
	}
	var item resourceBlueprint
	if err := json.Unmarshal(data, &item); err != nil {
		return resourceBlueprint{}, fmt.Errorf("parse resource blueprint %s: %w", path, err)
	}
	if item.Name != name {
		return resourceBlueprint{}, fmt.Errorf("resource blueprint %s: filename does not match name %q", path, item.Name)
	}
	if strings.TrimSpace(item.SuggestedTemplate) == "" {
		return resourceBlueprint{}, fmt.Errorf("resource blueprint %q missing suggested_template", name)
	}
	return item, nil
}

func normalizeResourceTemplateManifest(manifest *ResourceTemplateManifest) {
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]ResourceTemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]ResourceTemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
}

func validateResourceTemplateManifest(manifest ResourceTemplateManifest) error {
	if !stringInSlice(manifest.Name, canonicalResourceTemplateNames) {
		return fmt.Errorf("name %q is invalid", manifest.Name)
	}
	if strings.TrimSpace(manifest.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(manifest.Description) == "" {
		return errors.New("description is required")
	}
	if !stringInSlice(strings.TrimSpace(manifest.Driver), manifestpkg.AllowedDrivers) {
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

func validateResourceTemplateAssets(root, templateDir string, manifest ResourceTemplateManifest) error {
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
		path := filepath.Join(root, filepath.FromSlash(relPath))
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
	for _, relPath := range []string{"config/defaults.json", "config/schema.json"} {
		path := filepath.Join(templateDir, filepath.FromSlash(relPath))
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("deprecated template file retained: %s", relPath)
		}
	}
	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".sh") {
			relPath, relErr := filepath.Rel(templateDir, path)
			if relErr != nil {
				return relErr
			}
			if filepath.ToSlash(relPath) == "cli/install.sh" {
				return nil
			}
			return fmt.Errorf("canonical templates must not include bash files: %s", filepath.ToSlash(relPath))
		}
		return nil
	})
}

func validateResourceTemplateVar(key string, variable ResourceTemplateVar) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("template variable keys cannot be empty")
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

func seedResourceTemplateValues(values map[string]string, info ResourceTemplateInfo, blueprint *resourceBlueprint) {
	values["RESOURCE_TEMPLATE"] = info.Name
	values["RESOURCE_DRIVER"] = info.Manifest.Driver
	values["CURRENT_DATE"] = time.Now().UTC().Format("2006-01-02")
	if blueprint == nil {
		return
	}
	setTemplateValueIfBlank(values, "RESOURCE_NAME", blueprint.Name)
	setTemplateValueIfBlank(values, "RESOURCE_DISPLAY_NAME", blueprint.DisplayName)
	setTemplateValueIfBlank(values, "RESOURCE_DESCRIPTION", blueprint.Summary)
	setTemplateValueIfBlank(values, "RESOURCE_CATEGORY", blueprint.Category)
	setTemplateValueIfBlank(values, "RESOURCE_PORTABILITY_TIER", blueprint.PlatformSupport.PortabilityTier)
	setTemplateValueIfBlank(values, "RESOURCE_BLUEPRINT_NAME", blueprint.Name)
	setTemplateValueIfBlank(values, "RESOURCE_BLUEPRINT_STATUS", blueprint.Status)
	setTemplateValueIfBlank(values, "RESOURCE_BLUEPRINT_INTEGRATION_KIND", blueprint.IntegrationKind)
}

func applyResourceTemplateDefaults(values map[string]string, manifest ResourceTemplateManifest) {
	optionalKeys := make([]string, 0, len(manifest.OptionalVars))
	for key := range manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) == "" {
			values[key] = renderTemplateString(manifest.OptionalVars[key].Default, values)
		}
	}
	if strings.TrimSpace(values["RESOURCE_NAME"]) != "" {
		setTemplateValueIfBlank(values, "RESOURCE_ID", values["RESOURCE_NAME"])
	}
	if strings.TrimSpace(values["RESOURCE_DISPLAY_NAME"]) == "" && strings.TrimSpace(values["RESOURCE_NAME"]) != "" {
		values["RESOURCE_DISPLAY_NAME"] = kebabToDisplayName(values["RESOURCE_NAME"])
	}
	setTemplateValueIfBlank(values, "RESOURCE_DESCRIPTION", "Scaffolded resource generated from the "+manifest.Name+" template.")
	setTemplateValueIfBlank(values, "RESOURCE_CATEGORY", "operations")
	setTemplateValueIfBlank(values, "RESOURCE_PORTABILITY_TIER", "partial")
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
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if relPath == "template.json" || filepath.Base(path) == ".DS_Store" || entry.IsDir() {
			return nil
		}
		files = append(files, filepath.Join(destination, renderTemplateString(relPath, values)))
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
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if relPath == "template.json" || filepath.Base(path) == ".DS_Store" {
			return nil
		}
		targetPath := filepath.Join(destination, renderTemplateString(relPath, values))
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
		if looksLikeTextFile(data) {
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func populateResourceTemplatePathValues(root, destination string, values map[string]string) error {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return err
	}
	repoRoot := filepath.Clean(root)
	packagesDir, err := contract.TopLevelDir(root, "packages")
	if err != nil {
		return err
	}
	for key, dir := range map[string]string{
		"RESOURCE": filepath.Clean(destination),
		"CLI":      filepath.Join(destination, "cli"),
	} {
		repoRel, err := filepath.Rel(dir, repoRoot)
		if err != nil {
			return err
		}
		packagesRel, err := filepath.Rel(dir, packagesDir)
		if err != nil {
			return err
		}
		values["REPO_ROOT_REL_FROM_"+key] = filepath.ToSlash(repoRel)
		values["PACKAGES_REL_FROM_"+key] = filepath.ToSlash(packagesRel)
	}
	return nil
}

func validateGeneratedResourceTemplate(root string, info ResourceTemplateInfo) error {
	tempRoot, err := os.MkdirTemp("", "vrooli-resource-template-validate-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempRoot)
	values := resourceTemplateValidationSeedValues(info)
	destination := filepath.Join(tempRoot, "resource")
	if err := populateResourceTemplatePathValues(root, destination, values); err != nil {
		return err
	}
	seedResourceTemplateValues(values, info, nil)
	applyResourceTemplateDefaults(values, info.Manifest)
	if missing := missingResourceTemplateVars(values, info.Manifest.RequiredVars); len(missing) > 0 {
		return fmt.Errorf("missing required values: %s", strings.Join(missing, ", "))
	}
	if err := copyResourceTemplate(info.Path, destination, values); err != nil {
		return err
	}
	if err := verifyTemplate(destination); err != nil {
		return err
	}
	if err := verifyGeneratedResourceManifest(destination); err != nil {
		return err
	}
	return verifyGeneratedResourceGoModules(destination)
}

func verifyGeneratedResourceManifest(destination string) error {
	path := filepath.Join(destination, "resource.json")
	manifest, err := manifestpkg.Load(path)
	if err != nil {
		return fmt.Errorf("read generated resource manifest: %w", err)
	}
	if err := manifestpkg.Validate(manifest); err != nil {
		return fmt.Errorf("validate generated resource manifest: %w", err)
	}
	if issues := resourceenv.ValidateResourceManifest(destination, manifest); len(issues) > 0 {
		return fmt.Errorf("validate generated resource manifest policy: %s", strings.Join(issues, "; "))
	}
	return nil
}

func validateResourceTemplateGoModuleSource(info ResourceTemplateInfo) error {
	return filepath.WalkDir(info.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, target := range parseResourceLocalReplaceTargets(string(data)) {
			if strings.Contains(target, "{{") {
				continue
			}
			relPath, relErr := filepath.Rel(info.Path, path)
			if relErr != nil {
				relPath = path
			}
			return fmt.Errorf("%s contains hardcoded local replace target %q; use generator-computed placeholders", filepath.ToSlash(relPath), target)
		}
		return nil
	})
}

func verifyGeneratedResourceGoModules(destination string) error {
	return filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		moduleDir := filepath.Dir(path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read generated go.mod %s: %w", path, readErr)
		}
		for _, target := range parseResourceLocalReplaceTargets(string(data)) {
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(moduleDir, filepath.FromSlash(target))
			}
			if _, statErr := os.Stat(filepath.Clean(resolved)); statErr != nil {
				relPath, _ := filepath.Rel(destination, path)
				return fmt.Errorf("%s replace target %q does not resolve: %w", filepath.ToSlash(relPath), target, statErr)
			}
		}
		if !resourceModuleHasGoFiles(moduleDir) {
			return nil
		}
		cmd := exec.Command("bash", "-lc", "GOWORK=off go mod tidy")
		cmd.Dir = moduleDir
		cmd.Env = os.Environ()
		output, execErr := cmd.CombinedOutput()
		if execErr != nil {
			relPath, _ := filepath.Rel(destination, path)
			msg := strings.TrimSpace(string(output))
			if msg == "" {
				msg = execErr.Error()
			}
			return fmt.Errorf("%s generated module validation failed: %s", filepath.ToSlash(relPath), msg)
		}
		return nil
	})
}

func parseResourceLocalReplaceTargets(content string) []string {
	return parseLocalReplaceTargets(content)
}

func resourceModuleHasGoFiles(moduleDir string) bool {
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func resourceTemplateValidationSeedValues(info ResourceTemplateInfo) map[string]string {
	values := map[string]string{}
	requiredKeys := make([]string, 0, len(info.Manifest.RequiredVars))
	for key := range info.Manifest.RequiredVars {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)
	for _, key := range requiredKeys {
		switch key {
		case "RESOURCE_NAME":
			values[key] = info.Name + "-validation"
		case "RESOURCE_DISPLAY_NAME":
			values[key] = coalesce(info.Manifest.DisplayName, info.Name+" Validation")
		case "RESOURCE_DESCRIPTION":
			values[key] = coalesce(info.Manifest.Description, "Validation resource generated from "+info.Name)
		default:
			if fallback := strings.TrimSpace(info.Manifest.RequiredVars[key].Default); fallback != "" {
				values[key] = fallback
			} else {
				values[key] = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
			}
		}
	}
	return values
}

func setTemplateValueIfBlank(values map[string]string, key, value string) {
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

func stringInSlice(value string, allowed []string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
