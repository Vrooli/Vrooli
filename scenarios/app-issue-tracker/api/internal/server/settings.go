package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"app-issue-tracker-api/internal/logging"
)

// AgentSettings holds configuration for AI agent execution
type AgentSettings struct {
	MaxTurns        int    `json:"max_turns"`
	AllowedTools    string `json:"allowed_tools"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	SkipPermissions bool   `json:"skip_permissions"`
	AgentTag        string `json:"agent_tag"`
	Provider        string `json:"provider"`
	CLICommand      string `json:"cli_command"`
	Command         []string
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
		MaxTurns:        80,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		TimeoutSeconds:  600,
		SkipPermissions: true,
		Provider:        "claude-code",
		CLICommand:      "resource-claude-code",
		Command:         []string{"run", "--tag", "{{TAG}}", "-"},
	}

	// Try to load from agent-settings.json
	configPath := filepath.Join(scenarioRootPath, "initialization", "configuration", "agent-settings.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		logging.LogWarn("Could not load agent settings file", "path", configPath, "error", err)
		return fmt.Errorf("load agent settings: %w", err)
	}

	var config struct {
		AgentBackend struct {
			Provider        string `json:"provider"`
			SkipPermissions *bool  `json:"skip_permissions"`
		} `json:"agent_backend"`
		Providers map[string]struct {
			CLICommand string `json:"cli_command"`
			Operations map[string]struct {
				MaxTurns       int    `json:"max_turns"`
				AllowedTools   string `json:"allowed_tools"`
				TimeoutSeconds int    `json:"timeout_seconds"`
				Command        string `json:"command"`
			} `json:"operations"`
		} `json:"providers"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		logging.LogWarn("Failed to parse agent settings file", "path", configPath, "error", err)
		return fmt.Errorf("parse agent settings: %w", err)
	}

	// Extract provider settings
	provider := config.AgentBackend.Provider
	if provider == "" {
		provider = "claude-code"
	}

	if config.AgentBackend.SkipPermissions != nil {
		agentSettings.SkipPermissions = *config.AgentBackend.SkipPermissions
	}

	providerConfig, ok := config.Providers[provider]
	if !ok {
		err := fmt.Errorf("provider %q missing from agent settings", provider)
		logging.LogWarn("Provider missing from agent settings", "provider", provider)
		return err
	}

	// Load investigation operation settings
	investigateOp, ok := providerConfig.Operations["investigate"]
	if ok {
		if investigateOp.MaxTurns > 0 {
			agentSettings.MaxTurns = investigateOp.MaxTurns
		}
		if investigateOp.AllowedTools != "" {
			agentSettings.AllowedTools = investigateOp.AllowedTools
		}
		if investigateOp.TimeoutSeconds > 0 {
			agentSettings.TimeoutSeconds = investigateOp.TimeoutSeconds
		}
		if command := strings.TrimSpace(investigateOp.Command); command != "" {
			if parts, err := parseCommandParts(command); err != nil {
				logging.LogWarn("Failed to parse investigate command", "raw_command", command, "error", err)
			} else if len(parts) > 0 {
				agentSettings.Command = parts
			}
		}
	}

	agentSettings.Provider = provider
	if providerConfig.CLICommand != "" {
		agentSettings.CLICommand = providerConfig.CLICommand
	}

	logging.LogInfo(
		"Agent settings loaded",
		"provider", agentSettings.Provider,
		"cli_command", agentSettings.CLICommand,
		"command", strings.Join(agentSettings.Command, " "),
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

func parseCommandParts(input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	var parts []string
	var current strings.Builder
	var inQuote rune
	escaped := false

	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			inQuote = r
		case unicode.IsSpace(r):
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape sequence")
	}
	if inQuote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}
