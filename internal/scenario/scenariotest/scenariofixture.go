package scenariofixture

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	repocontracttest "github.com/vrooli/repo-contract-go/repocontracttest"
	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	hostreqspec "github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/scenario"
)

type ScenarioServiceOption func(*scenario.ServiceManifest)

type TemplateVar struct {
	Flag        string `json:"flag,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type TemplateManifest struct {
	Name         string                 `json:"name,omitempty"`
	DisplayName  string                 `json:"displayName,omitempty"`
	Description  string                 `json:"description,omitempty"`
	RequiredVars map[string]TemplateVar `json:"requiredVars,omitempty"`
	OptionalVars map[string]TemplateVar `json:"optionalVars,omitempty"`
}

type TemplateManifestOption func(*TemplateManifest)

func DefaultDisplayName(name string) string {
	return repocontracttest.DefaultDisplayName(name)
}

func ProjectServiceManifest(opts ...ScenarioServiceOption) scenario.ServiceManifest {
	manifest := scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        "project-alpha",
			DisplayName: "Project Alpha",
			Description: "Project-level fixture",
			Version:     "0.1.0",
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func ScenarioServiceManifest(name string, opts ...ScenarioServiceOption) scenario.ServiceManifest {
	manifest := scenario.ServiceManifest{
		Version: "1.0.0",
		Service: scenario.ServiceMetadata{
			Name:        name,
			DisplayName: DefaultDisplayName(name),
			Description: fmt.Sprintf("%s fixture", DefaultDisplayName(name)),
			Version:     "0.1.0",
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func ScenarioTemplateManifest(name string, opts ...TemplateManifestOption) TemplateManifest {
	manifest := TemplateManifest{
		Name:        name,
		DisplayName: "Demo Template",
		Description: "Template test fixture",
		RequiredVars: map[string]TemplateVar{
			"SCENARIO_ID":           {Flag: "id", Description: "Scenario id"},
			"SCENARIO_DISPLAY_NAME": {Flag: "display-name", Description: "Scenario name"},
			"SCENARIO_DESCRIPTION":  {Flag: "description", Description: "Scenario description"},
		},
		OptionalVars: map[string]TemplateVar{
			"AUTHOR": {Flag: "author", Description: "Author", Default: "Generator Agent"},
			"DATE":   {Flag: "date", Description: "Date", Default: "{{CURRENT_DATE}}"},
		},
	}
	for _, opt := range opts {
		opt(&manifest)
	}
	return manifest
}

func WithDescription(description string) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Service.Description = description
	}
}

func WithTemplateDisplayName(displayName string) TemplateManifestOption {
	return func(manifest *TemplateManifest) {
		manifest.DisplayName = displayName
	}
}

func WithTemplateDescription(description string) TemplateManifestOption {
	return func(manifest *TemplateManifest) {
		manifest.Description = description
	}
}

func WithTemplateRequiredVars(vars map[string]TemplateVar) TemplateManifestOption {
	return func(manifest *TemplateManifest) {
		manifest.RequiredVars = cloneTemplateVarMap(vars)
	}
}

func WithTemplateOptionalVars(vars map[string]TemplateVar) TemplateManifestOption {
	return func(manifest *TemplateManifest) {
		manifest.OptionalVars = cloneTemplateVarMap(vars)
	}
}

func WithDisplayName(displayName string) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Service.DisplayName = displayName
	}
}

func WithLifecycle(lifecycle scenario.Lifecycle) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Lifecycle = lifecycle
	}
}

func WithDependencies(dependencies scenario.Dependencies) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Dependencies = dependencies
	}
}

func WithPorts(ports map[string]scenario.Port) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Ports = ports
	}
}

func WithCLI(cli *scenario.CLIConfig) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		if cli != nil && cli.Enabled && cli.Adapter.Kind == "go_module" {
			if cli.SourceBuild == nil {
				cli.SourceBuild = &scenario.CLISourceBuildConfig{Kind: "go_module"}
			}
			if cli.Invoke.Kind == "" {
				cli.Invoke.Kind = "installed_command"
			}
			if cli.Invoke.Command == "" {
				cli.Invoke.Command = cli.Command
			}
			if cli.Freshness == nil {
				cli.Freshness = &scenario.CLIFreshnessCheck{Inputs: []string{"cli/**", ".vrooli/service.json"}}
			}
		}
		manifest.CLI = cli
	}
}

func WithEnvironment(env map[string]string) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.Environment = env
	}
}

func WithHostTools(tools ...hostreqspec.Declaration) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.HostTools = append([]hostreqspec.Declaration(nil), tools...)
	}
}

func WithHostSafeguards(safeguards ...hostreqspec.Declaration) ScenarioServiceOption {
	return func(manifest *scenario.ServiceManifest) {
		manifest.HostSafeguards = append([]hostreqspec.Declaration(nil), safeguards...)
	}
}

func WriteProjectService(t *testing.T, root string, manifest scenario.ServiceManifest) {
	t.Helper()
	testkitgo.WriteJSON(t, scenario.ProjectServicePath(root), manifest)
}

func WriteProjectResourceConfig(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	WriteProjectService(t, root, ProjectServiceManifest(
		WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				name: {Enabled: enabled},
			},
		}),
	))
}

func WriteScenarioService(t *testing.T, root, name string, manifest scenario.ServiceManifest) {
	t.Helper()
	if strings.TrimSpace(manifest.Service.Name) == "" {
		manifest.Service.Name = name
	}
	testkitgo.WriteJSON(t, scenario.ServicePath(root, name), manifest)
}

func WriteScenarioServiceAtPath(t *testing.T, path string, manifest scenario.ServiceManifest) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(path, ".vrooli", "service.json"), manifest)
}

func WriteScenarioCLIGoMod(t *testing.T, root, name, module string) {
	t.Helper()
	if strings.TrimSpace(module) == "" {
		module = name + "/cli"
	}
	testkitgo.WriteFile(t, filepath.Join(root, "scenarios", name, "cli", "go.mod"), "module "+module+"\n")
}

func WriteMalformedProjectService(t *testing.T, root, raw string) {
	t.Helper()
	testkitgo.WriteMalformedJSON(t, scenario.ProjectServicePath(root), raw, 0o644)
}

func WriteMalformedScenarioService(t *testing.T, root, name, raw string) {
	t.Helper()
	testkitgo.WriteMalformedJSON(t, scenario.ServicePath(root, name), raw, 0o644)
}

func ReadProjectService(t *testing.T, root string) scenario.ServiceManifest {
	t.Helper()
	return testkitgo.ReadJSONFileInto[scenario.ServiceManifest](t, scenario.ProjectServicePath(root))
}

func ReadScenarioService(t *testing.T, root, name string) scenario.ServiceManifest {
	t.Helper()
	return testkitgo.ReadJSONFileInto[scenario.ServiceManifest](t, scenario.ServicePath(root, name))
}

func WriteScenarioSetupOnlyFixture(t *testing.T, root, name string) {
	t.Helper()
	WriteScenarioService(t, root, name, ScenarioServiceManifest(
		name,
		WithDisplayName("Setup "+DefaultDisplayName(name)),
		WithDescription("Setup validation scenario"),
		WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup:   scenario.Phase{Steps: []scenario.PhaseStep{{Name: "write-file", Run: "mkdir -p build && printf 'ok\n' > build/setup.txt"}}},
		}),
	))
}

func WriteScenarioWithoutSetupFixture(t *testing.T, root, name string) {
	t.Helper()
	WriteScenarioService(t, root, name, ScenarioServiceManifest(
		name,
		WithDisplayName("No Setup "+DefaultDisplayName(name)),
		WithDescription("Scenario without setup phase"),
		WithLifecycle(scenario.Lifecycle{Version: "2.0.0"}),
	))
}

func WriteScenarioTestPhaseFixture(t *testing.T, root, name string) {
	t.Helper()
	testkitgo.WriteRelativeExecutable(t, root, filepath.Join("scenarios", name, "run-test.sh"), "#!/usr/bin/env bash\nset -e\nmkdir -p coverage\nprintf '%s\\n' \"$1\" > coverage/selector.txt\n")
	WriteScenarioService(t, root, name, ScenarioServiceManifest(
		name,
		WithDisplayName("Test "+DefaultDisplayName(name)),
		WithDescription("Test validation scenario"),
		WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Test:    scenario.Phase{Steps: []scenario.PhaseStep{{Name: "run-tests", Run: "./run-test.sh"}}},
		}),
	))
}

func WriteScenarioServiceWithPorts(t *testing.T, root, name string) {
	t.Helper()
	WriteScenarioService(t, root, name, ScenarioServiceManifest(
		name,
		WithDisplayName("Ports "+DefaultDisplayName(name)),
		WithDescription("Port validation scenario"),
		WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
			"ui":  {EnvVar: "UI_PORT", Range: "35000-39999"},
		}),
		WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{
				{Name: "start-api", Run: "sleep 10", Background: true},
				{Name: "start-ui", Run: "sleep 10", Background: true},
			}},
		}),
	))
}

func WriteScenarioTemplateFixture(t *testing.T, templateBase, name string) {
	t.Helper()
	testkitgo.WriteJSON(t, filepath.Join(templateBase, name, "template.json"), ScenarioTemplateManifest(name))
	testkitgo.WriteRelativeFile(t, filepath.Join(templateBase, name), "README.md", "# {{SCENARIO_DISPLAY_NAME}}\n\n{{SCENARIO_DESCRIPTION}}\n")
	testkitgo.WriteJSON(t, filepath.Join(templateBase, name, ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{
			"name":        "{{SCENARIO_ID}}",
			"displayName": "{{SCENARIO_DISPLAY_NAME}}",
			"description": "{{SCENARIO_DESCRIPTION}}",
		},
	})
	testkitgo.WriteJSON(t, filepath.Join(templateBase, name, "requirements", "index.json"), map[string]any{
		"owner": "{{AUTHOR}}",
		"date":  "{{DATE}}",
	})
}

func LifecycleScenarioManifest(name string, fixedPort *int, dependency string) scenario.ServiceManifest {
	ports := map[string]scenario.Port{
		"api": {EnvVar: "API_PORT", Range: "15000-19999"},
	}
	if fixedPort != nil {
		ports["api"] = scenario.Port{EnvVar: "API_PORT", Port: fixedPort}
	}

	manifest := ScenarioServiceManifest(
		name,
		WithDisplayName("Lifecycle "+DefaultDisplayName(name)),
		WithDescription("Lifecycle validation scenario"),
		WithPorts(ports),
		WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Health: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{{
					Name:     "api",
					Type:     "http",
					Target:   "http://127.0.0.1:${API_PORT}/health",
					Critical: true,
					Timeout:  1000,
				}},
				StartupGracePeriod: 1000,
				Timeout:            30000,
				Interval:           250,
			},
			Setup: scenario.Phase{
				Condition: &scenario.Condition{
					Checks: []scenario.ConditionCheck{{
						Type:    "binaries",
						Targets: []string{"api/mock-api"},
					}},
				},
				Steps: []scenario.PhaseStep{{
					Name: "build-api",
					Run:  "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health",
				}},
			},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{
				Name:       "start-api",
				Run:        "cd api && ./mock-api",
				Background: true,
				Condition:  &scenario.Condition{FileExists: "api/mock-api"},
			}}},
		}),
	)
	if dependency != "" {
		manifest.Dependencies = scenario.Dependencies{
			Scenarios: map[string]scenario.Dependency{
				dependency: {Type: "scenario", Required: true},
			},
		}
	}
	return manifest
}

func cloneTemplateVarMap(vars map[string]TemplateVar) map[string]TemplateVar {
	if len(vars) == 0 {
		return nil
	}
	cloned := make(map[string]TemplateVar, len(vars))
	for name, spec := range vars {
		cloned[name] = spec
	}
	return cloned
}

func intPtr(value int) *int {
	return &value
}

func WriteLifecycleScenarioService(t *testing.T, root, name string) {
	t.Helper()
	WriteScenarioService(t, root, name, LifecycleScenarioManifest(name, nil, ""))
}

func WriteLifecycleScenarioServiceAtPath(t *testing.T, path, name string) {
	t.Helper()
	WriteScenarioServiceAtPath(t, path, LifecycleScenarioManifest(name, nil, ""))
}

func WriteFixedPortLifecycleScenarioService(t *testing.T, root, name string, port int) {
	t.Helper()
	WriteScenarioService(t, root, name, LifecycleScenarioManifest(name, intPtr(port), ""))
}

func WriteBestEffortLifecycleScenarioService(t *testing.T, root, name, dependency string) {
	t.Helper()
	WriteScenarioService(t, root, name, LifecycleScenarioManifest(name, nil, dependency))
}
