package scenario_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	apiscenario "github.com/vrooli/api-core/scenario"
	repocontract "github.com/vrooli/repo-contract-go"
)

func TestName_FromEnv(t *testing.T) {
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return "/some/path", nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "my-scenario"
			}
			return ""
		},
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "my-scenario" {
		t.Errorf("expected my-scenario, got %s", name)
	}
}

func TestName_FromEnv_Trimmed(t *testing.T) {
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return "/some/path", nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "  my-scenario  "
			}
			return ""
		},
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "my-scenario" {
		t.Errorf("expected my-scenario, got %s", name)
	}
}

func TestName_FromDirectory(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "chart-generator", "api"), nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "chart-generator" {
		t.Errorf("expected chart-generator, got %s", name)
	}
}

func TestName_FromDirectory_Subdirectory(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) {
			return filepath.Join(root, "scenarios", "my-app", "api", "internal", "handlers"), nil
		},
		func(key string) string { return "" },
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "my-app" {
		t.Errorf("expected my-app, got %s", name)
	}
}

func TestName_EnvTakesPriority(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "from-dir", "api"), nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "from-env"
			}
			return ""
		},
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "from-env" {
		t.Errorf("expected from-env (env takes priority), got %s", name)
	}
}

func TestName_Unknown(t *testing.T) {
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return "/some/random/path", nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "unknown" {
		t.Errorf("expected unknown, got %s", name)
	}
}

func TestName_GetwdError(t *testing.T) {
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return "", errors.New("getwd failed") },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := apiscenario.Name()
	if name != "unknown" {
		t.Errorf("expected unknown on getwd error, got %s", name)
	}
}

func TestServiceName(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "test-app", "api"), nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	svc := apiscenario.ServiceName()
	if svc != "test-app-api" {
		t.Errorf("expected test-app-api, got %s", svc)
	}
}

func TestName_Cached(t *testing.T) {
	callCount := 0
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) {
			callCount++
			return filepath.Join(root, "scenarios", "cached-test", "api"), nil
		},
		func(key string) string { return "" },
	)
	defer cleanup()

	_ = apiscenario.Name()
	_ = apiscenario.Name()
	_ = apiscenario.Name()

	if callCount != 1 {
		t.Errorf("expected getwd to be called once, got %d", callCount)
	}
}

func TestReset(t *testing.T) {
	callCount := 0
	root := writeRepoFixture(t)
	cleanup := apiscenario.SetTestHooks(
		func() (string, error) {
			callCount++
			return filepath.Join(root, "scenarios", "reset-test", "api"), nil
		},
		func(key string) string { return "" },
	)
	defer cleanup()

	_ = apiscenario.Name()
	apiscenario.Reset()
	_ = apiscenario.Name()

	if callCount != 2 {
		t.Errorf("expected getwd to be called twice after reset, got %d", callCount)
	}
}

func writeRepoFixture(t *testing.T) string {
	t.Helper()
	sourceRoot := repoRoot(t)
	contract, err := repocontract.LoadDefault(sourceRoot)
	if err != nil {
		t.Fatalf("LoadDefault(source root) error = %v", err)
	}

	root := t.TempDir()
	for _, dir := range contract.RootMarkers().RequiredDirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	contractBytes, err := os.ReadFile(filepath.Join(sourceRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("ReadFile(repo-contract.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractBytes, 0o644); err != nil {
		t.Fatalf("WriteFile(repo-contract.json) error = %v", err)
	}

	scenarios := map[string]string{
		"chart-generator": "api",
		"my-app":          filepath.Join("api", "internal", "handlers"),
		"from-dir":        "api",
		"test-app":        "api",
		"cached-test":     "api",
		"reset-test":      "api",
	}
	for name, relPath := range scenarios {
		writeScenarioFixture(t, contract, root, name, relPath)
	}

	return root
}

func writeScenarioFixture(t *testing.T, contract *repocontract.Contract, root, name, relPath string) {
	t.Helper()
	scenarioRoot, err := contract.ScenarioRoot(root, name)
	if err != nil {
		t.Fatalf("ScenarioRoot(%q) error = %v", name, err)
	}
	if relPath != "" {
		if err := os.MkdirAll(filepath.Join(scenarioRoot, filepath.FromSlash(relPath)), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", relPath, err)
		}
	}

	servicePath, err := contract.ScenarioFile(root, name, "service")
	if err != nil {
		t.Fatalf("ScenarioFile(%q, service) error = %v", name, err)
	}
	data, err := json.Marshal(map[string]any{
		"service": map[string]any{
			"name": name,
		},
	})
	if err != nil {
		t.Fatalf("Marshal(%q) error = %v", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(service dir) error = %v", err)
	}
	if err := os.WriteFile(servicePath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", servicePath, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs(source root) error = %v", err)
	}
	return root
}
