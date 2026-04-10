package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"app-issue-tracker-api/internal/agentmanager"
)

const agentManagerProfileKey = "app-issue-tracker-investigations"

func (s *Server) initializeAgentManager() error {
	settings := GetAgentSettings()
	profileConfig, err := buildProfileConfig(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if !s.agentManager.IsAvailable(ctx) {
		return fmt.Errorf("agent-manager is not available")
	}
	if err := s.agentManager.Initialize(ctx, profileConfig); err != nil {
		return fmt.Errorf("initialize agent-manager profiles: %w", err)
	}
	return nil
}

func (s *Server) updateAgentManagerProfile() error {
	settings := GetAgentSettings()
	profileConfig, err := buildProfileConfig(settings)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.agentManager.UpdateProfile(ctx, profileConfig)
}

func buildProfileConfig(settings AgentSettings) (agentmanager.ProfileConfig, error) {
	runnerType, err := parseRunnerType(settings.RunnerType)
	if err != nil {
		return agentmanager.ProfileConfig{}, err
	}

	allowedTools := parseToolsList(settings.AllowedTools)

	return agentmanager.ProfileConfig{
		RunnerType:       runnerType,
		MaxTurns:         int32(settings.MaxTurns),
		TimeoutSeconds:   int32(settings.TimeoutSeconds),
		AllowedTools:     allowedTools,
		SkipPermissions:  settings.SkipPermissions,
		RequiresSandbox:  false,
		RequiresApproval: false,
	}, nil
}

func parseRunnerType(raw string) (agentmanager.RunnerType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "codex":
		return agentmanager.RunnerTypeCodex, nil
	case "opencode":
		return agentmanager.RunnerTypeOpenCode, nil
	case "claude-code", "":
		return agentmanager.RunnerTypeClaudeCode, nil
	default:
		return agentmanager.RunnerTypeClaudeCode, fmt.Errorf("unsupported runner type %q", raw)
	}
}

func parseToolsList(tools string) []string {
	if tools == "" {
		return nil
	}
	parts := strings.Split(tools, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
