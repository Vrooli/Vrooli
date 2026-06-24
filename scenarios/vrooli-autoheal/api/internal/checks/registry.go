// Package checks provides the health check registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

const (
	// DefaultCheckTimeout is the maximum time a single health check can run before timing out
	DefaultCheckTimeout = 30 * time.Second

	// MaxAutoHealActionsPerTick limits how many heal actions can run per tick to prevent timeout
	MaxAutoHealActionsPerTick = 5

	// MaxParallelHealActions limits concurrent heal action execution
	MaxParallelHealActions = 3

	// DefaultFastActionTimeout bounds quick diagnostic / cleanup actions
	// (logs, diagnose, cleanup-ports, kill). A short ceiling prevents a stuck
	// helper from blocking the tick.
	DefaultFastActionTimeout = 30 * time.Second

	// DefaultRestartActionTimeout bounds scenario restart / lifecycle actions
	// (restart, restart-clean, setup-restart, start, stop). Scenario restarts
	// can legitimately take minutes when a dependency requires a cold build.
	DefaultRestartActionTimeout = 5 * time.Minute

	// DefaultTimeoutRetryCooldown is the short cooldown applied after a heal
	// action times out, so the next tick can retry quickly instead of being
	// silenced by the failure-cooldown ratchet.
	DefaultTimeoutRetryCooldown = 30 * time.Second
)

// restartActionIDs enumerates action IDs that may run as long as a full
// scenario restart. Unknown actions default to the fast timeout.
var restartActionIDs = map[string]struct{}{
	"restart":       {},
	"restart-clean": {},
	"setup-restart": {},
	"recover-go":    {},
	"recover-pnpm":  {},
	"start":         {},
	"stop":          {},
}

// healOutcome is the tri-state outcome of an auto-heal attempt.
// Distinguishing "timed out" from "failed" keeps a slow-but-recoverable
// action from ratcheting ConsecutiveFailures into the exponential cooldown.
type healOutcome int

const (
	outcomeSuccess healOutcome = iota
	outcomeFailure
	outcomeTimeout
)

// AutoHealPolicy controls cooldown and retry behavior for auto-heal actions.
// This must be configured explicitly from user configuration.
type AutoHealPolicy struct {
	// BaseCooldown is applied after successful heals and for early failures.
	BaseCooldown time.Duration
	// MaxRestartAttempts is the failure threshold after which backoff increases exponentially.
	MaxRestartAttempts int
	// FastActionTimeout bounds quick diagnostic / cleanup actions.
	FastActionTimeout time.Duration
	// RestartActionTimeout bounds scenario restart / lifecycle actions.
	RestartActionTimeout time.Duration
	// TimeoutRetryCooldown is the short cooldown applied after a timeout outcome.
	TimeoutRetryCooldown time.Duration
}

// NewAutoHealPolicyFromGlobal creates a policy from global configuration values.
// Action timeouts and the timeout-retry cooldown fall back to package defaults
// when the provided seconds are zero.
func NewAutoHealPolicyFromGlobal(restartCooldownSeconds, maxRestartAttempts, fastActionTimeoutSeconds, restartActionTimeoutSeconds, timeoutRetrySeconds int) (AutoHealPolicy, error) {
	policy := AutoHealPolicy{
		BaseCooldown:         time.Duration(restartCooldownSeconds) * time.Second,
		MaxRestartAttempts:   maxRestartAttempts,
		FastActionTimeout:    time.Duration(fastActionTimeoutSeconds) * time.Second,
		RestartActionTimeout: time.Duration(restartActionTimeoutSeconds) * time.Second,
		TimeoutRetryCooldown: time.Duration(timeoutRetrySeconds) * time.Second,
	}
	policy.applyTimeoutDefaults()
	if err := policy.Validate(); err != nil {
		return AutoHealPolicy{}, err
	}
	return policy, nil
}

func (p *AutoHealPolicy) applyTimeoutDefaults() {
	if p.FastActionTimeout <= 0 {
		p.FastActionTimeout = DefaultFastActionTimeout
	}
	if p.RestartActionTimeout <= 0 {
		p.RestartActionTimeout = DefaultRestartActionTimeout
	}
	if p.TimeoutRetryCooldown <= 0 {
		p.TimeoutRetryCooldown = DefaultTimeoutRetryCooldown
	}
}

