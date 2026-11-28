package checks

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/coreset"
)

type AutoHealResult struct {
	CheckID             string        `json:"checkId"`
	Attempted           bool          `json:"attempted"`
	ActionResult        ActionResult  `json:"actionResult,omitempty"`
	TimedOut            bool          `json:"timedOut,omitempty"`          // Mirrors ActionResult.TimedOut for callers that only inspect AutoHealResult.
	Reason              string        `json:"reason,omitempty"`            // Why it wasn't attempted
	CooldownRemaining   time.Duration `json:"cooldownRemaining,omitempty"` // Time until next attempt allowed
	ConsecutiveFailures int           `json:"consecutiveFailures,omitempty"`
	Suspended           bool          `json:"suspended,omitempty"`
	SuspensionReason    string        `json:"suspensionReason,omitempty"`
	RecoveryRequestID   string        `json:"recoveryRequestId,omitempty"`
	SkipClass           string        `json:"skipClass,omitempty"` // Why it wasn't attempted, as one of the Skip* classes
	Started             bool          `json:"started,omitempty"`   // The heal was handed to the background and has not finished
}

// Skip classes name why a heal did not run. A skip is an outcome, never a
// failed heal: measures count it apart from attempts that ran and failed.
// Gate and hold classes (GateClass*) are also skip classes.
const (
	SkipNotEnabled   = "not-enabled"
	SkipNoAction     = "no-action"
	SkipSuspended    = "suspended"
	SkipCooldown     = "cooldown"
	SkipNotConfirmed = "not-confirmed"
	SkipRecovered    = "recovered"
	SkipAgentOwned   = "agent-owned"
	SkipInFlight     = "in-flight"
)

// healCandidate represents a check that needs healing with its metadata
type healCandidate struct {
	result         Result
	healable       HealableCheck
	selectedAction *RecoveryAction
	priority       int // lower = higher priority
	detectedPID    int // PID at detection time (0 if unknown); used to prevent killing a new process
}

// getHealPriority returns priority for a check (lower = more important).
// Order: the core recovery plane (0) > resources (1) > other scenarios (2) >
// everything else (3). The per-tick action cap used to cut web-console behind
// resource-whisper starts; the cap now only ever cuts non-core work.
func getHealPriority(checkID string) int {
	switch {
	case coreRecoveryCheck(checkID):
		return 0
	case strings.HasPrefix(checkID, "resource-"):
		return 1 // Resources (postgres, redis, etc.) are infrastructure
	case strings.HasPrefix(checkID, "scenario-"):
		return 2 // Scenarios depend on API and resources
	default:
		return 3 // Unknown checks get lowest priority
	}
}

// RunAutoHeal plans the heals for the failing checks and runs them to
// completion before returning: the synchronous form, for callers that need the
// outcome in hand. The tick uses StartAutoHeal instead, so a slow restart
// cannot hold up the next observation of the core.
func (r *Registry) RunAutoHeal(ctx context.Context, results []Result) []AutoHealResult {
	planned, jobs := r.planAutoHeal(ctx, results)
	r.executeHealJobs(ctx, jobs, func(result AutoHealResult) { planned = append(planned, result) })
	return planned
}

// StartAutoHeal plans the heals and runs them in the background, one goroutine
// per call. It returns at once with every skip and a "started" entry per heal;
// done receives each heal's result when it finishes. Before this, heals ran
// inside the tick one after another (a restart is 45-77 s), so a tick took up
// to 2 minutes and the core went unobserved for 592 s. A check with a heal in
// flight is skipped by the next plan, so an overlapping tick never doubles a
// restart.
func (r *Registry) StartAutoHeal(ctx context.Context, results []Result, done func(AutoHealResult)) []AutoHealResult {
	planned, jobs := r.planAutoHeal(ctx, results)
	for _, job := range jobs {
		planned = append(planned, AutoHealResult{CheckID: job.result.CheckID, Started: true,
			Reason: "heal " + job.selectedAction.ID + " started in the background; its result is recorded in action history"})
	}
	if len(jobs) > 0 {
		r.healWG.Add(1)
		go func() {
			defer r.healWG.Done()
			roundCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), HealRoundTimeout)
			defer cancel()
			r.executeHealJobs(roundCtx, jobs, done)
		}()
	}
	return planned
}

