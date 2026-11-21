package capacity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// metadataStore is a ledger whose install-sized floor no real filesystem can
// satisfy (the host is inside its floor) and whose metadata floor is zero, so a
// test exercises the class rules without assuming any host free space.
func metadataStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	impossible, none := int64(1)<<60, int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &impossible, MetadataDiskFloorBytes: &none})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store, dir
}

func TestMetadataDiskJobIsAdmittedWhereAnInstallSizedJobIsRefused(t *testing.T) {
	store, dir := metadataStore(t)
	ctx := context.Background()
	paths := []DiskJobPath{{Path: dir, Role: "lockfile", PeakBytes: 4 << 20}}

	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "install", Paths: paths}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("an ordinary job inside the host floor must be refused, got %v", err)
	}
	receipt, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "lock", Paths: paths, Class: DiskJobClassMetadata})
	if err != nil {
		t.Fatalf("a metadata job must be admitted against its own floor: %v", err)
	}
	if receipt.Status != "active" {
		t.Fatalf("receipt = %+v", receipt)
	}
	// A live metadata job lowers nobody else's floor.
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "install-2", Paths: paths}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("an active metadata job must not loosen the floor for an ordinary job, got %v", err)
	}
}

func TestMetadataDiskJobStillHonoursItsOwnFloor(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	zero, impossible := int64(0), int64(1)<<60
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &zero, MetadataDiskFloorBytes: &impossible})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _, err = store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "lock", Paths: []DiskJobPath{{Path: dir, Role: "lockfile", PeakBytes: 1 << 20}}, Class: DiskJobClassMetadata})
	if !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("a host truly out of space must still refuse a metadata job, got %v", err)
	}
}

func TestMetadataDiskJobCannotDeclareOrGrowBeyondItsBound(t *testing.T) {
	store, dir := metadataStore(t)
	ctx := context.Background()

	for name, paths := range map[string][]DiskJobPath{
		"one path over the bound":        {{Path: dir, Role: "big", PeakBytes: DiskJobMetadataMaxPeakBytes + 1}},
		"several paths summing over it":  {{Path: dir, Role: "a", PeakBytes: DiskJobMetadataMaxPeakBytes}, {Path: filepath.Join(dir, "other"), Role: "b", PeakBytes: 1}},
		"a path with no declared growth": {{Path: dir, Role: "none"}},
	} {
		if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: name, Paths: paths, Class: DiskJobClassMetadata}); !errors.Is(err, ErrInvalidClaim) {
			t.Errorf("%s: err = %v, want ErrInvalidClaim", name, err)
		}
	}
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "odd", Paths: []DiskJobPath{{Path: dir, Role: "x", PeakBytes: 1}}, Class: "tiny"}); !errors.Is(err, ErrInvalidClaim) {
		t.Errorf("an unknown class must be refused, got %v", err)
	}

	lease, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "grow", Paths: []DiskJobPath{{Path: dir, Role: "lockfile", PeakBytes: 1 << 20}}, Class: DiskJobClassMetadata})
	if err != nil {
		t.Fatal(err)
	}
	extra := t.TempDir()
	if _, err := store.ExtendDiskJob(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, []DiskJobPath{{Path: extra, Role: "tarballs", PeakBytes: DiskJobMetadataMaxPeakBytes}}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("extending a metadata job past its bound must be refused, got %v", err)
	}
	if _, err := store.SampleDiskJobPath(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, dir, "", "", false, false, 1<<20, DiskJobMetadataMaxPeakBytes+1); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("a measured extension past the bound must be refused, got %v", err)
	}
	if _, err := store.ExtendDiskJob(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart, []DiskJobPath{{Path: extra, Role: "more", PeakBytes: 1 << 20}}); err != nil {
		t.Fatalf("growth inside the bound is allowed: %v", err)
	}
}

func TestDefaultFloorsStayInstallSizedAndMetadataStaysBelowThem(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.diskFloorMinimum != 120<<30 || store.diskMetadataFloor != DiskJobMetadataFloorBytes {
		t.Fatalf("production floors changed: install %d, metadata %d", store.diskFloorMinimum, store.diskMetadataFloor)
	}
	if store.diskMetadataFloor >= store.diskFloorMinimum {
		t.Fatal("the metadata floor must stay below the install floor, or the class means nothing")
	}
}

// Other jobs' floors protect headroom for their own bulk writes. A bounded
// metadata write is held to its own floor, so a host already inside an active
// install's floor still admits it, while a second install is still refused.
func TestMetadataDiskJobIgnoresOtherJobsFloorsButStillCountsInTheirAdmission(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	_, _, free, err := diskFilesystem(dir)
	if err != nil {
		t.Skipf("probe free space: %v", err)
	}
	const margin = 2 << 30
	if free < 2*margin {
		t.Skip("not enough free space to build the scenario")
	}
	min, none := free-margin, int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(dir, "capacity.db"), MinimumDiskFloorBytes: &min, MetadataDiskFloorBytes: &none})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "install", Paths: []DiskJobPath{{Path: dir, Role: "store", PeakBytes: 1 << 30}}}); err != nil {
		t.Fatalf("the first install must fit: %v", err)
	}
	if _, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "install-2", Paths: []DiskJobPath{{Path: dir, Role: "store", PeakBytes: 3 << 29}}}); !errors.Is(err, ErrDiskAdmissionDenied) {
		t.Fatalf("a second install inside the active install's floor must be refused, got %v", err)
	}
	lease, token, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "o", JobID: "lock", Paths: []DiskJobPath{{Path: dir, Role: "lockfile", PeakBytes: 4 << 20}}, Class: DiskJobClassMetadata})
	if err != nil {
		t.Fatalf("a metadata job is held to its own floor, not the active install's: %v", err)
	}
	headroom, err := store.ObserveDiskJob(ctx, lease.ID, token, lease.ProcessID, lease.ProcessStart)
	if err != nil || len(headroom.Filesystems) != 1 || headroom.Filesystems[0].FloorBytes != 0 || headroom.Filesystems[0].OtherReservedBytes != 1<<30 {
		t.Fatalf("observation = %+v, %v", headroom, err)
	}
}
