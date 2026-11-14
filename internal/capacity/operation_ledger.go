package capacity

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// LedgerReference is an opaque reference to one operator's capacity authority.
// Consumers retain it across detached execution but do not open its storage or
// interpret its backend. Existing file references remain byte-for-byte stable.
type LedgerReference string

func DefaultLedgerReference(home string) (LedgerReference, error) {
	path, err := DefaultDBPath(home)
	return LedgerReference(path), err
}

func (r LedgerReference) Validate() error {
	path := string(r)
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) {
		return fmt.Errorf("capacity ledger reference is not canonical")
	}
	return nil
}

func openOperationReservationStore(ctx context.Context, opts OperationReservationOptions) (AdmitStore, error) {
	if opts.Ledger == "" {
		return openAdmitStore(ctx, opts.Admission)
	}
	if err := opts.Ledger.Validate(); err != nil {
		return nil, err
	}
	if opts.Admission.OpenStore != nil {
		return nil, fmt.Errorf("capacity reservation cannot override its retained ledger authority")
	}
	cfg := Config{DBPath: string(opts.Ledger)}
	if opts.Admission.Clock != nil {
		cfg.Clock = clockAdapter(opts.Admission.Clock)
	}
	return NewSQLiteStore(ctx, cfg)
}

func validOperationReservationIdentity(opts OperationReservationOptions) bool {
	return strings.TrimSpace(opts.OwnerID) != "" && strings.TrimSpace(opts.InstanceID) != "" && opts.AmountBytes > 0
}

// VerifyOperationRAM observes exact current lease authority; it never renews or
// reacquires a grant on behalf of a wrapper that may already be executing.
func VerifyOperationRAM(ctx context.Context, opts OperationReservationOptions, minimumRemaining time.Duration) error {
	if !validOperationReservationIdentity(opts) || minimumRemaining < 0 || minimumRemaining > time.Hour {
		return fmt.Errorf("invalid operation capacity verification identity or freshness budget")
	}
	store, err := openOperationReservationStore(ctx, opts)
	if err != nil {
		return err
	}
	defer store.Close()
	claims, err := store.ListClaims(ctx, ClaimFilter{OwnerKind: OwnerKindOp, OwnerID: opts.OwnerID, ResourceKind: ResourceKindRAM, Statuses: ActiveClaimStatuses()})
	if err != nil {
		return err
	}
	if len(claims) != 1 {
		return fmt.Errorf("operation has no unique live capacity reservation")
	}
	c := claims[0]
	now := time.Now()
	if opts.Admission.Clock != nil {
		now = opts.Admission.Clock()
	}
	if c.InstanceID != opts.InstanceID || c.AmountBytes != opts.AmountBytes || c.PreferredBytes != opts.AmountBytes || c.FloorBytes != opts.AmountBytes || c.Status != StatusReserved || !c.Protected || c.Priority != PriorityInteractive || c.ActivityState != ActivityActive || c.YieldWhenIdle || c.HeartbeatDeadlineAt == nil || !c.HeartbeatDeadlineAt.After(now.Add(minimumRemaining)) {
		return fmt.Errorf("operation capacity reservation is stale or differs from its admitted budget")
	}
	return nil
}

// ReleaseOperationRAM releases only this exact operation's claims. Its domain
// must first prove completion or certain non-admission; a failed caller alone
// is never that proof.
func ReleaseOperationRAM(ctx context.Context, opts OperationReservationOptions) error {
	if !validOperationReservationIdentity(opts) {
		return fmt.Errorf("operation capacity release requires exact owner and budget")
	}
	store, err := openOperationReservationStore(ctx, opts)
	if err != nil {
		return err
	}
	defer store.Close()
	claims, err := store.ListClaims(ctx, ClaimFilter{OwnerKind: OwnerKindOp, OwnerID: opts.OwnerID, ResourceKind: ResourceKindRAM, Statuses: ActiveClaimStatuses()})
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if claim.InstanceID != opts.InstanceID || claim.AmountBytes != opts.AmountBytes {
			return fmt.Errorf("refuse to release conflicting operation capacity owner")
		}
	}
	for _, claim := range claims {
		if _, err := store.ReleaseClaim(ctx, claim.ClaimID); err != nil {
			return err
		}
	}
	return nil
}
