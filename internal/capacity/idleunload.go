package capacity

import (
	"context"
	"fmt"
	"time"
)

// idleUnloadTTLFor resolves the effective autonomous idle-unload dwell for a
// claim: its own declared idle_unload_ttl, or the policy default_idle_unload_ttl
// when the claim does not declare one. Zero means "never autonomously unload".
func idleUnloadTTLFor(c CapacityClaim, policy Policy) time.Duration {
	if c.IdleUnloadTTL > 0 {
		return c.IdleUnloadTTL
	}
	return policy.DefaultIdleUnloadTTL
}

// IdleUnloadEligible reports whether a claim should be AUTONOMOUSLY unloaded
// (§Phase 3 keystone) — i.e. it has sat idle long enough that the broker should
// proactively free its VRAM, accepting a cold start on next use. This is a
// distinct axis from demand-driven reclaim (no requester is asking). A claim is
// eligible iff:
//
//   - it has a positive idle_unload_ttl (own or policy default), AND
//   - it is not protected (an interactive owner active right now is protected), AND
//   - it is an active VRAM claim, AND
//   - its activity_state is idle AND it has dwelt idle ≥ idle_unload_ttl, AND
//   - it has a degrade profile with a rung below its current amount (something to
//     unload to — a claim already at its floor is skipped).
//
// Age and utilization NEVER make a claim eligible — only the work-owner-reported
// idle state plus the declared TTL.
func IdleUnloadEligible(c CapacityClaim, policy Policy, now time.Time) bool {
	ttl := idleUnloadTTLFor(c, policy)
	if ttl <= 0 {
		return false
	}
	if c.Protected || !IsActiveClaimStatus(c.Status) || c.ResourceKind != ResourceKindVRAM {
		return false
	}
	if c.ActivityState != ActivityIdle {
		return false
	}
	if now.Before(idleSince(c).Add(ttl)) {
		return false
	}
	_, ok := idleUnloadAction(c)
	return ok
}

// idleUnloadAction builds the request-degrade action that drops an idle claim all
// the way to its lowest profile rung (floor / "unloaded"), freeing the most VRAM.
// ok is false when the claim has no profile or is already at (or below) its floor.
func idleUnloadAction(c CapacityClaim) (EscalationAction, bool) {
	if c.DegradeProfile == nil || len(c.DegradeProfile.Steps) == 0 {
		return EscalationAction{}, false
	}
	floor := c.DegradeProfile.Steps[0]
	for _, st := range c.DegradeProfile.Steps {
		if st.AmountBytes < floor.AmountBytes {
			floor = st
		}
	}
	if c.AmountBytes <= floor.AmountBytes {
		return EscalationAction{}, false
	}
	return EscalationAction{
		ClaimID:    c.ClaimID,
		OwnerID:    c.OwnerID,
		Action:     ActionRequestDegrade,
		ToStep:     floor.Label,
		FreesBytes: c.AmountBytes - floor.AmountBytes,
		Reason:     fmt.Sprintf("idle beyond idle_unload_ttl; autonomously unloading %q to %q (frees %s)", c.OwnerID, floor.Label, humanBytes(c.AmountBytes-floor.AmountBytes)),
	}, true
}

// PlanIdleUnload builds the (pure) plan of autonomous idle-unload degrades across
// a set of active claims. It names what the broker WOULD do; advisory mode logs
// the plan and enforce=on actuates it (via RunIdleUnload). The plan is a no-op for
// any claim that is not idle-unload-eligible.
func PlanIdleUnload(claims []CapacityClaim, policy Policy, now time.Time) EscalationPlan {
	var plan EscalationPlan
	for _, c := range claims {
		if !IdleUnloadEligible(c, policy, now) {
			continue
		}
		action, ok := idleUnloadAction(c)
		if !ok {
			continue
		}
		plan.Actions = append(plan.Actions, action)
		plan.FreedBytes += action.FreesBytes
	}
	plan.Satisfied = true // idle-unload has no deficit to satisfy; it is opportunistic
	return plan
}

// RunIdleUnload plans the autonomous idle-unload and, ONLY under enforce=on,
// actuates it through the existing degrade path (Actuate honors degrade_debounce,
// the profile floor step, and never strands a resource off-GPU on failure).
// Advisory/off returns the plan WITHOUT actuating — the broker logs what it WOULD
// unload. GC and sampling stay safe regardless; only this unload step is gated.
func RunIdleUnload(ctx context.Context, store ClaimRepository, claims []CapacityClaim, exec ApplyExecutor, policy Policy, enforce string, now time.Time) (EscalationPlan, ActuationResult, error) {
	plan := PlanIdleUnload(claims, policy, now)
	if enforce != EnforceOn || len(plan.Actions) == 0 {
		return plan, ActuationResult{}, nil
	}
	res, err := Actuate(ctx, plan, store, exec, policy, now)
	return plan, res, err
}
