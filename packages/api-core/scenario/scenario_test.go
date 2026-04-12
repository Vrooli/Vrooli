package scenario

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestName_FromEnv(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "/some/path", nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "my-scenario"
			}
			return ""
		},
	)
	defer cleanup()

	name := Name()
	if name != "my-scenario" {
		t.Errorf("expected my-scenario, got %s", name)
	}
}

func TestName_FromEnv_Trimmed(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "/some/path", nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "  my-scenario  "
			}
			return ""
		},
	)
	defer cleanup()

	name := Name()
	if name != "my-scenario" {
		t.Errorf("expected my-scenario, got %s", name)
	}
}

func TestName_FromDirectory(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "chart-generator", "api"), nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "chart-generator" {
		t.Errorf("expected chart-generator, got %s", name)
	}
}

func TestName_FromDirectory_Subdirectory(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "my-app", "api", "internal", "handlers"), nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "my-app" {
		t.Errorf("expected my-app, got %s", name)
	}
}

func TestName_EnvTakesPriority(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "from-dir", "api"), nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "from-env"
			}
			return ""
		},
	)
	defer cleanup()

	name := Name()
	if name != "from-env" {
		t.Errorf("expected from-env (env takes priority), got %s", name)
	}
}

func TestName_Unknown(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "/some/random/path", nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "unknown" {
		t.Errorf("expected unknown, got %s", name)
	}
}

func TestName_GetwdError(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "", errors.New("getwd failed") },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "unknown" {
		t.Errorf("expected unknown on getwd error, got %s", name)
	}
}

func TestServiceName(t *testing.T) {
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) { return filepath.Join(root, "scenarios", "test-app", "api"), nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	svc := ServiceName()
	if svc != "test-app-api" {
		t.Errorf("expected test-app-api, got %s", svc)
	}
}

func TestName_Cached(t *testing.T) {
	callCount := 0
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) {
			callCount++
			return filepath.Join(root, "scenarios", "cached-test", "api"), nil
		},
		func(key string) string { return "" },
	)
	defer cleanup()

	// Call multiple times
	_ = Name()
	_ = Name()
	_ = Name()

	// getwd should only be called once due to caching
	if callCount != 1 {
		t.Errorf("expected getwd to be called once, got %d", callCount)
	}
}

func TestReset(t *testing.T) {
	callCount := 0
	root := writeRepoFixture(t)
	cleanup := SetTestHooks(
		func() (string, error) {
			callCount++
			return filepath.Join(root, "scenarios", "reset-test", "api"), nil
		},
		func(key string) string { return "" },
	)
	defer cleanup()

	_ = Name()
	Reset()
	_ = Name()

	// After reset, getwd should be called again
	if callCount != 2 {
		t.Errorf("expected getwd to be called twice after reset, got %d", callCount)
	}
}

func writeRepoFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	for _, dir := range []string{
		".vrooli/schemas",
		"scenarios/chart-generator/.vrooli",
		"scenarios/chart-generator/api",
		"scenarios/my-app/.vrooli",
		"scenarios/my-app/api/internal/handlers",
		"scenarios/from-dir/.vrooli",
		"scenarios/from-dir/api",
		"scenarios/test-app/.vrooli",
		"scenarios/test-app/api",
		"scenarios/cached-test/.vrooli",
		"scenarios/cached-test/api",
		"scenarios/reset-test/.vrooli",
		"scenarios/reset-test/api",
		"resources",
		"packages",
		"cmd",
		"internal",
	} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	for _, scenario := range []string{"chart-generator", "my-app", "from-dir", "test-app", "cached-test", "reset-test"} {
		path := filepath.Join(root, "scenarios", scenario, ".vrooli", "service.json")
		if err := os.WriteFile(path, []byte(`{"service":{"name":"`+scenario+`"}}`), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	sourceRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("Abs(source root) error = %v", err)
	}
	contractBytes, err := os.ReadFile(filepath.Join(sourceRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("ReadFile(repo-contract.json) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractBytes, 0o644); err != nil {
		t.Fatalf("WriteFile(repo-contract.json) error = %v", err)
	}

	return root
}
