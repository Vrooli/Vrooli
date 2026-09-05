package packagefixture

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	repocontracttest "github.com/vrooli/repo-contract-go/repocontracttest"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	packagegov "github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

type PackageManifestOption func(*packagegov.Manifest)

type NodePackageManifest struct {
	Name                 string            `json:"name,omitempty"`
	Scripts              map[string]string `json:"scripts,omitempty"`
	Dependencies         map[string]string `json:"dependencies,omitempty"`
	DevDependencies      map[string]string `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string `json:"optionalDependencies,omitempty"`
}

func PackageManifest(name string, opts ...PackageManifestOption) packagegov.Manifest {
	manifest := packagegov.Manifest{
		Schema:  "schemas/package.schema.json",
		Version: "1.0.0",
		Package: packagegov.ManifestEntry{
			Name:              name,
			DisplayName:       defaultPackageDisplayName(name),
			Description:       fmt.Sprintf("%s fixture package", repocontracttest.DefaultDisplayName(name)),
			Kind:              packagegov.KindJSRuntime,
			ModuleIdentifiers: []string{defaultPackageDisplayName(name)},
			Adoption: packagegov.AdoptionPolicy{
				ScenarioAdoptable: true,
				AllowedConsumers:  []packagegov.ConsumerClass{packagegov.ConsumerScenarioUI},
				AdoptionModes:     []packagegov.AdoptionMode{packagegov.ModeFileDependency},
			},
			Lifecycle: packagegov.LifecyclePolicy{},
			Refresh: packagegov.RefreshPolicy{
				Strategy:                packagegov.RefreshScenarioSetup,
				RestartRunningConsumers: false,
			},
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func WithPackageDisplayName(displayName string) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.DisplayName = displayName
	}
}

func WithPackageDescription(description string) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Description = description
	}
}

func WithPackageKind(kind packagegov.PackageKind) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Kind = kind
	}
}

func WithPackageModuleIdentifiers(ids ...string) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.ModuleIdentifiers = append([]string(nil), ids...)
	}
}

func WithPackageGeneratedOutputs(outputs ...packagegov.GeneratedOutput) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.GeneratedOutputs = append([]packagegov.GeneratedOutput(nil), outputs...)
	}
}

func WithPackageAdoption(adoption packagegov.AdoptionPolicy) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Adoption = adoption
	}
}

func WithPackageAllowedConsumers(consumers ...packagegov.ConsumerClass) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Adoption.AllowedConsumers = append([]packagegov.ConsumerClass(nil), consumers...)
	}
}

func WithPackageAdoptionModes(modes ...packagegov.AdoptionMode) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Adoption.AdoptionModes = append([]packagegov.AdoptionMode(nil), modes...)
	}
}

func WithPackageScenarioAdoptable(enabled bool) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Adoption.ScenarioAdoptable = enabled
	}
}

func WithPackageLifecycle(lifecycle packagegov.LifecyclePolicy) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Lifecycle = lifecycle
	}
}

func WithPackageGenerateCommands(commands ...packagegov.CommandSpec) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Lifecycle.Generate = append([]packagegov.CommandSpec(nil), commands...)
	}
}

func WithPackageBuildCommands(commands ...packagegov.CommandSpec) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Lifecycle.Build = append([]packagegov.CommandSpec(nil), commands...)
	}
}

func WithPackageRefresh(strategy packagegov.RefreshStrategy, restart bool) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Refresh = packagegov.RefreshPolicy{
			Strategy:                strategy,
			RestartRunningConsumers: restart,
		}
	}
}

func WithPackageDocs(paths ...string) PackageManifestOption {
	return func(manifest *packagegov.Manifest) {
		manifest.Package.Docs = append([]string(nil), paths...)
	}
}

func WritePackageManifest(t *testing.T, root, name string, manifest packagegov.Manifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Package.Name) == "" {
		manifest.Package.Name = name
	}
	testkitgo.WriteJSON(t, filepath.Join(root, "packages", name, repocontractmeta.ProjectConfigDir, "package.json"), manifest)
}

func WriteNodePackageManifest(t *testing.T, path string, manifest NodePackageManifest) {
	t.Helper()
	testkitgo.WriteJSON(t, path, manifest)
}

func WriteScenarioUIPackageManifest(t *testing.T, root, scenarioName string, manifest NodePackageManifest) {
	t.Helper()
	WriteNodePackageManifest(t, filepath.Join(root, repocontractmeta.ScenarioDir, scenarioName, "ui", "package.json"), manifest)
}

func WriteTemplateScenarioUIPackageManifest(t *testing.T, root, templateName string, manifest NodePackageManifest) {
	t.Helper()
	WriteNodePackageManifest(t, filepath.Join(root, "templates", repocontractmeta.ScenarioDir, templateName, "ui", "package.json"), manifest)
}

func defaultPackageDisplayName(name string) string {
	if strings.HasPrefix(name, "@") || strings.Contains(name, "/") {
		return name
	}
	return "@vrooli/" + name
}
