package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newTestApp(root string) *App {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return root, nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	return app
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
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func writeTestScenarioService(t *testing.T, root, name, description string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "` + strings.Title(strings.ReplaceAll(name, "-", " ")) + `",
    "description": "` + description + `",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "develop": {
      "description": "Run the scenario",
      "steps": [
        {
          "name": "start-api",
          "run": "sleep 10",
          "background": true
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeProjectLifecycleFixture(t *testing.T, root string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	path := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "VROOLI_API_PORT",
      "port": 8092
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "data",
            "path": "data"
          }
        ]
      },
      "steps": [
        {
          "name": "capture-setup",
          "run": "mkdir -p data build && printf 'setup\n' >> build/setup-count.txt && printf '%s|%s|%s|%s|%s|%s|%s|%s|%s\n' \"${ENVIRONMENT:-}\" \"${RESOURCES:-}\" \"${YES:-}\" \"${SUDO_MODE:-}\" \"${TARGET:-}\" \"${LOCATION:-}\" \"${DRY_RUN:-false}\" \"${APP_ROOT:-}\" \"${SERVICE_JSON_PATH:-}\" > build/setup-env.txt && printf 'ready\n' > data/bootstrap.txt"
        },
        {
          "name": "add-data",
          "run": "printf 'data\n' >> data/bootstrap.txt"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "capture-develop",
          "run": "mkdir -p build && printf 'develop\n' >> build/develop-count.txt && printf '%s\n' \"${VROOLI_API_PORT:-}\" > build/develop-port.txt"
        }
      ]
    },
    "build": {
      "steps": [
        {
          "name": "capture-build",
          "run": "mkdir -p build && printf 'build\n' > build/build.txt"
        }
      ]
    },
    "clean": {
      "steps": [
        {
          "name": "capture-clean",
          "run": "mkdir -p build && printf 'clean\n' > build/clean.txt"
        }
      ]
    },
    "deploy": {
      "steps": [
        {
          "name": "capture-deploy",
          "run": "mkdir -p build && printf 'deploy\n' > build/deploy.txt"
        }
      ]
    },
    "backup": {
      "steps": [
        {
          "name": "capture-backup",
          "run": "mkdir -p build && printf 'backup\n' > build/backup.txt"
        }
      ]
    },
    "restore": {
      "steps": [
        {
          "name": "capture-restore",
          "run": "mkdir -p build && printf 'restore\n' > build/restore.txt"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	portRegistryPath := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.WriteFile(portRegistryPath, []byte("#!/usr/bin/env bash\nRESOURCE_PORTS=( [\"vrooli-api\"]=\"8092\" )\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryPath, err)
	}
	portRegistryJSONPath := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.WriteFile(portRegistryJSONPath, []byte("{\n  \"resource_ports\": {\n    \"vrooli-api\": 8092\n  },\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryJSONPath, err)
	}
}

func writeResourceStatusFixture(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	writeTestFile(t, root, ".vrooli/service.json", `{
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha"
  },
  "dependencies": {
    "resources": {
      "`+name+`": {
        "enabled": true
      }
    }
  }
}`)
	writeTestFile(t, root, filepath.Join("resources", name, "resource.json"), `{
  "name": "`+name+`",
  "display_name": "`+name+`",
  "description": "Fixture resource",
  "template": "legacy-adapter",
  "driver": "legacy-adapter",
  "legacy_adapter": {
    "owner": "CLI tests",
    "decision_deadline": "2026-12-31",
    "final_disposition": "migrate",
    "legacy_cli_path": "resources/`+name+`/cli.sh"
  },
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "partial",
    "windows": "unsupported"
  }
}`)
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	writeFakeExecutable(t, root, filepath.Join("resources", name, "cli.sh"), script)
}

func writeScenarioSetupOnlyFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Setup ` + strings.Title(name) + `",
    "description": "Setup validation scenario",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0",
    "setup": {
      "steps": [
        {
          "name": "write-file",
          "run": "mkdir -p build && printf 'ok\n' > build/setup.txt"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioWithoutSetupFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "No Setup ` + strings.Title(name) + `",
    "description": "Scenario without setup phase",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioTestPhaseFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	scriptPath := filepath.Join(root, "scenarios", name, "run-test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nset -e\nmkdir -p coverage\nprintf '%s\\n' \"$1\" > coverage/selector.txt\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Test ` + strings.Title(name) + `",
    "description": "Test validation scenario",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0",
    "test": {
      "steps": [
        {
          "name": "run-tests",
          "run": "./run-test.sh"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioServiceWithPorts(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Ports ` + strings.Title(name) + `",
    "description": "Port validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "sleep 10",
          "background": true
        },
        {
          "name": "start-ui",
          "run": "sleep 10",
          "background": true
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
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
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	writeScenarioPortRegistryFixture(t, root)
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Lifecycle ` + strings.Title(name) + `",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
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
      "startup_grace_period": 1000,
      "timeout": 30000,
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
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
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
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeLifecycleScenarioServiceAtPath(t *testing.T, root, scenarioPath, name string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	servicePath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(servicePath), err)
	}

	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Lifecycle ` + strings.Title(name) + `",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
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
      "startup_grace_period": 1000,
      "timeout": 30000,
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
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
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

func writeFixedPortLifecycleScenarioService(t *testing.T, root, name string, port int) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data := fmt.Sprintf(`{
  "version": "1.0.0",
  "service": {
    "name": %q,
    "displayName": "Lifecycle %s",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "port": %d
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
      "startup_grace_period": 1000,
      "timeout": 30000,
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
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
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
}`, name, strings.Title(name), port)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeBestEffortLifecycleScenarioService(t *testing.T, root, name, dependency string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data := fmt.Sprintf(`{
  "version": "1.0.0",
  "service": {
    "name": %q,
    "displayName": "Lifecycle %s",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
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
      "startup_grace_period": 1000,
      "timeout": 30000,
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
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
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
  },
  "dependencies": {
    "scenarios": {
      %q: {
        "type": "scenario",
        "required": true
      }
    }
  }
}`, name, strings.Title(name), dependency)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioPortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	portRegistryPath := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.MkdirAll(filepath.Dir(portRegistryPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(portRegistryPath), err)
	}
	if err := os.WriteFile(portRegistryPath, []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryPath, err)
	}
	portRegistryJSONPath := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.WriteFile(portRegistryJSONPath, []byte("{\n  \"resource_ports\": {},\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryJSONPath, err)
	}
}

func writeScenarioProcessRecord(t *testing.T, home, name, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", name, step+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + fmt.Sprintf("%d", pid) + `,
  "pgid": ` + fmt.Sprintf("%d", pid) + `,
  "process_id": "vrooli.develop.` + name + `.` + step + `",
  "phase": "develop",
  "scenario": "` + name + `",
  "step": "` + step + `",
  "command": "sleep 10",
  "working_dir": "` + filepath.Join("/repo/scenarios", name) + `",
  "log_file": "/tmp/` + name + `.log",
  "port": ` + fmt.Sprintf("%d", port) + `,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioProcessRecordWithWorkingDir(t *testing.T, home, name, step string, pid, port int, startedAt time.Time, workingDir string) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", name, step+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + fmt.Sprintf("%d", pid) + `,
  "pgid": ` + fmt.Sprintf("%d", pid) + `,
  "process_id": "vrooli.develop.` + name + `.` + step + `",
  "phase": "develop",
  "scenario": "` + name + `",
  "step": "` + step + `",
  "command": "sleep 10",
  "working_dir": "` + workingDir + `",
  "log_file": "/tmp/` + name + `.log",
  "port": ` + fmt.Sprintf("%d", port) + `,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
