package checks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Quiet failure is a defect (RULES.md R6): a suspension or a held restart on a
// failing check is carried on the check result itself, so it reaches the
// status summary, the incident pipeline and the steward without any consumer
// having to read a second list.

// SuspensionMaxAge bounds a suspension. After it, the check gets one more
// heal attempt; a failure re-suspends with a fresh timestamp.
const SuspensionMaxAge = 24 * time.Hour

// Gate classes name why a restart is held. They are also the reason class the
// steward reports.
const (
	GateClassEpoch       = "pressure-epoch"
	GateClassStarting    = "owner-starting"
	GateClassMaintenance = "owner-maintenance"
	GateClassUnreadable  = "gate-unreadable"
	GateClassSchema      = "registry-schema"
	// GateClassUpdateFence: the root-update initialization fence refused the
	// restart. GateClassTickCap: the per-tick action cap cut it. Both are
	// retried every tick and turn critical when held too long.
	GateClassUpdateFence = "update-fence"
	GateClassTickCap     = "tick-cap"
)

const (
	UpdateFenceRefusalKind   = "root-update-fence"
	UpdateFenceCriticalAfter = 15 * time.Minute
	TickCapCriticalAfter     = 10 * time.Minute
	// updateFenceMarker is the stable prefix `vrooli` prints when the root
	// update initialization fence refuses a command (cmd/vrooli/activation_guard.go).
	updateFenceMarker = "root update initialization fence"
)

// asUpdateFenceRefusal turns a failed action whose output names the root-update
// fence into a typed refusal: the action did not run, which is not a failure of
// the heal itself.
func asUpdateFenceRefusal(result ActionResult, target HealTarget) ActionResult {
	if result.Success || result.Refusal != nil {
		return result
	}
	if !strings.Contains(result.Output, updateFenceMarker) && !strings.Contains(result.Error, updateFenceMarker) {
		return result
	}
	result.Refusal = &ActionRefusal{
		Kind:              UpdateFenceRefusalKind,
		Reason:            fmt.Sprintf("%s refused by the root update initialization fence; retried every tick until it clears", result.ActionID),
		Target:            target,
		AttemptingCheckID: result.CheckID,
		SourceActionID:    result.ActionID,
		SourceTimestamp:   result.Timestamp,
	}
	return result
}

// GateVerdict is a recovery gate's answer. A denial is a deferral, never a
// permanent no: after CriticalAfter of continuous denial the check turns
// critical and opens an incident, and after OverrideAfter (zero: never) the
// restart proceeds anyway with the reason kept on the check. Only an owner's
// explicit closed maintenance fence never overrides.
type GateVerdict struct {
	Allowed       bool
	Class         string
	Reason        string
	CriticalAfter time.Duration
	OverrideAfter time.Duration
}

// GateAllowed is the verdict for an open gate.
var GateAllowed = GateVerdict{Allowed: true}

// healHold is one check's continuous gate denial.
type healHold struct {
	Since         time.Time
	Class         string
	Reason        string
	CriticalAfter time.Duration
	OverrideAfter time.Duration
	Active        bool
	Overridden    bool
	IncidentOpen  bool
}

// A scenario restart is never justified by one probe: a single timeout right
// after autoheal's own start, on a loaded host, took a healthy event bus down
// into a stuck restart. Restart-class repair of a scenario needs at least
// ConfirmProbes consecutive failed probes spanning ConfirmSpan. The streak lives
// in memory only, so a fresh autoheal process must observe the failure itself.
const (
	ConfirmProbes = 2
	ConfirmSpan   = 30 * time.Second
)

type probeStreak struct {
	Count       int
	First, Last time.Time
}

func (r *Registry) observeProbe(checkID string, healthy bool, at time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.streaks == nil {
		r.streaks = make(map[string]*probeStreak)
	}
	if healthy {
		delete(r.streaks, checkID)
		return
	}
	streak := r.streaks[checkID]
	if streak == nil {
		streak = &probeStreak{First: at}
		r.streaks[checkID] = streak
	}
	streak.Count++
	streak.Last = at
}

// failureConfirmed reports whether a check has failed on ConfirmProbes
// consecutive probes spanning at least ConfirmSpan, with the evidence for the
// skip reason when it has not.
func (r *Registry) failureConfirmed(checkID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	streak := r.streaks[checkID]
	return streak != nil && streak.Count >= r.confirmProbes && streak.Last.Sub(streak.First) >= r.confirmSpan
}

// SetFailureConfirmation overrides the default confirmation rule; tests that
// exercise unrelated heal behavior use (1, 0) to act on the first failed probe.
func (r *Registry) SetFailureConfirmation(probes int, span time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.confirmProbes, r.confirmSpan = probes, span
}

const holdIncidentAction = "gate-hold"

func (r *Registry) noteHold(checkID string, v GateVerdict, now time.Time) healHold {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.holds == nil {
		r.holds = make(map[string]*healHold)
	}
	hold := r.holds[checkID]
	if hold == nil || !hold.Active {
		open := hold != nil && hold.IncidentOpen
		hold = &healHold{Since: now, IncidentOpen: open}
		r.holds[checkID] = hold
	}
	hold.Active = true
	hold.Class, hold.Reason, hold.CriticalAfter, hold.OverrideAfter = v.Class, v.Reason, v.CriticalAfter, v.OverrideAfter
	return *hold
}

func (r *Registry) markHoldOverridden(checkID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if hold := r.holds[checkID]; hold != nil {
		hold.Overridden = true
	}
}

func (r *Registry) deactivateHold(checkID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if hold := r.holds[checkID]; hold != nil {
		hold.Active = false
		hold.Overridden = false
	}
}

