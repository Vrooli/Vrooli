package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/testenv"
)

// An owner that stops heartbeating loses its reservation at the deadline and
// learns that through a typed error, while a still-live owner can complete the
// settled job and attest cleanup.
func TestDiskJobLeaseLapseIsVisibleThroughThePublicBroker(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	clock := testenv.NewClock(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC))
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: filepath.Join(dir, "capacity.db"), Clock: clock, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	broker := NewBrokerWithSource(store, nil)
	lease, err := broker.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "owner", JobID: "job", Paths: []DiskJobPath{{Path: dir, Role: "cache", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(16 * time.Minute)
	if err := lease.Heartbeat(ctx); !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("late heartbeat = %v, want ErrDiskLeaseLapsed", err)
	}
	if _, err := lease.Extend(ctx, []DiskJobPath{{Path: dir, Role: "more", PeakBytes: 1 << 20}}); !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("late extend = %v, want ErrDiskLeaseLapsed", err)
	}
	receipt, err := lease.Complete(ctx)
	if err != nil || receipt.Status != "completed" || !receipt.CleanupVerified {
		t.Fatalf("completing a lapsed lease = %+v, %v", receipt, err)
	}
}
