package packagegov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const manifestRelPath = ".vrooli/package.json"

var (
	packageSchemaCacheMu sync.Mutex
	packageSchemaCache   = make(map[string]*jsonschema.Schema)
)

func LoadAll(root string) ([]Package, []ValidationIssue, error) {
	packagesDir, err := packageDir(root)
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read packages dir: %w", err)
	}

	var items []Package
	var issues []ValidationIssue
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Hidden directories under packages are build staging areas (for
		// example, .proto-gen-stage-*), not governed package roots.
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		rootPath := filepath.Join(packagesDir, entry.Name())
		manifestPath := filepath.Join(rootPath, manifestRelPath)
		if _, err := os.Stat(manifestPath); err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, ValidationIssue{
					Severity:    "error",
					Code:        "missing-package-manifest",
					Message:     "package root is missing .vrooli/package.json",
					Path:        manifestPath,
					PackageName: entry.Name(),
				})
				continue
			}
			return nil, nil, fmt.Errorf("stat package manifest %s: %w", manifestPath, err)
		}

		item, loadIssues, err := LoadPackage(rootPath)
		if err != nil {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "invalid-package-manifest",
				Message:     err.Error(),
				Path:        manifestPath,
				PackageName: entry.Name(),
			})
			continue
		}
		items = append(items, item)
		issues = append(issues, loadIssues...)
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, issues, nil
}

func LoadPackage(rootPath string) (Package, []ValidationIssue, error) {
	manifestPath := filepath.Join(rootPath, manifestRelPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Package{}, nil, fmt.Errorf("read manifest: %w", err)
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Clean(rootPath)))
	if err := validateManifestSchema(repoRoot, data); err != nil {
		return Package{}, nil, err
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Package{}, nil, fmt.Errorf("decode manifest: %w", err)
	}
	if decoder.More() {
		return Package{}, nil, fmt.Errorf("decode manifest: trailing JSON content")
	}
	if err := manifest.ValidateBasics(); err != nil {
		return Package{}, nil, err
	}

	item := Package{
		Name:         manifest.Package.Name,
		RootPath:     filepath.Clean(rootPath),
		ManifestPath: filepath.Clean(manifestPath),
		Manifest:     manifest,
	}
	return item, validateManifestSemantics(item), nil
}

func FindByName(items []Package, name string) (Package, bool) {
	name = strings.TrimSpace(name)
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return Package{}, false
}

func packageDir(root string) (string, error) {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", err
	}
	dir, err := contract.TopLevelDir(root, "packages")
	if err != nil {
		return "", err
	}
	return filepath.Clean(dir), nil
}

func validateManifestSchema(repoRoot string, data []byte) error {
	schema, err := loadPackageManifestSchema(repoRoot)
	if err != nil {
		return fmt.Errorf("load manifest schema: %w", err)
	}
	var payload any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode manifest for schema validation: %w", err)
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}

func loadPackageManifestSchema(repoRoot string) (*jsonschema.Schema, error) {
	schemaDir, err := packageSchemaDir(repoRoot)
	if err != nil {
		return nil, err
	}
	key := filepath.Clean(schemaDir)

	packageSchemaCacheMu.Lock()
	if schema, ok := packageSchemaCache[key]; ok {
		packageSchemaCacheMu.Unlock()
		return schema, nil
	}
	packageSchemaCacheMu.Unlock()

	commonBytes, err := os.ReadFile(filepath.Join(schemaDir, "common.schema.json"))
	if err != nil {
		return nil, fmt.Errorf("read common schema: %w", err)
	}
	packageBytes, err := os.ReadFile(filepath.Join(schemaDir, "package.schema.json"))
	if err != nil {
		return nil, fmt.Errorf("read package schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("common.schema.json", bytes.NewReader(commonBytes)); err != nil {
		return nil, fmt.Errorf("add common schema resource: %w", err)
	}
	if err := compiler.AddResource("https://vrooli.com/schemas/common.schema.json", bytes.NewReader(commonBytes)); err != nil {
		return nil, fmt.Errorf("add canonical common schema resource: %w", err)
	}
	if err := compiler.AddResource("package.schema.json", bytes.NewReader(packageBytes)); err != nil {
		return nil, fmt.Errorf("add package schema resource: %w", err)
	}
	if err := compiler.AddResource("https://vrooli.com/schemas/package.schema.json", bytes.NewReader(packageBytes)); err != nil {
		return nil, fmt.Errorf("add canonical package schema resource: %w", err)
	}
	schema, err := compiler.Compile("https://vrooli.com/schemas/package.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile package schema: %w", err)
	}

	packageSchemaCacheMu.Lock()
	packageSchemaCache[key] = schema
	packageSchemaCacheMu.Unlock()
	return schema, nil
}