// WaitForHeals blocks until every background heal has finished (shutdown, tests).
func (r *Registry) WaitForHeals() { r.healWG.Wait() }

func (r *Registry) healInFlight(checkID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, running := r.healing[checkID]
	return running
}

// planAutoHeal selects, orders and caps the heals for this tick and marks each
// as in flight. Everything it skips is returned with its class.
func (r *Registry) planAutoHeal(ctx context.Context, results []Result) ([]AutoHealResult, []healCandidate) {
	autoHealResults := make([]AutoHealResult, 0)
	candidates := make([]healCandidate, 0)
	now := r.now()
	_, hasPolicy := r.getAutoHealPolicy()
	r.refreshRecoveryOwnership(ctx)

	// Phase 1: Collect and filter candidates
	for _, result := range results {
		r.observeHealth(result)
		r.releaseHold(ctx, result)
		// Lift a stale suspension for every failing check, including ones whose
		// severity never triggers a heal: a suspension must not outlive its bound.
		r.expireStaleSuspension(result.CheckID, now)
		if admission, pending := r.nativeRecoveryPending(result.CheckID); pending && r.shouldTriggerAutoHeal(result) {
			autoHealResults = append(autoHealResults, pendingRecoveryResult(result.CheckID, admission))
			continue
		}
		if admission := r.requestScenarioRecoveryIfReady(ctx, result); admission.RequestID != "" {
			r.retainRecoveryAdmission(result.CheckID, admission)
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:           result.CheckID,
				Attempted:         false,
				Reason:            "coding-agent recovery receipt retained: " + admission.Outcome,
				SkipClass:         SkipAgentOwned,
				RecoveryRequestID: admission.RequestID,
			})
			if admission.NativePending {
				continue
			}
		}
		// Auto-heal trigger is per-check policy (critical or warning+critical).
		if !r.shouldTriggerAutoHeal(result) {
			continue
		}

		if !hasPolicy {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "auto-heal policy not configured",
				SkipClass: SkipNotEnabled,
			})
			continue
		}

		// Check if auto-heal is enabled for this check
		if !r.IsAutoHealEnabled(result.CheckID) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:   result.CheckID,
				Attempted: false,
				Reason:    "auto-heal not enabled for this check" + repairOwnerNote(result),
				SkipClass: SkipNotEnabled,
			})
			continue
		}

		if r.healInFlight(result.CheckID) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID: result.CheckID, Attempted: false, SkipClass: SkipInFlight,
				Reason: "a heal for this check is already running",
			})
			continue
		}

		// Check cooldown before attempting heal.
		// Use a snapshot to avoid races with concurrent tracker updates.
		tracker := r.getHealTrackerSnapshot(result.CheckID)
		if tracker.IsSuspended() {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             result.CheckID,
				Attempted:           false,
				Reason:              "auto-heal suspended: " + tracker.SuspensionReason,
				SkipClass:           SkipSuspended,
				ConsecutiveFailures: tracker.ConsecutiveFailures,
				Suspended:           true,
				SuspensionReason:    tracker.SuspensionReason,
			})
			continue
		}
		if tracker.IsInCooldownAt(now) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID:             result.CheckID,
				Attempted:           false,
				Reason:              fmt.Sprintf("in cooldown (%.0fs remaining)", tracker.CooldownRemainingAt(now).Seconds()),
				SkipClass:           SkipCooldown,
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
				Reason:    "check does not support healing" + repairOwnerNote(result),
				SkipClass: SkipNoAction,
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
				Reason:    "no auto-heal recovery action available" + repairOwnerNote(result),
				SkipClass: SkipNoAction,
			})
			continue
		}
		if strings.HasPrefix(result.CheckID, "scenario-") && selectedAction.ID != "start" && !r.failureConfirmed(result.CheckID) {
			autoHealResults = append(autoHealResults, AutoHealResult{
				CheckID: result.CheckID, Attempted: false, SkipClass: SkipNotConfirmed,
				Reason: fmt.Sprintf("failure not yet confirmed: a scenario restart needs %d consecutive failed probes at least %s apart", r.confirmProbes, r.confirmSpan),
			})
			continue
		}
		if _, gatedAction := gatedActionIDs[selectedAction.ID]; gatedAction {
			if allowed, class, reason := r.allowsGatedHeal(ctx, result.CheckID, selectedAction.ID); !allowed {
				autoHealResults = append(autoHealResults, AutoHealResult{
					CheckID: result.CheckID, Attempted: false, SkipClass: class, Reason: reason,
				})
				continue
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
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].priority < candidates[j].priority })

	// Phase 3: Limit number of actions. Heals still running from earlier ticks
	// use slots, and a candidate cut by the cap is a held restart, not a quiet
	// skip: it is retried next tick, and held past TickCapCriticalAfter it turns
	// the check critical and opens an incident. The core is never cut: a slot is
	// always found for every core candidate.
	limit := MaxAutoHealActionsPerTick - r.healInFlightCount()
	for core := 0; core < len(candidates) && candidates[core].priority == 0; core++ {
		limit = max(limit, core+1)
	}
	if len(candidates) > limit {
		for _, c := range candidates[limit:] {
			reason := fmt.Sprintf("deferred to next tick (max %d heals at once, lower priority)", MaxAutoHealActionsPerTick)
			hold := r.noteHold(c.result.CheckID, GateVerdict{Class: GateClassTickCap, Reason: reason, CriticalAfter: TickCapCriticalAfter}, now)
			if r.now().Sub(hold.Since) >= hold.CriticalAfter {
				r.openHoldIncident(ctx, c.result.CheckID, hold)
			}
			autoHealResults = append(autoHealResults, AutoHealResult{CheckID: c.result.CheckID, Attempted: false, SkipClass: GateClassTickCap, Reason: reason})
		}
		candidates = candidates[:limit]
	}
	r.mu.Lock()
	if r.healing == nil {
		r.healing = make(map[string]struct{})
	}
	for _, c := range candidates {
		r.healing[c.result.CheckID] = struct{}{}
	}
	r.mu.Unlock()
	return autoHealResults, candidates
}

