package capacity

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// SweepResult reports the outcome of a presence-driven sweep.
type SweepResult struct {
	// Refreshed are active resident claims whose owner is still observed holding
	// GPU memory; their heartbeat deadline was renewed so the stale sweep does
	// not expire a still-alive owner.
	Refreshed []CapacityClaim `json:"refreshed"`
	// Expired are claims whose heartbeat deadline lapsed with no observed owner;
	// they were swept to "expired" (the normal liveness sweep).
	Expired []CapacityClaim `json:"expired"`
	// Sampled is how many active VRAM claims had their observed usage / decaying
	// peak refreshed this pass (§Phase 2 telemetry).
	Sampled  int              `json:"sampled"`
	Deferred ExpiryExclusions `json:"deferred,omitempty"`
}

// Sweep is the resident-claim heartbeat driver (plan §7 engine gap).
//
// Third-party model-server containers (whisper, kyutai-stt) hold VRAM
// continuously but have no shell hook to heartbeat their own capacity claim, so
// without a driver their claim would expire at DefaultHeartbeatTTL while the
// container is still very much alive — making the ledger lie (and forcing the
// ttl_seconds=21600 stopgap the adopters use today). Sweep derives their
// liveness from the host snapshot instead: any active *non-op* claim whose owner
// is currently observed holding GPU memory has its heartbeat renewed; every
// active claim past its deadline with no observed owner is then swept to expired
// as usual.
//
// Op-scoped claims (owner_kind=="op", e.g. image-tools:job-<id>) are NOT
// presence-refreshed — they own their own claim→run→release lifecycle and a
// short TTL is correct for them. RAM/CPU claims are not refreshed either: V1
// presence is GPU-process-based, the only honest residency signal we have.
//
// Sweep is read-mostly and never enforces. It is meant to be driven
// periodically (system-monitor's collector loop, a cleanup pass, or
// `vrooli capacity sweep`). Refresh happens BEFORE expiry so an observed-alive
// owner whose deadline has technically lapsed is rescued rather than expired.
func Sweep(ctx context.Context, store ClaimRepository, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time) (SweepResult, error) {
	return SweepWithFootprints(ctx, store, snapshot, attr, policy, now, nil)
}

func SweepWithFootprints(ctx context.Context, store ClaimRepository, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time, resolve FootprintResolver) (SweepResult, error) {
	if attr == nil {
		attr = unknownAttributor{}
	}
	active, err := store.ListClaims(ctx, ClaimFilter{Statuses: ActiveClaimStatuses()})
	if err != nil {
		return SweepResult{}, err
	}

	usage := observeClaimUsage(ctx, active, snapshot, attr)
	var result SweepResult
	result.Deferred = ExpiryExclusions{}
	proofs := AbsenceProofs{}
	for _, claim := range active {
		if !claimRefreshable(claim) {
			continue
		}
		if usage.uncertain.Contains(claim.ClaimID) {
			result.Deferred[claim.ClaimID] = usage.uncertain[claim.ClaimID]
			continue
		}
		if len(usage.assigned[claim.ClaimID]) == 0 {
			if !gpuProcessAbsenceVerified(snapshot, claim, now) {
				result.Deferred[claim.ClaimID] = "GPU process absence is unverified"
			} else {
				proofs[claim.ClaimID] = AbsenceProof{Generation: claim.Generation, ObservedAt: snapshot.FieldProvenance["gpu_processes"].ObservedAt}
			}
			continue
		}
		// Heartbeat at the claim's current generation. Heartbeat does not bump the
		// generation, so a concurrent activity report is the only thing that can
		// invalidate this; on a stale generation we simply skip (the next sweep
		// retries with a fresh read).
		refreshed, hbErr := store.HeartbeatClaim(ctx, claim.ClaimID, claim.Generation, policy.DefaultHeartbeatTTL)
		if hbErr != nil {
			continue
		}
		result.Refreshed = append(result.Refreshed, refreshed)
	}

	expired, err := store.ExpireStaleClaims(ctx, now, proofs)
	if err != nil {
		return result, err
	}
	result.Expired = expired

	// Usage sampling (§Phase 2): fold each active VRAM claim's currently-observed
	// footprint into its decaying high-water mark. Pure telemetry — RecordObserved
	// never bumps generation and only writes active claims, so a just-expired claim
	// in `active` is a silent no-op. Piggybacks the snapshot the sweep already
	// holds (no extra nvidia-smi).
	result.Sampled = sampleClaimUsage(ctx, store, active, snapshot, usage, policy, now, resolve)
	return result, nil
}

// SweepCursorStore is the surface the cadence-gated sweep needs: the claim
// repository plus a persisted "last swept" cursor. The cursor must be persisted
// because the sweep is driven from short-lived processes (the `vrooli` CLI, the
// maintenance pass) that cannot hold an in-memory timer across invocations.
type SweepCursorStore interface {
	ClaimRepository
	LastSweepAt(ctx context.Context) (time.Time, bool, error)
	RecordSweepAt(ctx context.Context, at time.Time) error
}

