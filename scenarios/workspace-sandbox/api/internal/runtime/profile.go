// Package runtime hosts the cross-handler runtime helpers — profile
// resolution, resource-limit defaulting, and SSE wiring — so HTTP
// handlers can stay thin and focused on parsing/formatting.
//
// Profile resolution and home-overlay enforcement live here (not in
// handlers) because they're domain decisions that the agent-manager
// preflight + the workspace-sandbox exec gate consult independently.
// Centralizing them keeps the contract a single read.
//
// DOC: home-overlay seam — handler-side enforcement.
package runtime

import (
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/types"
)

// ProfileResolver pairs a ProfileStore with a default profile ID so
// handlers can resolve a request's `isolationLevel` without reaching
// into Handlers state for every call.
type ProfileResolver struct {
	Store     config.ProfileStore
	DefaultID string
	Caps      driver.DriverCapabilities
}

// Resolve returns the IsolationProfile for the requested ID, falling
// back to the configured default and finally to the builtin "full"
// profile. Errors map to typed types.IsolationProfileNotFoundError.
func (r *ProfileResolver) Resolve(requestedID string) (*config.IsolationProfile, error) {
	id := requestedID
	if id == "" {
		id = r.DefaultID
	}
	if id == "" {
		id = "full"
	}
	if r.Store == nil {
		return nil, types.NewIsolationProfileNotFoundError(id)
	}
	profile, err := r.Store.Get(id)
	if err != nil {
		return nil, types.NewIsolationProfileNotFoundError(id)
	}
	return profile, nil
}

// ResolveAndApply resolves the requested profile, applies the
// home-overlay decision, and composes the profile onto cfg. Returns
// HomeOverlayRequiredError (HTTP 409) when the profile requires a
// home overlay but the sandbox doesn't have one.
func (r *ProfileResolver) ResolveAndApply(sb *types.Sandbox, cfg *driverexec.BwrapConfig, requestedID string) error {
	profile, err := r.Resolve(requestedID)
	if err != nil {
		return err
	}
	if sb != nil {
		if decision := policy.DecideHomeOverlay(r.Caps, *profile, *sb); !decision.Allowed {
			return types.NewHomeOverlayRequiredError(sb.ID.String(), profile.ID, string(sb.HomeOverlayState))
		}
	}
	return driverexec.ApplyIsolationProfile(cfg, ConvertProfileToDriver(profile))
}

// ConvertProfileToDriver converts a config.IsolationProfile to the
// exec-package representation used by ApplyIsolationProfile.
func ConvertProfileToDriver(p *config.IsolationProfile) *driverexec.IsolationProfile {
	if p == nil {
		return nil
	}
	return &driverexec.IsolationProfile{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Builtin:        p.Builtin,
		NetworkAccess:  p.NetworkAccess,
		ReadOnlyBinds:  p.ReadOnlyBinds,
		ReadWriteBinds: p.ReadWriteBinds,
		Environment:    p.Environment,
		Hostname:       p.Hostname,
	}
}

// ApplyResourceLimitDefaults applies defaults from execCfg when req
// values are 0, and clamps to maximums (0 = no maximum).
func ApplyResourceLimitDefaults(req driverexec.ResourceLimits, execCfg config.ExecutionConfig) driverexec.ResourceLimits {
	defaults := execCfg.DefaultResourceLimits
	maxes := execCfg.MaxResourceLimits

	result := req

	if result.MemoryLimitMB == 0 && defaults.MemoryLimitMB > 0 {
		result.MemoryLimitMB = defaults.MemoryLimitMB
	}
	if result.CPUTimeSec == 0 && defaults.CPUTimeSec > 0 {
		result.CPUTimeSec = defaults.CPUTimeSec
	}
	if result.MaxProcesses == 0 && defaults.MaxProcesses > 0 {
		result.MaxProcesses = defaults.MaxProcesses
	}
	if result.MaxOpenFiles == 0 && defaults.MaxOpenFiles > 0 {
		result.MaxOpenFiles = defaults.MaxOpenFiles
	}
	if result.TimeoutSec == 0 && defaults.TimeoutSec > 0 {
		result.TimeoutSec = defaults.TimeoutSec
	}

	if maxes.MemoryLimitMB > 0 && result.MemoryLimitMB > maxes.MemoryLimitMB {
		result.MemoryLimitMB = maxes.MemoryLimitMB
	}
	if maxes.CPUTimeSec > 0 && result.CPUTimeSec > maxes.CPUTimeSec {
		result.CPUTimeSec = maxes.CPUTimeSec
	}
	if maxes.MaxProcesses > 0 && result.MaxProcesses > maxes.MaxProcesses {
		result.MaxProcesses = maxes.MaxProcesses
	}
	if maxes.MaxOpenFiles > 0 && result.MaxOpenFiles > maxes.MaxOpenFiles {
		result.MaxOpenFiles = maxes.MaxOpenFiles
	}
	if maxes.TimeoutSec > 0 && result.TimeoutSec > maxes.TimeoutSec {
		result.TimeoutSec = maxes.TimeoutSec
	}

	return result
}
