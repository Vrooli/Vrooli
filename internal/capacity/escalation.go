package capacity

import (
	"fmt"
	"sort"
	"time"
)

// Escalation ladder rungs (plan §7 Phase 4). The broker climbs the ladder only
// as far as needed and never past a rung the policy forbids.
const (
	ActionWarn           = "warn"            // surface an unclaimed/contended consumer; no action
	ActionRequestDegrade = "request-degrade" // ask the adopter to step down its profile
	ActionPreempt        = "preempt"         // stop the owner (last rung, config-gated)
)

// EscalationAction is one planned step against a reclaim target.
type EscalationAction struct {
	ClaimID    string `json:"claim_id"`
	OwnerID    string `json:"owner_id"`
	Action     string `json:"action"`
	ToStep     string `json:"to_step,omitempty"` // for request-degrade
	FreesBytes int64  `json:"frees_bytes"`       // bytes this action returns to the pool
	Reason     string `json:"reason,omitempty"`
}

// EscalationPlan is the ordered set of actions the broker would take to free
// `deficit` bytes for a requester, plus whether the plan actually satisfies it.
type EscalationPlan struct {
	Actions    []EscalationAction `json:"actions"`
	FreedBytes int64              `json:"freed_bytes"`
	Satisfied  bool               `json:"satisfied"`
	Deficit    int64              `json:"deficit"`
}

// PlanEscalation builds the reclaim plan to free `deficit` bytes for a request
// of priority `requesterPriority`, honoring the hard rules (plan §8.3, §11):
//
//   - A claim is reclaim-eligible ONLY IF it is idle beyond idle_grace AND
//     strictly lower priority than the requester AND not protected. Age and
//     utilization NEVER make a claim eligible.
//   - Degradation is always tried before preemption. request-degrade steps a
//     claim down to the highest profile rung that still frees enough; preempt
//     (stop) is the last rung and only when policy.PreemptEnabled.
//
// It is a PURE planner: it computes what the broker WOULD do. Applying the plan
// (invoking the adopter's degrade verb, stopping a container) is the
// orchestration layer's job.
func PlanEscalation(requesterPriority int, deficit int64, candidates []CapacityClaim, policy Policy, now time.Time) EscalationPlan {
	plan := EscalationPlan{Deficit: deficit}
	if deficit <= 0 {
		plan.Satisfied = true
		return plan
	}

	// Reclaim eligibility (§8.3): unprotected, active-status, idle beyond grace,
	// and either strictly-lower priority (the strict default) or an idle
	// yield-opted claim the requester sits at/above the floor of. reclaimEligibleFor
	// encodes the whole rule; for non-opt-in claims it is byte-identical to the
	// strict lower-priority test.
	eligible := make([]CapacityClaim, 0, len(candidates))
	for _, c := range candidates {
		if reclaimEligibleFor(c, requesterPriority, policy, now) {
			eligible = append(eligible, c)
		}
	}

	// Reclaim the lowest-priority, longest-idle claims first so we disturb the
	// least-important work and prefer those that have been idle longest.
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Priority != eligible[j].Priority {
			return eligible[i].Priority < eligible[j].Priority
		}
		return idleSince(eligible[i]).Before(idleSince(eligible[j]))
	})

	for _, c := range eligible {
		if plan.FreedBytes >= deficit {
			break
		}
		if action, ok := degradeAction(c, deficit-plan.FreedBytes); ok {
			plan.Actions = append(plan.Actions, action)
			plan.FreedBytes += action.FreesBytes
			continue
		}
		// No degrade rung frees anything (no profile / already at floor): the
		// only way to reclaim is to stop the owner — the last rung.
		if policy.PreemptEnabled {
			plan.Actions = append(plan.Actions, EscalationAction{
				ClaimID:    c.ClaimID,
				OwnerID:    c.OwnerID,
				Action:     ActionPreempt,
				FreesBytes: c.AmountBytes,
				Reason:     fmt.Sprintf("idle lower-priority claim has no degrade headroom; preempting frees %s", humanBytes(c.AmountBytes)),
			})
			plan.FreedBytes += c.AmountBytes
		} else {
			plan.Actions = append(plan.Actions, EscalationAction{
				ClaimID: c.ClaimID,
				OwnerID: c.OwnerID,
				Action:  ActionWarn,
				Reason:  "idle lower-priority claim could be reclaimed, but preempt is disabled by policy",
			})
		}
	}

	plan.Satisfied = plan.FreedBytes >= deficit
	return plan
}

// degradeAction returns the request-degrade action that frees the most bytes
// without dropping below what's needed, choosing the HIGHEST profile rung that
// still frees at least `need` (so we degrade as little as possible). If no rung
// below the current amount exists, ok is false (the claim is at its floor or
// has no profile — escalation must consider preemption instead).
func degradeAction(c CapacityClaim, need int64) (EscalationAction, bool) {
	if c.DegradeProfile == nil || len(c.DegradeProfile.Steps) == 0 {
		return EscalationAction{}, false
	}
	// Steps below the current amount, sorted descending (least disruptive first).
	type rung struct {
		label  string
		amount int64
	}
	var rungs []rung
	for _, st := range c.DegradeProfile.Steps {
		if st.AmountBytes < c.AmountBytes {
			rungs = append(rungs, rung{st.Label, st.AmountBytes})
		}
	}
	if len(rungs) == 0 {
		return EscalationAction{}, false
	}
	sort.SliceStable(rungs, func(i, j int) bool { return rungs[i].amount > rungs[j].amount })

	// Prefer the highest rung that frees enough; if none frees enough, take the
	// lowest rung (frees the most we can from this claim).
	chosen := rungs[len(rungs)-1]
	for _, r := range rungs {
		if c.AmountBytes-r.amount >= need {
			chosen = r
			break
		}
	}
	frees := c.AmountBytes - chosen.amount
	return EscalationAction{
		ClaimID:    c.ClaimID,
		OwnerID:    c.OwnerID,
		Action:     ActionRequestDegrade,
		ToStep:     chosen.label,
		FreesBytes: frees,
		Reason:     fmt.Sprintf("degrade %q to %q frees %s before any preempt", c.OwnerID, chosen.label, humanBytes(frees)),
	}, true
}

func idleSince(c CapacityClaim) time.Time {
	if c.LastActiveAt != nil {
		return *c.LastActiveAt
	}
	return c.CreatedAt
}
