package packagegov

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const manifestRelPath = ".vrooli/package.json"

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
	case KindJSRuntime, KindGeneratedTypeScript, KindGoRuntime, KindGoCLI, KindGoTestkit, KindInternalPlatform, KindSchemaOrContract:
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
