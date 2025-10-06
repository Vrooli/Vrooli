package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
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
	agentSettings     AgentSettings
	agentSettingsMu   sync.RWMutex
	agentSettingsOnce sync.Once
	scenarioRootPath  string
)

// LoadAgentSettings loads agent settings from configuration file
// This is called once at startup to initialize settings
func LoadAgentSettings(scenarioRoot string) AgentSettings {
	agentSettingsOnce.Do(func() {
		scenarioRootPath = scenarioRoot
		reloadAgentSettings()
	})

	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return agentSettings
}

// ReloadAgentSettings reloads settings from disk (for runtime updates)
func ReloadAgentSettings() {
	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()
	reloadAgentSettings()
}

// reloadAgentSettings performs the actual loading (must be called with lock held or from Once)
func reloadAgentSettings() {
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
		log.Printf("Warning: Could not load agent-settings.json: %v (using defaults)", err)
		return
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
		log.Printf("Warning: Failed to parse agent-settings.json: %v (using defaults)", err)
		return
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
		log.Printf("Warning: Provider '%s' not found in agent-settings.json (using defaults)", provider)
		return
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
				log.Printf("Warning: Failed to parse investigate command '%s': %v (using default)", command, err)
			} else if len(parts) > 0 {
				agentSettings.Command = parts
			}
		}
	}

	agentSettings.Provider = provider
	if providerConfig.CLICommand != "" {
		agentSettings.CLICommand = providerConfig.CLICommand
	}

	log.Printf("Loaded agent settings: provider=%s, cli=%s, command=%v, max_turns=%d, timeout=%ds (%dmin)",
		agentSettings.Provider, agentSettings.CLICommand, agentSettings.Command, agentSettings.MaxTurns, agentSettings.TimeoutSeconds, agentSettings.TimeoutSeconds/60)
}

// GetAgentSettings returns current agent settings (thread-safe)
func GetAgentSettings() AgentSettings {
	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return agentSettings
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