// SweepIfDue runs Sweep only when at least policy.SweepInterval has elapsed since
// the last recorded sweep, then records the new sweep time. due=false (zero
// result) means the call was debounced. This is what the opportunistic callers
// (admission, list, reconcile) use so a burst of reads does not re-collect the
// GPU snapshot and re-sweep on every call; the always-on maintenance pass calls
// Sweep directly.
func SweepIfDue(ctx context.Context, store SweepCursorStore, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time) (SweepResult, bool, error) {
	interval := policy.SweepInterval
	if interval <= 0 {
		interval = DefaultSweepInterval
	}
	last, ok, err := store.LastSweepAt(ctx)
	if err != nil {
		return SweepResult{}, false, err
	}
	if ok && now.Sub(last) < interval {
		return SweepResult{}, false, nil
	}
	result, err := Sweep(ctx, store, snapshot, attr, policy, now)
	if err != nil {
		return SweepResult{}, false, err
	}
	if recErr := store.RecordSweepAt(ctx, now); recErr != nil {
		return result, true, recErr
	}
	return result, true, nil
}

// MaybeSweep is the best-effort opportunistic-sweep helper for read paths that
// do NOT already hold a snapshot (admission, `capacity list`). It checks the
// debounce FIRST so a burst of reads does not collect a GPU snapshot (shell out
// to nvidia-smi) on every call — only a due call senses. It is a no-op when the
// store lacks the cursor surface, and skips entirely when sensing is unavailable
// (a failed snapshot must NEVER drive expiry, or a transient hiccup would
// falsely expire a live resident). Any error is returned for the caller to log;
// it is never fatal to the host operation the sweep rides on.
func MaybeSweep(ctx context.Context, store ClaimRepository, source CapacitySource, attr Attributor, policy Policy, clockNow func() time.Time) (SweepResult, bool, error) {
	return MaybeSweepWithFootprints(ctx, store, source, attr, policy, clockNow, nil)
}

func MaybeSweepWithFootprints(ctx context.Context, store ClaimRepository, source CapacitySource, attr Attributor, policy Policy, clockNow func() time.Time, resolve FootprintResolver) (SweepResult, bool, error) {
	if clockNow == nil {
		clockNow = time.Now
	}
	now := clockNow()
	cursor, ok := store.(SweepCursorStore)
	if !ok || source == nil {
		return SweepResult{}, false, nil
	}
	interval := policy.SweepInterval
	if interval <= 0 {
		interval = DefaultSweepInterval
	}
	last, seen, err := cursor.LastSweepAt(ctx)
	if err != nil {
		return SweepResult{}, false, err
	}
	if seen && now.Sub(last) < interval {
		return SweepResult{}, false, nil // debounced: no snapshot collected
	}
	snapshot, snapErr := source.Snapshot(ctx)
	if snapErr != nil {
		return SweepResult{}, false, nil
	}
	if attr == nil {
		attr = NewDockerAttributor()
	}
	now = clockNow() // Evaluate provenance after sensing, never against a pre-probe clock.
	result, err := SweepWithFootprints(ctx, cursor, snapshot, attr, policy, now, resolve)
	if err != nil {
		return SweepResult{}, false, err
	}
	if recErr := cursor.RecordSweepAt(ctx, now); recErr != nil {
		return result, true, recErr
	}
	return result, true, nil
}

// claimRefreshable reports whether a claim is eligible for presence-driven
// heartbeat: an active, GPU-backed, non-op claim. Op claims own their own
// lifecycle; non-vram claims have no GPU-process residency signal in V1.
func claimRefreshable(c CapacityClaim) bool {
	if !IsActiveClaimStatus(c.Status) {
		return false
	}
	if c.OwnerKind == OwnerKindOp {
		return false
	}
	return c.ResourceKind == ResourceKindVRAM
}

// completeGPUProcessProbe distinguishes a successful empty process list from
// unknown or partial sensing. Missing status is not evidence of absence.
func completeGPUProcessProbe(snapshot hostinventory.Snapshot) bool {
	return snapshot.ProbeStatuses["nvidia_gpu"] == "ok" && snapshot.ProbeStatuses["nvidia_gpu_processes"] == "ok"
}

// gpuProcessAbsenceVerified requires a complete, recent observation at or after
// this claim's known liveness/activity boundary. The default heartbeat TTL is
// also the maximum accepted absence-observation age (30s by default); changing
// policy sweep cadence cannot silently expand this safety bound.
func gpuProcessAbsenceVerified(snapshot hostinventory.Snapshot, claim CapacityClaim, now time.Time) bool {
	if !completeGPUProcessProbe(snapshot) {
		return false
	}
	return (AbsenceProof{Generation: claim.Generation, ObservedAt: snapshot.FieldProvenance["gpu_processes"].ObservedAt}).validFor(claim, now)
}

func (proof AbsenceProof) validFor(claim CapacityClaim, now time.Time) bool {
	observed := proof.ObservedAt
	if proof.Generation <= 0 || proof.Generation != claim.Generation || observed.IsZero() || observed.After(now) || now.Sub(observed) > DefaultHeartbeatTTL {
		return false
	}
	boundary := claim.CreatedAt
	if claim.UpdatedAt.After(boundary) {
		boundary = claim.UpdatedAt
	}
	for _, at := range []*time.Time{claim.LastHeartbeatAt, claim.LastActiveAt} {
		if at != nil && at.After(boundary) {
			boundary = *at
		}
	}
	return !observed.Before(boundary)
}