// Validate ensures the policy is safe and usable.
func (p AutoHealPolicy) Validate() error {
	if p.BaseCooldown <= 0 {
		return fmt.Errorf("base cooldown must be > 0")
	}
	if p.MaxRestartAttempts < 1 {
		return fmt.Errorf("max restart attempts must be >= 1")
	}
	if p.FastActionTimeout <= 0 {
		return fmt.Errorf("fast action timeout must be > 0")
	}
	if p.RestartActionTimeout <= 0 {
		return fmt.Errorf("restart action timeout must be > 0")
	}
	if p.TimeoutRetryCooldown <= 0 {
		return fmt.Errorf("timeout retry cooldown must be > 0")
	}
	return nil
}

// MaxFailureCooldown caps exponential backoff so auto-heal cannot be silenced
// indefinitely. Without a cap, a few-dozen failed restarts pushed cooldowns
// into the multi-day range, leaving broken scenarios effectively unrecoverable
// until manual intervention.
const MaxFailureCooldown = 1 * time.Hour

// CalculateFailureCooldown returns the cooldown after a failed auto-heal attempt.
// For failures below MaxRestartAttempts, BaseCooldown is used.
// Once failures reach MaxRestartAttempts, cooldown grows exponentially:
// BaseCooldown * 2^(consecutiveFailures - MaxRestartAttempts + 1), capped at
// MaxFailureCooldown.
func (p AutoHealPolicy) CalculateFailureCooldown(consecutiveFailures int) time.Duration {
	if consecutiveFailures <= 0 {
		return p.BaseCooldown
	}
	if consecutiveFailures < p.MaxRestartAttempts {
		return capCooldown(p.BaseCooldown)
	}

	shift := consecutiveFailures - p.MaxRestartAttempts + 1
	// Saturate the shift before computing the multiplier to avoid integer
	// overflow at very large failure counts (1 << 62 would still be valid as
	// an int64, but 1 << 63 wraps negative). 30 is well past the point the
	// cap kicks in for any sane BaseCooldown.
	if shift > 30 {
		return capCooldown(MaxFailureCooldown)
	}
	multiplier := time.Duration(1 << shift)
	cooldown := multiplier * p.BaseCooldown
	if cooldown < 0 || cooldown > MaxFailureCooldown {
		return MaxFailureCooldown
	}
	return cooldown
}

func capCooldown(d time.Duration) time.Duration {
	if d > MaxFailureCooldown {
		return MaxFailureCooldown
	}
	return d
}

// HealTracker tracks the healing state for a single check
type HealTracker struct {
	LastAttempt         time.Time `json:"lastAttempt"`
	LastSuccess         time.Time `json:"lastSuccess"`
	ConsecutiveFailures int       `json:"consecutiveFailures"`
	ConsecutiveTimeouts int       `json:"consecutiveTimeouts,omitempty"`
	TotalAttempts       int       `json:"totalAttempts"`
	TotalSuccesses      int       `json:"totalSuccesses"`
	TotalTimeouts       int       `json:"totalTimeouts,omitempty"`
	CooldownUntil       time.Time `json:"cooldownUntil"`
}

// IsInCooldown returns true if the check is still in cooldown period
func (ht *HealTracker) IsInCooldown() bool {
	return time.Now().Before(ht.CooldownUntil)
}

// IsInCooldownAt returns true if the check is in cooldown at the provided time.
func (ht *HealTracker) IsInCooldownAt(now time.Time) bool {
	return now.Before(ht.CooldownUntil)
}

// CooldownRemaining returns the time remaining in cooldown, or 0 if not in cooldown
func (ht *HealTracker) CooldownRemaining() time.Duration {
	if !ht.IsInCooldown() {
		return 0
	}
	return time.Until(ht.CooldownUntil)
}

// CooldownRemainingAt returns remaining cooldown at the provided time.
func (ht *HealTracker) CooldownRemainingAt(now time.Time) time.Duration {
	if !ht.IsInCooldownAt(now) {
		return 0
	}
	return ht.CooldownUntil.Sub(now)
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
	autoHealPolicy   *AutoHealPolicy
	platform         *platform.Capabilities
	config           ConfigProvider
	healTrackerStore HealTrackerStore // Optional persistence for heal trackers
	clock            Clock
}

// Clock provides a seam for time-based logic.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
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
		clock:        realClock{},
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

// SetAutoHealPolicy configures cooldown/backoff behavior for auto-heal.
// This is required for RunAutoHeal to execute actions.
func (r *Registry) SetAutoHealPolicy(policy AutoHealPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	p := policy
	r.autoHealPolicy = &p
	return nil
}

