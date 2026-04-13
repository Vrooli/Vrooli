package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/testfixture"
	"github.com/vrooli/vrooli/internal/testutil"
)

func newTestApp(root string) *App {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return root, nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	return app
}

func loadScenarioStateForTest(root string) ([]scenario.Scenario, map[string]process.ScenarioRuntime, error) {
	app, ctx := newConfiguredCommandContext(root, globalOptions{}, io.Discard, io.Discard)
	service, err := app.newScenarioService(ctx)
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
	service, err := app.newScenarioService(ctx)
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
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeFakeExecutable(t *testing.T, root, rel, contents string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func waitForTestFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func reserveFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	return testutil.ProjectRoot(t)
}

func writeTestScenarioService(t *testing.T, root, name, description string) {
	t.Helper()
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName(testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription(description),
		testfixture.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
		}),
		testfixture.WithLifecycle(scenario.Lifecycle{
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
	testfixture.WritePortRegistry(t, root, map[string]int{"vrooli-api": 8092})
	testfixture.WriteProjectService(t, root, testfixture.ProjectServiceManifest(
		testfixture.WithDescription("Project-level lifecycle fixture"),
		testfixture.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "VROOLI_API_PORT", Port: intPtr(8092)},
		}),
		testfixture.WithLifecycle(scenario.Lifecycle{
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
			Build:   scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-build", Run: "mkdir -p build && printf 'build\n' > build/build.txt"}}},
			Clean:   scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-clean", Run: "mkdir -p build && printf 'clean\n' > build/clean.txt"}}},
			Deploy:  scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-deploy", Run: "mkdir -p build && printf 'deploy\n' > build/deploy.txt"}}},
			Backup:  scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-backup", Run: "mkdir -p build && printf 'backup\n' > build/backup.txt"}}},
			Restore: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "capture-restore", Run: "mkdir -p build && printf 'restore\n' > build/restore.txt"}}},
		}),
	))
}

func writeResourceStatusFixture(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	testfixture.WriteProjectService(t, root, testfixture.ProjectServiceManifest(
		testfixture.WithDependencies(scenario.Dependencies{
			Resources: map[string]scenario.Dependency{name: {Enabled: true}},
		}),
	))
	testfixture.WriteResourceManifest(t, root, name, testfixture.ResourceManifest(
		name,
		testfixture.WithResourceDescription("Fixture resource"),
		testfixture.WithLegacyCLIPath(filepath.Join("resources", name, "cli.sh")),
	))
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	testfixture.WriteResourceCLI(t, root, name, script)
}

func writeScenarioSetupOnlyFixture(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName("Setup "+testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription("Setup validation scenario"),
		testfixture.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Setup:   scenario.Phase{Steps: []scenario.PhaseStep{{Name: "write-file", Run: "mkdir -p build && printf 'ok\n' > build/setup.txt"}}},
		}),
	))
}

func writeScenarioWithoutSetupFixture(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName("No Setup "+testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription("Scenario without setup phase"),
		testfixture.WithLifecycle(scenario.Lifecycle{Version: "2.0.0"}),
	))
}

func writeScenarioTestPhaseFixture(t *testing.T, root, name string) {
	t.Helper()
	writeFakeExecutable(t, root, filepath.Join("scenarios", name, "run-test.sh"), "#!/usr/bin/env bash\nset -e\nmkdir -p coverage\nprintf '%s\\n' \"$1\" > coverage/selector.txt\n")
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName("Test "+testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription("Test validation scenario"),
		testfixture.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Test:    scenario.Phase{Steps: []scenario.PhaseStep{{Name: "run-tests", Run: "./run-test.sh"}}},
		}),
	))
}

func writeScenarioServiceWithPorts(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName("Ports "+testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription("Port validation scenario"),
		testfixture.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "15000-19999"},
			"ui":  {EnvVar: "UI_PORT", Range: "35000-39999"},
		}),
		testfixture.WithLifecycle(scenario.Lifecycle{
			Version: "2.0.0",
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{
				{Name: "start-api", Run: "sleep 10", Background: true},
				{Name: "start-ui", Run: "sleep 10", Background: true},
			}},
		}),
	))
}

