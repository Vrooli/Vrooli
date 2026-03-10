package main

import (
	"context"
	"sync"
	"time"
)

// DependencyKind classifies what a capability depends on.
type DependencyKind string

const (
	DependencyScenario DependencyKind = "scenario"
	DependencyResource DependencyKind = "resource"
)

// CapabilityStatus represents the current state of a capability.
type CapabilityStatus string

const (
	StatusAvailable   CapabilityStatus = "available"
	StatusUnavailable CapabilityStatus = "unavailable"
	StatusUnknown     CapabilityStatus = "unknown"
)

// CapabilityDef describes a capability that depends on an external service.
type CapabilityDef struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	DependencyKind DependencyKind `json:"dependencyKind"`
	DependencySlug string         `json:"dependencySlug"`
	Features       []string       `json:"features"`
}

// CapabilityState is a CapabilityDef enriched with runtime status.
type CapabilityState struct {
	CapabilityDef
	Status    CapabilityStatus `json:"status"`
	Message   string           `json:"message,omitempty"`
	CheckedAt string           `json:"checkedAt,omitempty"`
}

// StatusChecker probes a dependency and returns its status.
type StatusChecker interface {
	Check(ctx context.Context) (CapabilityStatus, string)
}

// knownCapabilities is the single source of truth for all declared dependencies.
var knownCapabilities = []CapabilityDef{
	{
		ID:             "workspace-sandbox",
		Name:           "Workspace Sandbox",
		Description:    "Approved changes tracking and commit preview for sandbox-managed files",
		DependencyKind: DependencyScenario,
		DependencySlug: "workspace-sandbox",
		Features:       []string{"Approved changes panel", "Commit preview filtering"},
	},
}

// CapabilityRegistry tracks dependency availability with caching.
type CapabilityRegistry struct {
	defs     []CapabilityDef
	checkers map[string]StatusChecker

	mu       sync.RWMutex
	cached   []CapabilityState
	cachedAt time.Time
	cacheTTL time.Duration
}

// NewCapabilityRegistry creates a new registry.
func NewCapabilityRegistry(defs []CapabilityDef, checkers map[string]StatusChecker, cacheTTL time.Duration) *CapabilityRegistry {
	return &CapabilityRegistry{
		defs:     defs,
		checkers: checkers,
		cacheTTL: cacheTTL,
	}
}

// Resolve returns the current state of all capabilities, using cached results
// when available and not expired.
func (r *CapabilityRegistry) Resolve(ctx context.Context) []CapabilityState {
	r.mu.RLock()
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]CapabilityState, len(r.cached))
		copy(result, r.cached)
		r.mu.RUnlock()
		return result
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if r.cached != nil && time.Since(r.cachedAt) < r.cacheTTL {
		result := make([]CapabilityState, len(r.cached))
		copy(result, r.cached)
		return result
	}

	now := time.Now().UTC()
	states := make([]CapabilityState, len(r.defs))
	for i, def := range r.defs {
		state := CapabilityState{
			CapabilityDef: def,
			Status:        StatusUnknown,
			CheckedAt:     now.Format(time.RFC3339),
		}
		if checker, ok := r.checkers[def.ID]; ok {
			state.Status, state.Message = checker.Check(ctx)
		}
		states[i] = state
	}

	r.cached = states
	r.cachedAt = now

	result := make([]CapabilityState, len(states))
	copy(result, states)
	return result
}

// IsAvailable returns true if the given capability is currently available.
func (r *CapabilityRegistry) IsAvailable(ctx context.Context, capabilityID string) bool {
	for _, s := range r.Resolve(ctx) {
		if s.ID == capabilityID {
			return s.Status == StatusAvailable
		}
	}
	return false
}
