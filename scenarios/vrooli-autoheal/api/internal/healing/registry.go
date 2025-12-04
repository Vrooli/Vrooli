// Package healing provides recovery action execution and healing strategies.
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package healing

import (
	"context"
	"sync"

	"vrooli-autoheal/internal/checks"
)

// Registry manages healers for checks.
// It provides a centralized location for healing logic,
// separating detection concerns (checks) from healing concerns (healers).
// [REQ:HEAL-ACTION-001]
type Registry struct {
	mu      sync.RWMutex
	healers map[string]Healer
}

// NewRegistry creates a new healer registry.
func NewRegistry() *Registry {
	return &Registry{
		healers: make(map[string]Healer),
	}
}

// Register adds a healer to the registry.
// The healer will be associated with its CheckID().
func (r *Registry) Register(healer Healer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healers[healer.CheckID()] = healer
}

// Unregister removes a healer from the registry.
func (r *Registry) Unregister(checkID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.healers, checkID)
}

// Get returns the healer for a check, or nil if not found.
func (r *Registry) Get(checkID string) Healer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.healers[checkID]
}

// IsHealable returns true if a healer is registered for the check.
func (r *Registry) IsHealable(checkID string) bool {
	return r.Get(checkID) != nil
}

// GetActions returns the available recovery actions for a check.
// Returns nil if no healer is registered for the check.
func (r *Registry) GetActions(checkID string, lastResult *checks.Result) []checks.RecoveryAction {
	healer := r.Get(checkID)
	if healer == nil {
		return nil
	}
	return healer.Actions(lastResult)
}

// Execute runs a recovery action for a check.
// Returns an error result if no healer is registered for the check.
func (r *Registry) Execute(ctx context.Context, checkID, actionID string, lastResult *checks.Result) checks.ActionResult {
	healer := r.Get(checkID)
	if healer == nil {
		return checks.ActionResult{
			ActionID: actionID,
			CheckID:  checkID,
			Success:  false,
			Error:    "no healer registered for check: " + checkID,
		}
	}
	return healer.Execute(ctx, actionID, lastResult)
}

// List returns all registered check IDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.healers))
	for id := range r.healers {
		ids = append(ids, id)
	}
	return ids
}

// HealerAdapter adapts a HealableCheck to the Healer interface.
// This allows existing HealableCheck implementations to work with
// the new healing registry while migrating to strategies.
type HealerAdapter struct {
	check checks.HealableCheck
}

// NewHealerAdapter creates an adapter for a HealableCheck.
func NewHealerAdapter(check checks.HealableCheck) *HealerAdapter {
	return &HealerAdapter{check: check}
}

// CheckID returns the check ID.
func (a *HealerAdapter) CheckID() string {
	return a.check.ID()
}

// Actions returns the recovery actions from the underlying check.
func (a *HealerAdapter) Actions(lastResult *checks.Result) []checks.RecoveryAction {
	return a.check.RecoveryActions(lastResult)
}

// Execute runs an action on the underlying check.
func (a *HealerAdapter) Execute(ctx context.Context, actionID string, lastResult *checks.Result) checks.ActionResult {
	return a.check.ExecuteAction(ctx, actionID)
}

// Ensure HealerAdapter implements Healer.
var _ Healer = (*HealerAdapter)(nil)
