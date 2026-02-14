// Package checks provides the health check registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"vrooli-autoheal/internal/platform"
)

// Auto-heal cooldown configuration
const (
	// DefaultHealCooldown is the minimum time between heal attempts for the same check
	DefaultHealCooldown = 2 * time.Minute

	// MaxConsecutiveFailures is the number of consecutive heal failures before backing off
	MaxConsecutiveFailures = 3

	// BackoffCooldown is the extended cooldown after MaxConsecutiveFailures
	BackoffCooldown = 10 * time.Minute

	// MaxBackoffCooldown is the maximum cooldown time after repeated failures
	MaxBackoffCooldown = 30 * time.Minute

	// DefaultCheckTimeout is the maximum time a single health check can run before timing out
	DefaultCheckTimeout = 30 * time.Second

	// MaxAutoHealActionsPerTick limits how many heal actions can run per tick to prevent timeout
	MaxAutoHealActionsPerTick = 5

	// MaxParallelHealActions limits concurrent heal action execution
	MaxParallelHealActions = 3

	// DefaultAutoHealActionTimeout bounds a single auto-heal action execution.
	// This prevents one stuck action from blocking the entire tick for too long.
	DefaultAutoHealActionTimeout = 90 * time.Second
)

// HealTracker tracks the healing state for a single check
type HealTracker struct {
	LastAttempt         time.Time `json:"lastAttempt"`
	LastSuccess         time.Time `json:"lastSuccess"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	TotalAttempts       int       `json:"totalAttempts"`
	TotalSuccesses      int       `json:"totalSuccesses"`
	CooldownUntil       time.Time `json:"cooldownUntil"`
}

// IsInCooldown returns true if the check is still in cooldown period
func (ht *HealTracker) IsInCooldown() bool {
	return time.Now().Before(ht.CooldownUntil)
}

// CooldownRemaining returns the time remaining in cooldown, or 0 if not in cooldown
func (ht *HealTracker) CooldownRemaining() time.Duration {
	if !ht.IsInCooldown() {
		return 0
	}
	return time.Until(ht.CooldownUntil)
}

// CalculateCooldown determines the next cooldown duration based on failure count
func CalculateCooldown(consecutiveFailures int) time.Duration {
	if consecutiveFailures < MaxConsecutiveFailures {
		return DefaultHealCooldown
	}

	// Exponential backoff: 2^(failures - MaxConsecutiveFailures) * BackoffCooldown
	// Capped at MaxBackoffCooldown
	multiplier := 1 << (consecutiveFailures - MaxConsecutiveFailures) // 2^n
	cooldown := time.Duration(multiplier) * BackoffCooldown

	if cooldown > MaxBackoffCooldown {
		return MaxBackoffCooldown
	}
	return cooldown
}

// HealTrackerStore abstracts persistence of heal tracker state.
// This interface decouples the registry from the persistence package.
type HealTrackerStore interface {
	SaveHealTracker(ctx context.Context, checkID string, tracker *HealTracker) error
	GetAllHealTrackers(ctx context.Context) (map[string]*HealTracker, error)
	DeleteHealTracker(ctx context.Context, checkID string) error
}

// Registry manages health checks
type Registry struct {
	mu               sync.RWMutex
	checks           map[string]Check
	results          map[string]Result
	lastRun          map[string]time.Time
	healTrackers     map[string]*HealTracker // Track healing state per check
	platform         *platform.Capabilities
	config           ConfigProvider
	healTrackerStore HealTrackerStore // Optional persistence for heal trackers
}

// NewRegistry creates a new health check registry with the given platform capabilities.
// Platform is injected to allow testing and avoid hidden dependency creation.
func NewRegistry(plat *platform.Capabilities) *Registry {
	return &Registry{
		checks:       make(map[string]Check),
		results:      make(map[string]Result),
		lastRun:      make(map[string]time.Time),
		healTrackers: make(map[string]*HealTracker),
		platform:     plat,
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

// SetHealTrackerStore sets the store for persisting heal tracker state.
// When set, heal tracker state will be saved to the database after each heal attempt
// and loaded on startup via LoadHealTrackers.
// [REQ:HEAL-ACTION-001]
func (r *Registry) SetHealTrackerStore(store HealTrackerStore) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healTrackerStore = store
}

// LoadHealTrackers loads heal tracker state from the persistence store.
// Should be called during startup to restore state from the database.
// [REQ:HEAL-ACTION-001]
func (r *Registry) LoadHealTrackers(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.healTrackerStore == nil {
		return nil // No store configured, nothing to load
	}

	trackers, err := r.healTrackerStore.GetAllHealTrackers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load heal trackers: %w", err)
	}

	// Merge loaded trackers into in-memory map
	for checkID, tracker := range trackers {
		r.healTrackers[checkID] = tracker
	}

	return nil
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

// RunChecksForIDs runs checks for a specific list of check IDs.
// Used to re-check items after autoheal to update their status.
func (r *Registry) RunChecksForIDs(ctx context.Context, checkIDs []string) []Result {
	r.mu.RLock()
	var checksToRun []Check
	for _, id := range checkIDs {
		if check, exists := r.checks[id]; exists {
			checksToRun = append(checksToRun, check)
		}
	}
	r.mu.RUnlock()

	results := make([]Result, 0, len(checksToRun))
	for _, check := range checksToRun {
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

// runCheck executes a single check with timeout and stores the result
func (r *Registry) runCheck(ctx context.Context, check Check) Result {
	start := time.Now()

	// Create per-check timeout context
	checkCtx, cancel := context.WithTimeout(ctx, DefaultCheckTimeout)
	defer cancel()

	// Run check with timeout - use channel to capture result
	resultCh := make(chan Result, 1)
	go func() {
		resultCh <- check.Run(checkCtx)
	}()

	var result Result
	select {
	case result = <-resultCh:
		// Check completed normally
	case <-checkCtx.Done():
		// Check timed out
		result = Result{
			CheckID: check.ID(),
			Status:  StatusCritical,
			Message: fmt.Sprintf("Check timed out after %s", DefaultCheckTimeout),
			Details: map[string]interface{}{
				"error":   "timeout",
				"timeout": DefaultCheckTimeout.String(),
			},
		}
	}

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

// GetAllResults returns all stored check results for currently registered checks.
// Results for unregistered checks (orphaned results) are filtered out.
func (r *Registry) GetAllResults() []Result {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]Result, 0, len(r.results))
	for _, result := range r.results {
		// Only include results for currently registered checks
		if _, registered := r.checks[result.CheckID]; registered {
			results = append(results, result)
		}
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
	CheckID             string        `json:"checkId"`
	Attempted           bool          `json:"attempted"`
	ActionResult        ActionResult  `json:"actionResult,omitempty"`
	Reason              string        `json:"reason,omitempty"`            // Why it wasn't attempted
	CooldownRemaining   time.Duration `json:"cooldownRemaining,omitempty"` // Time until next attempt allowed
	ConsecutiveFailures int           `json:"consecutiveFailures,omitempty"`
}

// healCandidate represents a check that needs healing with its metadata
type healCandidate struct {
	result         Result
	healable       HealableCheck
	selectedAction *RecoveryAction
	priority       int // lower = higher priority
}

// getHealPriority returns priority for a check (lower = more important)
// Priority order: API (0) > Resources (1) > Scenarios (2) > Others (3)
func getHealPriority(checkID string) int {
	switch {
	case checkID == "vrooli-api":
		return 0 // API is most critical - other services depend on it
	case len(checkID) > 9 && checkID[:9] == "resource-":
		return 1 // Resources (postgres, redis, etc.) are infrastructure
	case len(checkID) > 9 && checkID[:9] == "scenario-":
		return 2 // Scenarios depend on API and resources
	default:
		return 3 // Unknown checks get lowest priority
	}
}

// RunAutoHeal attempts to auto-heal any critical checks that have auto-heal enabled.
// It runs the first available recovery action for each failing check.
// Implements cooldown and rate limiting to prevent thrashing on flapping services.
// Actions are prioritized (API > resources > scenarios) and run in parallel with limits.
// Returns a list of auto-heal results.
// [REQ:CONFIG-CHECK-001] [REQ:HEAL-ACTION-001]
func (r *Registry) RunAutoHeal(ctx context.Context, results []Result) []AutoHealResult {
	autoHealResults := make([]AutoHealResult, 0)
	candidates := make([]healCandidate, 0)

	// Phase 1: Collect and filter candidates
	for _, result := range results {
		// Auto-heal trigger is per-check policy (critical or warning+critical).
		if !r.shouldTriggerAutoHeal(result) {
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

		// Check cooldown before attempting heal
		tracker := r.getOrCreateHealTracker(result.CheckID)
		if tracker.IsInCooldown() {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             result.CheckID,
				Attempted:           false,
				Reason:              fmt.Sprintf("in cooldown (%.0fs remaining)", tracker.CooldownRemaining().Seconds()),
				CooldownRemaining:   tracker.CooldownRemaining(),
				ConsecutiveFailures: tracker.ConsecutiveFailures,
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

		selectedAction := selectAutoHealAction(result.CheckID, actions)

		if selectedAction == nil {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "no auto-heal recovery action available",
			})
			continue
		}

		// Add to candidates list
		candidates = append(candidates, healCandidate{
			result:         result,
			healable:       healable,
			selectedAction: selectedAction,
			priority:       getHealPriority(result.CheckID),
		})
	}

	// Phase 2: Sort candidates by priority (lower priority number = more important)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].priority < candidates[i].priority {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Phase 3: Limit number of actions per tick
	if len(candidates) > MaxAutoHealActionsPerTick {
		// Mark excess candidates as skipped
		for _, c := range candidates[MaxAutoHealActionsPerTick:] {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   c.result.CheckID,
				Attempted: false,
				Reason:    fmt.Sprintf("deferred to next tick (max %d actions per tick, lower priority)", MaxAutoHealActionsPerTick),
			})
		}
		candidates = candidates[:MaxAutoHealActionsPerTick]
	}

	// Phase 4: Execute heal actions with per-action timeout.
	// We intentionally avoid waiting on a global worker group here; a single stuck
	// action must not block all future ticks behind a permanent tick_in_progress lock.
	if len(candidates) == 0 {
		return autoHealResults
	}

	for _, c := range candidates {
		// Check context before executing
		select {
		case <-ctx.Done():
			actionResult := ActionResult{
				ActionID:  c.selectedAction.ID,
				CheckID:   c.result.CheckID,
				Timestamp: time.Now(),
				Success:   false,
				Error:     "context cancelled",
				Message:   "Heal action cancelled due to timeout",
			}
			r.updateHealTracker(c.result.CheckID, false)
			updatedTracker := r.getOrCreateHealTracker(c.result.CheckID)
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             c.result.CheckID,
				Attempted:           true,
				ActionResult:        actionResult,
				CooldownRemaining:   updatedTracker.CooldownRemaining(),
				ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
			})
			continue
		default:
		}

		actionCtx, cancel := context.WithTimeout(ctx, DefaultAutoHealActionTimeout)
		actionResult := r.executeAutoHealActionWithTimeout(actionCtx, c)
		cancel()

		// Update heal tracker based on result
		r.updateHealTracker(c.result.CheckID, actionResult.Success)

		// Get updated tracker for result
		updatedTracker := r.getOrCreateHealTracker(c.result.CheckID)

		autoHealResults = append(autoHealResults, AutoHealResult{
			CheckID:             c.result.CheckID,
			Attempted:           true,
			ActionResult:        actionResult,
			CooldownRemaining:   updatedTracker.CooldownRemaining(),
			ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
		})
	}

	return autoHealResults
}

func (r *Registry) executeAutoHealActionWithTimeout(ctx context.Context, candidate healCandidate) ActionResult {
	resultCh := make(chan ActionResult, 1)
	go func() {
		resultCh <- candidate.healable.ExecuteAction(ctx, candidate.selectedAction.ID)
	}()

	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		reason := "action timed out"
		if ctx.Err() == context.Canceled {
			reason = "action cancelled"
		}
		return ActionResult{
			ActionID:  candidate.selectedAction.ID,
			CheckID:   candidate.result.CheckID,
			Timestamp: time.Now(),
			Success:   false,
			Error:     reason,
			Message:   "Auto-heal action did not complete before timeout",
			Duration:  DefaultAutoHealActionTimeout,
		}
	}
}

func (r *Registry) shouldTriggerAutoHeal(result Result) bool {
	if result.Details != nil {
		if eligible, ok := result.Details["autoHealEligible"].(bool); ok && !eligible {
			return false
		}
	}

	// Safety default: only critical triggers unless explicitly widened.
	policy := "critical"
	if r.config != nil {
		if configured := r.config.GetAutoHealOn(result.CheckID); configured != "" {
			policy = configured
		}
	}

	switch policy {
	case "warning+critical":
		return result.Status == StatusWarning || result.Status == StatusCritical
	default:
		return result.Status == StatusCritical
	}
}

func selectAutoHealAction(checkID string, actions []RecoveryAction) *RecoveryAction {
	// Controlled dangerous action policy for scenarios:
	// allow "restart" auto-execution for scenario checks only.
	if strings.HasPrefix(checkID, "scenario-") {
		for _, action := range actions {
			if action.Available && action.ID == "restart" {
				return &action
			}
		}
	}

	// Default policy: first available safe action.
	for _, action := range actions {
		if action.Available && !action.Dangerous {
			return &action
		}
	}

	return nil
}

// getOrCreateHealTracker returns the heal tracker for a check, creating one if needed
func (r *Registry) getOrCreateHealTracker(checkID string) *HealTracker {
	r.mu.Lock()
	defer r.mu.Unlock()

	tracker, exists := r.healTrackers[checkID]
	if !exists {
		tracker = &HealTracker{}
		r.healTrackers[checkID] = tracker
	}
	return tracker
}

// updateHealTracker updates the heal tracker after a heal attempt and persists to store
func (r *Registry) updateHealTracker(checkID string, success bool) {
	r.mu.Lock()

	tracker, exists := r.healTrackers[checkID]
	if !exists {
		tracker = &HealTracker{}
		r.healTrackers[checkID] = tracker
	}

	now := time.Now()
	tracker.LastAttempt = now
	tracker.TotalAttempts++

	if success {
		tracker.LastSuccess = now
		tracker.TotalSuccesses++
		tracker.ConsecutiveFailures = 0
		// Still apply a cooldown after success to prevent rapid re-triggering
		tracker.CooldownUntil = now.Add(DefaultHealCooldown)
	} else {
		tracker.ConsecutiveFailures++
		// Calculate backoff cooldown based on consecutive failures
		cooldown := CalculateCooldown(tracker.ConsecutiveFailures)
		tracker.CooldownUntil = now.Add(cooldown)
	}

	// Persist to store if configured (async to not block)
	store := r.healTrackerStore
	trackerCopy := *tracker
	r.mu.Unlock()

	if store != nil {
		// Use background context for persistence - don't block on this
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Ignore errors - persistence is best-effort, in-memory state is authoritative
			_ = store.SaveHealTracker(ctx, checkID, &trackerCopy)
		}()
	}
}

// GetHealTracker returns the heal tracker for a check (for API exposure)
func (r *Registry) GetHealTracker(checkID string) (*HealTracker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tracker, exists := r.healTrackers[checkID]
	if !exists {
		return nil, false
	}
	// Return a copy to prevent external modification
	trackerCopy := *tracker
	return &trackerCopy, true
}

// GetAllHealTrackers returns all heal trackers (for API exposure)
func (r *Registry) GetAllHealTrackers() map[string]HealTracker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]HealTracker, len(r.healTrackers))
	for id, tracker := range r.healTrackers {
		result[id] = *tracker
	}
	return result
}

// ResetHealTracker resets the heal tracker for a check (for manual intervention)
func (r *Registry) ResetHealTracker(checkID string) {
	r.mu.Lock()
	delete(r.healTrackers, checkID)
	store := r.healTrackerStore
	r.mu.Unlock()

	// Delete from persistent store if configured (async)
	if store != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = store.DeleteHealTracker(ctx, checkID)
		}()
	}
}
