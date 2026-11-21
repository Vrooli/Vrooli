package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestDiskReservationsAreAtomicAcrossStoreHandles(t *testing.T) {
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
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, store := range []*SQLiteStore{first, second} {
		wg.Add(1)
		go func(s *SQLiteStore) {
			defer wg.Done()
			_, reserveErr := s.ReserveDisk(ctx, "test-owner", dir, peak, 0)
			results <- reserveErr
		}(store)
	}
	wg.Wait()
	close(results)
	granted, denied := 0, 0
	for err := range results {
		switch {
		case err == nil:
			granted++
		case errors.Is(err, ErrDiskAdmissionDenied):
			denied++
		default:
			t.Fatalf("unexpected reservation result: %v", err)
		}
	}
	if granted != 1 || denied != 1 {
		t.Fatalf("concurrent reservations granted=%d denied=%d, want one each", granted, denied)
	}
}

func TestDiskReservationHeartbeatAndRelease(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	reservation, err := store.ReserveDisk(ctx, "owner", dir, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.HeartbeatDisk(ctx, reservation.ID); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if err := store.ReleaseDisk(ctx, reservation.ID); err != nil {
		t.Fatalf("release: %v", err)
	}
	if err := store.HeartbeatDisk(ctx, reservation.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("heartbeat after release = %v, want ErrNotFound", err)
	}
}

func TestListDiskHeadroomRefreshesAggregateAcrossLeaseTypes(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(root, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	jobPeak, legacyPeak, floor := int64(2<<20), int64(3<<20), int64(4<<20)
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{
		OwnerID: "job-owner", JobID: "headroom-job", FloorBytes: 2 << 20,
		Paths: []DiskJobPath{{Path: root, Role: "job-write", PeakBytes: jobPeak}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReserveDisk(ctx, "legacy-owner", root, legacyPeak, floor); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.ListDiskHeadroom(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ObservedAt.IsZero() || len(snapshot.Filesystems) != 1 {
		t.Fatalf("headroom snapshot = %+v", snapshot)
	}
	filesystem := snapshot.Filesystems[0]
	if filesystem.ActiveReservedBytes != jobPeak+legacyPeak || filesystem.FloorBytes != floor || filesystem.CurrentFreeBytes <= 0 {
		t.Fatalf("refreshed aggregate filesystem = %+v", filesystem)
	}
	if got, want := filesystem.SafeAdditionalPeakBytes, filesystem.CurrentFreeBytes-filesystem.ActiveReservedBytes-filesystem.FloorBytes; got != want {
		t.Fatalf("safe additional peak = %d, want current free minus all active peaks and floor: %d", got, want)
	}
	store.diskFloorMinimum = filesystem.CurrentFreeBytes - filesystem.ActiveReservedBytes + 10
	belowFloor, err := store.ListDiskHeadroom(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := belowFloor.Filesystems[0].SafeAdditionalPeakBytes; got != -10 {
		t.Fatalf("under-floor observation = %d, want negative remaining headroom", got)
	}
}

func TestLegacyDiskAdmissionCountsMultiFilesystemJobLeases(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Fatal(err)
	}
	peak := free * 3 / 5
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{
		OwnerID: "job-owner", JobID: "job-1", FloorBytes: 0,
		Paths: []DiskJobPath{{Path: dir, Role: "staging", PeakBytes: peak}},
	}); err != nil {
		t.Fatalf("reserve job lease: %v", err)
	}
	if _, err := store.ReserveDisk(ctx, "legacy-owner", dir, peak, 0); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("legacy scalar admission ignored active job lease: %v", err)
	}
}
