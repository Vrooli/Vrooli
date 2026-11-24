package capacity

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/testenv"
)

// slowSource models a loaded host: every snapshot takes a fixed time.
type slowSource struct {
	delay time.Duration
	inv   hostinventory.Snapshot
}

func (s slowSource) Snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	select {
	case <-time.After(s.delay):
		return s.inv, nil
	case <-ctx.Done():
		return hostinventory.Snapshot{}, ctx.Err()
	}
}

// longestAdmissionLockHold measures, by polling, the longest continuous stretch
// during which another party could not take the admission lock for dbPath while
// run executes.
func longestAdmissionLockHold(t *testing.T, store *SQLiteStore, run func()) time.Duration {
	t.Helper()
	var (
		mu      sync.Mutex
		longest time.Duration
	)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var blockedSince time.Time
		for {
			select {
			case <-stop:
				return
			default:
			}
			probe, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
			release, err := store.LockAdmission(probe)
			cancel()
			now := time.Now()
			if err == nil {
				release()
				blockedSince = time.Time{}
			} else {
				if blockedSince.IsZero() {
					blockedSince = now
				}
				mu.Lock()
				if d := now.Sub(blockedSince); d > longest {
					longest = d
				}
				mu.Unlock()
			}
			time.Sleep(time.Millisecond)
		}
	}()
	run()
	close(stop)
	<-done
	mu.Lock()
	defer mu.Unlock()
	return longest
}

// A strict no-reclaim startup must not hold the fleet-wide admission lock while
// it senses the host. Sensing here takes 600 ms; the lock may be held only for
// the ledger read, decision and write.
func TestNoReclaimAdmissionDoesNotHoldAdmissionLockWhileSensing(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeResourceManifest(t, root, "model", `{"acceleration":{"claim":{"default_preferred_bytes":1073741824,"floor_bytes":1073741824,"default_priority":"batch"}}}`)
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	clk := func() time.Time { return time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC) }
	open := func(ctx context.Context) (AdmitStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clockAdapter(clk)})
	}
	probe, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clockAdapter(clk)})
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	const sensing = 600 * time.Millisecond
	opts := AdmitOptions{Root: root, ResourceName: "model", NoReclaim: true, OpenStore: open, Clock: clk, EnforceEnv: EnforceAdvisory, Source: slowSource{delay: sensing, inv: snapshotWith(16, 0)}}
	var admitErr error
	started := time.Now()
	hold := longestAdmissionLockHold(t, probe, func() {
		_, admitErr = AdmitResource(ctx, opts)
	})
	if admitErr != nil {
		t.Fatal(admitErr)
	}
	if elapsed := time.Since(started); elapsed < sensing {
		t.Fatalf("admission returned in %v, before sensing finished: the test no longer exercises sensing", elapsed)
	}
	if hold > sensing/3 {
		t.Fatalf("admission lock was unavailable for %v during a %v sensing pass: it is being held across sensing", hold, sensing)
	}
	t.Logf("admission lock longest hold %v while sensing took %v", hold, sensing)
}

func TestSenseThenLockSensesBeforeTheLockAndRedoesAgedSensing(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	opts := AdmissionOptions{Source: slowSource{delay: 300 * time.Millisecond, inv: snapshotWith(16, 0)}}
	hold := longestAdmissionLockHold(t, store, func() {
		sensing, release, err := SenseThenLock(ctx, store, opts)
		if err != nil {
			t.Error(err)
			return
		}
		defer release()
		if age := time.Since(sensing.SensedAt()); age > MaxAdmissionSensingAge {
			t.Errorf("sensing handed to the decision is %v old", age)
		}
	})
	if hold > 100*time.Millisecond {
		t.Fatalf("lock unavailable for %v while sensing", hold)
	}
}

// Heartbeat and completion of a disk job decide nothing and must not queue
// behind the admission lock: a release that cannot get the lock inside its
// short bounded context failed jobs that had already succeeded.
func TestDiskJobHeartbeatAndCompleteDoNotWaitForAdmissionLock(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	clock := testenv.NewClock(time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC))
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), Clock: clock, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	job, capability, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "owner", JobID: "j", Paths: []DiskJobPath{{Path: dir, Role: "output", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	release, err := store.LockAdmission(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	short := func() (context.Context, context.CancelFunc) { return context.WithTimeout(ctx, 2*time.Second) }
	hbCtx, cancel := short()
	defer cancel()
	if err := store.HeartbeatDiskJob(hbCtx, job.ID, capability, job.ProcessID, job.ProcessStart); err != nil {
		t.Fatalf("heartbeat queued behind the admission lock: %v", err)
	}
	doneCtx, cancelDone := short()
	defer cancelDone()
	if _, err := store.CompleteDiskJob(doneCtx, job.ID, capability, job.ProcessID, job.ProcessStart); err != nil {
		t.Fatalf("completion queued behind the admission lock: %v", err)
	}

	// An admission, by contrast, is a decision and must still wait for it.
	blocked, cancelBlocked := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancelBlocked()
	if _, _, err := store.ReserveDiskJob(blocked, DiskJobRequest{OwnerID: "owner", JobID: "j2", Paths: []DiskJobPath{{Path: dir, Role: "output", PeakBytes: 1 << 20}}}); err == nil {
		t.Fatal("a new admission proceeded while another party held the admission lock")
	}
}

// Calibration: the probe really does see a held lock, so a zero reading above
// means the lock was not held rather than that it could not be observed.
func TestLongestAdmissionLockHoldSeesAHeldLock(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	// The holder uses a second store: the in-process gate is keyed by path, so
	// it blocks the probe exactly as another process would.
	hold := longestAdmissionLockHold(t, store, func() {
		release, err := store.LockAdmission(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(300 * time.Millisecond)
		release()
	})
	if hold < 200*time.Millisecond {
		t.Fatalf("probe reported %v for a lock held 300 ms", hold)
	}
}
