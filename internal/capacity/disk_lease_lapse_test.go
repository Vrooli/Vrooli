package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

type lapseFixture struct {
	t     *testing.T
	ctx   context.Context
	dir   string
	clock *testenv.Clock
	store *SQLiteStore
	free  int64
}

func newLapseFixture(t *testing.T) *lapseFixture {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	clock := testenv.NewClock(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC))
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), Clock: clock, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &lapseFixture{t: t, ctx: ctx, dir: dir, clock: clock, store: store, free: free}
}

// reserve admits a job owned by this test process for three fifths of the free
// space, so a second such job fits only if the first stops counting.
func (f *lapseFixture) reserve(owner string) (DiskJobReceipt, string) {
	f.t.Helper()
	job, capability, err := f.store.ReserveDiskJob(f.ctx, DiskJobRequest{OwnerID: owner, JobID: "job", Paths: []DiskJobPath{{Path: f.dir, Role: "cache", PeakBytes: f.free * 3 / 5}}})
	if err != nil {
		f.t.Fatalf("reserve %s: %v", owner, err)
	}
	return job, capability
}

func (f *lapseFixture) admits() bool {
	f.t.Helper()
	_, _, err := f.store.ReserveDiskJob(f.ctx, DiskJobRequest{OwnerID: "probe", JobID: "probe", Paths: []DiskJobPath{{Path: f.dir, Role: "cache", PeakBytes: f.free * 3 / 5}}})
	if err != nil && !errors.Is(err, ErrDiskAdmissionDenied) {
		f.t.Fatal(err)
	}
	return err == nil
}

// A lease is a heartbeat promise, not a process promise: a live owner that
// stopped heartbeating stops counting against admission at its deadline, and
// the receipt says nobody verified cleanup.
func TestLapsedLeaseOfLiveOwnerStopsBlockingAdmissionAndIsSettledUnverified(t *testing.T) {
	f := newLapseFixture(t)
	job, _ := f.reserve("silent-live-owner")
	f.clock.Advance(diskReservationTTL - time.Second)
	if f.admits() {
		t.Fatal("a lease inside its deadline stopped counting")
	}
	f.clock.Advance(2 * time.Second)
	if !f.admits() {
		t.Fatal("a lease past its deadline still blocks admission")
	}
	got, err := f.store.GetDiskJob(f.ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "reconciled" || got.CleanupVerified {
		t.Fatalf("settled job = status %q cleanup_verified=%t, want reconciled/unverified", got.Status, got.CleanupVerified)
	}
}

func TestHeartbeatKeepsALeaseAndNeverRevivesALapsedOne(t *testing.T) {
	f := newLapseFixture(t)
	job, capability := f.reserve("owner")
	pid, start := job.ProcessID, job.ProcessStart
	for i := 0; i < 3; i++ {
		f.clock.Advance(diskReservationTTL - time.Minute)
		if err := f.store.HeartbeatDiskJob(f.ctx, job.ID, capability, pid, start); err != nil {
			t.Fatalf("heartbeat %d inside the deadline: %v", i, err)
		}
	}
	if f.admits() {
		t.Fatal("a heartbeating lease stopped counting")
	}
	f.clock.Advance(diskReservationTTL + time.Second)
	err := f.store.HeartbeatDiskJob(f.ctx, job.ID, capability, pid, start)
	if !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("late heartbeat error = %v, want ErrDiskLeaseLapsed", err)
	}
	// The refusal committed the settle: capacity is free and stays free, the
	// late heartbeat did not take it back.
	if !f.admits() {
		t.Fatal("late heartbeat revived a lapsed lease")
	}
}