func (r *Registry) healInFlightCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.healing)
}

// executeHealJobs runs the planned heals one after another, each with its own
// action timeout, and reports every result through emit. Before each action it
// re-runs the health check to guard against TOCTOU races (a heal planned at
// T=0 whose target is healthy by T=90 s) and verifies the process PID has not
// changed, which means a new process replaced the unhealthy one.
func (r *Registry) executeHealJobs(ctx context.Context, candidates []healCandidate, emit func(AutoHealResult)) {
	for _, c := range candidates {
		r.executeHealJob(ctx, c, emit)
		r.mu.Lock()
		delete(r.healing, c.result.CheckID)
		r.mu.Unlock()
	}
}

// executeHealJob runs one planned heal.
func (r *Registry) executeHealJob(ctx context.Context, c healCandidate, emit func(AutoHealResult)) {
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
		r.recordHealIncident(ctx, c.result.CheckID, c.selectedAction.ID, actionResult, outcomeFailure, updatedTracker)
		emit(AutoHealResult{
			CheckID:             c.result.CheckID,
			Attempted:           true,
			ActionResult:        actionResult,
			CooldownRemaining:   updatedTracker.CooldownRemainingAt(r.now()),
			ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
			Suspended:           updatedTracker.IsSuspended(),
			SuspensionReason:    updatedTracker.SuspensionReason,
		})
		return
	default:
	}

	// Pre-heal recheck: re-run the health check to verify the target is
	// still unhealthy. This prevents the TOCTOU race where detection happens
	// long before execution (e.g., due to SQLITE_BUSY delays or earlier
	// heal actions taking time).
	if skipped := r.preHealRecheck(ctx, c, emit); skipped {
		return
	}
	// A native investigation may have been admitted after collection or by
	// another requester. Refresh before the mechanical effect, independently
	// of failure streaks and cooldowns (which healthy observations can reset).
	r.refreshRecoveryOwnership(ctx)
	if admission, pending := r.nativeRecoveryPending(c.result.CheckID); pending {
		emit(pendingRecoveryResult(c.result.CheckID, admission))
		return
	}
	// The owner can enter coordinated maintenance while this tick is
	// rechecking health. Refresh the same gate immediately before invoking
	// the action so a stale phase-one observation cannot race the lifecycle.
	if _, gatedAction := gatedActionIDs[c.selectedAction.ID]; gatedAction {
		if allowed, class, reason := r.allowsGatedHeal(ctx, c.result.CheckID, c.selectedAction.ID); !allowed {
			emit(AutoHealResult{
				CheckID: c.result.CheckID, Attempted: false, SkipClass: class, Reason: reason,
			})
			return
		}
	}

	actionCtx, cancel := context.WithTimeout(ctx, r.actionTimeoutFor(c.selectedAction.ID))
	actionResult := r.executeAutoHealActionWithTimeout(actionCtx, c)
	cancel()

	// A heal refused by the root-update fence never ran. It is not a failed
	// attempt (no failure count, cooldown or suspension): the check stays
	// failing, the refusal is held on it, and the next tick retries.
	if actionResult.Refusal != nil && actionResult.Refusal.Kind == UpdateFenceRefusalKind {
		reason := actionResult.Refusal.Reason
		hold := r.noteHold(c.result.CheckID, GateVerdict{Class: GateClassUpdateFence, Reason: reason, CriticalAfter: UpdateFenceCriticalAfter}, r.now())
		if r.now().Sub(hold.Since) >= hold.CriticalAfter {
			r.openHoldIncident(ctx, c.result.CheckID, hold)
		}
		emit(AutoHealResult{CheckID: c.result.CheckID, Attempted: false, SkipClass: GateClassUpdateFence, Reason: reason})
		return
	}
	r.deactivateHold(c.result.CheckID)

	// Update heal tracker based on result
	outcome := outcomeFromActionResult(actionResult)
	r.updateHealTracker(c.result.CheckID, outcome)

	// Get updated tracker for result
	updatedTracker := r.getHealTrackerSnapshot(c.result.CheckID)
	r.recordHealIncident(ctx, c.result.CheckID, c.selectedAction.ID, actionResult, outcome, updatedTracker)
	emit(AutoHealResult{
		CheckID:             c.result.CheckID,
		Attempted:           true,
		ActionResult:        actionResult,
		TimedOut:            actionResult.TimedOut,
		CooldownRemaining:   updatedTracker.CooldownRemainingAt(r.now()),
		ConsecutiveFailures: updatedTracker.ConsecutiveFailures,
		Suspended:           updatedTracker.IsSuspended(),
		SuspensionReason:    updatedTracker.SuspensionReason,
	})
}

