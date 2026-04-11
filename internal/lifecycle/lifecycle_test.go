package lifecycle

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func TestRunnerStartStopRestart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	start, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if start.Health != "healthy" {
		t.Fatalf("health = %q, want healthy", start.Health)
	}
	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	live := process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after start, got %#v", live)
	}
	firstPID := live[0].PID

	setupNeeded, _, err := runner.SetupNeeded(start.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded after start: %v", err)
	}
	if setupNeeded {
		t.Fatalf("expected setup to be current after start")
	}

	restarted, err := runner.Restart("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if restarted.Health != "healthy" {
		t.Fatalf("restart health = %q, want healthy", restarted.Health)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after restart: %v", err)
	}
	live = process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("expected 1 live record after restart, got %#v", live)
	}
	if live[0].PID == firstPID {
		t.Fatalf("expected new PID after restart, still %d", firstPID)
	}

	if err := runner.Stop("alpha", StopOptions{}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after stop: %v", err)
	}
	if len(process.LiveRecords(records)) != 0 {
		t.Fatalf("expected no live records after stop: %#v", records)
	}
}

func TestSetupNeededDetectsUpdatedSources(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	result, err := runner.Start("alpha", StartOptions{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	sourcePath := filepath.Join(root, "scenarios", "alpha", "api", "handler.go")
	if err := os.WriteFile(sourcePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", sourcePath, err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(sourcePath, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", sourcePath, err)
	}

	setupNeeded, reasons, err := runner.SetupNeeded(result.Scenario, false)
	if err != nil {
		t.Fatalf("SetupNeeded: %v", err)
	}
	if !setupNeeded {
		t.Fatalf("expected setup to be needed after touching source")
	}
	if len(reasons) == 0 {
		t.Fatalf("expected setup reasons to be populated")
	}

	_ = runner.Stop("alpha", StopOptions{})
}

func TestFileDependencySpecsIgnoresNonDependencyTopLevelFields(t *testing.T) {
	packageJSON := filepath.Join(t.TempDir(), "package.json")
	data := `{
  "name": "fixture",
  "version": "1.0.0",
  "scripts": {
    "build": "vite build"
  },
  "dependencies": {
    "@local/pkg-a": "file:../pkg-a",
    "react": "^18.0.0"
  },
  "devDependencies": {
    "@local/pkg-b": "file:../pkg-b"
  },
  "optionalDependencies": {
    "@local/pkg-c": "file:../pkg-c"
  }
}`
	if err := os.WriteFile(packageJSON, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", packageJSON, err)
	}

	specs, err := fileDependencySpecs(packageJSON)
	if err != nil {
		t.Fatalf("fileDependencySpecs: %v", err)
	}

	want := []string{"file:../pkg-a", "file:../pkg-b", "file:../pkg-c"}
	if len(specs) != len(want) {
		t.Fatalf("spec count = %d, want %d (%v)", len(specs), len(want), specs)
	}
	for i, spec := range specs {
		if spec != want[i] {
			t.Fatalf("spec[%d] = %q, want %q", i, spec, want[i])
		}
	}
}

func TestEnsureScenarioDatabaseUsesPostgresResourceLibs(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "scripts", "resources"), 0o755); err != nil {
		t.Fatalf("mkdir scripts/resources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "resources", "port_registry.sh"), []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write port_registry.sh: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(root, "resources", "postgres", "config"), 0o755); err != nil {
		t.Fatalf("mkdir postgres config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "postgres", "lib"), 0o755); err != nil {
		t.Fatalf("mkdir postgres lib: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "config", "defaults.sh"), []byte("#!/usr/bin/env bash\n"), 0o644); err != nil {
		t.Fatalf("write defaults.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "lib", "common.sh"), []byte(`#!/usr/bin/env bash
postgres::common::is_running() { return 0; }
`), 0o644); err != nil {
		t.Fatalf("write common.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "resources", "postgres", "lib", "database.sh"), []byte(`#!/usr/bin/env bash
postgres::database::create() { printf '%s\n' "$2" > "$APP_ROOT/create.txt"; }
postgres::database::execute_file() { printf '%s|%s\n' "$2" "$3" > "$APP_ROOT/schema.txt"; }
postgres::database::migrate() { printf '%s|%s\n' "$2" "$3" > "$APP_ROOT/migrate.txt"; }
`), 0o644); err != nil {
		t.Fatalf("write database.sh: %v", err)
	}

	scenarioPath := filepath.Join(root, "scenarios", "alpha")
	if err := os.MkdirAll(filepath.Join(scenarioPath, "initialization", "postgres"), 0o755); err != nil {
		t.Fatalf("mkdir initialization/postgres: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "initialization", "postgres", "schema.sql"), []byte("create table if not exists test();\n"), 0o644); err != nil {
		t.Fatalf("write schema.sql: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, "initialization", "postgres", "migration_001.sql"), []byte("-- migration\n"), 0o644); err != nil {
		t.Fatalf("write migration_001.sql: %v", err)
	}

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	item := scenario.Scenario{
		Slug: "alpha",
		Path: scenarioPath,
	}
	if err := runner.ensureScenarioDatabase(item, map[string]string{"POSTGRES_DB": "alpha_db"}, io.Discard); err != nil {
		t.Fatalf("ensureScenarioDatabase: %v", err)
	}

	createData, err := os.ReadFile(filepath.Join(root, "create.txt"))
	if err != nil {
		t.Fatalf("read create.txt: %v", err)
	}
	if got := string(createData); got != "alpha_db\n" {
		t.Fatalf("create.txt = %q", got)
	}

	schemaData, err := os.ReadFile(filepath.Join(root, "schema.txt"))
	if err != nil {
		t.Fatalf("read schema.txt: %v", err)
	}
	if got := string(schemaData); got != filepath.Join(scenarioPath, "initialization", "postgres", "schema.sql")+"|alpha_db\n" {
		t.Fatalf("schema.txt = %q", got)
	}

	migrateData, err := os.ReadFile(filepath.Join(root, "migrate.txt"))
	if err != nil {
		t.Fatalf("read migrate.txt: %v", err)
	}
	if got := string(migrateData); got != filepath.Join(scenarioPath, "initialization", "postgres")+"|alpha_db\n" {
		t.Fatalf("migrate.txt = %q", got)
	}
}

func writeLifecycleFixture(t *testing.T, root, name string) {
	t.Helper()

	portRegistry := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.MkdirAll(filepath.Dir(portRegistry), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(portRegistry), err)
	}
	if err := os.WriteFile(portRegistry, []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistry, err)
	}

	scenarioDir := filepath.Join(root, "scenarios", name)
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	servicePath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Lifecycle ` + name + `",
    "description": "Lifecycle validation fixture",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "22000-22010"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "checks": [
        {
          "name": "api",
          "type": "http",
          "target": "http://127.0.0.1:${API_PORT}/health",
          "critical": true,
          "timeout": 1000
        }
      ],
      "startup_grace_period": 250,
      "timeout": 5000,
      "interval": 250
    },
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": [
              "api/mock-api"
            ]
          }
        ]
      },
      "steps": [
        {
          "name": "build-api",
          "run": "mkdir -p api public && printf 'package main\n' > api/handler.go && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./mock-api",
          "background": true,
          "condition": {
            "file_exists": "api/mock-api"
          }
        }
      ]
    }
  }
}`
	if err := os.WriteFile(servicePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}
}
