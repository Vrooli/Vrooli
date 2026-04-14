package main

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/vroolicli"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
	testkitvrooli "github.com/vrooli/vrooli/packages/testkit-go/vrooli"
)

type App = vroolicli.App
type commandContext = vroolicli.CommandContext
type scenarioSubprocessSpec = scenarioexec.SubprocessSpec

func newTestApp(root string) *App {
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) { return root, nil }
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{Stale: false}, nil
	}
	app.EnsureScenarioCLIFn = func(root, home, name string) error { return nil }
	app.EnsureResourceCLIFn = func(root, home, name string) error { return nil }
	return app
}

func newConfiguredCommandContext(root string, globals globalOptions, stdout, stderr io.Writer) (*App, *commandContext) {
	app := configuredApp()
	return app, app.NewCommandContext(root, globals, stdout, stderr)
}

func runCleanupCommand(root string, parsed parsedArgs, stdout, stderr io.Writer) error {
	app := newTestApp(root)
	ctx := app.NewCommandContext(root, parsed.Globals, stdout, stderr)
	handler, ok := app.Registry().TopLevelHandler("cleanup")
	if !ok {
		return rootcli.NewUnknownCommandError("cleanup", nil)
	}
	return handler(ctx, parsed.Args)
}

func loadScenarioStateForTest(root string) ([]scenario.Scenario, map[string]process.ScenarioRuntime, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.NewScenarioService(ctx)
	if err != nil {
		return nil, nil, err
	}
	inventory, err := service.Inventory()
	if err != nil {
		return nil, nil, err
	}

	items := make([]scenario.Scenario, 0, len(inventory))
	runtimes := make(map[string]process.ScenarioRuntime, len(inventory))
	for _, item := range inventory {
		items = append(items, item.Scenario)
		if item.Runtime.ProcessCount > 0 {
			runtimes[item.Scenario.Slug] = item.Runtime
		}
	}
	return items, runtimes, nil
}

func loadScenarioDetailForTest(root, name string) (scenario.Scenario, process.ScenarioRuntime, string, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.NewScenarioService(ctx)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	detail, err := service.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, "", err
	}
	return detail.Scenario, detail.Runtime, detail.Details.Health, nil
}

func writeTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	testkitgo.WriteRelativeFile(t, root, rel, contents)
}

func writeFakeExecutable(t *testing.T, root, rel, contents string) string {
	t.Helper()
	return testkitgo.WriteRelativeExecutable(t, root, rel, contents)
}

func writeTestScenarioService(t *testing.T, root, name, description string) {
	t.Helper()
	testkitvrooli.WriteScenarioService(t, root, name, testkitvrooli.ScenarioServiceManifest(
		name,
		testkitvrooli.WithDisplayName(testkitvrooli.DefaultDisplayName(name)),
		testkitvrooli.WithDescription(description),
		testkitvrooli.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
		}),
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Develop: scenario.Phase{
				Description: "Run the scenario",
				Steps: []scenario.PhaseStep{{
					Name:       "start-api",
					Run:        "sleep 10",
					Background: true,
				}},
			},
		}),
	))
}

