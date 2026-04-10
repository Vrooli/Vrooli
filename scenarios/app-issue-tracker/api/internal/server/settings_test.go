package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeAgentSettingsFile(t *testing.T, root string, runnerType string, maxTurns int) {
	t.Helper()
	configDir := filepath.Join(root, "initialization", "configuration")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	data := map[string]any{
		"agent_manager": map[string]any{
			"runner_type":      runnerType,
			"max_turns":        maxTurns,
			"allowed_tools":    "Read",
			"timeout_seconds":  90,
			"skip_permissions": true,
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

	writeAgentSettingsFile(t, first, "claude-code", 42)
	writeAgentSettingsFile(t, second, "codex", 55)

	resetAgentSettingsForTest()
	t.Cleanup(resetAgentSettingsForTest)

	settings, err := LoadAgentSettings(first)
	if err != nil {
		t.Fatalf("first load failed: %v", err)
	}
	if settings.RunnerType != "claude-code" {
		t.Fatalf("expected runner claude-code, got %s", settings.RunnerType)
	}

	settings, err = LoadAgentSettings(second)
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if settings.RunnerType != "codex" {
		t.Fatalf("expected runner codex, got %s", settings.RunnerType)
	}
}

func TestReloadAgentSettingsRequiresLoad(t *testing.T) {
	resetAgentSettingsForTest()
	if err := ReloadAgentSettings(); err == nil {
		t.Fatalf("expected error when reloading without LoadAgentSettings call")
	}
}
