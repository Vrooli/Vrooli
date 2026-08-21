package capacity

import (
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// stepCandidate is one grantable amount considered by Decide.
type stepCandidate struct {
	label  string
	amount int64
}

// Decide renders an admission verdict for a request against a live snapshot,
// the current ledger, and policy (plan §8.5). It is a PURE function: no side
// effects, no nvidia-smi/docker calls (capacity arrives via the injected
// snapshot). `now` is the decision time, used only to evaluate idle-grace.
//
// The ledger passed in should be the active claims on the same resource (the
// caller filters); claims belonging to the requester itself should be excluded
// when re-deciding an existing owner so it is not counted against itself.
func Decide(req CapacityRequest, snapshot hostinventory.Snapshot, ledger []CapacityClaim, policy Policy, now time.Time) Verdict {
	if policy.IdleGrace < 0 {
		policy = DefaultPolicy()
	}

	candidates, err := buildCandidates(req)
	if err != nil {
		return Verdict{Kind: VerdictDeny, Reason: err.Error()}
	}

	total, observedUsed, ok, resolveWarn := resolveCapacity(req, snapshot)
	if !ok {
		return Verdict{Kind: VerdictDeny, Reason: resolveWarn}
	}

	// A host that is swapping heavily has already run out of memory; granting
	// more memory-hungry work makes it worse rather than better. AvailableBytes
	// alone cannot see this — once pages are on disk it can look healthy while
	// the machine thrashes.
	if req.ResourceKind == ResourceKindRAM && SwapPressure(snapshot, policy.SwapPressureThresholdPct) {
		return Verdict{
			Kind: VerdictDeny,
			Reason: fmt.Sprintf("host is under swap pressure (at or above %d%% of swap in use); "+
				"admitting more memory would deepen the thrashing", policy.SwapPressureThresholdPct),
		}
	}

	// effectiveUsed is the more conservative of observed usage and the sum of
	// outstanding ledger commitments on this resource — so two simultaneous
	// grants before either materializes don't both think the space is free.
	committed := sumActiveAmounts(req, ledger)
	effectiveUsed := observedUsed
	if committed > effectiveUsed {
		effectiveUsed = committed
	}
	available := total - effectiveUsed
	if available < 0 {
		available = 0
	}

	// Reclaim accounting: idle-eligible reclaimable claims can be reclaimed right
	// now; the broader reclaimable set could be reclaimed once it goes idle. Both
	// honor the idle-yield rule, so an idle yield-opted higher-priority claim
	// (e.g. interactive whisper) counts toward what a batch requester can reclaim.
	var reclaimNowBytes, potentialBytes int64
	var reclaimTargets []string
	aheadCount := 0
	for _, c := range ledger {
		if !IsActiveClaimStatus(c.Status) {
			continue
		}
		if potentialReclaimEligibleFor(c, req.Priority, policy) {
			potentialBytes += c.AmountBytes
			if isReclaimEligible(c, effectiveIdleGrace(c, policy), now) {
				reclaimNowBytes += c.AmountBytes
				reclaimTargets = append(reclaimTargets, c.ClaimID)
			}
			continue
		}
		if c.Priority >= req.Priority {
			aheadCount++
		}
	}

	availableNow := available + reclaimNowBytes
	availablePotential := available + potentialBytes

	// Choose the best step that fits availableNow.
	top := candidates[0]
	floor := candidates[len(candidates)-1]
	for _, cand := range candidates {
		if cand.amount <= availableNow {
			v := Verdict{GrantedBytes: cand.amount, Step: cand.label}
			if cand.amount > available {
				v.ReclaimTargets = reclaimTargets
				v.ReclaimBytes = cand.amount - available
				v.Warnings = append(v.Warnings, fmt.Sprintf("grant requires reclaiming %d idle lower-priority claim(s)", len(reclaimTargets)))
			}
			if cand.label == top.label && cand.amount == top.amount {
				v.Kind = VerdictGrant
			} else {
				v.Kind = VerdictDegrade
				v.Reason = fmt.Sprintf("preferred %d exceeds available %d; granted step %q (%d)", top.amount, availableNow, cand.label, cand.amount)
			}
			if resolveWarn != "" {
				v.Warnings = append(v.Warnings, resolveWarn)
			}
			return v
		}
	}

	// Nothing fits now. If the floor could fit once lower-priority active claims
	// go idle, queue; otherwise deny.
	if floor.amount <= availablePotential {
		return Verdict{
			Kind:          VerdictQueue,
			QueuePosition: aheadCount + 1,
			Reason:        fmt.Sprintf("floor %d exceeds available %d; waiting for %d reclaimable claim(s) to idle", floor.amount, availableNow, len(potentialReclaimable(req, ledger, policy))),
		}
	}
	return Verdict{
		Kind:   VerdictDeny,
		Reason: fmt.Sprintf("insufficient %s: floor %d exceeds available %d even after reclaiming idle/lower-priority claims", req.ResourceKind, floor.amount, availableNow),
	}
}

// buildCandidates derives the descending grantable steps for a request. When a
// degrade profile is declared its steps are authoritative; otherwise we
// synthesize {preferred} and (if distinct and positive) {floor}.
func buildCandidates(req CapacityRequest) ([]stepCandidate, error) {
	if req.PreferredBytes < 0 || req.FloorBytes < 0 {
		return nil, fmt.Errorf("%w: byte amounts must be non-negative", ErrInvalidClaim)
	}
	if req.FloorBytes > req.PreferredBytes && req.PreferredBytes > 0 {
		return nil, fmt.Errorf("%w: floor %d exceeds preferred %d", ErrInvalidClaim, req.FloorBytes, req.PreferredBytes)
	}
	var cands []stepCandidate
	if req.Profile != nil && len(req.Profile.Steps) > 0 {
		for _, st := range req.Profile.Steps {
			if st.AmountBytes < 0 {
				return nil, fmt.Errorf("%w: profile step %q has negative amount", ErrInvalidClaim, st.Label)
			}
			cands = append(cands, stepCandidate{label: st.Label, amount: st.AmountBytes})
		}
	} else {
		cands = append(cands, stepCandidate{label: "preferred", amount: req.PreferredBytes})
		if req.FloorBytes > 0 && req.FloorBytes < req.PreferredBytes {
			cands = append(cands, stepCandidate{label: "floor", amount: req.FloorBytes})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].amount > cands[j].amount })
	return cands, nil
}

