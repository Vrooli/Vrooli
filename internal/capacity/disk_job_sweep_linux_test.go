//go:build linux

package capacity

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// An owner whose process is gone keeps its lease until the deadline (detached
// children of the producer may still be writing); at the deadline it is
// settled like any lapsed lease and its receipt says cleanup was not verified.
func TestExitedProducerLeaseIsHeldUntilItsDeadlineThenSettledUnverified(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	clock := testenv.NewClock(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC))
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), Clock: clock, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	producer := exec.Command("sleep", "30")
	if err := producer.Start(); err != nil {
		t.Fatal(err)
	}
	pid := producer.Process.Pid
	start, err := processStartIdentity(pid)
	if err != nil {
		t.Fatal(err)
	}
	job, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "dead-owner", JobID: "j1", ProcessID: pid, ProcessStart: start, Paths: []DiskJobPath{{Path: dir, Role: "output", PeakBytes: free * 3 / 5}}})
	if err != nil {
		t.Fatal(err)
	}
	big := DiskJobRequest{OwnerID: "next-owner", JobID: "j2", Paths: []DiskJobPath{{Path: dir, Role: "cache", PeakBytes: free * 3 / 5}}}
	if err := producer.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = producer.Wait()

	clock.Advance(diskReservationTTL - time.Minute)
	if _, _, err := store.ReserveDiskJob(ctx, big); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("an exited owner's lease was released before its deadline: %v", err)
	}
	clock.Advance(2 * time.Minute)
	if _, _, err := store.ReserveDiskJob(ctx, big); err != nil {
		t.Fatalf("lapsed lease still blocks admission: %v", err)
	}
	got, err := store.GetDiskJob(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "reconciled" || got.CleanupVerified {
		t.Fatalf("settled job receipt = status %q cleanup_verified=%t, want reconciled with cleanup unverified", got.Status, got.CleanupVerified)
	}
}
