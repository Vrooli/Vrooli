package checks

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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
// Priority order: Resources (1) > Scenarios (2) > Others (3)
func getHealPriority(checkID string) int {
	switch {
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
		var actions []RecoveryAction
		if contextAware, ok := healable.(ContextAwareHealableCheck); ok {
			actions = contextAware.RecoveryActionsWithContext(ctx, &result)
		} else {
			actions = healable.RecoveryActions(&result)
		}

		selectedAction := selectAutoHealAction(result, actions)

		if selectedAction == nil {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "no auto-heal recovery action available",
			})
			continue
		}
		if _, restartAction := restartActionIDs[selectedAction.ID]; restartAction {
			if gate := r.recoveryOwnershipGate(); gate != nil {
				if allowed, reason := gate.AllowsAutoHealRestart(ctx, result.CheckID, selectedAction.ID); !allowed {
					autoHealResults = append(autoHealResults, AutoHealResult{
						CheckID: result.CheckID, Attempted: false, Reason: reason,
					})
					continue
				}
			}
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
		outcome := outcomeFromActionResult(actionResult)
		r.updateHealTracker(c.result.CheckID, outcome)

		// Get updated tracker for result
		updatedTracker := r.getHealTrackerSnapshot(c.result.CheckID)
		r.recordHealIncident(ctx, c.result.CheckID, c.selectedAction.ID, actionResult, outcome, updatedTracker)

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

func (r *Registry) recordHealIncident(ctx context.Context, checkID, actionID string, actionResult ActionResult, outcome healOutcome, tracker HealTracker) {
	reporter := r.getHealIncidentReporter()
	if reporter == nil {
		return
	}
	switch outcome {
	case outcomeSuccess:
		if err := reporter.ResolveHealIncident(ctx, checkID, actionID); err != nil {
			return
		}
	case outcomeFailure:
		policy, ok := r.getAutoHealPolicy()
		if !ok || tracker.ConsecutiveFailures < policy.MaxRestartAttempts {
			return
		}
		lastError := actionResult.Error
		if lastError == "" {
			lastError = actionResult.Message
		}
		_ = reporter.OpenHealIncident(ctx, checkID, actionID, lastError, tracker.ConsecutiveFailures)
	}
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

	// A stopped scenario has a safe, idempotent recovery path. Prefer it over
	// restart: restart is intentionally subject to the runtime recovery
	// ownership gate, while starting a process that is already known to be down
	// cannot race a running-process recovery controller.
	if strings.HasPrefix(checkID, "scenario-") && result.Details != nil {
		if status, ok := result.Details["scenarioStatus"].(string); ok && strings.EqualFold(status, "stopped") {
			for _, action := range actions {
				if action.Available && action.ID == "start" {
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