// resolveCapacity returns (total, observedUsed, ok, warn) for the request's
// resource. ok=false means the request cannot be satisfied on this host (e.g.
// VRAM requested but no matching GPU); warn carries the reason.
func resolveCapacity(req CapacityRequest, snapshot hostinventory.Snapshot) (total, used int64, ok bool, warn string) {
	switch req.ResourceKind {
	case ResourceKindVRAM:
		gpu, found := selectGPU(req.GPUIndex, snapshot.GPUs)
		if !found {
			return 0, 0, false, "no matching GPU available for vram request"
		}
		return int64(gpu.VRAMBytes), int64(gpu.VRAMUsedBytes), true, ""
	case ResourceKindRAM:
		total = int64(snapshot.Memory.TotalBytes)
		if total <= 0 {
			return 0, 0, false, "host RAM total is unknown"
		}
		if snapshot.Memory.AvailableBytes > 0 {
			used = total - int64(snapshot.Memory.AvailableBytes)
			if used < 0 {
				used = 0
			}
		}
		return total, used, true, ""
	case ResourceKindCPU:
		if snapshot.CPU.Cores <= 0 {
			return 0, 0, false, "host CPU capacity is unknown"
		}
		// CPU claims use millicores as the generic amount unit: one logical
		// core is 1000 units. This keeps the frozen claim schema unchanged while
		// allowing the same ledger and admission algorithm to enforce CPU.
		total = int64(snapshot.CPU.Cores) * 1000
		used, warn = observedCPUMillis(snapshot)
		return total, used, true, warn
	default:
		return 0, 0, false, fmt.Sprintf("unknown resource kind %q", req.ResourceKind)
	}
}

// observedCPUMillis estimates how much CPU the host is already using, in the
// same millicore unit the claims use.
//
// This used to return a literal 0, which meant admission believed every core
// was free no matter what the host was doing. The snapshot already carried the
// answer: it collects Load and Swap on every capture, and both were discarded.
// Admission was therefore blind to load while being the component whose whole
// job is deciding whether the host can take more work — so a saturated machine
// admitted phases exactly as readily as an idle one.
//
// NormalizedLoad1 is preferred: it is already load-per-core, so it does not
// have to be divided by a core count that may differ from the one the snapshot
// reports. Raw Load1 is the fallback. Both are run-queue length rather than
// true utilization, so a value above 1.0 per core means processes are waiting;
// it is clamped to the total, because "more than fully busy" is still just
// fully busy as far as remaining capacity is concerned.
//
// A snapshot with no load signal at all reports 0 used, with a warning. That is
// the old behaviour, kept only as an explicit, visible degradation rather than
// as the silent default it used to be.
func observedCPUMillis(snapshot hostinventory.Snapshot) (int64, string) {
	total := int64(snapshot.CPU.Cores) * 1000

	perCore := snapshot.Load.NormalizedLoad1
	if perCore <= 0 && snapshot.Load.Load1 > 0 && snapshot.CPU.Cores > 0 {
		perCore = snapshot.Load.Load1 / float64(snapshot.CPU.Cores)
	}
	if perCore <= 0 {
		return 0, "host CPU load is unknown; admission is treating the host as idle"
	}

	used := int64(perCore * float64(total))
	if used > total {
		used = total
	}
	if used < 0 {
		used = 0
	}
	return used, ""
}

