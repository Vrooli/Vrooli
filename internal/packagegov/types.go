package packagegov

import (
	"fmt"
	"path/filepath"
	"regexp"
)

type PackageKind string

const (
	KindJSRuntime           PackageKind = "js_runtime"
	KindGeneratedTypeScript PackageKind = "generated_typescript"
	KindGoRuntime           PackageKind = "go_runtime"
	KindGoCLI               PackageKind = "go_cli"
	KindInternalPlatform    PackageKind = "internal_platform"
	KindSchemaOrContract    PackageKind = "schema_or_contract"
)

type ConsumerClass string

const (
	ConsumerScenarioUI       ConsumerClass = "scenario_ui"
	ConsumerScenarioAPI      ConsumerClass = "scenario_api"
	ConsumerScenarioCLI      ConsumerClass = "scenario_cli"
	ConsumerScenarioTest     ConsumerClass = "scenario_test"
	ConsumerTemplateUI       ConsumerClass = "template_ui"
	ConsumerTemplateAPI      ConsumerClass = "template_api"
	ConsumerTemplateCLI      ConsumerClass = "template_cli"
	ConsumerResourceRuntime  ConsumerClass = "resource_runtime"
	ConsumerInternalPlatform ConsumerClass = "internal_platform"
)

type AdoptionMode string

const (
	ModeFileDependency    AdoptionMode = "file_dependency"
	ModeGoModuleReplace   AdoptionMode = "go_module_replace"
	ModeGeneratedArtifact AdoptionMode = "generated_artifact"
	ModePublishedSemver   AdoptionMode = "published_semver"
)

type RefreshStrategy string

const (
	RefreshScenarioSetup     RefreshStrategy = "scenario_setup"
	RefreshGenerateThenSetup RefreshStrategy = "generate_then_setup"
	RefreshRestartConsumers  RefreshStrategy = "restart_running_consumers"
	RefreshRebuildCLI        RefreshStrategy = "rebuild_cli_consumers"
	RefreshNone              RefreshStrategy = "none"
)

type CommandSpec struct {
	Name    string   `json:"name"`
	Run     []string `json:"run"`
	Outputs []string `json:"outputs,omitempty"`
}

type Manifest struct {
	Schema  string        `json:"$schema"`
	Version string        `json:"version"`
	Package ManifestEntry `json:"package"`
}

type ManifestEntry struct {
	Name              string            `json:"name"`
	DisplayName       string            `json:"display_name,omitempty"`
	Description       string            `json:"description,omitempty"`
	Kind              PackageKind       `json:"kind"`
	Language          string            `json:"language,omitempty"`
	ModuleIdentifiers []string          `json:"module_identifiers,omitempty"`
	GeneratedOutputs  []GeneratedOutput `json:"generated_outputs,omitempty"`
	ModuleBoundary    *ModuleBoundary   `json:"module_boundary,omitempty"`
	Adoption          AdoptionPolicy    `json:"adoption"`
	Lifecycle         LifecyclePolicy   `json:"lifecycle"`
	Refresh           RefreshPolicy     `json:"refresh"`
	Docs              []string          `json:"docs,omitempty"`
}

type ModuleBoundary struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type GeneratedOutput struct {
	Name        string          `json:"name"`
	Identifiers []string        `json:"identifiers,omitempty"`
	Consumers   []ConsumerClass `json:"consumers,omitempty"`
}

type AdoptionPolicy struct {
	ScenarioAdoptable bool            `json:"scenario_adoptable"`
	AllowedConsumers  []ConsumerClass `json:"allowed_consumers,omitempty"`
	AdoptionModes     []AdoptionMode  `json:"adoption_modes,omitempty"`
}

type LifecyclePolicy struct {
	Generate []CommandSpec `json:"generate,omitempty"`
	Build    []CommandSpec `json:"build,omitempty"`
	Test     []CommandSpec `json:"test,omitempty"`
}

type RefreshPolicy struct {
	Strategy                RefreshStrategy `json:"strategy"`
	RestartRunningConsumers bool            `json:"restart_running_consumers"`
}

type Package struct {
	Name         string
	RootPath     string
	ManifestPath string
	Manifest     Manifest
}

type Dependent struct {
	PackageName      string        `json:"package_name"`
	ConsumerName     string        `json:"consumer_name"`
	ConsumerPath     string        `json:"consumer_path"`
	ConsumerClass    ConsumerClass `json:"consumer_class"`
	AdoptionMode     AdoptionMode  `json:"adoption_mode"`
	DependencyFile   string        `json:"dependency_file"`
	DependencyTarget string        `json:"dependency_target,omitempty"`
	Version          string        `json:"version,omitempty"`
}

type ValidationIssue struct {
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Path        string `json:"path,omitempty"`
	PackageName string `json:"package_name,omitempty"`
}

func (m Manifest) ValidateBasics() error {
	if m.Schema != "schemas/package.schema.json" {
		return fmt.Errorf("$schema = %q, want schemas/package.schema.json", m.Schema)
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !semverPattern.MatchString(m.Version) {
		return fmt.Errorf("version %q is not valid semver", m.Version)
	}
	if m.Package.Name == "" {
		return fmt.Errorf("package.name is required")
	}
	if m.Package.Kind == "" {
		return fmt.Errorf("package.kind is required")
	}
	if len(m.Package.ModuleIdentifiers) == 0 {
		return fmt.Errorf("package.module_identifiers is required")
	}
	return nil
}

var semverPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

func normalizeIssues(issues []ValidationIssue) []ValidationIssue {
	out := make([]ValidationIssue, 0, len(issues))
	seen := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		issue.Path = filepath.Clean(issue.Path)
		key := issue.Severity + "\x00" + issue.Code + "\x00" + issue.Path + "\x00" + issue.Message + "\x00" + issue.PackageName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}
