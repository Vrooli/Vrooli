package agentmanager

import (
	"log/slog"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
)

// DefaultAgentMaxTurns is the canonical per-run turn budget for swarm-manager
// agents. Settings defaults, governance fallbacks, and the configuration UI
// should all mirror this value so that deleting and recreating a profile does
// not silently drop the cap back to a smaller number.
const DefaultAgentMaxTurns int32 = 600

// ProfileConfig contains agent profile configuration.
type ProfileConfig struct {
	RunnerType       domainpb.RunnerType
	Model            string
	ModelPreset      domainpb.ModelPreset
	MaxTurns         int32
	TimeoutSeconds   int32
	AllowedTools     []string
	SkipPermissions  bool
	RequiresSandbox  bool
	RequiresApproval bool
}

// SettingsReader provides agent settings from an external source (e.g. settings store)
// without creating a direct import dependency on the settings package.
type SettingsReader interface {
	LoadAgentSettings() (maxTurns, timeoutSeconds int32, requiresApproval bool, err error)
}

// ProfileConfigFromSettings creates a ProfileConfig by overlaying settings values
// on top of the defaults. Zero/negative values for maxTurns or timeoutSeconds
// are ignored, preserving the default.
func ProfileConfigFromSettings(maxTurns, timeoutSeconds int32, requiresApproval bool) *ProfileConfig {
	cfg := DefaultProfileConfig()
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	if timeoutSeconds > 0 {
		cfg.TimeoutSeconds = timeoutSeconds
	}
	cfg.RequiresApproval = requiresApproval
	return cfg
}

// DefaultProfileConfig returns the default configuration for swarm-manager agents.
func DefaultProfileConfig() *ProfileConfig {
	return &ProfileConfig{
		RunnerType:  domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		ModelPreset: domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:    DefaultAgentMaxTurns,
		// 60 minute timeout for research and implementation prep.
		TimeoutSeconds: 3600,
		AllowedTools: []string{
			"Read",
			"Write",
			"Edit",
			"Glob",
			"Grep",
			"Bash",
		},
		SkipPermissions:  false,
		RequiresSandbox:  false,
		RequiresApproval: true,
	}
}

// SetSettingsReader assigns a SettingsReader for runtime profile config resolution.
// This is safe to call after construction, before any spawns occur.
func (s *AgentService) SetSettingsReader(r SettingsReader) {
	s.settingsReader = r
}

// GetProfileID returns the current profile ID.
func (s *AgentService) GetProfileID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.profileID
}

func (s *AgentService) buildProfile(cfg *ProfileConfig) *domainpb.AgentProfile {
	return &domainpb.AgentProfile{
		Name:                 s.profileName,
		ProfileKey:           s.profileKey,
		Description:          "Agent profile for swarm-manager research and execution",
		RunnerType:           cfg.RunnerType,
		Model:                cfg.Model,
		ModelPreset:          cfg.ModelPreset,
		MaxTurns:             cfg.MaxTurns,
		Timeout:              durationpb.New(time.Duration(cfg.TimeoutSeconds) * time.Second),
		AllowedTools:         cfg.AllowedTools,
		SkipPermissionPrompt: cfg.SkipPermissions,
		RequiresSandbox:      cfg.RequiresSandbox,
		RequiresApproval:     cfg.RequiresApproval,
		CreatedBy:            "swarm-manager",
	}
}

// resolveProfileConfig returns a ProfileConfig derived from the settings store
// when available, falling back to hardcoded defaults on error or when no
// SettingsReader is configured.
func (s *AgentService) resolveProfileConfig() *ProfileConfig {
	if s.settingsReader != nil {
		maxTurns, timeout, approval, err := s.settingsReader.LoadAgentSettings()
		if err == nil {
			return ProfileConfigFromSettings(maxTurns, timeout, approval)
		}
		slog.Warn("settings read failed, using defaults", "error", err)
	}
	return DefaultProfileConfig()
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	if s.profileKey == "" {
		return nil
	}
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
		Defaults:   s.buildProfile(s.resolveProfileConfig()),
	}
}
