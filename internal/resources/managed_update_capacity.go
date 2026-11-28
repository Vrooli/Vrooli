package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
)

const (
	managedUpdateRoleMemory  = 512 << 20
	managedUpdatePeakRoles   = 3 // qualification + target candidate + primary/restoration
	managedUpdateCapacityTTL = 10 * time.Minute
)

// This is an immutable budget declaration, not a durable claim of free RAM.
// Current lease authority lives in the canonical capacity ledger. After a
// lapse, a replacement row requires a fresh no-reclaim decision.
type ManagedUpdateCapacityBudget struct {
	OwnerID     string                   `json:"owner_id"`
	InstanceID  string                   `json:"instance_id"`
	AmountBytes int64                    `json:"amount_bytes"`
	LedgerPath  capacity.LedgerReference `json:"ledger_path"`
}

func managedUpdateCapacityBudget(u ManagedServiceUpdate) ManagedUpdateCapacityBudget {
	return ManagedUpdateCapacityBudget{OwnerID: "resource-update:" + u.Resource + ":" + u.ID, InstanceID: u.ID, AmountBytes: managedUpdatePeakRoles * managedUpdateRoleMemory}
}

func (b ManagedUpdateCapacityBudget) validate(u ManagedServiceUpdate) error {
	expected := managedUpdateCapacityBudget(u)
	if b.OwnerID != expected.OwnerID || b.InstanceID != expected.InstanceID || b.AmountBytes != expected.AmountBytes || b.LedgerPath.Validate() != nil {
		return fmt.Errorf("update capacity budget differs from its exact operation and peak role ceilings")
	}
	return nil
}

func reserveManagedUpdateCapacity(ctx context.Context, u ManagedServiceUpdate) error {
	if u.Capacity == nil {
		if nativeOpenRGBUsesPrivateLocks(u.Resource) {
			return fmt.Errorf("native OpenRGB update has no admitted RAM budget")
		}
		return nil // Preserve legacy Linux admitted records.
	}
	if err := u.Capacity.validate(u); err != nil {
		return err
	}
	_, err := capacity.ReserveOperationRAM(ctx, managedUpdateCapacityOptionsFor(u))
	return err
}
func verifyManagedUpdateCapacity(ctx context.Context, u ManagedServiceUpdate) error {
	if u.Capacity == nil {
		if nativeOpenRGBUsesPrivateLocks(u.Resource) {
			return fmt.Errorf("native OpenRGB update has no admitted RAM budget")
		}
		return nil
	}
	if err := u.Capacity.validate(u); err != nil {
		return err
	}
	return capacity.VerifyOperationRAM(ctx, managedUpdateCapacityOptionsFor(u), 30*time.Second)
}

func releaseManagedUpdateCapacity(ctx context.Context, u ManagedServiceUpdate) error {
	if u.Capacity == nil {
		return nil
	}
	if !u.Phase.Terminal() {
		return fmt.Errorf("only a terminal update can release its capacity reservation")
	}
	return releaseManagedUpdateCapacityClaims(ctx, u)
}

// The admission owner may also release a rejected operation only after proving
// it never entered the durable ledger. Serving cleanup must use the terminal
// wrapper above; a caller error is not evidence that native effects stopped.
func releaseManagedUpdateCapacityClaims(ctx context.Context, u ManagedServiceUpdate) error {
	if u.Capacity == nil {
		return nil
	}
	if err := u.Capacity.validate(u); err != nil {
		return err
	}

	return capacity.ReleaseOperationRAM(ctx, managedUpdateCapacityOptionsFor(u))
}

func managedUpdateCapacityOptions(b ManagedUpdateCapacityBudget) capacity.OperationReservationOptions {
	return capacity.OperationReservationOptions{OwnerID: b.OwnerID, InstanceID: b.InstanceID, AmountBytes: b.AmountBytes, TTL: managedUpdateCapacityTTL, Ledger: b.LedgerPath}
}
func managedUpdateCapacityOptionsFor(u ManagedServiceUpdate) capacity.OperationReservationOptions {
	return managedUpdateCapacityOptions(*u.Capacity)
}