func writeProjectLifecycleFixture(t *testing.T, root string) {
	t.Helper()
	testkitvrooli.WritePortRegistry(t, root, map[string]int{"vrooli-api": 8092})
	testkitvrooli.WriteProjectService(t, root, testkitvrooli.ProjectServiceManifest(
		testkitvrooli.WithDescription("Project-level lifecycle fixture"),
		testkitvrooli.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "VROOLI_API_PORT", Port: intPtr(8092)},
		}),
		testkitvrooli.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup: scenario.Phase{
				Condition: &scenario.Condition{
					Checks: []scenario.ConditionCheck{{Type: "data", Path: "data"}},
				},
				Steps: []scenario.PhaseStep{
					{Name: "capture-setup", Run: "mkdir -p data build && printf 'setup\n' >> build/setup-count.txt && printf '%s|%s|%s|%s|%s|%s|%s|%s|%s\n' \"${ENVIRONMENT:-}\" \"${RESOURCES:-}\" \"${YES:-}\" \"${SUDO_MODE:-}\" \"${TARGET:-}\" \"${LOCATION:-}\" \"${DRY_RUN:-false}\" \"${APP_ROOT:-}\" \"${SERVICE_JSON_PATH:-}\" > build/setup-env.txt && printf 'ready\n' > data/bootstrap.txt"},
					{Name: "add-data", Run: "printf 'data\n' >> data/bootstrap.txt"},
				},
			},
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-develop", Run: "mkdir -p build && printf 'develop\n' >> build/develop-count.txt && printf '%s\n' \"${VROOLI_API_PORT:-}\" > build/develop-port.txt"}}},
			Clean:   scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-clean", Run: "mkdir -p build && printf 'clean\n' > build/clean.txt"}}},
			Backup:  scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-backup", Run: "mkdir -p build && printf 'backup\n' > build/backup.txt"}}},
			Restore: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-restore", Run: "mkdir -p build && printf 'restore\n' > build/restore.txt"}}},
		}),
	))
}

func writeResourceStatusFixture(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	testkitvrooli.WriteProjectService(t, root, testkitvrooli.ProjectServiceManifest(
		testkitvrooli.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{name: {Enabled: true}},
		}),
	))
	testkitvrooli.WriteResourceManifest(t, root, name, testkitvrooli.ResourceManifest(
		name,
		testkitvrooli.WithResourceDescription("Fixture resource"),
	))
	testkitvrooli.WriteResourceCLI(t, root, name, "#!/usr/bin/env bash\nprintf 'ok\\n'\n")
}

func writeScenarioSetupOnlyFixture(t *testing.T, root, name string) {
	t.Helper()
	testkitvrooli.WriteScenarioSetupOnlyFixture(t, root, name)
}

func writeScenarioWithoutSetupFixture(t *testing.T, root, name string) {
	t.Helper()
	testkitvrooli.WriteScenarioWithoutSetupFixture(t, root, name)
}

func writeScenarioTestPhaseFixture(t *testing.T, root, name string) {
	t.Helper()
	testkitvrooli.WriteScenarioTestPhaseFixture(t, root, name)
}

func writeScenarioServiceWithPorts(t *testing.T, root, name string) {
	t.Helper()
	testkitvrooli.WriteScenarioServiceWithPorts(t, root, name)
}

func writeScenarioTemplateFixture(t *testing.T, templateBase, name string) {
	t.Helper()
	testkitvrooli.WriteScenarioTemplateFixture(t, templateBase, name)
}

func writeLifecycleScenarioService(t *testing.T, root, name string) {
	t.Helper()
	testkitvrooli.WriteLifecycleScenarioService(t, root, name)
}

func writeLifecycleScenarioServiceAtPath(t *testing.T, root, scenarioPath, name string) {
	t.Helper()
	testkitvrooli.WriteLifecycleScenarioServiceAtPath(t, root, scenarioPath, name)
}

func writeFixedPortLifecycleScenarioService(t *testing.T, root, name string, port int) {
	t.Helper()
	testkitvrooli.WriteFixedPortLifecycleScenarioService(t, root, name, port)
}

func writeBestEffortLifecycleScenarioService(t *testing.T, root, name, dependency string) {
	t.Helper()
	testkitvrooli.WriteBestEffortLifecycleScenarioService(t, root, name, dependency)
}

func writeScenarioPortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	testkitvrooli.WriteScenarioPortRegistryFixture(t, root)
}

func writeScenarioProcessRecord(t *testing.T, home, name, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	writeScenarioProcessRecordWithWorkingDir(t, home, name, step, pid, port, startedAt, filepath.Join("/repo/scenarios", name))
}

func writeScenarioProcessRecordWithWorkingDir(t *testing.T, home, name, step string, pid, port int, startedAt time.Time, workingDir string) {
	t.Helper()
	testkitvrooli.WriteScenarioProcessRecordWithWorkingDir(t, home, name, step, pid, port, startedAt, workingDir)
}

func intPtr(value int) *int {
	return &value
}
