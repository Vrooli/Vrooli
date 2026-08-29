// Package checks provides the health check registry
// [REQ:HEALTH-REGISTRY-001] [REQ:HEALTH-REGISTRY-002] [REQ:HEALTH-REGISTRY-003] [REQ:HEALTH-REGISTRY-004]
package checks

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	mathrand "math/rand"
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

	// DefaultHealInterlockWindow is deliberately short: it contains cross-check
	// disagreement without indefinitely masking a genuinely flapping target.
	DefaultHealInterlockWindow = 30 * time.Second
)

// longRunningActionIDs enumerates action IDs that receive the long lifecycle
// timeout. Timeout budgeting and recovery ownership are separate concerns:
// starting a stopped scenario is long-running but does not need the restart
// coordination gate.
var longRunningActionIDs = map[string]struct{}{
	"restart":       {},
	"restart-clean": {},
	"setup-restart": {},
	"recover-go":    {},
	"recover-pnpm":  {},
	"start":         {},
	"stop":          {},
	"reclaim":       {},
}

// gatedActionIDs enumerates actions that require runtime recovery ownership.
// A start action is intentionally absent: starting a scenario already known
// to be stopped cannot race a running-process recovery controller.
var gatedActionIDs = map[string]struct{}{
	"restart":       {},
	"restart-clean": {},
	"setup-restart": {},
	"recover-go":    {},
	"recover-pnpm":  {},
	"stop":          {},
	"reclaim":       {},
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
	SuspendedAt         time.Time `json:"suspendedAt,omitempty"`
	SuspensionReason    string    `json:"suspensionReason,omitempty"`
	Disposition         string    `json:"disposition,omitempty"`
	DispositionAt       time.Time `json:"dispositionAt,omitempty"`
	SuccessHistory      string    `json:"successHistory,omitempty"`
}

const (
	HealDispositionHealed    = "healed"
	HealDispositionRetired   = "retired"
	HealDispositionEscalated = "escalated"
	HealSuccessNever         = "never_succeeded"
	HealSuccessPrevious      = "previously_succeeded"
)

func (ht *HealTracker) IsSuspended() bool { return !ht.SuspendedAt.IsZero() }

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

// RecoveryOwnershipGate prevents autoheal from racing the runtime supervisor's
// pressure-epoch recovery controller. It is intentionally narrow: health
// checks continue to run, but restart-class actions defer while ownership is
// held elsewhere.
type RecoveryOwnershipGate interface {
	AllowsAutoHealRestart(ctx context.Context, checkID, actionID string) (allowed bool, reason string)
}

// HealIncidentReporter is the narrow persistence seam for operator-facing
// escalation. The registry owns the threshold decision; the incident service
// owns durable lifecycle records.
type HealIncidentReporter interface {
	OpenHealIncident(ctx context.Context, checkID, actionID, lastError string, consecutiveFailures int) error
	ResolveHealIncident(ctx context.Context, checkID, actionID string) error
}

// Registry manages health checks
type Registry struct {
	mu                   sync.RWMutex
	checks               map[string]Check
	results              map[string]Result
	lastRun              map[string]time.Time
	healTrackers         map[string]*HealTracker // Track healing state per check
	autoHealPolicy       *AutoHealPolicy
	platform             *platform.Capabilities
	config               ConfigProvider
	healTrackerStore     HealTrackerStore // Optional persistence for heal trackers
	recoveryGate         RecoveryOwnershipGate
	healIncidentReporter HealIncidentReporter
	clock                Clock
	interlockMu          sync.Mutex
	recentHealActions    map[HealTarget]RecentHealAction
	healInterlockWindow  time.Duration
}

// SetRecoveryOwnershipGate installs the cross-controller restart ownership
// boundary. Nil restores standalone autoheal behavior.
func (r *Registry) SetRecoveryOwnershipGate(gate RecoveryOwnershipGate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryGate = gate
}

// SetHealIncidentReporter wires durable escalation without coupling the check
// registry to a particular persistence implementation.
func (r *Registry) SetHealIncidentReporter(reporter HealIncidentReporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healIncidentReporter = reporter
}

