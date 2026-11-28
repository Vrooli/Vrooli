package localqemu

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity"
)

type fakeResourceLease struct {
	heartbeats int
	activities int
	releases   int
}

func (l *fakeResourceLease) Heartbeat(context.Context) error { l.heartbeats++; return nil }
func (l *fakeResourceLease) Release(context.Context) error   { l.releases++; return nil }
func (l *fakeResourceLease) ReportActivity(context.Context, bool) error {
	l.activities++
	return nil
}

type fakeDiskLease struct {
	receipt    capacity.DiskJobReceipt
	heartbeats int
	releases   int
}

func (l *fakeDiskLease) Heartbeat(context.Context) error { l.heartbeats++; return nil }
func (l *fakeDiskLease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{}, nil
}
func (l *fakeDiskLease) Extend(context.Context, []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *fakeDiskLease) SamplePath(context.Context, string, int64, int64) (capacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *fakeDiskLease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	l.releases++
	return l.receipt, nil
}
func (l *fakeDiskLease) Release(ctx context.Context) error    { _, err := l.Complete(ctx); return err }
func (l *fakeDiskLease) Reservation() capacity.DiskJobReceipt { return l.receipt }

type fakeCapacityBroker struct {
	resource    *fakeResourceLease
	disk        *fakeDiskLease
	request     capacity.DiskJobRequest
	owner       string
	ram         int64
	cpu         int64
	found       bool
	jobs        []capacity.DiskJobReceipt
	diskErr     error
	resourceErr error
	verdict     capacity.Verdict
	releaseRuns int
	acquires    int
	reattaches  int
}

func (b *fakeCapacityBroker) AcquireNoReclaim(_ context.Context, owner string, ram, cpu int64) (capacity.Lease, capacity.Verdict, error) {
	b.acquires++
	b.owner, b.ram, b.cpu = owner, ram, cpu
	verdict := b.verdict
	if verdict.Kind == "" {
		verdict = capacity.Verdict{Kind: "grant"}
	}
	if verdict.Kind != "grant" {
		return nil, verdict, b.resourceErr
	}
	return b.resource, verdict, b.resourceErr
}
func (b *fakeCapacityBroker) ReattachResources(_ context.Context, owner string, ram, cpu int64) (capacity.RenewableLease, bool, error) {
	b.reattaches++
	b.owner, b.ram, b.cpu = owner, ram, cpu
	return b.resource, b.found, nil
}
func (b *fakeCapacityBroker) ReserveDiskJob(_ context.Context, request capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	b.request = request
	if b.diskErr != nil {
		return nil, b.diskErr
	}
	return b.disk, nil
}
func (b *fakeCapacityBroker) ReattachDiskJob(_ context.Context, owner, job, id string) (capacity.DiskJobLease, error) {
	b.reattaches++
	if owner != b.request.OwnerID && b.request.OwnerID != "" {
		return nil, errors.New("unexpected disk owner")
	}
	if id != b.disk.receipt.ID {
		return nil, errors.New("unexpected disk lease id")
	}
	_ = job
	return b.disk, nil
}
func (b *fakeCapacityBroker) GetDiskJob(_ context.Context, owner, id string) (capacity.DiskJobReceipt, error) {
	if owner != b.disk.receipt.OwnerID || id != b.disk.receipt.ID {
		return capacity.DiskJobReceipt{}, errors.New("unexpected disk receipt lookup")
	}
	return b.disk.receipt, nil
}
func (b *fakeCapacityBroker) ListDiskJobs(context.Context, string, bool) ([]capacity.DiskJobReceipt, error) {
	return b.jobs, nil
}
func (b *fakeCapacityBroker) ReconcileDiskJob(context.Context, string, capacity.DiskJobRecoveryEvidence) (capacity.DiskJobReceipt, error) {
	return b.disk.receipt, nil
}
func (b *fakeCapacityBroker) ReleaseRun(context.Context, string, string) error {
	b.releaseRuns++
	return nil
}

func TestBrokerAdmissionAcquiresProfileRAMCPUDiskAndRenews(t *testing.T) {
	resource := &fakeResourceLease{}
	disk := &fakeDiskLease{receipt: capacity.DiskJobReceipt{ID: "lease-qemu-1"}}
	broker := &fakeCapacityBroker{resource: resource, disk: disk}
	gate := NewBrokerAdmission(broker)
	lease, err := gate.Acquire(context.Background(), AdmissionRequest{InstanceID: "qemu-1", Workdir: filepath.Join(t.TempDir(), "guest"), Profile: ProfileHeadlessLinux, Memory: "2048", CPUs: 2, DiskPeak: 3 << 30})
	if err != nil {
		t.Fatal(err)
	}
	if broker.owner != "compute-manager:qemu-1:local-qemu" || broker.ram != 2<<30 || broker.cpu != 2000 {
		t.Fatalf("resource request = owner %q RAM %d CPU %d", broker.owner, broker.ram, broker.cpu)
	}
	if broker.request.OwnerID != capacityOwnerID || broker.request.JobID != "qemu-1" || broker.request.FloorBytes != 120<<30 || len(broker.request.Paths) != 1 || broker.request.Paths[0].PeakBytes != 3<<30 {
		t.Fatalf("disk request = %#v", broker.request)
	}
	if lease.Receipt.DiskLeaseID != "lease-qemu-1" || lease.Receipt.Profile != ProfileHeadlessLinux || lease.Receipt.DiskPath != broker.request.Paths[0].Path {
		t.Fatalf("persistable receipt = %#v", lease.Receipt)
	}
	if err := lease.Heartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if resource.activities != 1 || resource.heartbeats != 1 || resource.releases != 1 || disk.heartbeats != 1 || disk.releases != 1 {
		t.Fatalf("lease calls resource=%#v disk=%#v", resource, disk)
	}
}