// SeedStartupJitter staggers the first run of interval checks so aligned
// intervals don't burst together (a synchronized "thundering herd" that pegs a
// core). For each check WITHOUT an existing lastRun (i.e. not restored from
// persistence), it seeds lastRun to `now - offset` where offset is a per-check
// random value in [0, interval). Because shouldRunCheck fires when
// time.Since(lastRun) >= interval, the check first becomes due at
// `now + (interval - offset)`, spreading first runs uniformly across the first
// interval window. Checks already seeded from persisted results are left
// untouched so restart behavior is unchanged.
//
// rng may be nil, in which case a time-seeded source is used. Passing a
// deterministic source makes the spread assertable in tests.
func (r *Registry) SeedStartupJitter(now time.Time, rng *rand.Rand) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, check := range r.checks {
		if _, exists := r.lastRun[id]; exists {
			continue
		}
		interval := time.Duration(check.IntervalSeconds()) * time.Second
		if interval <= 0 {
			continue
		}
		offset := time.Duration(rng.Int63n(int64(interval)))
		r.lastRun[id] = now.Add(-offset)
	}
}

// SetClock sets the time source for cooldown calculations (used by tests).
func (r *Registry) SetClock(clock Clock) {
	if clock == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clock = clock
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

	// Check interval.
	// Caller must hold at least a read lock for r.lastRun access.
	lastRun, exists := r.lastRun[check.ID()]

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
	TimedOut            bool          `json:"timedOut,omitempty"`          // Mirrors ActionResult.TimedOut for callers that only inspect AutoHealResult.
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
	detectedPID    int // PID at detection time (0 if unknown); used to prevent killing a new process
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
	now := r.now()
	_, hasPolicy := r.getAutoHealPolicy()

	// Phase 1: Collect and filter candidates
	for _, result := range results {
		// Auto-heal trigger is per-check policy (critical or warning+critical).
		if !r.shouldTriggerAutoHeal(result) {
			continue
		}

		if !hasPolicy {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "auto-heal policy not configured",
			})
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

		// Check cooldown before attempting heal.
		// Use a snapshot to avoid races with concurrent tracker updates.
		tracker := r.getHealTrackerSnapshot(result.CheckID)
		if tracker.IsInCooldownAt(now) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             result.CheckID,
				Attempted:           false,
				Reason:              fmt.Sprintf("in cooldown (%.0fs remaining)", tracker.CooldownRemainingAt(now).Seconds()),
				CooldownRemaining:   tracker.CooldownRemainingAt(now),
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

		selectedAction := selectAutoHealAction(result, actions)

		if selectedAction == nil {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "no auto-heal recovery action available",
			})
			continue
		}

		// Capture PID at detection time for TOCTOU protection.
		var detectedPID int
		if result.Details != nil {
			if pid, ok := result.Details["detectedPID"].(int); ok {
				detectedPID = pid
			} else if pidFloat, ok := result.Details["detectedPID"].(float64); ok {
				detectedPID = int(pidFloat)
			}
		}

		// Add to candidates list
		candidates = append(candidates, healCandidate{
			result:         result,
			healable:       healable,
			selectedAction: selectedAction,
			priority:       getHealPriority(result.CheckID),
			detectedPID:    detectedPID,
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
	// Before each action, re-run the health check to guard against TOCTOU races
	// (e.g., a slow tick detects unhealthy at T=0, but by T=90s a fresh process is healthy).
	// Also verify that the process PID hasn't changed, which indicates a new process
	// replaced the unhealthy one and the heal action is no longer needed.
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
			r.updateHealTracker(c.result.CheckID, outcomeFailure)
			updatedTracker := r.getHealTrackerSnapshot(c.result.CheckID)
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             c.result.CheckID,
				Attempted:           true,
				ActionResult:        actionResult,
				CooldownRemaining:   updatedTracker.CooldownRemainingAt(r.now()),
				ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
			})
			continue
		default:
		}

		// Pre-heal recheck: re-run the health check to verify the target is
		// still unhealthy. This prevents the TOCTOU race where detection happens
		// long before execution (e.g., due to SQLITE_BUSY delays or earlier
		// heal actions taking time).
		if skipped := r.preHealRecheck(ctx, c, &autoHealResults); skipped {
			continue
		}

		actionCtx, cancel := context.WithTimeout(ctx, r.actionTimeoutFor(c.selectedAction.ID))
		actionResult := r.executeAutoHealActionWithTimeout(actionCtx, c)
		cancel()

		// Update heal tracker based on result
		r.updateHealTracker(c.result.CheckID, outcomeFromActionResult(actionResult))

		// Get updated tracker for result
		updatedTracker := r.getHealTrackerSnapshot(c.result.CheckID)

		autoHealResults = append(autoHealResults, AutoHealResult{
			CheckID:             c.result.CheckID,
			Attempted:           true,
			ActionResult:        actionResult,
			TimedOut:            actionResult.TimedOut,
			CooldownRemaining:   updatedTracker.CooldownRemainingAt(r.now()),
			ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
		})
	}

	return autoHealResults
}

