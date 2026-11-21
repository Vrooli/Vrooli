package capacity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/vrooli/vrooli/internal/testenv"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDiskJobPathSampleSchemaAddsColumnsWithoutDroppingHistory(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE capacity_disk_job_path_samples(sample_id INTEGER PRIMARY KEY,job_lease_id TEXT,canonical_path TEXT,filesystem_id TEXT,bytes INTEGER,baseline_bytes INTEGER,delta_bytes INTEGER,observed_at TEXT); INSERT INTO capacity_disk_job_path_samples VALUES(1,'lease','/out','dev:test',20,10,10,'2026-11-11T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDiskJobPathSampleColumns(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var bytes, baseline, delta, reserved, exceeded int64
	if err := db.QueryRow(`SELECT bytes,baseline_bytes,delta_bytes,reserved_peak_bytes,exceeded_reserved_peak FROM capacity_disk_job_path_samples WHERE sample_id=1`).Scan(&bytes, &baseline, &delta, &reserved, &exceeded); err != nil {
		t.Fatal(err)
	}
	if bytes != 20 || baseline != 10 || delta != 10 || reserved != 1 || exceeded != 1 {
		t.Fatalf("legacy receipt changed or not marked conservative: %d %d %d %d %d", bytes, baseline, delta, reserved, exceeded)
	}
}

func TestDiskJobPathSamplesBindCanonicalPathAndRejectRegression(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	target := filepath.Join(dir, "output")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "output-alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	lease, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "sample-owner", JobID: "sample-job", Paths: []DiskJobPath{{Path: target, Role: "output", PeakBytes: 1024}}})
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, alias, "", "", false, false, 4096, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Paths) != 1 || len(r.Paths[0].Samples) != 1 || r.Paths[0].Samples[0].BaselineBytes != 4096 || r.Paths[0].Samples[0].DeltaBytes != 0 || r.Paths[0].Samples[0].ExceedsPeak {
		t.Fatalf("baseline receipt=%+v", r.Paths)
	}
	r, err = store.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, target, "", "", false, false, 8192, 0)
	if err != nil {
		t.Fatal(err)
	}
	last := r.Paths[0].Samples[len(r.Paths[0].Samples)-1]
	if last.DeltaBytes != 4096 || !last.ExceedsPeak || r.Paths[0].Path != target {
		t.Fatalf("checkpoint receipt=%+v", r.Paths[0])
	}
	if _, err := store.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, target, "", "", false, false, 7000, 0); err == nil {
		t.Fatal("accepted negative checkpoint delta")
	}
	if _, err := store.SampleDiskJobPath(ctx, lease.ID, "wrong-capability", lease.ProcessID, lease.ProcessStart, target, "", "", false, false, 9000, 0); err == nil {
		t.Fatal("accepted stale/incorrect credential")
	}
}

func TestDiskJobPathSampleExtensionIsAtomicAndPreservesFloor(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	target := filepath.Join(dir, "output")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "first", JobID: "sample", Paths: []DiskJobPath{{Path: target, Role: "output", PeakBytes: free / 5}}})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "second", JobID: "contender", Paths: []DiskJobPath{{Path: dir, Role: "cache", PeakBytes: free / 5}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.SampleDiskJobPath(ctx, first.ID, token, first.ProcessID, first.ProcessStart, target, "", "", false, false, 128, free*9/10); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("extension should preserve competitor and floor: %v", err)
	}
	got, err := store.GetDiskJob(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Paths[0].Samples) != 0 || got.Paths[0].PeakBytes != free/5 {
		t.Fatalf("denied sample+extension was not atomic: %+v", got.Paths[0])
	}
}

