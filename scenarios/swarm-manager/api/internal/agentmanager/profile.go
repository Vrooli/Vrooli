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
//
// RequiresApproval (legacy proto field 12) was removed in
// agent-sandbox-audit-foundation Phase 3b. Operator-gated apply is now
// expressed via SandboxConfig.ManualReview on the agent-manager side; we
// represent that here via the ManualReview bool, which is forwarded onto
// the AgentProfile.SandboxConfig at build time.
type ProfileConfig struct {
	RunnerType      domainpb.RunnerType
	Model           string
	ModelPreset     domainpb.ModelPreset
	MaxTurns        int32
	TimeoutSeconds  int32
	AllowedTools    []string
	SkipPermissions bool
	RequiresSandbox bool
	ManualReview    bool
}

// SettingsReader provides agent settings from an external source (e.g. settings store)
// without creating a direct import dependency on the settings package.
type SettingsReader interface {
	LoadAgentSettings() (maxTurns, timeoutSeconds int32, err error)
}

// ProfileConfigFromSettings creates a ProfileConfig by overlaying settings values
// on top of the defaults. Zero/negative values for maxTurns or timeoutSeconds
// are ignored, preserving the default. ManualReview is taken from the default
// (per-profile concern; not a global setting).
func ProfileConfigFromSettings(maxTurns, timeoutSeconds int32) *ProfileConfig {
	cfg := DefaultProfileConfig()
	if maxTurns > 0 {
		cfg.MaxTurns = maxTurns
	}
	if timeoutSeconds > 0 {
		cfg.TimeoutSeconds = timeoutSeconds
	}
	return cfg
}

// DefaultProfileConfig returns the default configuration for swarm-manager agents.
//
// Swarm-manager agents run Sandboxed with manual review: research and
// workshop agents frequently write files (plans, docs, specs) and those
// diffs should be human-reviewable. Under the auditability contract,
// ManualReview=true defers apply at run end until an operator approves
// via one of the three viewing surfaces (GCT, agent-manager, workspace-sandbox).
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
		SkipPermissions: false,
		RequiresSandbox: true,
		ManualReview:    true,
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
		// Express operator-gated apply via SandboxConfig.ManualReview
		// per the auditability contract (Phase 3b cutover).
		SandboxConfig: &domainpb.SandboxConfig{ManualReview: cfg.ManualReview},
		CreatedBy:     "swarm-manager",
	}
}

// resolveProfileConfig returns a ProfileConfig derived from the settings store
// when available, falling back to hardcoded defaults on error or when no
// SettingsReader is configured.
func (s *AgentService) resolveProfileConfig() *ProfileConfig {
	if s.settingsReader != nil {
		maxTurns, timeout, err := s.settingsReader.LoadAgentSettings()
		if err == nil {
			return ProfileConfigFromSettings(maxTurns, timeout)
		}
		slog.Warn("settings read failed, using defaults", "error", err)
	}
	return DefaultProfileConfig()
}

func (s *AgentService) defaultProfileRef() *apipb.ProfileRef {
	if s.profileKey == "" {
		return nil
	}
	// UpdateExisting=true makes swarm-manager's code-declared profile the
	// source of truth: every dispatch overwrites the DB row with the current
	// defaults, so a code change to DefaultProfileConfig() takes effect on
	// the next run without requiring manual profile edits. This prevents
	// the code/DB desync that caused sandboxed-but-requires-approval runs
	// to silently accumulate in NEEDS_REVIEW in 2026-04.
	return &apipb.ProfileRef{
		ProfileKey:     s.profileKey,
		Defaults:       s.buildProfile(s.resolveProfileConfig()),
		UpdateExisting: true,
	}
}
