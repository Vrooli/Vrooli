package capacity

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// OperationReservationOptions reserves peak additional RAM for an isolated
// operation. This remains pending overhead until explicitly released: measured
// RSS cannot tell which part of its peak budget has materialized. No resident
// may be reclaimed to make room for this reservation.
type OperationReservationOptions struct {
	OwnerID, InstanceID string
	AmountBytes         int64
	TTL                 time.Duration
	Admission           AdmitOptions
	Ledger              LedgerReference
}

func ReserveOperationRAM(ctx context.Context, opts OperationReservationOptions) (CapacityClaim, error) {
	if strings.TrimSpace(opts.OwnerID) == "" || strings.TrimSpace(opts.InstanceID) == "" || opts.AmountBytes <= 0 || opts.TTL < time.Minute || opts.TTL > time.Hour {
		return CapacityClaim{}, fmt.Errorf("operation reservation requires identity, positive RAM and a bounded lease")
	}
	store, err := openOperationReservationStore(ctx, opts)
	if err != nil {
		return CapacityClaim{}, err
	}
	defer store.Close()
	release, err := LockAdmission(ctx, store)
	if err != nil {
		return CapacityClaim{}, err
	}
	defer release()
	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return CapacityClaim{}, err
	}
	enforce := opts.Admission.EnforceEnv
	if strings.TrimSpace(enforce) == "" {
		enforce = strings.TrimSpace(os.Getenv(EnvEnforce))
	}
	effective := policy.EffectiveEnforce(enforce)
	if effective == EnforceOff {
		return CapacityClaim{}, fmt.Errorf("operation reservation unavailable: capacity policy is off")
	}
	// Expire lapsed operation leases through the canonical sweep before looking
	// for a replay. A later grant is a new row and a fresh capacity decision,
	// never resurrection of an expired claim.
	now := time.Now().UTC()
	if opts.Admission.Clock != nil {
		now = opts.Admission.Clock().UTC()
	}
	if _, err := store.ExpireStaleClaims(ctx, now, nil); err != nil {
		return CapacityClaim{}, err
	}
	claims, err := store.ListClaims(ctx, ClaimFilter{OwnerKind: OwnerKindOp, OwnerID: opts.OwnerID, ResourceKind: ResourceKindRAM, Statuses: ActiveClaimStatuses()})
	if err != nil {
		return CapacityClaim{}, err
	}
	if len(claims) > 1 {
		return CapacityClaim{}, fmt.Errorf("operation reservation has ambiguous active claims")
	}
	if len(claims) == 1 {
		c := claims[0]
		if c.InstanceID != opts.InstanceID || c.AmountBytes != opts.AmountBytes || c.PreferredBytes != opts.AmountBytes || c.FloorBytes != opts.AmountBytes || !c.Protected || c.Status != StatusReserved || c.ActivityState != ActivityActive {
			return CapacityClaim{}, fmt.Errorf("operation reservation conflicts with existing ownership or budget")
		}
		// HeartbeatClaim refuses resurrection after the lease deadline. Missing
		// admission acknowledgements reconcile the same active row, never a new one.
		return store.HeartbeatClaim(ctx, c.ClaimID, c.Generation, opts.TTL)
	}
	req := CapacityRequest{OwnerKind: OwnerKindOp, OwnerID: opts.OwnerID, InstanceID: opts.InstanceID, ResourceKind: ResourceKindRAM, PreferredBytes: opts.AmountBytes, FloorBytes: opts.AmountBytes, Priority: PriorityInteractive, NoReclaim: true, Protected: true, TTL: opts.TTL}
	v, err := EvaluateAdmission(ctx, store, req, AdmissionOptions{Source: opts.Admission.Source, MemoryPressure: opts.Admission.MemoryPressure, Attributor: opts.Admission.Attributor, Policy: policy, Enforce: effective, Clock: opts.Admission.Clock})
	if err != nil {
		return CapacityClaim{}, err
	}
	if !v.Granted() || v.GrantedBytes != opts.AmountBytes || len(v.ReclaimTargets) != 0 {
		return CapacityClaim{}, fmt.Errorf("operation RAM admission refused: %s", v.Reason)
	}
	c := req.Claim(v)
	c.Status, c.ActivityState = StatusReserved, ActivityActive
	return store.CreateClaim(ctx, c, opts.TTL)
}
