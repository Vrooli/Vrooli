// Responsibility: resolve and validate sandbox configuration and scope paths.
package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"path/filepath"
	"strings"
)

// resolveSandboxConfig produces the effective SandboxConfig for a run.
//
// Contract: the returned config is always non-nil. Callers (including
// tryAutoApproval) rely on this invariant; a nil return historically caused
// silent fall-through to NEEDS_REVIEW for empty sandboxes because there was
// no acceptance config to consult.
//
// Precedence (later overrides earlier):
//  1. Zero-valued default
//  2. profile.SandboxConfig (if present)
//  3. req.SandboxConfig (inline override, if present)
//
// Phase G: Profile/req AllowedPaths/DeniedPaths are merged into the resolved
// SandboxConfig.Acceptance so they become *enforced* at apply-at-run-end
// rather than passed as advisory env vars to runners. This is the
// agent-sandbox-audit-foundation policy-to-sandbox handoff.
func (o *Orchestrator) resolveSandboxConfig(req CreateRunRequest, profile *domain.AgentProfile) (*domain.SandboxConfig, error) {
	// Start from the auditability-contract defaults (Mode=Protected,
	// AutoApply=true, ApplyOnFailure=true, NetworkMode=localhost,
	// NoLock=true). Profile and request overrides clone over the top.
	// Without this baseline, a request with no profile and no inline
	// config would zero-value the struct, dropping Mode to unspecified
	// and silently downgrading to host-tracked execution.
	defaults := domain.DefaultSandboxConfig()
	cfg := defaults
	if profile != nil && profile.SandboxConfig != nil {
		cfg = cloneSandboxConfig(profile.SandboxConfig)
	}
	if req.SandboxConfig != nil {
		cfg = mergeSandboxConfig(cfg, req.SandboxConfig)
	}

	// Backfill enum/string fields that the override left at the proto
	// zero-value. Callers (notably swarm-manager) often send a partial
	// SandboxConfig containing only Acceptance overrides; without this
	// backfill the wholesale-replace clone above would silently strip
	// Mode and NetworkMode to "unspecified", silently downgrading
	// protected runs to tracking. Pointer-typed fields (AutoApply,
	// ApplyOnFailure) and structural fields (Lifecycle, Acceptance) are
	// left intentional-explicit; bool fields (ManualReview, NoLock) are
	// left at the override's value because zero is operator-visible
	// "off" rather than "not provided".
	if cfg.Mode == domain.SandboxModeUnspecified {
		cfg.Mode = defaults.Mode
	}
	if cfg.NetworkMode == "" {
		cfg.NetworkMode = defaults.NetworkMode
	}

	// Push path policy from profile/request into the acceptance layer so
	// workspace-sandbox actually enforces it at apply time. The runner-side
	// advisory env vars are kept for the tracking-mode capability matrix
	// but the load-bearing enforcement now lives at the sandbox boundary.
	allowedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.AllowedPaths })
	if req.AllowedPaths != nil {
		allowedPaths = req.AllowedPaths
	}
	deniedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.DeniedPaths })
	if req.DeniedPaths != nil {
		deniedPaths = req.DeniedPaths
	}
	cfg.Acceptance.Allow.PathGlobs = mergeUnique(cfg.Acceptance.Allow.PathGlobs, allowedPaths)
	cfg.Acceptance.Deny.PathGlobs = mergeUnique(cfg.Acceptance.Deny.PathGlobs, deniedPaths)

	cfg = normalizeSandboxConfig(cfg)
	if err := validateSandboxConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// mergeSandboxConfig applies an inline config as a sparse override instead of
// replacing the profile config wholesale. Proto scalar zero values cannot
// distinguish absence from an explicit false, so only non-zero scalar values
// and explicitly-present pointer/structural values override here. This keeps a
// ManualReview-only request from accidentally deleting its profile's lifecycle
// and acceptance contract.
func mergeSandboxConfig(base, override *domain.SandboxConfig) *domain.SandboxConfig {
	if base == nil {
		base = domain.DefaultSandboxConfig()
	}
	merged := cloneSandboxConfig(base)
	if override == nil {
		return merged
	}
	if override.Mode != domain.SandboxModeUnspecified {
		merged.Mode = override.Mode
	}
	if override.NetworkMode != "" {
		merged.NetworkMode = override.NetworkMode
	}
	// A populated config carries an explicit ManualReview=false; a sparse
	// ManualReview-only inline message cannot represent false in proto3 and
	// therefore leaves the profile value intact.
	if override.ManualReview || sandboxConfigHasExplicitScalars(override) {
		merged.ManualReview = override.ManualReview
	}
	if override.AutoApply != nil {
		v := *override.AutoApply
		merged.AutoApply = &v
	}
	if override.ApplyOnFailure != nil {
		v := *override.ApplyOnFailure
		merged.ApplyOnFailure = &v
	}
	if override.NoLock {
		merged.NoLock = true
	}
	if !sandboxLifecycleIsZero(override.Lifecycle) {
		merged.Lifecycle = cloneSandboxConfig(override).Lifecycle
	}
	if !sandboxAcceptanceIsZero(override.Acceptance) {
		merged.Acceptance = cloneSandboxConfig(override).Acceptance
	}
	if override.WritePolicy != nil {
		merged.WritePolicy = cloneSandboxConfig(override).WritePolicy
	}
	return merged
}

func sandboxConfigHasExplicitScalars(cfg *domain.SandboxConfig) bool {
	return cfg.Mode != domain.SandboxModeUnspecified || cfg.NetworkMode != "" || cfg.AutoApply != nil || cfg.ApplyOnFailure != nil || cfg.NoLock
}