func (r *Registry) getHealIncidentReporter() HealIncidentReporter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.healIncidentReporter
}

func (r *Registry) recoveryOwnershipGate() RecoveryOwnershipGate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.recoveryGate
}

// Clock provides a seam for time-based logic.
type Clock = TimeSource

type TimeSource interface {
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
		checks:              make(map[string]Check),
		results:             make(map[string]Result),
		lastRun:             make(map[string]time.Time),
		healTrackers:        make(map[string]*HealTracker),
		recentHealActions:   make(map[HealTarget]RecentHealAction),
		healInterlockWindow: DefaultHealInterlockWindow,
		platform:            plat,
		clock:               realClock{},
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
// rng may be nil, in which case the operating system entropy source is used.
// Passing a deterministic source makes the spread assertable in tests.
func (r *Registry) SeedStartupJitter(now time.Time, rng *mathrand.Rand) {
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
		var offset time.Duration
		if rng != nil {
			offset = time.Duration(rng.Int63n(int64(interval)))
		} else {
			offset = secureJitterOffset(interval)
		}
		r.lastRun[id] = now.Add(-offset)
	}
}

// secureJitterOffset supplies startup spreading without using a
// cryptographically-weak PRNG in the production default. Tests can still pass
// math/rand.Rand to make the schedule deterministic.
func secureJitterOffset(interval time.Duration) time.Duration {
	if interval <= 0 {
		return 0
	}
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(interval)))
	if err != nil {
		// Jitter is an optimization, not a correctness condition. A stable
		// midpoint keeps startup safe if the OS entropy source is unavailable.
		return interval / 2
	}
	return time.Duration(n.Int64())
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
		if tracker.LastSuccess.IsZero() {
			tracker.SuccessHistory = HealSuccessNever
		} else {
			tracker.SuccessHistory = HealSuccessPrevious
		}
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
	if !r.platformCompatible(check) {
		return false
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

func (r *Registry) platformCompatible(check Check) bool {
	platforms := check.Platforms()
	if len(platforms) == 0 {
		return true
	}
	for _, p := range platforms {
		if p == r.platform.Platform {
			return true
		}
	}
	return false
}

func (r *Registry) checkEnabled(check Check) bool {
	return r.config == nil || r.config.IsCheckEnabled(check.ID())
}

func (r *Registry) notApplicableResult(check Check) Result {
	return Result{
		CheckID: check.ID(),
		Status:  StatusNotApplicable,
		Message: fmt.Sprintf("%s is not applicable on platform %s", check.Title(), r.platform.Platform),
		Details: map[string]interface{}{
			"platform":           r.platform.Platform,
			"supportedPlatforms": check.Platforms(),
		},
		Timestamp: time.Now().UTC(),
	}
}

// RunAll executes all registered checks that should run
func (r *Registry) RunAll(ctx context.Context, forceAll bool) []Result {
	r.mu.RLock()
	checks := make([]Check, 0, len(r.checks))
	for _, check := range r.checks {
		if !r.checkEnabled(check) {
			continue
		}
		if !r.platformCompatible(check) {
			if forceAll || !r.lastRunExists(check.ID()) || r.shouldRunCheck(check, false) {
				checks = append(checks, check)
			}
			continue
		}
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
			var result Result
			if r.platformCompatible(check) {
				result = r.runCheck(ctx, check)
			} else {
				result = r.notApplicableResult(check)
				r.storeResult(result)
			}
			results = append(results, result)
		}
	}

	return results
}

func (r *Registry) lastRunExists(checkID string) bool {
	_, exists := r.lastRun[checkID]
	return exists
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
			var result Result
			if r.platformCompatible(check) {
				result = r.runCheck(ctx, check)
			} else {
				result = r.notApplicableResult(check)
				r.storeResult(result)
			}
			results = append(results, result)
		}
	}

	return results
}

func (r *Registry) storeResult(result Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[result.CheckID] = result
	r.lastRun[result.CheckID] = result.Timestamp
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