func (r *Registry) executeAutoHealActionWithTimeout(ctx context.Context, candidate healCandidate) ActionResult {
	start := time.Now()
	resultCh := make(chan ActionResult, 1)
	go func() {
		resultCh <- candidate.healable.ExecuteAction(ctx, candidate.selectedAction.ID)
	}()

	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		timedOut := ctx.Err() != context.Canceled
		reason := "action timed out"
		message := "Auto-heal action did not complete before timeout"
		if !timedOut {
			reason = "action cancelled"
			message = "Auto-heal action cancelled before completing"
		}
		return ActionResult{
			ActionID:  candidate.selectedAction.ID,
			CheckID:   candidate.result.CheckID,
			Timestamp: time.Now(),
			Success:   false,
			TimedOut:  timedOut,
			Error:     reason,
			Message:   message,
			Duration:  time.Since(start),
		}
	}
}

// actionTimeoutFor resolves the deadline for an individual auto-heal action.
// Restart-class actions (full scenario restarts) get a generous budget so a
// dependency cold-build doesn't time them out. Everything else gets the short
// budget — unknown actions fall back to fast so a misbehaving action becomes
// a timeout retry rather than a silent budget hog.
func (r *Registry) actionTimeoutFor(actionID string) time.Duration {
	policy, ok := r.getAutoHealPolicy()
	fast := DefaultFastActionTimeout
	restart := DefaultRestartActionTimeout
	if ok {
		fast = policy.FastActionTimeout
		restart = policy.RestartActionTimeout
	}
	if _, isRestart := restartActionIDs[actionID]; isRestart {
		return restart
	}
	return fast
}

// outcomeFromActionResult collapses the (Success, TimedOut) pair on
// ActionResult into the tri-state healOutcome used by the heal tracker.
func outcomeFromActionResult(r ActionResult) healOutcome {
	switch {
	case r.Success:
		return outcomeSuccess
	case r.TimedOut:
		return outcomeTimeout
	default:
		return outcomeFailure
	}
}

