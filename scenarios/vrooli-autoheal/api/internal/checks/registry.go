// Package checks provides the health check registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	"sync"
	"time"

	"vrooli-autoheal/internal/platform"
)

// Registry manages health checks
type Registry struct {
	mu       sync.RWMutex
	checks   map[string]Check
	results  map[string]Result
	lastRun  map[string]time.Time
	platform *platform.Capabilities
	config   ConfigProvider
}

// NewRegistry creates a new health check registry with the given platform capabilities.
// Platform is injected to allow testing and avoid hidden dependency creation.
func NewRegistry(plat *platform.Capabilities) *Registry {
	return &Registry{
		checks:   make(map[string]Check),
		results:  make(map[string]Result),
		lastRun:  make(map[string]time.Time),
		platform: plat,
	}
}

// Register adds a health check to the registry
func (r *Registry) Register(check Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[check.ID()] = check
}

// Unregister removes a health check from the registry
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checks, id)
	delete(r.results, id)
	delete(r.lastRun, id)
}

// SetConfigProvider sets the configuration provider for the registry.
// This controls which checks run and which have auto-heal enabled.
// [REQ:CONFIG-CHECK-001]
func (r *Registry) SetConfigProvider(cp ConfigProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cp
}

// shouldRunCheck determines if a check should run based on config, platform, and interval
// [REQ:CONFIG-CHECK-001]
func (r *Registry) shouldRunCheck(check Check, forceAll bool) bool {
	// Check if check is enabled in config (if config provider is set)
	if r.config != nil && !r.config.IsCheckEnabled(check.ID()) {
		return false
	}

	// Check platform compatibility
	platforms := check.Platforms()
	if len(platforms) > 0 {
		compatible := false
		for _, p := range platforms {
			if p == r.platform.Platform {
				compatible = true
				break
			}
		}
		if !compatible {
			return false
		}
	}

	// If forceAll, skip interval check
	if forceAll {
		return true
	}

	// Check interval
	r.mu.RLock()
	lastRun, exists := r.lastRun[check.ID()]
	r.mu.RUnlock()

	if !exists {
		return true
	}

	interval := time.Duration(check.IntervalSeconds()) * time.Second
	return time.Since(lastRun) >= interval
}

// RunAll executes all registered checks that should run
func (r *Registry) RunAll(ctx context.Context, forceAll bool) []Result {
	r.mu.RLock()
	checks := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		if r.shouldRunCheck(check, forceAll) {
			checks = append(checks, check)
		}
	}
	r.mu.RUnlock()

	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		select {
		case <-ctx.Done():
			return results
		default:
			result := r.runCheck(ctx, check)
			results = append(results, result)
		}
	}

	return results
}

// runCheck executes a single check and stores the result
func (r *Registry) runCheck(ctx context.Context, check Check) Result {
	start := time.Now()
	result := check.Run(ctx)
	result.Duration = time.Since(start)
	result.Timestamp = start

	r.mu.Lock()
	r.results[check.ID()] = result
	r.lastRun[check.ID()] = start
	r.mu.Unlock()

	return result
}

// GetResult returns the last result for a specific check
func (r *Registry) GetResult(checkID string) (Result, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, exists := r.results[checkID]
	return result, exists
}

// SetResult stores a result without running the check.
// Used to pre-populate the registry from persisted data on startup.
func (r *Registry) SetResult(result Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[result.CheckID] = result
	// Also update lastRun so interval checks work correctly
	r.lastRun[result.CheckID] = result.Timestamp
}

// GetAllResults returns all stored check results
func (r *Registry) GetAllResults() []Result {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]Result, 0, len(r.results))
	for _, result := range r.results {
		results = append(results, result)
	}
	return results
}

// GetSummary returns an aggregate health summary.
// Delegates to ComputeSummary for the domain logic.
func (r *Registry) GetSummary() Summary {
	return ComputeSummary(r.GetAllResults())
}

// ListChecks returns info about all registered checks
func (r *Registry) ListChecks() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]Info, 0, len(r.checks))
	for _, check := range r.checks {
		infos = append(infos, Info{
			ID:              check.ID(),
			Title:           check.Title(),
			Description:     check.Description(),
			Importance:      check.Importance(),
			Category:        check.Category(),
			IntervalSeconds: check.IntervalSeconds(),
			Platforms:       check.Platforms(),
		})
	}
	return infos
}

// GetCheck returns a check by ID
func (r *Registry) GetCheck(id string) (Check, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	check, exists := r.checks[id]
	return check, exists
}

// GetHealableCheck returns a HealableCheck by ID if the check supports recovery actions
// [REQ:HEAL-ACTION-001]
func (r *Registry) GetHealableCheck(id string) (HealableCheck, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	check, exists := r.checks[id]
	if !exists {
		return nil, false
	}
	healable, ok := check.(HealableCheck)
	return healable, ok
}

// IsHealable returns true if the check with the given ID supports recovery actions
// [REQ:HEAL-ACTION-001]
func (r *Registry) IsHealable(id string) bool {
	_, ok := r.GetHealableCheck(id)
	return ok
}

// IsAutoHealEnabled returns whether auto-healing is enabled for a check
// Returns false if no config provider is set or if the check doesn't support healing
// [REQ:CONFIG-CHECK-001]
func (r *Registry) IsAutoHealEnabled(id string) bool {
	if r.config == nil {
		return false
	}
	// Check if auto-heal is enabled AND the check is healable
	if !r.config.IsAutoHealEnabled(id) {
		return false
	}
	return r.IsHealable(id)
}

// AutoHealResult represents the outcome of an auto-heal attempt
type AutoHealResult struct {
	CheckID      string       `json:"checkId"`
	Attempted    bool         `json:"attempted"`
	ActionResult ActionResult `json:"actionResult,omitempty"`
	Reason       string       `json:"reason,omitempty"` // Why it wasn't attempted
}

// RunAutoHeal attempts to auto-heal any critical checks that have auto-heal enabled.
// It runs the first available recovery action for each failing check.
// Returns a list of auto-heal results.
// [REQ:CONFIG-CHECK-001] [REQ:HEAL-ACTION-001]
func (r *Registry) RunAutoHeal(ctx context.Context, results []Result) []AutoHealResult {
	autoHealResults := make([]AutoHealResult, 0)

	for _, result := range results {
		// Only auto-heal critical checks
		if result.Status != StatusCritical {
			continue
		}

		// Check if auto-heal is enabled for this check
		if !r.IsAutoHealEnabled(result.CheckID) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "auto-heal not enabled for this check",
			})
			continue
		}

		// Get the healable check
		healable, ok := r.GetHealableCheck(result.CheckID)
		if !ok {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "check does not support healing",
			})
			continue
		}

		// Get available recovery actions
		actions := healable.RecoveryActions(&result)

		// Find the first available, non-dangerous action
		var selectedAction *RecoveryAction
		for _, action := range actions {
			if action.Available && !action.Dangerous {
				selectedAction = &action
				break
			}
		}

		if selectedAction == nil {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "no safe recovery action available",
			})
			continue
		}

		// Execute the recovery action
		actionResult := healable.ExecuteAction(ctx, selectedAction.ID)

		autoHealResults = append(autoHealResults, AutoHealResult{
			CheckID:      result.CheckID,
			Attempted:    true,
			ActionResult: actionResult,
		})
	}

	return autoHealResults
}
