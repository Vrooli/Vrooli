package capacity

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Capacity upshift (§8.8 hysteresis) is the symmetric counterpart to the degrade
// path: when a claim was stepped DOWN its profile under GPU pressure (large-v3 →
// medium → small) and the contending consumer later frees its VRAM, an idle
// degraded claim should climb back toward its preferred size so it is ready
// before its owner is next used. The degrade ladder reclaims capacity; upshift
// returns it. Without this, a whisper degraded to `small` stays there
// indefinitely even after the GPU empties — the exact gap the upshift opt-in
// (`profile.upshift:true`) and the `upshift_headroom` lever declared but never
// actuated.
//
// Upshift is idle-time/background only (driven from the maintenance sweep, never
// synchronously on an active request) so a model-resize never blips an in-flight
// transcription, and it is enforce-gated exactly like degrade: advisory logs the
// would-upshift, enforce=on actuates. Anti-thrash reuses degrade_debounce — a
// just-resized claim is not immediately resized again — alongside the
// upshift_headroom hysteresis floor and the idle-only requirement.

// UpshiftEligible reports whether a claim is a candidate for capacity upshift —
// the priority/profile/idleness gate, independent of how much headroom is free
// (PlanUpshift applies the headroom test). A claim qualifies iff:
//
//   - it is an active VRAM claim currently in the degraded status, AND
//   - its declared profile opts into upshift (profile.upshift == true) AND has a
//     rung above its current amount to climb to, AND
//   - it is reported idle (never upshift while transcribing — age/util never
//     decide this), AND
//   - it is not protected.
//
// A claim already at (or above) its preferred amount, or with no profile, is not
// eligible (nothing to climb to).
func UpshiftEligible(c CapacityClaim, _ Policy) bool {
	if c.Protected || !IsActiveClaimStatus(c.Status) || c.ResourceKind != ResourceKindVRAM {
		return false
	}
	if c.Status != StatusDegraded {
		return false
	}
	if c.ActivityState != ActivityIdle {
		return false
	}
	if c.DegradeProfile == nil || !c.DegradeProfile.Upshift {
		return false
	}
	return c.AmountBytes < upshiftCeiling(c)
}

// upshiftCeiling is the largest amount a claim may climb to: its preferred amount,
// but never above the top profile rung (the profile is authoritative).
func upshiftCeiling(c CapacityClaim) int64 {
	ceiling := c.PreferredBytes
	if c.DegradeProfile != nil {
		for _, st := range c.DegradeProfile.Steps {
			if st.AmountBytes > ceiling {
				ceiling = st.AmountBytes
			}
		}
		// Cap at preferred when preferred is set and below the top rung.
		if c.PreferredBytes > 0 {
			ceiling = c.PreferredBytes
		}
	}
	return ceiling
}

// PlanUpshift returns the request-upshift action that steps a degraded, idle claim
// UP to the HIGHEST profile rung that (a) is above its current amount, (b) is at or
// below its preferred ceiling, and (c) whose growth (rung − current) fits within
// freeHeadroom — but only when freeHeadroom clears the policy.UpshiftHeadroom
// hysteresis floor. ok is false when the claim is not upshift-eligible, headroom is
// below the floor, or no rung fits. PURE: the caller computes freeHeadroom and
// decides whether to actuate (§8.8 — enforce only).
func PlanUpshift(c CapacityClaim, freeHeadroom int64, policy Policy, _ time.Time) (EscalationAction, bool) {
	if !UpshiftEligible(c, policy) {
		return EscalationAction{}, false
	}
	floor := policy.UpshiftHeadroom
	if floor < 0 {
		floor = DefaultUpshiftHeadroom
	}
	// Hysteresis: never upshift unless a comfortable buffer of free VRAM exists, so
	// a small transient dip in GPU pressure does not trigger an immediate resize
	// that the next batch claim would just reverse.
	if freeHeadroom < floor {
		return EscalationAction{}, false
	}
	target, ok := bestUpshiftRung(c, freeHeadroom)
	if !ok {
		return EscalationAction{}, false
	}
	return EscalationAction{
		ClaimID:    c.ClaimID,
		OwnerID:    c.OwnerID,
		Action:     ActionRequestUpshift,
		ToStep:     target.Label,
		FreesBytes: 0, // upshift consumes capacity; it frees nothing
		Reason: fmt.Sprintf("idle degraded claim %q can climb %q→%q (%s free ≥ %s headroom)",
			c.OwnerID, humanBytes(c.AmountBytes), target.Label, humanBytes(freeHeadroom), humanBytes(floor)),
	}, true
}