// allowsGatedHeal asks the recovery gate. A denial is a bounded deferral: it is
// recorded on the check (overlayHealState), becomes critical and opens an
// incident once held past the verdict's bound, and is overridden after that
// bound when the verdict allows it. It is never a quiet skip.
func (r *Registry) allowsGatedHeal(ctx context.Context, checkID, actionID string) (allowed bool, class, reason string) {
	gate := r.recoveryOwnershipGate()
	if gate == nil {
		return true, "", ""
	}
	verdict := gate.AllowsAutoHealRestart(ctx, checkID, actionID)
	if verdict.Allowed {
		r.deactivateHold(checkID)
		return true, "", ""
	}
	hold := r.noteHold(checkID, verdict, r.now())
	held := r.now().Sub(hold.Since)
	if held >= verdict.CriticalAfter {
		r.openHoldIncident(ctx, checkID, hold)
	}
	if verdict.OverrideAfter > 0 && held >= verdict.OverrideAfter {
		r.markHoldOverridden(checkID)
		return true, "", ""
	}
	return false, verdict.Class, verdict.Reason
}

func pendingRecoveryResult(checkID string, admission RecoveryAdmission) AutoHealResult {
	return AutoHealResult{CheckID: checkID, Attempted: false, RecoveryRequestID: admission.RequestID, SkipClass: SkipAgentOwned,
		Reason: "native coding-agent recovery owns this target until exact execution exit: " + admission.Outcome}
}