func sandboxLifecycleIsZero(lifecycle domain.SandboxLifecycleConfig) bool {
	return len(lifecycle.CheckpointOn) == 0 && len(lifecycle.StopOn) == 0 && len(lifecycle.DeleteOn) == 0 && lifecycle.TTL == 0 && lifecycle.IdleTimeout == 0
}

func sandboxAcceptanceIsZero(acceptance domain.SandboxAcceptanceConfig) bool {
	return acceptance.Mode == "" && !acceptance.IgnoreBinary && len(acceptance.Allow.PathGlobs) == 0 && len(acceptance.Allow.Extensions) == 0 && len(acceptance.Deny.PathGlobs) == 0 && len(acceptance.Deny.Extensions) == 0
}

func profilePaths(p *domain.AgentProfile, get func(*domain.AgentProfile) []string) []string {
	if p == nil {
		return nil
	}
	return get(p)
}

func mergeUnique(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range b {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func cloneSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Lifecycle.CheckpointOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.CheckpointOn...)
	clone.Lifecycle.StopOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.StopOn...)
	clone.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.DeleteOn...)
	clone.Acceptance.Allow = cloneSandboxCriteria(cfg.Acceptance.Allow)
	clone.Acceptance.Deny = cloneSandboxCriteria(cfg.Acceptance.Deny)
	if cfg.WritePolicy != nil {
		clone.WritePolicy = &domain.WorkspaceWritePolicy{Paths: append([]string(nil), cfg.WritePolicy.Paths...)}
	}
	if cfg.AutoApply != nil {
		v := *cfg.AutoApply
		clone.AutoApply = &v
	}
	if cfg.ApplyOnFailure != nil {
		v := *cfg.ApplyOnFailure
		clone.ApplyOnFailure = &v
	}
	return &clone
}

func cloneSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	return domain.SandboxFileCriteria{
		PathGlobs:  append([]string(nil), criteria.PathGlobs...),
		Extensions: append([]string(nil), criteria.Extensions...),
	}
}

func normalizeSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode == "" {
		cfg.Acceptance.Mode = "allowlist"
	}
	cfg.Acceptance.Allow = normalizeSandboxCriteria(cfg.Acceptance.Allow)
	cfg.Acceptance.Deny = normalizeSandboxCriteria(cfg.Acceptance.Deny)

	// Default lifecycle cleanup for auto-apply sandboxes.
	//
	// Under the auditability contract (Phase 3b), AutoApply=true (the
	// contract default unless ManualReview=true) means the sandbox is
	// applied at run end. Once applied, leaving the sandbox active
	// indefinitely blocks future runs on the same scope path and leaks
	// overlay mounts. We default deleteOn to ["terminal"] so the sandbox
	// is cleaned up after any terminal event when ManualReview is off.
	//
	// ManualReview=true sandboxes intentionally persist past run end so
	// operators can review; their TTL GC is owned by workspace-sandbox
	// LifecycleReconciler (Phase 4).
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Lifecycle.CheckpointOn) == 0 && len(cfg.Lifecycle.DeleteOn) == 0 && len(cfg.Lifecycle.StopOn) == 0 {
		cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{
			domain.SandboxLifecycleTurnCompleted,
			domain.SandboxLifecycleTurnFailed,
			domain.SandboxLifecycleTurnCancelled,
		}
		// Use the terminal event emitted by finalize so default sandboxes are
		// deleted instead of remaining checkpointed.
		cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	}

	return cfg
}

func normalizeSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	paths := make([]string, 0, len(criteria.PathGlobs))
	seenPaths := make(map[string]bool)
	for _, p := range criteria.PathGlobs {
		p = strings.TrimSpace(p)
		if p == "" || seenPaths[p] {
			continue
		}
		seenPaths[p] = true
		paths = append(paths, p)
	}

	exts := make([]string, 0, len(criteria.Extensions))
	seenExts := make(map[string]bool)
	for _, ext := range criteria.Extensions {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ext = strings.ToLower(ext)
		if seenExts[ext] {
			continue
		}
		seenExts[ext] = true
		exts = append(exts, ext)
	}

	criteria.PathGlobs = paths
	criteria.Extensions = exts
	return criteria
}

func validateSandboxConfig(cfg *domain.SandboxConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.WritePolicy != nil && cfg.Mode.Effective() != domain.SandboxModeProtected {
		return domain.NewValidationError("sandboxConfig.writePolicy", "runtime write policy requires protected mode")
	}
	if cfg.Acceptance.Mode != "" && cfg.Acceptance.Mode != "allowlist" {
		return domain.NewValidationError("sandboxConfig.acceptance.mode", "unsupported acceptance mode")
	}
	if cfg.Lifecycle.TTL < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.ttl", "ttl cannot be negative")
	}
	if cfg.Lifecycle.IdleTimeout < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.idleTimeout", "idleTimeout cannot be negative")
	}
	for _, p := range append(cfg.Acceptance.Allow.PathGlobs, cfg.Acceptance.Deny.PathGlobs...) {
		if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
			return domain.NewValidationErrorWithHint(
				"sandboxConfig.acceptance.pathGlobs",
				"path globs must be project-root relative",
				"Remove the leading '/' and use project-root relative patterns",
			)
		}
	}
	// Warn when AutoApply is on (the contract default) but no allow
	// criteria are configured. This is valid (empty allow = accept all
	// non-denied files), but surprising enough to warrant a log line —
	// especially since an empty deny (from proto serialization)
	// previously caused silent universal denial.
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Acceptance.Allow.PathGlobs) == 0 &&
		len(cfg.Acceptance.Allow.Extensions) == 0 {
		obs.Component("sandbox-config").Info("autoApply enabled with no allow criteria; all non-denied files will be applied at run end")
	}
	return nil
}