func TestDiskJobConcurrentPathCheckpointsSerializeWithoutDoubleCounting(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	db := filepath.Join(dir, "capacity.db")
	first, err := NewSQLiteStore(ctx, Config{DBPath: db, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := NewSQLiteStore(ctx, Config{DBPath: db, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	target := filepath.Join(dir, "output")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	lease, token, err := first.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "concurrent-sampler", JobID: "one", Paths: []DiskJobPath{{Path: target, Role: "output", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, store := range []*SQLiteStore{first, second} {
		wg.Add(1)
		go func(s *SQLiteStore) {
			defer wg.Done()
			_, e := s.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, target, "", "", false, false, 4096, 0)
			errs <- e
		}(store)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	got, err := first.GetDiskJob(ctx, lease.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Paths[0].Samples) != 2 || got.Paths[0].Samples[0].BaselineBytes != 4096 || got.Paths[0].Samples[1].DeltaBytes != 0 {
		t.Fatalf("concurrent checkpoint history=%+v", got.Paths[0].Samples)
	}
}

func TestDiskJobPathSampleRefusesCrashedOwnerUntilVerifiedReconciliation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	target := filepath.Join(dir, "output")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDiskJobCrashHelper$")
	cmd.Env = append(os.Environ(), "VROOLI_CAPACITY_CRASH_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	deadline := time.Now().Add(3 * time.Second)
	var processStart string
	for time.Now().Before(deadline) {
		processStart, err = processStartIdentity(cmd.Process.Pid)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("read child process identity: %v", err)
	}
	lease, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "crash-owner", JobID: "crash-job", ProcessID: cmd.Process.Pid, ProcessStart: processStart, Paths: []DiskJobPath{{Path: target, Role: "output", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil { // Wait returns the expected killed-process status.
		cmd.Process = nil
	}
	cmd.Process = nil
	if _, err := store.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, target, "", "", false, false, 4096, 0); err == nil {
		t.Fatal("sample accepted after owner process crashed")
	}
	active, err := store.ListDiskJobs(ctx, true)
	if err != nil || len(active) != 1 || active[0].ID != lease.ID {
		t.Fatalf("crashed lease was not retained: %+v, %v", active, err)
	}
	if _, err := store.ReconcileDiskJob(ctx, lease.ID, false, true); err == nil {
		t.Fatal("reconciled crash without verified process-stop evidence")
	}
	reconciled, err := store.ReconcileDiskJob(ctx, lease.ID, true, true)
	if err != nil {
		t.Fatalf("verified crash reconciliation: %v", err)
	}
	if reconciled.Status != "reconciled" || !reconciled.CleanupVerified {
		t.Fatalf("crash receipt=%+v", reconciled)
	}
}

func TestDiskJobCrashHelper(t *testing.T) {
	if os.Getenv("VROOLI_CAPACITY_CRASH_HELPER") != "1" {
		return
	}
	time.Sleep(time.Hour)
}

func TestDiskJobDeduplicatesCanonicalAliasesAndPersistsActualDelta(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	zeroFloor := int64(0)
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(root, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	r, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "image-tools", JobID: "dedupe-1", Paths: []DiskJobPath{{Path: target, Role: "output", PeakBytes: 4096}, {Path: alias, Role: "output-alias", PeakBytes: 8192}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Paths) != 1 || len(r.Filesystems) != 1 || r.Filesystems[0].PeakBytes != 8192 {
		t.Fatalf("alias receipt = %+v", r)
	}
	if err := os.WriteFile(filepath.Join(target, "proof.bin"), []byte("proof"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CompleteDiskJob(ctx, r.ID, token, r.ProcessID, r.ProcessStart); err != nil {
		t.Fatal(err)
	}
	completed, err := store.GetDiskJob(ctx, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || !completed.CleanupVerified || completed.Filesystems[0].ActualDeltaBytes == nil || completed.Filesystems[0].ActualDeltaBasis != diskJobActualDeltaBasis {
		t.Fatalf("completion receipt = %+v", completed)
	}
}

func TestDiskJobHeartbeatDeadlineDoesNotReleaseActiveReservation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	clock := testenv.NewClock(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), Clock: clock, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	lease, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "first-owner", JobID: "stale-heartbeat", Paths: []DiskJobPath{{Path: dir, Role: "output", PeakBytes: free * 3 / 5}}})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * diskReservationTTL)
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "second-owner", JobID: "must-remain-denied", Paths: []DiskJobPath{{Path: dir, Role: "cache", PeakBytes: free * 3 / 5}}}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("stale heartbeat lease was released: %v", err)
	}
	rows, err := store.ListDiskJobs(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != lease.ID || rows[0].Status != "active" || rows[0].HeartbeatDeadline.After(clock.Now()) {
		t.Fatalf("stale lease state=%+v", rows)
	}
}

func TestDiskJobConcurrentOwnersCannotDoubleAdmitSameFilesystem(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	zeroFloor := int64(0)
	first, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	peak := free * 3 / 5
	stores := []*SQLiteStore{first, second}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i, store := range stores {
		wg.Add(1)
		go func(i int, s *SQLiteStore) {
			defer wg.Done()
			_, _, reserveErr := s.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: fmt.Sprintf("owner-%d", i), JobID: fmt.Sprintf("job-%d", i), FloorBytes: 0, Paths: []DiskJobPath{{Path: dir, Role: "same-cache-filesystem", PeakBytes: peak}}})
			results <- reserveErr
		}(i, store)
	}
	wg.Wait()
	close(results)
	granted, denied := 0, 0
	for reserveErr := range results {
		switch {
		case reserveErr == nil:
			granted++
		case errors.Is(reserveErr, ErrDiskAdmissionDenied):
			denied++
		default:
			t.Fatalf("unexpected concurrent reservation error: %v", reserveErr)
		}
	}
	if granted != 1 || denied != 1 {
		t.Fatalf("same-filesystem concurrent jobs granted=%d denied=%d, want exactly one each", granted, denied)
	}
}

func TestDiskJobReservationIsAtomicAcrossFilesystemsAndRetainsStaleLease(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(root, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	job, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "owner", JobID: "atomic-1", Paths: []DiskJobPath{{Path: root, Role: "cache", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.HeartbeatDiskJob(ctx, job.ID, token, job.ProcessID+1, job.ProcessStart); err == nil {
		t.Fatal("heartbeat from stale/different process identity succeeded")
	}
	if err := store.HeartbeatDiskJob(ctx, job.ID, "wrong-capability", job.ProcessID, job.ProcessStart); err == nil {
		t.Fatal("heartbeat with wrong capability succeeded")
	}
	if _, err := store.ReconcileDiskJob(ctx, job.ID, true, true); err == nil {
		t.Fatal("reconciled a lease while its bound owner process is alive")
	}
	if _, err := store.ExtendDiskJob(ctx, job.ID, token, job.ProcessID, job.ProcessStart, []DiskJobPath{{Path: root, Role: "overflow", PeakBytes: 1 << 60}}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("unsafe extension error = %v, want ErrDiskAdmissionDenied", err)
	}
	if extended, err := store.ExtendDiskJob(ctx, job.ID, token, job.ProcessID, job.ProcessStart, []DiskJobPath{{Path: root, Role: "cache", PeakBytes: 2 << 20}}); err != nil {
		t.Fatalf("safe extension failed: %v", err)
	} else if extended.Filesystems[0].PeakBytes != 2<<20 {
		t.Fatalf("extended receipt peak=%d, want %d", extended.Filesystems[0].PeakBytes, 2<<20)
	}
	rows, err := store.ListDiskJobs(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Status != "active" {
		t.Fatalf("active lease was not retained: %+v", rows)
	}
	if _, err := store.CompleteDiskJob(ctx, job.ID, token, job.ProcessID, job.ProcessStart); err != nil {
		t.Fatal(err)
	}
	_, _, err = store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "owner", JobID: "atomic-2", FloorBytes: 0, Paths: []DiskJobPath{{Path: root, Role: "probe", PeakBytes: 1}}})
	if err != nil && !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("second admission: %v", err)
	}
}

func TestDiskJobEmptyExtensionDoesNotReturnReceiptWithoutAuthorization(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(root, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	job, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "owner", JobID: "empty-extension", Paths: []DiskJobPath{{Path: root, Role: "cache", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, capability string
		pid              int
		start            string
	}{{name: "wrong capability", capability: "wrong-capability", pid: job.ProcessID, start: job.ProcessStart}, {name: "wrong process", capability: token, pid: job.ProcessID + 1, start: job.ProcessStart}} {
		t.Run(tc.name, func(t *testing.T) {
			if receipt, err := store.ExtendDiskJob(ctx, job.ID, tc.capability, tc.pid, tc.start, nil); err == nil {
				t.Fatalf("empty extension returned receipt %+v without validating owner identity", receipt)
			}
		})
	}
}
