package project

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqcheck"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/process"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/testfixture"
	"github.com/vrooli/vrooli/internal/testutil"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-13

func TestStatusAggregatesResourcesAndScenarios(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
		Dependencies: scenario.Dependencies{
			Resources: map[string]scenario.Dependency{"redis": {Enabled: true}},
		},
	})
	writeScenarioService(t, root, "alpha")
	writeResourceManifest(t, root, "redis")
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)
	writeScenarioProcess(t, home, "alpha", 18081)

	controller := New(root, home, io.Discard, io.Discard)
	controller.MaintenanceSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
		return maintenance.ProcessSnapshot{TrackedProcesses: 1}, nil
	}
	controller.MaintenanceLocksFn = func() ([]maintenance.LockInfo, error) {
		return nil, nil
	}
	report, err := controller.Status(StatusOptions{Fast: true})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	if got := report.Summary["resources_running"]; got != 1 {
		t.Fatalf("resources_running = %d, want 1", got)
	}
	if got := report.Summary["scenarios_running"]; got != 1 {
		t.Fatalf("scenarios_running = %d, want 1", got)
	}
	if got := report.Summary["maintenance_orphan_processes"]; got != 0 {
		t.Fatalf("maintenance_orphan_processes = %d, want 0", got)
	}
	if report.Maintenance == nil {
		t.Fatalf("maintenance snapshot missing")
	}
	if len(report.Resources) != 1 || report.Resources[0].Resource.Name != "redis" {
		t.Fatalf("resources = %#v", report.Resources)
	}
	if len(report.Scenarios) != 1 || report.Scenarios[0].Name != "alpha" {
		t.Fatalf("scenarios = %#v", report.Scenarios)
	}
}

func TestRunProjectPhaseRejectsUndefinedPhase(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
		Lifecycle: scenario.Lifecycle{
			Develop: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "noop", Run: "true"}}},
		},
	})

	controller := New(root, home, io.Discard, io.Discard)
	if err := controller.RunProjectPhase("deploy", nil); err == nil {
		t.Fatal("expected undefined phase to fail")
	}
}

func TestDoctorReportsToolingPortAndServiceManifest(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
	})
	t.Setenv("VROOLI_API_PORT", "18092")

	controller := New(root, home, io.Discard, io.Discard)
	controller.MaintenanceSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
		return maintenance.ProcessSnapshot{}, nil
	}
	controller.MaintenanceLocksFn = func() ([]maintenance.LockInfo, error) {
		return nil, nil
	}
	controller.HostReqValidateFn = func(root, home string) (hostreqcheck.Report, error) {
		return hostreqcheck.Report{
			Findings: []hostreqcheck.Finding{
				{Code: hostreqcheck.FindingUndeclaredReference, OwnerKind: "scenario", OwnerName: "web-console", Requirement: "ffmpeg"},
				{Code: hostreqcheck.FindingMissingHandler, OwnerKind: "scenario", OwnerName: "scenario-to-desktop", Requirement: "websockify"},
			},
		}, nil
	}
	report, err := controller.Doctor()
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	joined := make([]string, 0, len(report.Checks))
	for _, check := range report.Checks {
		joined = append(joined, check.Name+"="+check.Status)
	}
	output := strings.Join(joined, "\n")
	if !strings.Contains(output, "api_port_18092=") {
		t.Fatalf("doctor checks missing api port: %s", output)
	}
	if !strings.Contains(output, "service_json=present") {
		t.Fatalf("doctor checks missing service manifest: %s", output)
	}
	if !strings.Contains(output, "orphan_processes=ok") {
		t.Fatalf("doctor checks missing orphan status: %s", output)
	}
	if !strings.Contains(output, "stale_port_locks=ok") {
		t.Fatalf("doctor checks missing stale lock status: %s", output)
	}
	if !strings.Contains(output, "listener_inspection=") {
		t.Fatalf("doctor checks missing listener inspection status: %s", output)
	}
	if !strings.Contains(output, "hostreq_undeclared_references=warning") {
		t.Fatalf("doctor checks missing undeclared host requirement summary: %s", output)
	}
	if !strings.Contains(output, "hostreq_missing_handlers=warning") {
		t.Fatalf("doctor checks missing missing-handler summary: %s", output)
	}
	if !strings.Contains(output, "hostreq_root_overreach=ok") {
		t.Fatalf("doctor checks missing root-overreach summary: %s", output)
	}
}