// preHealRecheck re-runs the health check for a candidate and optionally verifies
// that the PID hasn't changed since detection. If the target is now healthy or the
// PID changed (indicating a new process replaced the unhealthy one), the candidate
// is skipped and a non-attempted result is appended.
// Returns true if the candidate was skipped.
func (r *Registry) preHealRecheck(ctx context.Context, c healCandidate, results *[]AutoHealResult) bool {
	recheckCtx, cancel := context.WithTimeout(ctx, DefaultCheckTimeout)
	defer cancel()

	// Re-run the check to see if it's still unhealthy.
	r.mu.RLock()
	check, exists := r.checks[c.result.CheckID]
	r.mu.RUnlock()
	if !exists {
		return false // Check was unregistered; let the caller handle it
	}

	freshResult := check.Run(recheckCtx)

	// If the check is now healthy, skip the heal action.
	if freshResult.Status == StatusOK {
		*results = append(*results, AutoHealResult{
			CheckID:   c.result.CheckID,
			Attempted: false,
			Reason:    "pre-heal recheck passed: target is now healthy",
		})
		return true
	}

	// PID-pinning: if we captured a PID at detection time and the current PID
	// differs, a new process has started. Skip the heal to avoid killing it.
	if c.detectedPID > 0 && freshResult.Details != nil {
		var currentPID int
		if pid, ok := freshResult.Details["detectedPID"].(int); ok {
			currentPID = pid
		} else if pidFloat, ok := freshResult.Details["detectedPID"].(float64); ok {
			currentPID = int(pidFloat)
		}
		if currentPID > 0 && currentPID != c.detectedPID {
			*results = append(*results, AutoHealResult{
				CheckID:   c.result.CheckID,
				Attempted: false,
				Reason: fmt.Sprintf(
					"PID changed since detection (was %d, now %d): new process started, skipping heal",
					c.detectedPID, currentPID,
				),
			})
			return true
		}
	}

	return false
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

func selectAutoHealAction(result Result, actions []RecoveryAction) *RecoveryAction {
	checkID := result.CheckID

	// Policy: orphan cleanup may auto-run the core maintenance cleanup action.
	if checkID == "vrooli-orphans" {
		for _, action := range actions {
			if action.Available && action.ID == "kill" {
				return &action
			}
		}
	}

	// Specialized scenario policy: when a drift signature is detected, prefer
	// the targeted recovery action over a plain restart loop. Language-specific
	// recoveries (recover-go / recover-pnpm) are cheaper than setup-restart
	// and the signature gating in the scenario check ensures they only fire
	// when their healable pattern is actually present.
	if strings.HasPrefix(checkID, "scenario-") && result.Details != nil {
		if recommended, ok := result.Details["recommendedAction"].(string); ok && recommended != "" {
			for _, action := range actions {
				if action.Available && action.ID == recommended {
					return &action
				}
			}
		}
	}

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

// getHealTrackerSnapshot returns a copy of tracker state for safe concurrent reads.
// It creates the tracker entry if missing.
func (r *Registry) getHealTrackerSnapshot(checkID string) HealTracker {
	r.mu.Lock()
	defer r.mu.Unlock()

	tracker, exists := r.healTrackers[checkID]
	if !exists {
		tracker = &HealTracker{}
		r.healTrackers[checkID] = tracker
	}

	return *tracker
}

// updateHealTracker updates the heal tracker after a heal attempt and persists to store.
//
// The outcome distinguishes between success, genuine failure, and timeout.
// Timeouts apply a short retry cooldown and do not ratchet ConsecutiveFailures —
// a slow-but-recoverable action should be retried on the next tick, not silenced
// by the exponential failure backoff. After MaxRestartAttempts consecutive
// timeouts the tracker falls through to the failure path so a permanently-stuck
// action does eventually cool down.
func (r *Registry) updateHealTracker(checkID string, outcome healOutcome) {
	r.mu.Lock()

	tracker, exists := r.healTrackers[checkID]
	if !exists {
		tracker = &HealTracker{}
		r.healTrackers[checkID] = tracker
	}

	now := r.clock.Now()
	tracker.LastAttempt = now
	tracker.TotalAttempts++

	policy := r.autoHealPolicy
	switch outcome {
	case outcomeSuccess:
		tracker.LastSuccess = now
		tracker.TotalSuccesses++
		tracker.ConsecutiveFailures = 0
		tracker.ConsecutiveTimeouts = 0
		// Apply base cooldown after success to prevent rapid re-triggering.
		if policy != nil {
			tracker.CooldownUntil = now.Add(policy.BaseCooldown)
		} else {
			tracker.CooldownUntil = now
		}
	case outcomeTimeout:
		tracker.TotalTimeouts++
		tracker.ConsecutiveTimeouts++
		// Safety cap: after MaxRestartAttempts consecutive timeouts a stuck
		// action falls through to the regular failure ratchet so it eventually
		// cools down on the exponential backoff.
		if policy != nil && tracker.ConsecutiveTimeouts > policy.MaxRestartAttempts {
			tracker.ConsecutiveFailures++
			cooldown := policy.CalculateFailureCooldown(tracker.ConsecutiveFailures)
			tracker.CooldownUntil = now.Add(cooldown)
		} else if policy != nil {
			tracker.CooldownUntil = now.Add(policy.TimeoutRetryCooldown)
		} else {
			tracker.CooldownUntil = now.Add(DefaultTimeoutRetryCooldown)
		}
	case outcomeFailure:
		fallthrough
	default:
		tracker.ConsecutiveFailures++
		tracker.ConsecutiveTimeouts = 0
		if policy != nil {
			cooldown := policy.CalculateFailureCooldown(tracker.ConsecutiveFailures)
			tracker.CooldownUntil = now.Add(cooldown)
		} else {
			tracker.CooldownUntil = now
		}
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

func (r *Registry) now() time.Time {
	r.mu.RLock()
	clock := r.clock
	r.mu.RUnlock()
	if clock == nil {
		return time.Now()
	}
	return clock.Now()
}

func (r *Registry) getAutoHealPolicy() (*AutoHealPolicy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.autoHealPolicy == nil {
		return nil, false
	}
	p := *r.autoHealPolicy
	return &p, true
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