// SwapPressure reports whether the host is swapping heavily enough that
// admitting more memory-hungry work would make things worse rather than better.
//
// Swap usage is a lagging signal of memory exhaustion that the RAM figures miss:
// once pages are on disk, AvailableBytes can look healthy while the machine
// thrashes. The snapshot has carried Swap all along without anything reading it.
//
// A host with no swap configured is not under swap pressure, which is why the
// zero-total case returns false rather than dividing by zero.
func SwapPressure(snapshot hostinventory.Snapshot, thresholdPct int) bool {
	if thresholdPct <= 0 {
		return false // the check is disabled
	}
	if snapshot.Swap.TotalBytes == 0 {
		return false // no swap configured is not swap pressure
	}
	usedPct := 100 * (1 - float64(snapshot.Swap.FreeBytes)/float64(snapshot.Swap.TotalBytes))
	return usedPct >= float64(thresholdPct)
}

// selectGPU picks the GPU matching the requested index, defaulting to index 0,
// then to the first GPU present.
func selectGPU(index *int, gpus []hostinventory.GPU) (hostinventory.GPU, bool) {
	if len(gpus) == 0 {
		return hostinventory.GPU{}, false
	}
	want := 0
	if index != nil {
		want = *index
	}
	for _, g := range gpus {
		if g.Index == want {
			return g, true
		}
	}
	if index != nil {
		return hostinventory.GPU{}, false
	}
	return gpus[0], true
}

// sumActiveAmounts sums the granted bytes of active claims on the same resource
// (and GPU, for vram) as the request.
func sumActiveAmounts(req CapacityRequest, ledger []CapacityClaim) int64 {
	var sum int64
	for _, c := range ledger {
		if !IsActiveClaimStatus(c.Status) || c.ResourceKind != req.ResourceKind {
			continue
		}
		if req.ResourceKind == ResourceKindVRAM && !sameGPU(req.GPUIndex, c.GPUIndex) {
			continue
		}
		sum += c.AmountBytes
	}
	return sum
}

func sameGPU(a, b *int) bool {
	av, bv := 0, 0
	if a != nil {
		av = *a
	}
	if b != nil {
		bv = *b
	}
	return av == bv
}

// potentialReclaimEligibleFor reports whether claim c could ever be reclaimed to
// satisfy a requester of priority requesterPriority — IGNORING current activity
// (it is the priority/protection gate, not the idleness test). A claim qualifies
// when it is unprotected and active-status AND either:
//
//   - strict default: its priority is strictly lower than the requester's, OR
//   - idle-yield (§8.3): it opted into yield_when_idle AND the requester's
//     priority is at or above the idle-yield floor. This relaxes the strict
//     lower-priority rule to permit EQUAL-priority reclaim of a yielded claim,
//     but ONLY for claims that explicitly opted in.
//
// Claims without the flag keep the strict rule byte-for-byte. Pair this with
// isReclaimEligible for "reclaimable right now".
func potentialReclaimEligibleFor(c CapacityClaim, requesterPriority int, policy Policy) bool {
	if c.Protected || !IsActiveClaimStatus(c.Status) {
		return false
	}
	if c.Priority < requesterPriority {
		return true // strict default: strictly-lower priority is reclaimable
	}
	if c.YieldWhenIdle && requesterPriority >= idleYieldFloor(policy) {
		return true // idle-yield opt-in: yields to floor-and-above work
	}
	return false
}

// reclaimEligibleFor reports whether claim c may be reclaimed RIGHT NOW for a
// requester of priority requesterPriority: it must pass the priority/protection
// gate AND have dwelt idle beyond idle_grace. Active claims (and non-opt-in
// higher/equal-priority claims) are never eligible — age/utilization never make
// a claim eligible, only reported idle state does.
func reclaimEligibleFor(c CapacityClaim, requesterPriority int, policy Policy, now time.Time) bool {
	return potentialReclaimEligibleFor(c, requesterPriority, policy) && isReclaimEligible(c, effectiveIdleGrace(c, policy), now)
}

func effectiveIdleGrace(c CapacityClaim, policy Policy) time.Duration {
	if c.IdleGrace > 0 {
		return c.IdleGrace
	}
	return policy.IdleGrace
}

// idleYieldFloor normalizes the policy's idle-yield floor, defaulting a zero
// value (a Policy constructed without DefaultPolicy) to batch — matching
// DefaultPolicy — rather than 0, which would let any requester reclaim.
func idleYieldFloor(policy Policy) int {
	if policy.IdleYieldFloor <= 0 {
		return PriorityBatch
	}
	return policy.IdleYieldFloor
}

// isReclaimEligible reports whether a claim may be reclaimed right now: it must
// be reported idle and have dwelt idle for at least the grace period. Age and
// utilization never make a claim eligible — only reported idle state does.
func isReclaimEligible(c CapacityClaim, grace time.Duration, now time.Time) bool {
	if c.ActivityState != ActivityIdle {
		return false
	}
	since := c.LastActiveAt
	if since == nil {
		// Never reported active: idle since creation.
		since = &c.CreatedAt
	}
	return !now.Before(since.Add(grace))
}

func potentialReclaimable(req CapacityRequest, ledger []CapacityClaim, policy Policy) []CapacityClaim {
	var out []CapacityClaim
	for _, c := range ledger {
		if potentialReclaimEligibleFor(c, req.Priority, policy) {
			out = append(out, c)
		}
	}
	return out
}
