package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"app-issue-tracker-api/internal/logging"
)

// AgentSettings holds configuration for AI agent execution
type AgentSettings struct {
	RunnerType      string `json:"runner_type"`
	MaxTurns        int    `json:"max_turns"`
	AllowedTools    string `json:"allowed_tools"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	SkipPermissions bool   `json:"skip_permissions"`
}

var (
	agentSettings    AgentSettings
	agentSettingsMu  sync.RWMutex
	scenarioRootPath string
	agentSettingsErr error
)

// LoadAgentSettings loads agent settings from configuration file. If called with a
// different scenario root than previous invocations, the settings are reloaded so
// tests and multi-scenario runs can supply their own configuration roots.
func LoadAgentSettings(scenarioRoot string) (AgentSettings, error) {
	cleaned := strings.TrimSpace(scenarioRoot)
	if cleaned == "" {
		return AgentSettings{}, fmt.Errorf("scenario root must not be empty")
	}
	cleaned = filepath.Clean(cleaned)

	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()

	if scenarioRootPath == "" || filepath.Clean(scenarioRootPath) != cleaned {
		scenarioRootPath = cleaned
		agentSettingsErr = reloadAgentSettingsLocked()
	} else if agentSettingsErr != nil {
		// Allow callers to recover from a previous failure by retrying the load.
		agentSettingsErr = reloadAgentSettingsLocked()
	}

	return agentSettings, agentSettingsErr
}

// ReloadAgentSettings reloads settings from disk (for runtime updates)
func ReloadAgentSettings() error {
	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()
	if strings.TrimSpace(scenarioRootPath) == "" {
		return fmt.Errorf("agent settings scenario root not configured")
	}
	agentSettingsErr = reloadAgentSettingsLocked()
	return agentSettingsErr
}

// reloadAgentSettingsLocked performs the actual loading. Caller must hold agentSettingsMu.
func reloadAgentSettingsLocked() error {
	// Default settings (aligned with ecosystem-manager)
	agentSettings = AgentSettings{
		RunnerType:      "claude-code",
		MaxTurns:        80,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		TimeoutSeconds:  600,
		SkipPermissions: true,
	}

	// Try to load from agent-settings.json
	configPath := filepath.Join(scenarioRootPath, "initialization", "configuration", "agent-settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		logging.LogWarn("Could not load agent settings file", "path", configPath, "error", err)
		return fmt.Errorf("load agent settings: %w", err)
	}

	var config struct {
		AgentManager struct {
			RunnerType      string `json:"runner_type"`
			MaxTurns        int    `json:"max_turns"`
			AllowedTools    string `json:"allowed_tools"`
			TimeoutSeconds  int    `json:"timeout_seconds"`
			SkipPermissions *bool  `json:"skip_permissions"`
		} `json:"agent_manager"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		logging.LogWarn("Failed to parse agent settings file", "path", configPath, "error", err)
		return fmt.Errorf("parse agent settings: %w", err)
	}

	if config.AgentManager.RunnerType != "" {
		agentSettings.RunnerType = strings.TrimSpace(config.AgentManager.RunnerType)
	}
	if config.AgentManager.MaxTurns > 0 {
		agentSettings.MaxTurns = config.AgentManager.MaxTurns
	}
	if config.AgentManager.AllowedTools != "" {
		agentSettings.AllowedTools = strings.TrimSpace(config.AgentManager.AllowedTools)
	}
	if config.AgentManager.TimeoutSeconds > 0 {
		agentSettings.TimeoutSeconds = config.AgentManager.TimeoutSeconds
	}
	if config.AgentManager.SkipPermissions != nil {
		agentSettings.SkipPermissions = *config.AgentManager.SkipPermissions
	}

	logging.LogInfo(
		"Agent settings loaded",
		"runner_type", agentSettings.RunnerType,
		"max_turns", agentSettings.MaxTurns,
		"timeout_seconds", agentSettings.TimeoutSeconds,
		"timeout_minutes", agentSettings.TimeoutSeconds/60,
	)

	return nil
}

// GetAgentSettings returns current agent settings (thread-safe)
func GetAgentSettings() AgentSettings {
	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return agentSettings
}

// resetAgentSettingsForTest clears cached agent settings allowing tests to provide
// isolated configuration roots. It is a no-op in production code paths.
func resetAgentSettingsForTest() {
	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()
	agentSettings = AgentSettings{}
	scenarioRootPath = ""
	agentSettingsErr = nil
}

// setAgentSettingsForTest seeds the cached agent settings for unit tests.
func setAgentSettingsForTest(settings AgentSettings, root string) {
	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()
	agentSettings = settings
	scenarioRootPath = root
	agentSettingsErr = nil
}
