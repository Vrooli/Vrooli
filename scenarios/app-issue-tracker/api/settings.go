package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// AgentSettings holds configuration for AI agent execution
type AgentSettings struct {
	MaxTurns         int    `json:"max_turns"`
	AllowedTools     string `json:"allowed_tools"`
	TimeoutSeconds   int    `json:"timeout_seconds"`
	SkipPermissions  bool   `json:"skip_permissions"`
	AgentTag         string `json:"agent_tag"`
	Provider         string `json:"provider"`
	CLICommand       string `json:"cli_command"`
}

var (
	agentSettings     AgentSettings
	agentSettingsMu   sync.RWMutex
	agentSettingsOnce sync.Once
)

// LoadAgentSettings loads agent settings from configuration file
func LoadAgentSettings(scenarioRoot string) AgentSettings {
	agentSettingsOnce.Do(func() {
		// Default settings (aligned with ecosystem-manager)
		agentSettings = AgentSettings{
			MaxTurns:        80,
			AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
			TimeoutSeconds:  600,
			SkipPermissions: true,
			Provider:        "claude-code",
			CLICommand:      "resource-claude-code",
		}

		// Try to load from agent-settings.json
		configPath := filepath.Join(scenarioRoot, "initialization", "configuration", "agent-settings.json")
		data, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("Warning: Could not load agent-settings.json: %v (using defaults)", err)
			return
		}

		var config struct {
			AgentBackend struct {
				Provider string `json:"provider"`
			} `json:"agent_backend"`
			Providers map[string]struct {
				CLICommand string `json:"cli_command"`
				Operations map[string]struct {
					MaxTurns       int    `json:"max_turns"`
					AllowedTools   string `json:"allowed_tools"`
					TimeoutSeconds int    `json:"timeout_seconds"`
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
		}

		agentSettings.Provider = provider
		if providerConfig.CLICommand != "" {
			agentSettings.CLICommand = providerConfig.CLICommand
		}

		log.Printf("Loaded agent settings: provider=%s, max_turns=%d, timeout=%ds",
			agentSettings.Provider, agentSettings.MaxTurns, agentSettings.TimeoutSeconds)
	})

	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return agentSettings
}

// GetAgentSettings returns current agent settings (thread-safe)
func GetAgentSettings() AgentSettings {
	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return agentSettings
}