func (r *Registry) openHoldIncident(ctx context.Context, checkID string, hold healHold) {
	r.mu.Lock()
	h := r.holds[checkID]
	if h == nil || h.IncidentOpen {
		r.mu.Unlock()
		return
	}
	h.IncidentOpen = true
	r.mu.Unlock()
	if reporter := r.getHealIncidentReporter(); reporter != nil {
		_ = reporter.OpenHealIncident(ctx, checkID, holdIncidentAction, fmt.Sprintf("%s: %s", hold.Class, hold.Reason), 0)
	}
}

// releaseHold ends a check's hold once the check is healthy again.
func (r *Registry) releaseHold(ctx context.Context, result Result) {
	if result.Status != StatusOK && result.Status != StatusNotApplicable {
		return
	}
	r.mu.Lock()
	hold := r.holds[result.CheckID]
	delete(r.holds, result.CheckID)
	r.mu.Unlock()
	if hold != nil && hold.IncidentOpen {
		if reporter := r.getHealIncidentReporter(); reporter != nil {
			_ = reporter.ResolveHealIncident(ctx, result.CheckID, holdIncidentAction)
		}
	}
}

// expireStaleSuspension lifts a suspension older than SuspensionMaxAge into one
// more attempt. The failure streak stays one below the limit, so a further
// failure re-suspends with a fresh timestamp and a healthy result clears it.
func (r *Registry) expireStaleSuspension(checkID string, now time.Time) bool {
	r.trackerWriteMu.Lock()
	defer r.trackerWriteMu.Unlock()
	r.mu.Lock()
	tracker := r.healTrackers[checkID]
	if tracker == nil || !tracker.IsSuspended() || now.Sub(tracker.SuspendedAt) < SuspensionMaxAge || r.autoHealPolicy == nil {
		r.mu.Unlock()
		return false
	}
	log.Printf("vrooli-autoheal: suspension of %s is older than %s; re-probing", checkID, SuspensionMaxAge)
	tracker.SuspendedAt = time.Time{}
	tracker.SuspensionReason = ""
	tracker.ConsecutiveFailures = r.autoHealPolicy.MaxRestartAttempts - 1
	tracker.CooldownUntil = now
	store := r.healTrackerStore
	snapshot := *tracker
	r.mu.Unlock()
	persistTrackerSnapshot(store, checkID, snapshot)
	return true
}

// overlayHealState annotates a failing result with the heal state that
// otherwise hides in logs: a suspension, or a restart held by a gate. A hold
// that has lasted past its bound makes the check critical; the unannotated
// status stays in Details so heal triggering does not change with the overlay.
func (r *Registry) overlayHealState(result Result, now time.Time) Result {
	if result.Status == StatusOK || result.Status == StatusNotApplicable {
		return result
	}
	r.mu.RLock()
	var tracker HealTracker
	if t := r.healTrackers[result.CheckID]; t != nil {
		tracker = *t
	}
	var hold healHold
	if h := r.holds[result.CheckID]; h != nil {
		hold = *h
	}
	r.mu.RUnlock()
	if !tracker.IsSuspended() && !hold.Active {
		return result
	}
	details := make(map[string]interface{}, len(result.Details)+8)
	for k, v := range result.Details {
		details[k] = v
	}
	details["statusBeforeOverlay"] = string(result.Status)
	if tracker.IsSuspended() {
		age := now.Sub(tracker.SuspendedAt).Round(time.Second)
		details["autoHealSuspended"] = true
		details["suspendedForSeconds"] = int(age.Seconds())
		details["suspensionReason"] = tracker.SuspensionReason
		result.Message += fmt.Sprintf(" [auto-heal suspended for %s: %s]", age, tracker.SuspensionReason)
	}
	if hold.Active {
		held := now.Sub(hold.Since).Round(time.Second)
		details["healHeld"] = true
		details["healHeldClass"] = hold.Class
		details["healHeldSeconds"] = int(held.Seconds())
		details["healHeldReason"] = hold.Reason
		if hold.Overridden {
			details["healHoldOverridden"] = true
			result.Message += fmt.Sprintf(" [restart gate %s overridden after %s: %s]", hold.Class, held, hold.Reason)
		} else {
			result.Message += fmt.Sprintf(" [restart held %s by %s: %s]", held, hold.Class, hold.Reason)
		}
		if held >= hold.CriticalAfter {
			result.Status = StatusCritical
		}
	}
	result.Details = details
	return result
}

// effectiveStatus is the result status before the heal-state overlay.
func effectiveStatus(result Result) Status {
	if result.Details != nil {
		if raw, ok := result.Details["statusBeforeOverlay"].(string); ok && raw != "" {
			return Status(raw)
		}
	}
	return result.Status
}

// OwnedElsewhere marks a failing result whose repair belongs to someone other
// than autoheal: host pressure, which no restart here relieves, and setup-owned
// state autoheal must never apply. A check with no safe supported action says
// plainly who owns the repair, on the result, in the heal skip reason and in the
// incident, instead of offering a fake action or failing without an owner.
func OwnedElsewhere(result Result, owner string) Result {
	if result.Status == StatusOK || result.Status == StatusNotApplicable {
		return result
	}
	details := make(map[string]interface{}, len(result.Details)+1)
	for k, v := range result.Details {
		details[k] = v
	}
	details["repairOwner"] = owner
	result.Details = details
	result.Message += fmt.Sprintf(" [no automatic repair here: %s]", owner)
	return result
}

// repairOwnerNote is the owner suffix for a heal skip reason, empty when the
// check does not name one.
func repairOwnerNote(result Result) string {
	if owner, _ := result.Details["repairOwner"].(string); owner != "" {
		return " (repair owner: " + owner + ")"
	}
	return ""
}
