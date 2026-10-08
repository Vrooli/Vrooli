package checks

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const interlockRefusalKind = "recent-cross-check-start"

// RecentHealAction is the successful lifecycle action retained by the
// in-process interlock. The durable action ledger remains the system of record;
// this short-lived index exists only to make the safety decision atomic.
type RecentHealAction struct {
	CheckID  string
	ActionID string
	Target   HealTarget
	At       time.Time
}

// SetHealInterlockWindow applies the operator-configured safety window.
func (r *Registry) SetHealInterlockWindow(window time.Duration) error {
	if window <= 0 {
		return fmt.Errorf("heal interlock window must be > 0")
	}
	r.interlockMu.Lock()
	defer r.interlockMu.Unlock()
	r.healInterlockWindow = window
	return nil
}

// ExecuteAction is the shared manual-action boundary. Scheduled auto-heal uses
// the same guarded executor after it has selected an action.
func (r *Registry) ExecuteAction(ctx context.Context, checkID, actionID string) ActionResult {
	healable, ok := r.GetHealableCheck(checkID)
	if !ok {
		return ActionResult{ActionID: actionID, CheckID: checkID, Success: false, Error: "check does not support healing", Timestamp: r.now()}
	}
	var lastResult *Result
	if result, exists := r.GetResult(checkID); exists {
		lastResult = &result
	}
	action := recoveryActionByID(ctx, healable, lastResult, actionID)
	return r.executeGuardedAction(ctx, healable, action)
}

func recoveryActionByID(ctx context.Context, healable HealableCheck, lastResult *Result, actionID string) RecoveryAction {
	var actions []RecoveryAction
	if aware, ok := healable.(ContextAwareHealableCheck); ok {
		actions = aware.RecoveryActionsWithContext(ctx, lastResult)
	} else {
		actions = healable.RecoveryActions(lastResult)
	}
	for _, action := range actions {
		if action.ID == actionID {
			return action
		}
	}
	return RecoveryAction{ID: actionID}
}

func (r *Registry) executeGuardedAction(ctx context.Context, healable HealableCheck, action RecoveryAction) ActionResult {
	checkID := healable.ID()
	target := healTargetFor(healable)
	now := r.now()

	if action.Dangerous {
		if refusal := r.interlockRefusal(now, target, checkID); refusal != nil {
			reason := fmt.Sprintf("heal interlock refused %s for %s: check %s recently ran %s; attempting check %s must wait %s", action.ID, target.String(), refusal.SourceCheckID, refusal.SourceActionID, checkID, refusal.Window)
			refusal.Reason = reason
			return ActionResult{
				ActionID: action.ID, CheckID: checkID, Success: false,
				Message: "Destructive heal action refused by safety interlock",
				Error:   reason, Timestamp: now, Refusal: refusal,
			}
		}
	}

	result := healable.ExecuteAction(ctx, action.ID)
	if result.CheckID == "" {
		result.CheckID = checkID
	}
	if result.ActionID == "" {
		result.ActionID = action.ID
	}
	if result.Timestamp.IsZero() {
		result.Timestamp = now
	}
	if result.Success && recordsRecentStart(action.ID) {
		r.recordRecentHealAction(RecentHealAction{CheckID: checkID, ActionID: action.ID, Target: target, At: r.now()})
	}
	return result
}

func (r *Registry) interlockRefusal(now time.Time, target HealTarget, attemptingCheckID string) *ActionRefusal {
	r.interlockMu.Lock()
	defer r.interlockMu.Unlock()
	recent, exists := r.recentHealActions[target]
	if !exists {
		return nil
	}
	if now.Sub(recent.At) >= r.healInterlockWindow {
		delete(r.recentHealActions, target)
		return nil
	}
	// A check may verify or repair its own lifecycle without deadlocking itself.
	if recent.CheckID == attemptingCheckID {
		return nil
	}
	return &ActionRefusal{
		Kind: interlockRefusalKind, Target: target,
		SourceCheckID: recent.CheckID, AttemptingCheckID: attemptingCheckID,
		SourceActionID: recent.ActionID, SourceTimestamp: recent.At,
		Window: r.healInterlockWindow.String(),
	}
}

func (r *Registry) recordRecentHealAction(action RecentHealAction) {
	r.interlockMu.Lock()
	defer r.interlockMu.Unlock()
	r.recentHealActions[action.Target] = action
}

func recordsRecentStart(actionID string) bool {
	switch actionID {
	case "start", "restart", "restart-clean", "setup-restart", "recover-go", "recover-pnpm", "respawn-companion":
		return true
	default:
		return false
	}
}

func healTargetFor(check HealableCheck) HealTarget {
	if provider, ok := check.(HealTargetProvider); ok {
		target := provider.HealTarget()
		target.Kind = strings.TrimSpace(target.Kind)
		target.Name = strings.TrimSpace(target.Name)
		if target.Name != "" {
			return target
		}
	}
	id := strings.TrimSpace(check.ID())
	if strings.HasPrefix(id, "resource-") {
		name := strings.TrimPrefix(id, "resource-")
		name = strings.TrimSuffix(name, "-mode-drift")
		return HealTarget{Kind: "resource", Name: name}
	}
	if strings.HasPrefix(id, "scenario-") {
		return HealTarget{Kind: "scenario", Name: strings.TrimPrefix(id, "scenario-")}
	}
	return HealTarget{Kind: "check", Name: id}
}
