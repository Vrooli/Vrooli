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
// swarm-manager runs always auto-accept on success. Sandbox is the
// auditability layer — per-run file diffs are preserved in
// workspace-sandbox regardless. There is no manual-review knob; runs
// are either successful (auto-applied) or failed (sandbox preserved
// for inspection). The ManualReview field was removed 2026-04-28 to
// keep the contract uniform across every swarm-manager skill.
//
// If a future workflow needs operator-gated apply, that is a separate
// plan and should not re-introduce the field on this struct — it
// should live in its own scenario or per-task knob on the agent-manager
// side.
type ProfileConfig struct {
	RunnerType      domainpb.RunnerType
	Model           string
	ModelPreset     domainpb.ModelPreset
	MaxTurns        int32
	TimeoutSeconds  int32
	AllowedTools    []string
	SkipPermissions bool
	// SandboxMode selects the per-run sandbox execution mode. The
	// zero-value (Unspecified) lets agent-manager apply its
	// DefaultSandboxConfig (Mode=Protected). Set to SandboxModeOff
	// only for profiles that legitimately need to run on the host
	// without a sandbox; every other value yields a sandboxed run.
	// See agent-manager's domain.DeriveRunMode for the contract.
	SandboxMode domainpb.SandboxMode
}

// SettingsReader provides agent settings from an external source (e.g. settings store)
// without creating a direct import dependency on the settings package.
type SettingsReader interface {
	LoadAgentSettings() (maxTurns, timeoutSeconds int32, err error)
}

// ProfileConfigFromSettings creates a ProfileConfig by overlaying settings values
// on top of the defaults. Zero/negative values for maxTurns or timeoutSeconds
// are ignored, preserving the default.
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
// Swarm-manager agents run Sandboxed and auto-accept on success.
// Sandbox is the auditability layer — workspace-sandbox preserves the
// per-run diff regardless of run outcome — so there is no need to gate
// apply on operator approval. Runs that fail leave the sandbox in
// place for inspection; runs that succeed apply atomically.
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
		// Explicitly request the protected-mode sandbox so the dispatch
		// path is unambiguous and an inadvertent SandboxConfig change
		// elsewhere cannot silently drop swarm-manager runs into Tracking.
		SandboxMode: domainpb.SandboxMode_SANDBOX_MODE_PROTECTED,
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
	profile := &domainpb.AgentProfile{
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
		CreatedBy:            "swarm-manager",
	}
	// Always carry an explicit SandboxConfig.Mode through to
	// agent-manager; agent-manager.resolveSandboxConfig fills in the
	// rest of the contract defaults (auto-apply, no manual review).
	// Sending the SandboxConfig with only the Mode field set is the
	// correct partial-override pattern. See agent-manager
	// resolveSandboxConfig for the backfill semantics.
	if cfg.SandboxMode != domainpb.SandboxMode_SANDBOX_MODE_UNSPECIFIED {
		profile.SandboxConfig = &domainpb.SandboxConfig{Mode: cfg.SandboxMode}
	}
	return profile
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
	// Profile defaults are reconciled from .vrooli/agent-profiles at startup.
	// Run creation only references the stable profile key so dispatch cannot
	// overwrite DB edits with code-declared defaults.
	return &apipb.ProfileRef{
		ProfileKey: s.profileKey,
	}
}