func TestBrokerAdmissionRefusesUnknownOrMismatchedProfileBeforeBrokerEffect(t *testing.T) {
	broker := &fakeCapacityBroker{resource: &fakeResourceLease{}, disk: &fakeDiskLease{}}
	gate := NewBrokerAdmission(broker)
	for _, req := range []AdmissionRequest{
		{InstanceID: "qemu-1", Workdir: filepath.Join(t.TempDir(), "guest"), Profile: "other", Memory: "2048", CPUs: 2, DiskPeak: 1 << 30},
		{InstanceID: "qemu-1", Workdir: filepath.Join(t.TempDir(), "guest"), Profile: ProfileHeadlessLinux, Memory: "4096", CPUs: 4, DiskPeak: 1 << 30},
	} {
		if _, err := gate.Acquire(context.Background(), req); err == nil {
			t.Fatalf("Acquire(%#v) accepted unknown/mismatched profile budget", req)
		}
	}
	if broker.acquires != 0 || broker.reattaches != 0 {
		t.Fatalf("profile refusal touched broker: %#v", broker)
	}
}

func TestBrokerAdmissionRefusesRAMCPUAndDiskDenialsBeforeAnyQEMUCreate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		broker *fakeCapacityBroker
	}{
		{name: "RAM CPU unavailable", broker: &fakeCapacityBroker{resourceErr: errors.New("insufficient host capacity"), verdict: capacity.Verdict{Kind: "deny", Reason: "insufficient RAM/CPU"}}},
		{name: "disk floor unavailable", broker: &fakeCapacityBroker{resource: &fakeResourceLease{}, diskErr: errors.New("120 GiB floor denied")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := NewBrokerAdmission(tc.broker)
			_, err := gate.Acquire(context.Background(), AdmissionRequest{InstanceID: "qemu-gate-test", Workdir: filepath.Join(t.TempDir(), "guest"), Profile: ProfileHeadlessLinux, Memory: "2048", CPUs: 2, DiskPeak: 2 << 30})
			if err == nil {
				t.Fatal("capacity denial was accepted")
			}
			if tc.broker.acquires != 1 {
				t.Fatalf("unexpected broker calls after denial: %#v", tc.broker)
			}
			if tc.name == "RAM CPU unavailable" && (tc.broker.request.OwnerID != "" || tc.broker.resource != nil) {
				t.Fatalf("RAM/CPU denial proceeded to disk reservation: %#v", tc.broker)
			}
			if tc.name == "disk floor unavailable" && (tc.broker.request.FloorBytes != 120<<30 || tc.broker.resource.releases != 1) {
				t.Fatalf("disk refusal did not use/release the composite admission: %#v", tc.broker)
			}
		})
	}
}

func TestBrokerAdmissionReleasesResourcesWhenDiskReservationRefuses(t *testing.T) {
	resource := &fakeResourceLease{}
	broker := &fakeCapacityBroker{resource: resource, diskErr: errors.New("disk floor denied")}
	gate := NewBrokerAdmission(broker)
	_, err := gate.Acquire(context.Background(), AdmissionRequest{InstanceID: "qemu-1", Workdir: filepath.Join(t.TempDir(), "guest"), Profile: ProfileHeadlessLinux, Memory: "2048", CPUs: 2, DiskPeak: 2 << 30})
	if err == nil || resource.releases != 1 {
		t.Fatalf("disk refusal err=%v resource releases=%d", err, resource.releases)
	}
}

func TestBrokerAdmissionReattachesExactPersistedPairAndDiskLease(t *testing.T) {
	resource := &fakeResourceLease{}
	disk := &fakeDiskLease{receipt: capacity.DiskJobReceipt{ID: "lease-qemu-2"}}
	disk.receipt = capacity.DiskJobReceipt{ID: "lease-qemu-2", OwnerID: capacityOwnerID, JobID: "qemu-2", Status: "active", Paths: []capacity.DiskJobPathReceipt{{Path: filepath.Join(t.TempDir(), "qemu-2"), PeakBytes: 2 << 30}}, Filesystems: []capacity.DiskJobFilesystem{{FloorBytes: 120 << 30}}}
	broker := &fakeCapacityBroker{resource: resource, disk: disk, found: true, request: capacity.DiskJobRequest{OwnerID: capacityOwnerID}}
	gate := NewBrokerAdmission(broker)
	receipt := CapacityReceipt{ResourceOwnerID: "compute-manager:qemu-2:local-qemu", RAMBytes: 2 << 30, CPUMilli: 2000, DiskOwnerID: capacityOwnerID, DiskJobID: "qemu-2", DiskLeaseID: "lease-qemu-2", DiskPath: disk.receipt.Paths[0].Path, DiskPeakBytes: 2 << 30, DiskFloorBytes: 120 << 30, Profile: ProfileHeadlessLinux}
	lease, err := gate.Reattach(context.Background(), receipt)
	if err != nil {
		t.Fatal(err)
	}
	if lease.Disk == nil || resource.activities != 1 || broker.acquires != 0 {
		t.Fatalf("reattached lease=%#v resource=%#v broker=%#v", lease, resource, broker)
	}
	if err := lease.Heartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if resource.releases != 1 || disk.releases != 1 || broker.releaseRuns != 1 {
		t.Fatalf("reattached lease was not fully released: resource=%#v disk=%#v release runs=%d", resource, disk, broker.releaseRuns)
	}
}