func (r *Registry) requestScenarioRecoveryIfReady(ctx context.Context, result Result) RecoveryAdmission {
	if !r.shouldTriggerAutoHeal(result) {
		return RecoveryAdmission{}
	}
	if !strings.HasPrefix(result.CheckID, "scenario-") {
		return RecoveryAdmission{}
	}
	if !r.IsAutoHealEnabled(result.CheckID) {
		return RecoveryAdmission{}
	}
	policy, ok := r.getAutoHealPolicy()
	if !ok {
		return RecoveryAdmission{}
	}
	tracker := r.getHealTrackerSnapshot(result.CheckID)
	if tracker.ConsecutiveFailures < policy.MaxRestartAttempts || !r.recoveryGraceElapsed(result.CheckID, r.now()) {
		return RecoveryAdmission{}
	}
	requester := r.getRecoveryRequester()
	if requester == nil {
		return RecoveryAdmission{}
	}
	// Agent investigations consume host resources and can restart services.
	// Escalation must not escape pressure, startup or maintenance coordination.
	if allowed, _, _ := r.allowsGatedHeal(ctx, result.CheckID, "restart"); !allowed {
		return RecoveryAdmission{}
	}
	scenario := strings.TrimPrefix(result.CheckID, "scenario-")
	reason := result.Message
	if reason == "" {
		reason = "scenario health check remains unhealthy"
	}
	admission, err := requester(ctx, scenario, fmt.Sprintf("mechanical autoheal exhausted after continuous unhealthy period: %s", reason))
	if err != nil {
		log.Printf("vrooli-autoheal: control-plane coding-agent recovery request for %s failed: %v", scenario, err)
		// An ambiguous dispatch can still return its admitted identity. Keep
		// that ownership instead of immediately racing it with another repair.
		return admission
	}
	return admission
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
		resultCh <- r.executeGuardedAction(ctx, candidate.healable, *candidate.selectedAction)
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
	if _, isLongRunning := longRunningActionIDs[actionID]; isLongRunning {
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
func (r *Registry) preHealRecheck(ctx context.Context, c healCandidate, emit func(AutoHealResult)) bool {
	// Re-run the check to see if it's still unhealthy.
	r.mu.RLock()
	check, exists := r.checks[c.result.CheckID]
	r.mu.RUnlock()
	if !exists {
		return false // Check was unregistered; let the caller handle it
	}
	recheckCtx, cancel := context.WithTimeout(ctx, timeoutForCheck(check))
	defer cancel()

	freshResult := check.Run(recheckCtx)

	// If the check is now healthy, skip the heal action.
	if freshResult.Status == StatusOK {
		emit(AutoHealResult{
			CheckID:   c.result.CheckID,
			Attempted: false,
			Reason:    "pre-heal recheck passed: target is now healthy",
			SkipClass: SkipRecovered,
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
			emit(AutoHealResult{
				CheckID:   c.result.CheckID,
				Attempted: false,
				SkipClass: SkipRecovered,
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
	policy := AutoHealOnCriticalSignature
	if r.config != nil {
		if configured := r.config.GetAutoHealOn(result.CheckID); configured != "" {
			policy = configured
		}
	}

	status := effectiveStatus(result)
	switch policy {
	case AutoHealOnWarningCritical:
		return status == StatusWarning || status == StatusCritical
	case AutoHealOnCriticalSignature:
		// Critical always, plus warnings the check has positively classified
		// with a specific root cause and a targeted recovery action.
		//
		// This exists because the two states are not comparable. A bare
		// warning means "something looks off" and healing it is a guess. A
		// signature-backed warning means the check identified the failure --
		// a dead UI port, a go.sum drift signature -- and named the action
		// that repairs it. Widening all the way to "warning+critical" would
		// let every soft signal restart a scenario; this widens only to the
		// warnings that carry evidence.
		if status == StatusCritical {
			return true
		}
		return status == StatusWarning && hasHealableDegradation(result)
	default:
		return status == StatusCritical
	}
}

// Auto-heal trigger policies, in increasing order of breadth.
const (
	// AutoHealOnCritical heals only critical results.
	AutoHealOnCritical = "critical"
	// AutoHealOnCriticalSignature heals critical results plus warnings that
	// carry an identified, healable root cause. This is the default.
	AutoHealOnCriticalSignature = "critical+signature"
	// AutoHealOnWarningCritical heals every warning and critical result.
	AutoHealOnWarningCritical = "warning+critical"
)

// hasHealableDegradation reports whether a check positively classified this
// warning as a degradation it knows how to repair. Requires BOTH the explicit
// marker and a named recommended action, so a check cannot opt into
// unattended healing without also saying what to run.
func hasHealableDegradation(result Result) bool {
	if result.Details == nil {
		return false
	}
	healable, ok := result.Details["healableDegradation"].(bool)
	if !ok || !healable {
		return false
	}
	action, ok := result.Details["recommendedAction"].(string)
	return ok && action != ""
}

func selectAutoHealAction(result Result, actions []RecoveryAction) *RecoveryAction {
	// A successful diagnostic is not a successful repair. Apply this boundary
	// before recommended-action and specialized selection paths as well.
	automatic := make([]RecoveryAction, 0, len(actions))
	for _, action := range actions {
		if !action.ManualOnly {
			automatic = append(automatic, action)
		}
	}
	actions = automatic
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

// observeHealth tracks continuous unhealthy time independently from action
// cooldowns, and clears a suspension the moment the check is healthy again, for
// every check. A restart attempt can fail while the target remains unhealthy;
// the grace window must survive those attempts and be persisted so an autoheal
// restart does not reset the escalation clock.
func (r *Registry) observeHealth(result Result) {
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
	now := result.Timestamp
	if now.IsZero() {
		now = r.now()
	}
	r.observeProbe(result.CheckID, result.Status == StatusOK || result.Status == StatusNotApplicable, now)
	r.mu.Lock()
	tracker, exists := r.healTrackers[result.CheckID]
	if !exists {
		tracker = &HealTracker{}
		r.healTrackers[result.CheckID] = tracker
	}
	changed := false
	if result.Status == StatusOK || result.Status == StatusNotApplicable {
		if !tracker.UnhealthySince.IsZero() || tracker.ConsecutiveFailures > 0 || tracker.ConsecutiveTimeouts > 0 || tracker.IsSuspended() {
			tracker.UnhealthySince = time.Time{}
			tracker.ConsecutiveFailures = 0
			tracker.ConsecutiveTimeouts = 0
			tracker.SuspendedAt = time.Time{}
			tracker.SuspensionReason = ""
			tracker.Disposition = HealDispositionHealed
			tracker.DispositionAt = now
			tracker.CooldownUntil = now
			changed = true
		}
	} else if tracker.UnhealthySince.IsZero() {
		tracker.UnhealthySince = now
		changed = true
	}
	store := r.healTrackerStore
	copy := *tracker
	r.mu.Unlock()
	if changed {
		persistTrackerSnapshot(store, result.CheckID, copy)
	}
}

func (r *Registry) recoveryGraceElapsed(checkID string, now time.Time) bool {
	r.mu.RLock()
	grace := r.recoveryGracePeriod
	tracker, exists := r.healTrackers[checkID]
	var unhealthySince time.Time
	if exists && tracker != nil {
		unhealthySince = tracker.UnhealthySince
	}
	r.mu.RUnlock()
	if grace <= 0 || !exists || unhealthySince.IsZero() {
		return grace <= 0
	}
	return !now.Before(unhealthySince.Add(grace))
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
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
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
		tracker.SuspendedAt = time.Time{}
		tracker.SuspensionReason = ""
		tracker.Disposition = HealDispositionHealed
		tracker.DispositionAt = now
		tracker.SuccessHistory = HealSuccessPrevious
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
		if tracker.LastSuccess.IsZero() {
			tracker.SuccessHistory = HealSuccessNever
		} else {
			tracker.SuccessHistory = HealSuccessPrevious
		}
		tracker.ConsecutiveTimeouts = 0
		if policy != nil {
			cooldown := policy.CalculateFailureCooldown(tracker.ConsecutiveFailures)
			tracker.CooldownUntil = now.Add(cooldown)
		} else {
			tracker.CooldownUntil = now
		}
	}
	if policy != nil && tracker.ConsecutiveFailures >= policy.MaxRestartAttempts && outcome != outcomeSuccess && !r.persistentRecoveryCheckLocked(checkID) {
		tracker.SuspendedAt = now
		tracker.SuspensionReason = fmt.Sprintf("reached consecutive failure limit (%d)", policy.MaxRestartAttempts)
		tracker.Disposition = HealDispositionEscalated
		tracker.DispositionAt = now
	}

	store := r.healTrackerStore
	trackerCopy := *tracker
	r.mu.Unlock()
	persistTrackerSnapshot(store, checkID, trackerCopy)
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

func (r *Registry) GetSuspendedHealTrackers() map[string]HealTracker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]HealTracker)
	for id, tracker := range r.healTrackers {
		if tracker.IsSuspended() {
			result[id] = *tracker
		}
	}
	return result
}

// ResumeHealTracker clears only the bounded-retry suspension and failure
// streak. Lifetime counters and the escalated disposition remain as evidence.
func (r *Registry) ResumeHealTracker(checkID string) bool {
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
	r.mu.Lock()
	tracker, exists := r.healTrackers[checkID]
	if !exists || !tracker.IsSuspended() {
		r.mu.Unlock()
		return false
	}
	tracker.SuspendedAt = time.Time{}
	tracker.SuspensionReason = ""
	tracker.ConsecutiveFailures = 0
	tracker.ConsecutiveTimeouts = 0
	tracker.CooldownUntil = r.clock.Now()
	store := r.healTrackerStore
	copy := *tracker
	r.mu.Unlock()
	persistTrackerSnapshot(store, checkID, copy)
	return true
}

// ReconcileHealTrackerDispositions classifies restored legacy trackers after
// canonical checks are registered. Missing checks are retired; registered
// checks already beyond the configured bound are escalated and suspended.
func (r *Registry) ReconcileHealTrackerDispositions(ctx context.Context) error {
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
	r.mu.Lock()
	if r.autoHealPolicy == nil {
		r.mu.Unlock()
		return fmt.Errorf("auto-heal policy not configured")
	}
	now := r.clock.Now()
	changed := make(map[string]HealTracker)
	escalated := make(map[string]HealTracker)
	for id, tracker := range r.healTrackers {
		if _, registered := r.checks[id]; !registered {
			if tracker.Disposition != HealDispositionRetired {
				tracker.Disposition = HealDispositionRetired
				tracker.DispositionAt = now
				tracker.SuspendedAt = time.Time{}
				tracker.SuspensionReason = ""
				changed[id] = *tracker
			}
			continue
		}
		if r.persistentRecoveryCheckLocked(id) {
			// The configured core set is a durable recovery obligation. Its
			// retry cadence is bounded by the normal cooldown, capped at one
			// hour, but an old failure streak must never permanently disable
			// the only mechanism that can restore the recovery plane.
			if tracker.IsSuspended() {
				tracker.SuspendedAt = time.Time{}
				tracker.SuspensionReason = ""
				tracker.CooldownUntil = now
				changed[id] = *tracker
			}
			continue
		}
		if tracker.ConsecutiveFailures >= r.autoHealPolicy.MaxRestartAttempts && !tracker.IsSuspended() {
			tracker.SuspendedAt = now
			tracker.SuspensionReason = fmt.Sprintf("restored above consecutive failure limit (%d)", r.autoHealPolicy.MaxRestartAttempts)
			tracker.Disposition = HealDispositionEscalated
			tracker.DispositionAt = now
			changed[id] = *tracker
			escalated[id] = *tracker
		}
	}
	store := r.healTrackerStore
	reporter := r.healIncidentReporter
	r.mu.Unlock()
	if batchStore, ok := store.(interface {
		SaveHealTrackers(context.Context, map[string]HealTracker) error
	}); ok {
		if err := batchStore.SaveHealTrackers(ctx, changed); err != nil {
			return fmt.Errorf("persist heal tracker dispositions: %w", err)
		}
	} else if store != nil {
		for id, tracker := range changed {
			copy := tracker
			if err := store.SaveHealTracker(ctx, id, &copy); err != nil {
				return fmt.Errorf("persist heal tracker disposition %s: %w", id, err)
			}
		}
	}
	if reporter != nil {
		for id, tracker := range escalated {
			if err := reporter.OpenHealIncident(ctx, id, "auto-heal", tracker.SuspensionReason, tracker.ConsecutiveFailures); err != nil {
				return fmt.Errorf("open restored heal incident %s: %w", id, err)
			}
		}
	}
	return nil
}

// persistentRecoveryCheck identifies the recovery plane that must keep
// retrying after repeated failures. Optional checks retain bounded suspension
// so a broken optional scenario cannot consume the whole recovery budget. The
// operator's configured core seed is authoritative for scenario membership;
// the directly coupled access/model resources remain protected with it.
func (r *Registry) persistentRecoveryCheck(checkID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.persistentRecoveryCheckLocked(checkID)
}

func (r *Registry) persistentRecoveryCheckLocked(checkID string) bool {
	checkID = strings.TrimSpace(checkID)
	if coreRecoveryCheck(checkID) {
		return true
	}
	_, supervised := r.supervisedChecks[checkID]
	return supervised && strings.HasPrefix(checkID, "scenario-")
}

// coreRecoveryCheck is the operator's core: the declared core seed (which
// always includes the required recovery plane) and the access/model resources
// coupled to it. It is the set healed first and checked first.
func coreRecoveryCheck(checkID string) bool {
	checkID = strings.TrimSpace(checkID)
	switch checkID {
	case "infra-cloudflared", "resource-cloudflared", "resource-openrouter", "resource-opencode":
		return true
	}
	scenario, ok := strings.CutPrefix(checkID, "scenario-")
	if !ok {
		return false
	}
	return slices.Contains(coreset.RequiredOperationalScenarios, scenario) || coreset.IsCoreSeed(scenario)
}

// ResetHealTracker resets the heal tracker for a check (for manual intervention)
func (r *Registry) ResetHealTracker(checkID string) {
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
	r.mu.Lock()
	delete(r.healTrackers, checkID)
	store := r.healTrackerStore
	r.mu.Unlock()

	if store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.DeleteHealTracker(ctx, checkID); err != nil {
			log.Printf("vrooli-autoheal: reset tracker persistence failed for %s: %v", checkID, err)
		}
	}
}