func writeScenarioTemplateFixture(t *testing.T, templateBase, name string) {
	t.Helper()
	manifestPath := filepath.Join(templateBase, name, "template.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(manifestPath), err)
	}
	manifest := `{
  "name": "` + name + `",
  "displayName": "Demo Template",
  "description": "Template test fixture",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id", "description": "Scenario id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name", "description": "Scenario name"},
    "SCENARIO_DESCRIPTION": {"flag": "description", "description": "Scenario description"}
  },
  "optionalVars": {
    "AUTHOR": {"flag": "author", "description": "Author", "default": "Generator Agent"},
    "DATE": {"flag": "date", "description": "Date", "default": "{{CURRENT_DATE}}"}
  }
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write %s: %v", manifestPath, err)
	}
	writeTestFile(t, filepath.Join(templateBase, name), "README.md", "# {{SCENARIO_DISPLAY_NAME}}\n\n{{SCENARIO_DESCRIPTION}}\n")
	writeTestFile(t, filepath.Join(templateBase, name), ".vrooli/service.json", `{"service":{"name":"{{SCENARIO_ID}}","displayName":"{{SCENARIO_DISPLAY_NAME}}","description":"{{SCENARIO_DESCRIPTION}}"}}`)
	writeTestFile(t, filepath.Join(templateBase, name), "requirements/index.json", `{"owner":"{{AUTHOR}}","date":"{{DATE}}"}`)
}

func writeLifecycleScenarioService(t *testing.T, root, name string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	testfixture.WriteScenarioService(t, root, name, lifecycleScenarioManifest(name, nil, ""))
}

func writeLifecycleScenarioServiceAtPath(t *testing.T, root, scenarioPath, name string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	testfixture.WriteScenarioServiceAtPath(t, scenarioPath, lifecycleScenarioManifest(name, nil, ""))
}

func writeFixedPortLifecycleScenarioService(t *testing.T, root, name string, port int) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	testfixture.WriteScenarioService(t, root, name, lifecycleScenarioManifest(name, intPtr(port), ""))
}

func writeBestEffortLifecycleScenarioService(t *testing.T, root, name, dependency string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	testfixture.WriteScenarioService(t, root, name, lifecycleScenarioManifest(name, nil, dependency))
}

func writeScenarioPortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	testfixture.WritePortRegistry(t, root, nil)
}

func writeScenarioProcessRecord(t *testing.T, home, name, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	writeScenarioProcessRecordWithWorkingDir(t, home, name, step, pid, port, startedAt, filepath.Join("/repo/scenarios", name))
}

func writeScenarioProcessRecordWithWorkingDir(t *testing.T, home, name, step string, pid, port int, startedAt time.Time, workingDir string) {
	t.Helper()
	testfixture.WriteScenarioProcessRecord(t, home, name, step, process.Record{
		PID:        pid,
		PGID:       pid,
		ProcessID:  fmt.Sprintf("vrooli.develop.%s.%s", name, step),
		Phase:      "develop",
		Scenario:   name,
		Step:       step,
		Command:    "sleep 10",
		WorkingDir: workingDir,
		LogFile:    fmt.Sprintf("/tmp/%s.log", name),
		Port:       port,
		StartedAt:  startedAt.UTC(),
		Status:     "running",
	})
}

func lifecycleScenarioManifest(name string, fixedPort *int, dependency string) scenario.ServiceManifest {
	ports := map[string]scenario.Port{
		"api": {EnvVar: "API_PORT", Range: "15000-19999"},
	}
	if fixedPort != nil {
		ports["api"] = scenario.Port{EnvVar: "API_PORT", Port: fixedPort}
	}

	manifest := testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName("Lifecycle "+testfixture.DefaultDisplayName(name)),
		testfixture.WithDescription("Lifecycle validation scenario"),
		testfixture.WithPorts(ports),
		testfixture.WithLifecycle(scenario.Lifecycle{
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

func intPtr(value int) *int {
	return &value
}
