package project

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-11

func TestStatusAggregatesResourcesAndScenarios(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeProjectService(t, root, `{
  "service": { "name": "project-alpha" },
  "dependencies": {
    "resources": {
      "redis": { "enabled": true }
    }
  }
}`)
	writeScenarioService(t, root, "alpha")
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)
	writeScenarioProcess(t, home, "alpha", 18081)

	controller := New(root, home, io.Discard, io.Discard)
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

	writeProjectService(t, root, `{
  "service": { "name": "project-alpha" },
  "lifecycle": {
    "develop": {
      "steps": [{ "name": "noop", "run": "true" }]
    }
  }
}`)

	controller := New(root, home, io.Discard, io.Discard)
	if err := controller.RunProjectPhase("deploy", nil); err == nil {
		t.Fatal("expected undefined phase to fail")
	}
}

func TestDoctorReportsToolingPortAndServiceManifest(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectService(t, root, `{"service":{"name":"project-alpha"}}`)
	t.Setenv("VROOLI_API_PORT", "18092")

	controller := New(root, home, io.Discard, io.Discard)
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
}

func TestStatusSupportsResourceAndScenarioFilters(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectService(t, root, `{"service":{"name":"project-alpha"}}`)
	writeScenarioService(t, root, "alpha")
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)
	writeScenarioProcess(t, home, "alpha", 18081)

	controller := New(root, home, io.Discard, io.Discard)
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
	writeProjectService(t, root, `{
  "service": { "name": "project-alpha" },
  "lifecycle": {
    "build": {
      "steps": [
        {
          "name": "write-build-file",
          "run": "mkdir -p build && printf 'built\n' > build/build.txt"
        }
      ]
    }
  }
}`)
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

func TestLoadProjectFallsBackToDirectoryNameWhenServiceNameMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project-alpha")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	writeProjectService(t, root, `{"service":{}}`)

	projectScenario, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if projectScenario.Slug != "project-alpha" {
		t.Fatalf("project slug = %q, want project-alpha", projectScenario.Slug)
	}
}

func writeProjectService(t *testing.T, root, contents string) {
	t.Helper()
	path := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioService(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "service": { "name": "` + name + `", "displayName": "` + name + `" },
  "ports": {
    "api": { "env_var": "API_PORT", "range": "18080-18090" }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeResourceCLI(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	path := filepath.Join(root, "resources", name, "cli.sh")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioProcess(t *testing.T, home, name string, port int) {
	t.Helper()
	record := process.Record{
		PID:       os.Getpid(),
		PGID:      os.Getpid(),
		Scenario:  name,
		Step:      "start-api",
		Port:      port,
		StartedAt: time.Now().Add(-time.Minute).UTC(),
		Status:    "running",
	}
	if err := process.WriteScenarioRecord(home, name, "start-api", record); err != nil {
		t.Fatalf("WriteScenarioRecord: %v", err)
	}
}

func writeProjectPortRegistry(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "scripts", "resources")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(filepath.Join(path, "port_registry.sh"), []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write port_registry.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "port_registry.json"), []byte("{\n  \"resource_ports\": {},\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write port_registry.json: %v", err)
	}
}