// bestUpshiftRung picks the highest profile rung above the claim's current amount,
// capped at its preferred ceiling, whose additional bytes fit within freeHeadroom.
func bestUpshiftRung(c CapacityClaim, freeHeadroom int64) (DegradeStep, bool) {
	if c.DegradeProfile == nil {
		return DegradeStep{}, false
	}
	ceiling := upshiftCeiling(c)
	var rungs []DegradeStep
	for _, st := range c.DegradeProfile.Steps {
		if st.AmountBytes > c.AmountBytes && st.AmountBytes <= ceiling && st.AmountBytes-c.AmountBytes <= freeHeadroom {
			rungs = append(rungs, st)
		}
	}
	if len(rungs) == 0 {
		return DegradeStep{}, false
	}
	// Highest fitting rung first — climb as far as the headroom allows in one step.
	sort.SliceStable(rungs, func(i, j int) bool { return rungs[i].AmountBytes > rungs[j].AmountBytes })
	return rungs[0], true
}

// PlanUpshiftAll builds the (pure) plan of opportunistic upshifts across a set of
// active claims, computing each claim's free GPU headroom from the host snapshot.
// When several claims share a GPU, the running free headroom is decremented as
// each upshift is planned so the pass never over-commits VRAM. It names what the
// broker WOULD do; advisory logs the plan and enforce=on actuates it (RunUpshift).
func PlanUpshiftAll(claims []CapacityClaim, snapshot hostinventory.Snapshot, policy Policy, now time.Time) EscalationPlan {
	var plan EscalationPlan
	// Per-GPU running free headroom, seeded lazily from the snapshot + ledger so
	// each chosen upshift shrinks what later candidates on the same GPU can take.
	headroomByGPU := map[int]int64{}
	seeded := map[int]bool{}

	// Process degraded idle claims most-degraded-first (lowest current amount) so a
	// claim that gave up the most VRAM gets first refusal on the freed headroom.
	ordered := append([]CapacityClaim(nil), claims...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].AmountBytes < ordered[j].AmountBytes })

	for _, c := range ordered {
		if !UpshiftEligible(c, policy) {
			continue
		}
		idx := gpuIndexOf(c)
		if !seeded[idx] {
			headroomByGPU[idx] = gpuFreeHeadroom(snapshot, claims, idx)
			seeded[idx] = true
		}
		action, ok := PlanUpshift(c, headroomByGPU[idx], policy, now)
		if !ok {
			continue
		}
		target, _ := stepAmount(c.DegradeProfile, action.ToStep)
		headroomByGPU[idx] -= target - c.AmountBytes
		plan.Actions = append(plan.Actions, action)
	}
	plan.Satisfied = true // upshift is opportunistic; there is no deficit to satisfy
	return plan
}

// gpuFreeHeadroom returns the genuinely-free VRAM (bytes) on a GPU: its total minus
// the more conservative of observed usage and the sum of active ledger commitments
// on that GPU (mirrors Decide's effectiveUsed). Negative results clamp to 0.
func gpuFreeHeadroom(snapshot hostinventory.Snapshot, ledger []CapacityClaim, gpuIndex int) int64 {
	idx := gpuIndex
	gpu, found := selectGPU(&idx, snapshot.GPUs)
	if !found {
		return 0
	}
	var committed int64
	for _, c := range ledger {
		if !IsActiveClaimStatus(c.Status) || c.ResourceKind != ResourceKindVRAM {
			continue
		}
		if gpuIndexOf(c) != gpuIndex {
			continue
		}
		committed += c.AmountBytes
	}
	used := int64(gpu.VRAMUsedBytes)
	if committed > used {
		used = committed
	}
	free := int64(gpu.VRAMBytes) - used
	if free < 0 {
		free = 0
	}
	return free
}

func gpuIndexOf(c CapacityClaim) int {
	if c.GPUIndex != nil {
		return *c.GPUIndex
	}
	return 0
}

// RunUpshift plans opportunistic upshifts and, ONLY under enforce=on, actuates them
// through the same ApplyExecutor seam the degrade path uses (Actuate honors the
// debounce window, runs the adopter's resize verb with --upshift, and never strands
// the claim on a verb failure). Advisory/off returns the plan WITHOUT actuating —
// the caller logs what it WOULD upshift. The snapshot supplies per-GPU free
// headroom; an empty/absent snapshot simply yields no actions (fail-safe).
func RunUpshift(ctx context.Context, store ClaimRepository, claims []CapacityClaim, snapshot hostinventory.Snapshot, exec ApplyExecutor, policy Policy, enforce string, now time.Time) (EscalationPlan, ActuationResult, error) {
	plan := PlanUpshiftAll(claims, snapshot, policy, now)
	if enforce != EnforceOn || len(plan.Actions) == 0 {
		return plan, ActuationResult{}, nil
	}
	res, err := Actuate(ctx, plan, store, exec, policy, now)
	return plan, res, err
}
