package scenario

import (
	"errors"
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
	cleanup := SetTestHooks(
		func() (string, error) { return "/home/user/Vrooli/scenarios/chart-generator/api", nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "chart-generator" {
		t.Errorf("expected chart-generator, got %s", name)
	}
}

func TestName_FromDirectory_Subdirectory(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "/home/user/Vrooli/scenarios/my-app/api/internal/handlers", nil },
		func(key string) string { return "" },
	)
	defer cleanup()

	name := Name()
	if name != "my-app" {
		t.Errorf("expected my-app, got %s", name)
	}
}

func TestName_EnvTakesPriority(t *testing.T) {
	cleanup := SetTestHooks(
		func() (string, error) { return "/home/user/Vrooli/scenarios/from-dir/api", nil },
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
	cleanup := SetTestHooks(
		func() (string, error) { return "/scenarios/test-app/api", nil },
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
	cleanup := SetTestHooks(
		func() (string, error) {
			callCount++
			return "/scenarios/cached-test/api", nil
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
	cleanup := SetTestHooks(
		func() (string, error) {
			callCount++
			return "/scenarios/reset-test/api", nil
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
