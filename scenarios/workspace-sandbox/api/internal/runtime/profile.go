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
	"fmt"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/types"
)

// ProfileResolver resolves an isolation-profile ID against an
// immutable, in-memory snapshot of the registry. Snapshot semantics
// are deliberate: once a resolver is built, the underlying ProfileStore
// can mutate (Save/Delete) without affecting future Resolve calls on
// this instance.
//
// Round 4 Phase 9 (2026-04-29): replaced the per-call Store dip with
// snapshot semantics. The handler holds an atomic.Pointer to the
// canonical snapshot and rebuilds it on admin Save/Delete, so the
// public API stays consistent without coupling the resolver to a
// long-lived Store reference.
type ProfileResolver struct {
	// Profiles is the per-resolver snapshot of {ID → profile}. A nil
	// or empty map is treated as "no profile registered" — Resolve
	// returns IsolationProfileNotFoundError for any non-empty
	// requestedID.
	Profiles map[string]config.IsolationProfile

	// DefaultID is the profile ID returned when callers pass an empty
	// requestedID. Empty string falls through to the builtin "full"
	// profile; if that's missing too, Resolve returns the typed error.
	DefaultID string

	// Caps captures the active driver's capabilities. Used by
	// ResolveAndApply to evaluate the home-overlay decision.
	Caps driver.DriverCapabilities
}

// LoadProfiles builds a snapshot map from a ProfileStore. Returns an
// error wrapping the underlying List() failure so callers can fail
// startup cleanly.
//
// Use at startup (or after admin Save/Delete) to capture the current
// registry; pass the resulting map into a ProfileResolver. The map is
// immutable from the resolver's point of view — never mutate it after
// construction.
func LoadProfiles(store config.ProfileStore) (map[string]config.IsolationProfile, error) {
	if store == nil {
		return nil, fmt.Errorf("LoadProfiles: store is nil")
	}
	profiles, err := store.List()
	if err != nil {
		return nil, fmt.Errorf("LoadProfiles: list: %w", err)
	}
	out := make(map[string]config.IsolationProfile, len(profiles))
	for _, p := range profiles {
		out[p.ID] = p
	}
	return out, nil
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
	if p, ok := r.Profiles[id]; ok {
		// Return a copy so callers cannot mutate the snapshot via the
		// returned pointer.
		copy := p
		return &copy, nil
	}
	return nil, types.NewIsolationProfileNotFoundError(id)
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
		MaskPaths:      p.MaskPaths,
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