func TestExtendAndSampleRefuseALapsedLease(t *testing.T) {
	f := newLapseFixture(t)
	job, capability := f.reserve("owner")
	f.clock.Advance(diskReservationTTL + time.Second)
	if _, err := f.store.ExtendDiskJob(f.ctx, job.ID, capability, job.ProcessID, job.ProcessStart, []DiskJobPath{{Path: f.dir, Role: "more", PeakBytes: 1 << 20}}); !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("extend error = %v, want ErrDiskLeaseLapsed", err)
	}
	if _, err := f.store.SampleDiskJobPath(f.ctx, job.ID, capability, job.ProcessID, job.ProcessStart, f.dir, "", "", false, false, 1, 0); !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("sample error = %v, want ErrDiskLeaseLapsed", err)
	}
	if _, err := f.store.ObserveDiskJob(f.ctx, job.ID, capability, job.ProcessID, job.ProcessStart); !errors.Is(err, ErrDiskLeaseLapsed) {
		t.Fatalf("observe error = %v, want ErrDiskLeaseLapsed", err)
	}
}

// A job that finished and verified cleanup must not fail because its lease
// lapsed first (a slow release, a stalled heartbeat): completion attests
// cleanup and records the measured delta.
func TestLiveOwnerCanCompleteALapsedLease(t *testing.T) {
	f := newLapseFixture(t)
	job, capability := f.reserve("owner")
	f.clock.Advance(diskReservationTTL + time.Second)
	if _, err := f.store.SettleLapsedDiskLeases(f.ctx); err != nil {
		t.Fatal(err)
	}
	got, err := f.store.CompleteDiskJob(f.ctx, job.ID, capability, job.ProcessID, job.ProcessStart)
	if err != nil {
		t.Fatalf("completing a lapsed lease: %v", err)
	}
	if got.Status != "completed" || !got.CleanupVerified {
		t.Fatalf("completed lapsed job = status %q cleanup_verified=%t", got.Status, got.CleanupVerified)
	}
	if _, err := f.store.CompleteDiskJob(f.ctx, job.ID, capability, job.ProcessID, job.ProcessStart); err == nil {
		t.Fatal("a job completed twice")
	}
}

func TestListActiveDiskJobsAfterSettleExcludesLapsedLeases(t *testing.T) {
	f := newLapseFixture(t)
	f.reserve("owner")
	f.clock.Advance(diskReservationTTL + time.Second)
	if _, err := f.store.SettleLapsedDiskLeases(f.ctx); err != nil {
		t.Fatal(err)
	}
	active, err := f.store.ListDiskJobs(f.ctx, true)
	if err != nil || len(active) != 0 {
		t.Fatalf("active jobs after settle = %d (%v)", len(active), err)
	}
}

// The periodic lease-expiry pass settles disk leases, so a ledger nobody
// visits does not keep counting dead reservations.
func TestExpireStaleClaimsSettlesLapsedDiskLeases(t *testing.T) {
	f := newLapseFixture(t)
	job, _ := f.reserve("owner")
	f.clock.Advance(diskReservationTTL + time.Second)
	if _, err := f.store.ExpireStaleClaims(f.ctx, f.clock.Now(), nil); err != nil {
		t.Fatal(err)
	}
	got, err := f.store.GetDiskJob(f.ctx, job.ID)
	if err != nil || got.Status != "reconciled" {
		t.Fatalf("job after expiry pass = %q (%v)", got.Status, err)
	}
}

func TestLapsedLegacyReservationExpiresAndCannotBeHeartbeated(t *testing.T) {
	f := newLapseFixture(t)
	res, err := f.store.ReserveDisk(f.ctx, "legacy-owner", f.dir, f.free*3/5, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.clock.Advance(diskReservationTTL - time.Second)
	if err := f.store.HeartbeatDisk(f.ctx, res.ID); err != nil {
		t.Fatalf("heartbeat inside the deadline: %v", err)
	}
	f.clock.Advance(diskReservationTTL + time.Second)
	if err := f.store.HeartbeatDisk(f.ctx, res.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("late legacy heartbeat error = %v, want ErrNotFound", err)
	}
	if !f.admits() {
		t.Fatal("lapsed legacy reservation still blocks admission")
	}
}
