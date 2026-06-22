package maintenance

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// seedCapacityClaim opens a temp ledger, records one resident VRAM claim with a
// tiny TTL (so it is immediately past its heartbeat deadline), and returns the
// db path + the claim id.
func seedCapacityClaim(t *testing.T, owner string) (string, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := capacity.NewSQLiteStore(context.Background(), capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()
	gpu := 0
	created, err := store.CreateClaim(context.Background(), capacity.CapacityClaim{
		OwnerKind:    capacity.OwnerKindResource,
		OwnerID:      owner,
		ResourceKind: capacity.ResourceKindVRAM,
		GPUIndex:     &gpu,
		AmountBytes:  3 << 30,
		Priority:     capacity.PriorityService,
	}, time.Nanosecond)
	if err != nil {
		t.Fatalf("CreateClaim() error = %v", err)
	}
	return dbPath, created.ClaimID
}

// withCapacitySweepSeams swaps the injectable seams for the duration of a test
// and restores them after.
func withCapacitySweepSeams(t *testing.T, dbPath string, snap hostinventory.Snapshot, snapErr error, attr capacity.Attributor, now time.Time) {
	t.Helper()
	origOpen, origSnap, origAttr, origNow := openCapacityStoreFn, capacitySnapshotFn, capacityAttributorFn, capacityNowFn
	openCapacityStoreFn = func(ctx context.Context, _ string) (capacitySweepStore, error) {
		return capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	}
	capacitySnapshotFn = func(context.Context) (hostinventory.Snapshot, error) { return snap, snapErr }
	capacityAttributorFn = func() capacity.Attributor { return attr }
	capacityNowFn = func() time.Time { return now }
	t.Cleanup(func() {
		openCapacityStoreFn, capacitySnapshotFn, capacityAttributorFn, capacityNowFn = origOpen, origSnap, origAttr, origNow
	})
}

// The maintenance pass expires a dead resident's claim and surfaces it as a
// StopReport item.
func TestSweepCapacityClaimsExpiresDeadResident(t *testing.T) {
	ctx := context.Background()
	dbPath, claimID := seedCapacityClaim(t, "kyutai-stt")
	// Empty snapshot (no observed owner) + a now far past the deadline.
	withCapacitySweepSeams(t, dbPath, hostinventory.Snapshot{}, nil, fakeMaintAttributor{}, time.Now().Add(time.Hour))

	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "kyutai-stt" {
		t.Fatalf("items = %+v, want one expired item for kyutai-stt (claim %s)", items, claimID)
	}
}

// A sensing failure is a clean no-op: the claim is NOT expired even though its
// deadline lapsed (never expire what we cannot verify on the GPU).
func TestSweepCapacityClaimsSensingDownNoExpiry(t *testing.T) {
	ctx := context.Background()
	dbPath, claimID := seedCapacityClaim(t, "whisper")
	withCapacitySweepSeams(t, dbPath, hostinventory.Snapshot{}, errMaintSensing, fakeMaintAttributor{}, time.Now().Add(time.Hour))

	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %+v, want none (sensing down)", items)
	}
	// Confirm the claim survived.
	store, err := capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("reopen store error = %v", err)
	}
	defer store.Close()
	got, err := store.GetClaim(ctx, claimID)
	if err != nil {
		t.Fatalf("GetClaim() error = %v", err)
	}
	if got.Status != capacity.StatusGranted {
		t.Errorf("status = %q, want granted (sensing-down must not expire)", got.Status)
	}
}

// No ledger yet on the host → nothing to sweep, no error.
func TestSweepCapacityClaimsNoLedgerIsNoop(t *testing.T) {
	ctx := context.Background()
	c := NewController(t.TempDir(), t.TempDir())
	items, err := c.sweepCapacityClaims(ctx)
	if err != nil {
		t.Fatalf("sweepCapacityClaims() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %+v, want none (no ledger)", items)
	}
}

type fakeMaintAttributor struct{}

func (fakeMaintAttributor) Attribute(context.Context, int) capacity.Attribution {
	return capacity.Attribution{OwnerID: capacity.OwnerUnknown}
}

var errMaintSensing = maintSensingError("nvidia-smi unavailable")

type maintSensingError string

func (e maintSensingError) Error() string { return string(e) }
