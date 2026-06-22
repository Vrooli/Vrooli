package capacity

import (
	"context"
	"strings"
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
	if attr == nil {
		attr = unknownAttributor{}
	}
	active, err := store.ListClaims(ctx, ClaimFilter{Statuses: ActiveClaimStatuses()})
	if err != nil {
		return SweepResult{}, err
	}

	var result SweepResult
	for _, claim := range active {
		if !claimRefreshable(claim) {
			continue
		}
		if !claimObserved(ctx, claim, snapshot, attr) {
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

	expired, err := store.ExpireStaleClaims(ctx, now)
	if err != nil {
		return result, err
	}
	result.Expired = expired
	return result, nil
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

// claimObserved reports whether any observed GPU process on the claim's GPU
// attributes to the claim's owner (loose match, mirroring reconcile's
// matchClaim).
func claimObserved(ctx context.Context, claim CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor) bool {
	owner := strings.TrimSpace(claim.OwnerID)
	if owner == "" {
		return false
	}
	for _, proc := range snapshot.GPUProcesses {
		idx := proc.GPUIndex
		if !sameGPU(claim.GPUIndex, &idx) {
			continue
		}
		a := attr.Attribute(ctx, proc.PID)
		for _, cand := range []string{a.OwnerID, NormalizeOwnerName(a.ContainerName), strings.TrimPrefix(a.ContainerName, "/")} {
			cand = strings.TrimSpace(cand)
			if cand == "" || cand == OwnerUnknown {
				continue
			}
			if ownerMatches(owner, cand) {
				return true
			}
		}
	}
	return false
}