func TestStatusSupportsResourceAndScenarioFilters(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
	})
	writeScenarioService(t, root, "alpha")
	writeResourceManifest(t, root, "redis")
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)
	writeScenarioProcess(t, home, "alpha", 18081)

	controller := New(root, home, io.Discard, io.Discard)
	controller.MaintenanceSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
		return maintenance.ProcessSnapshot{}, nil
	}
	controller.MaintenanceLocksFn = func() ([]maintenance.LockInfo, error) {
		return nil, nil
	}
	resourcesOnly, err := controller.Status(StatusOptions{Fast: true, ResourcesOnly: true})
	if err != nil {
		t.Fatalf("Status(resources only): %v", err)
	}
	if len(resourcesOnly.Resources) != 1 || len(resourcesOnly.Scenarios) != 0 {
		t.Fatalf("resourcesOnly = %#v", resourcesOnly)
	}

	scenariosOnly, err := controller.Status(StatusOptions{Fast: true, ScenariosOnly: true})
	if err != nil {
		t.Fatalf("Status(scenarios only): %v", err)
	}
	if len(scenariosOnly.Scenarios) != 1 || len(scenariosOnly.Resources) != 0 {
		t.Fatalf("scenariosOnly = %#v", scenariosOnly)
	}
}

func TestRunProjectPhaseExecutesDefinedLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
		Lifecycle: scenario.Lifecycle{
			Build: scenario.Phase{
				Steps: []scenario.PhaseStep{{Name: "write-build-file", Run: "mkdir -p build && printf 'built\n' > build/build.txt"}},
			},
		},
	})
	writeProjectPortRegistry(t, root)

	controller := New(root, home, io.Discard, io.Discard)
	if err := controller.RunProjectPhase("build", nil); err != nil {
		t.Fatalf("RunProjectPhase(build): %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "build.txt"))
	if err != nil {
		t.Fatalf("read build output: %v", err)
	}
	if strings.TrimSpace(string(data)) != "built" {
		t.Fatalf("build output = %q", string(data))
	}
}

func TestRunProjectPhaseUsesInjectedPhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "project-alpha"},
		Lifecycle: scenario.Lifecycle{
			Build: scenario.Phase{Steps: []scenario.PhaseStep{{Name: "noop", Run: "true"}}},
		},
	})

	controller := New(root, home, io.Discard, io.Discard)
	called := false
	controller.NewPhaseRunner = func(root, home string, stdout, stderr io.Writer) (PhaseRunner, error) {
		return phaseRunnerFunc(func(name, phase string, opts lifecycle.PhaseOptions) error {
			called = true
			if name != "project-alpha" || phase != "build" {
				t.Fatalf("runner called with name=%q phase=%q", name, phase)
			}
			return nil
		}), nil
	}

	if err := controller.RunProjectPhase("build", nil); err != nil {
		t.Fatalf("RunProjectPhase(build): %v", err)
	}
	if !called {
		t.Fatalf("expected injected phase runner to be used")
	}
}

func TestLoadProjectFallsBackToDirectoryNameWhenServiceNameMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project-alpha")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{}})

	projectScenario, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if projectScenario.Slug != "project-alpha" {
		t.Fatalf("project slug = %q, want project-alpha", projectScenario.Slug)
	}
}

type phaseRunnerFunc func(name, phase string, opts lifecycle.PhaseOptions) error

func (fn phaseRunnerFunc) RunPhase(name, phase string, opts lifecycle.PhaseOptions) error {
	return fn(name, phase, opts)
}

func writeScenarioService(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteScenarioService(t, root, name, testfixture.ScenarioServiceManifest(
		name,
		testfixture.WithDisplayName(name),
		testfixture.WithPorts(map[string]scenario.Port{
			"api": {EnvVar: "API_PORT", Range: "18080-18090"},
		}),
	))
}

func writeResourceCLI(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	testfixture.WriteResourceCLI(t, root, name, script)
	installedPath := filepath.Join(root, "test-bin", "resource-"+name)
	testutil.WriteExecutable(t, installedPath, script)
	t.Setenv("PATH", filepath.Dir(installedPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeResourceManifest(t *testing.T, root, name string) {
	t.Helper()
	testfixture.WriteResourceManifest(t, root, name, manifestpkg.ResourceManifest{
		Name:            name,
		Driver:          "external-cli",
		Binary:          "resource-" + name,
		PortabilityTier: "full",
		Platforms:       manifestpkg.ResourcePlatforms{Linux: "supported"},
	})
}

func writeScenarioProcess(t *testing.T, home, name string, port int) {
	t.Helper()
	testfixture.WriteScenarioProcessRecord(t, home, name, "start-api", process.Record{
		PID:       os.Getpid(),
		PGID:      os.Getpid(),
		Scenario:  name,
		Step:      "start-api",
		Port:      port,
		StartedAt: time.Now().Add(-time.Minute).UTC(),
		Status:    "running",
	})
}

func writeProjectPortRegistry(t *testing.T, root string) {
	t.Helper()
	testfixture.WritePortRegistry(t, root, nil)
}
