package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeAgentSettingsFile(t *testing.T, root string, provider string, cli string) {
	t.Helper()
	configDir := filepath.Join(root, "initialization", "configuration")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	data := map[string]any{
		"agent_backend": map[string]any{
			"provider": provider,
		},
		"providers": map[string]any{
			provider: map[string]any{
				"cli_command": cli,
				"operations": map[string]any{
					"investigate": map[string]any{
						"max_turns":       42,
						"allowed_tools":   "Read",
						"timeout_seconds": 90,
						"command":         "run --tag {{TAG}} -",
					},
				},
			},
		},
	}

	file := filepath.Join(configDir, "agent-settings.json")
	contents, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal agent settings: %v", err)
	}
	if err := os.WriteFile(file, contents, 0o644); err != nil {
		t.Fatalf("failed to write agent settings: %v", err)
	}
}

func TestLoadAgentSettingsAllowsScenarioSwitch(t *testing.T) {
	tempDir := t.TempDir()
	first := filepath.Join(tempDir, "scenario1")
	second := filepath.Join(tempDir, "scenario2")

	writeAgentSettingsFile(t, first, "claude-code", "resource-claude-code")
	writeAgentSettingsFile(t, second, "custom-provider", "custom-cli")

	resetAgentSettingsForTest()
	t.Cleanup(resetAgentSettingsForTest)

	settings, err := LoadAgentSettings(first)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if settings.Provider != "claude-code" {
		t.Fatalf("expected provider claude-code, got %s", settings.Provider)
	}

	settings, err = LoadAgentSettings(second)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if settings.Provider != "custom-provider" {
		t.Fatalf("expected provider custom-provider, got %s", settings.Provider)
	}
	if settings.CLICommand != "custom-cli" {
		t.Fatalf("expected cli command custom-cli, got %s", settings.CLICommand)
	}
}

func TestReloadAgentSettingsRequiresLoad(t *testing.T) {
	resetAgentSettingsForTest()
	if err := ReloadAgentSettings(); err == nil {
		t.Fatalf("expected error when reloading without LoadAgentSettings call")
	}
}
