package capacity

import (
	"context"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// SpecLoader resolves a resource's declared capacity block by name (the seam over
// reading resource.json). Production uses LoadResourceClaimSpec under the source
// root; tests inject a fake. ok is false (nil error) when the resource declares no
// capacity block — those owners are never adopted (reconcile still warns).
type SpecLoader func(resourceName string) (ResourceClaimSpec, bool, error)

// AdoptObservedResidents creates claims for observed-but-unclaimed GPU consumers
// whose resource.json declares a capacity block (claim-on-observe, contract C6).
// This adopts residents that predate the admission hook (e.g. kyutai-stt,
// speaker-verification — up for weeks, never re-admitted) WITHOUT a restart, and
// self-heals any future pre-existing resident into the ledger.
//
// It is, by construction:
//
//   - DECLARED-BLOCK-ONLY: an observed consumer whose owner declares no capacity
//     block is left to reconcile's warn (undeclared consumers are never adopted).
//   - IDEMPOTENT: an owner that already holds an active claim is skipped, so a
//     second sweep is a no-op (no duplicate/stacked claim).
//   - ADVISORY-SAFE: it only records a claim from the declared spec; it never
//     enforces or actuates.
//
// Returns the claims created this pass.
func AdoptObservedResidents(ctx context.Context, store ClaimRepository, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, loadSpec SpecLoader, now time.Time) ([]CapacityClaim, error) {
	if attr == nil || loadSpec == nil {
		return nil, nil
	}
	active, err := store.ListClaims(ctx, ClaimFilter{Statuses: ActiveClaimStatuses()})
	if err != nil {
		return nil, err
	}
	activeVRAM := make([]CapacityClaim, 0, len(active))
	for _, c := range active {
		if c.ResourceKind == ResourceKindVRAM {
			activeVRAM = append(activeVRAM, c)
		}
	}

	var created []CapacityClaim
	adopted := make(map[string]bool) // owners claimed this pass (dedupe within the sweep)
	for _, proc := range snapshot.GPUProcesses {
		if int64(proc.UsedBytes) < policy.TrackingThreshold {
			continue
		}
		a := attr.Attribute(ctx, proc.PID)
		// Already covered by an active claim → nothing to adopt.
		if _, matched := matchClaim(activeVRAM, a, proc); matched {
			continue
		}
		name, spec, ok := resolveDeclaredOwner(a, proc, loadSpec)
		if !ok {
			continue // undeclared observed consumer — reconcile warns, adoption skips
		}
		if adopted[name] || ownerHasActiveClaim(activeVRAM, name) {
			continue
		}
		adopted[name] = true

		req := spec.toRequest(name)
		claim := CapacityClaim{
			OwnerKind:      OwnerKindResource,
			OwnerID:        name,
			ResourceKind:   req.ResourceKind,
			GPUIndex:       req.GPUIndex,
			AmountBytes:    req.PreferredBytes,
			PreferredBytes: req.PreferredBytes,
			FloorBytes:     req.FloorBytes,
			Priority:       req.Priority,
			Protected:      req.Protected,
			YieldWhenIdle:  req.YieldWhenIdle,
			IdleUnloadTTL:  req.IdleUnloadTTL,
			IdleGrace:      req.IdleGrace,
			Status:         StatusGranted,
			DegradeProfile: req.Profile,
		}
		recorded, createErr := store.CreateClaim(ctx, claim, req.TTL)
		if createErr != nil {
			continue // best-effort: one failed adoption never aborts the rest
		}
		created = append(created, recorded)
	}
	return created, nil
}

// resolveDeclaredOwner applies the shared canonical identity guard before
// declaration lookup. Ledger/spec presence cannot arbitrate conflicting sources.
func resolveDeclaredOwner(a Attribution, proc hostinventory.GPUProcess, loadSpec SpecLoader) (string, ResourceClaimSpec, bool) {
	candidates := ownerCandidates(a, proc)
	if len(candidates) != 1 {
		return "", ResourceClaimSpec{}, false
	}
	for name := range candidates {
		spec, ok, err := loadSpec(name)
		if err == nil && ok {
			return name, spec, true
		}
	}
	return "", ResourceClaimSpec{}, false
}

// ownerHasActiveClaim reports whether any active claim's owner matches `name`.
func ownerHasActiveClaim(active []CapacityClaim, name string) bool {
	for _, c := range active {
		if ownerMatches(strings.TrimSpace(c.OwnerID), name) {
			return true
		}
	}
	return false
}