func packageSchemaDir(repoRoot string) (string, error) {
	candidate := filepath.Join(filepath.Clean(repoRoot), repocontractmeta.ProjectConfigDir, "schemas")
	if _, err := os.Stat(filepath.Join(candidate, "package.schema.json")); err == nil {
		return candidate, nil
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to determine package schema path")
	}
	fallback := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", repocontractmeta.ProjectConfigDir, "schemas"))
	if _, err := os.Stat(filepath.Join(fallback, "package.schema.json")); err != nil {
		return "", fmt.Errorf("package schema not found in %s or %s", candidate, fallback)
	}
	return fallback, nil
}

//nolint:gocyclo // manifest semantics validate independent package identity, dependency, and artifact rules.
func validateManifestSemantics(item Package) []ValidationIssue {
	var issues []ValidationIssue

	if item.Manifest.Package.DisplayName == "" {
		issues = append(issues, ValidationIssue{
			Severity:    "warning",
			Code:        "missing-package-display-name",
			Message:     "package.display_name should document the canonical public identifier",
			Path:        item.ManifestPath,
			PackageName: item.Name,
		})
	}

	if item.Manifest.Package.Adoption.ScenarioAdoptable && len(item.Manifest.Package.Adoption.AllowedConsumers) == 0 {
		issues = append(issues, ValidationIssue{
			Severity:    "error",
			Code:        "missing-allowed-consumers",
			Message:     "scenario-adoptable package must declare package.adoption.allowed_consumers",
			Path:        item.ManifestPath,
			PackageName: item.Name,
		})
	}

	if item.Manifest.Package.Adoption.ScenarioAdoptable && len(item.Manifest.Package.Adoption.AdoptionModes) == 0 {
		issues = append(issues, ValidationIssue{
			Severity:    "error",
			Code:        "missing-adoption-modes",
			Message:     "scenario-adoptable package must declare package.adoption.adoption_modes",
			Path:        item.ManifestPath,
			PackageName: item.Name,
		})
	}

	if !isAllowedPackageKind(item.Manifest.Package.Kind) {
		issues = append(issues, ValidationIssue{
			Severity:    "error",
			Code:        "invalid-package-kind",
			Message:     fmt.Sprintf("package.kind %q is not allowed", item.Manifest.Package.Kind),
			Path:        item.ManifestPath,
			PackageName: item.Name,
		})
	}

	if !isAllowedRefreshStrategy(item.Manifest.Package.Refresh.Strategy) {
		issues = append(issues, ValidationIssue{
			Severity:    "error",
			Code:        "invalid-refresh-strategy",
			Message:     fmt.Sprintf("package.refresh.strategy %q is not allowed", item.Manifest.Package.Refresh.Strategy),
			Path:        item.ManifestPath,
			PackageName: item.Name,
		})
	}

	seenIdentifiers := make(map[string]struct{}, len(item.Manifest.Package.ModuleIdentifiers))
	for _, identifier := range item.Manifest.Package.ModuleIdentifiers {
		identifier = strings.TrimSpace(identifier)
		if identifier == "" {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "empty-module-identifier",
				Message:     "package.module_identifiers must not contain empty values",
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
			continue
		}
		if _, ok := seenIdentifiers[identifier]; ok {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "duplicate-module-identifier",
				Message:     fmt.Sprintf("package.module_identifiers repeats %q", identifier),
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
			continue
		}
		seenIdentifiers[identifier] = struct{}{}
	}

	for _, consumer := range item.Manifest.Package.Adoption.AllowedConsumers {
		if !isAllowedConsumerClass(consumer) {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "invalid-allowed-consumer",
				Message:     fmt.Sprintf("package.adoption.allowed_consumers includes unsupported value %q", consumer),
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
		}
	}

	for _, mode := range item.Manifest.Package.Adoption.AdoptionModes {
		if !isAllowedAdoptionMode(mode) {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "invalid-adoption-mode",
				Message:     fmt.Sprintf("package.adoption.adoption_modes includes unsupported value %q", mode),
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
		}
	}

	for _, section := range []struct {
		name     string
		commands []CommandSpec
	}{
		{name: "generate", commands: item.Manifest.Package.Lifecycle.Generate},
		{name: "build", commands: item.Manifest.Package.Lifecycle.Build},
		{name: "test", commands: item.Manifest.Package.Lifecycle.Test},
	} {
		for _, command := range section.commands {
			if strings.TrimSpace(command.Name) == "" {
				issues = append(issues, ValidationIssue{
					Severity:    "error",
					Code:        "invalid-lifecycle-command",
					Message:     fmt.Sprintf("package.lifecycle.%s command name is required", section.name),
					Path:        item.ManifestPath,
					PackageName: item.Name,
				})
			}
			if len(command.Run) == 0 {
				issues = append(issues, ValidationIssue{
					Severity:    "error",
					Code:        "invalid-lifecycle-command",
					Message:     fmt.Sprintf("package.lifecycle.%s command %q must include run entries", section.name, command.Name),
					Path:        item.ManifestPath,
					PackageName: item.Name,
				})
			}
		}
	}

	for _, output := range item.Manifest.Package.GeneratedOutputs {
		if strings.TrimSpace(output.Name) == "" {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "invalid-generated-output",
				Message:     "package.generated_outputs entries must include name",
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
		}
		for _, consumer := range output.Consumers {
			if !isAllowedConsumerClass(consumer) {
				issues = append(issues, ValidationIssue{
					Severity:    "error",
					Code:        "invalid-generated-output-consumer",
					Message:     fmt.Sprintf("generated output %q includes unsupported consumer %q", output.Name, consumer),
					Path:        item.ManifestPath,
					PackageName: item.Name,
				})
			}
		}
	}

	repoRoot := filepath.Dir(filepath.Dir(item.RootPath))
	for _, docPath := range item.Manifest.Package.Docs {
		docPath = strings.TrimSpace(docPath)
		if docPath == "" {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "invalid-doc-path",
				Message:     "package.docs must not contain empty paths",
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
			continue
		}
		if _, err := os.Stat(filepath.Join(repoRoot, docPath)); err != nil {
			issues = append(issues, ValidationIssue{
				Severity:    "error",
				Code:        "missing-doc-path",
				Message:     fmt.Sprintf("package.docs references missing file %q", docPath),
				Path:        item.ManifestPath,
				PackageName: item.Name,
			})
		}
	}

	return issues
}

func isAllowedPackageKind(kind PackageKind) bool {
	switch kind {
	case KindJSRuntime, KindGeneratedTypeScript, KindGoRuntime, KindGoCLI, KindPythonRuntime, KindRustRuntime, KindInternalPlatform, KindSchemaOrContract:
		return true
	default:
		return false
	}
}

func isAllowedRefreshStrategy(strategy RefreshStrategy) bool {
	switch strategy {
	case RefreshScenarioSetup, RefreshGenerateThenSetup, RefreshRestartConsumers, RefreshRebuildCLI, RefreshNone:
		return true
	default:
		return false
	}
}

func isAllowedConsumerClass(class ConsumerClass) bool {
	switch class {
	case ConsumerScenarioUI, ConsumerScenarioAPI, ConsumerScenarioCLI, ConsumerScenarioTest, ConsumerTemplateUI, ConsumerTemplateAPI, ConsumerTemplateCLI, ConsumerResourceRuntime, ConsumerInternalPlatform:
		return true
	default:
		return false
	}
}

func isAllowedAdoptionMode(mode AdoptionMode) bool {
	switch mode {
	case ModeFileDependency, ModeGoModuleReplace, ModeGeneratedArtifact, ModePublishedSemver:
		return true
	default:
		return false
	}
}
