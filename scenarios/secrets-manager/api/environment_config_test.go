//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"strings"
	"testing"

	"secrets-manager-api/internal/testutil/mocks"
)

func TestLoadStartupEnvironment(t *testing.T) {
	t.Run("accepts explicit booleans", func(t *testing.T) {
		settings, err := loadStartupEnvironment(mocks.FakeEnv{
			"VROOLI_DESKTOP_MODE":     "true",
			"SECRETS_MANAGER_SKIP_DB": "false",
		})
		if err != nil {
			t.Fatalf("loadStartupEnvironment() error = %v", err)
		}
		if !settings.desktopMode || settings.skipDB {
			t.Fatalf("settings = %+v, want desktop=true skipDB=false", settings)
		}
	})

	t.Run("rejects malformed boolean", func(t *testing.T) {
		_, err := loadStartupEnvironment(mocks.FakeEnv{"SECRETS_MANAGER_SKIP_DB": "sometimes"})
		if err == nil || !strings.Contains(err.Error(), "SECRETS_MANAGER_SKIP_DB") {
			t.Fatalf("loadStartupEnvironment() error = %v, want named validation error", err)
		}
	})
}

func TestOptionalScenarioDirectory(t *testing.T) {
	t.Run("accepts an absolute lifecycle directory", func(t *testing.T) {
		directory, err := optionalScenarioDirectory(mocks.FakeEnv{"VROOLI_SCENARIO_DIR": "/tmp/scenario"})
		if err != nil || directory != "/tmp/scenario" {
			t.Fatalf("optionalScenarioDirectory() = %q, %v", directory, err)
		}
	})

	t.Run("rejects a relative lifecycle directory", func(t *testing.T) {
		_, err := optionalScenarioDirectory(mocks.FakeEnv{"VROOLI_SCENARIO_DIR": "relative"})
		if err == nil || !strings.Contains(err.Error(), "absolute path") {
			t.Fatalf("optionalScenarioDirectory() error = %v, want absolute-path error", err)
		}
	})
}
