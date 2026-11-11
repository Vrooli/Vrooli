package ai

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image-tools/internal/jobs"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

type fakeDiskAdmission struct {
	leases  []*fakeDiskLease
	request sharedcapacity.DiskJobRequest
	fail    bool
}

func (f *fakeDiskAdmission) ReserveDiskJob(_ context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	f.request = request
	if f.fail {
		return nil, errors.New("disk owner refused")
	}
	l := &fakeDiskLease{reservation: sharedcapacity.DiskJobReceipt{ID: request.JobID, OwnerID: request.OwnerID, JobID: request.JobID, Status: "active", Filesystems: []sharedcapacity.DiskJobFilesystem{{FilesystemID: "output-fs", PeakBytes: request.Paths[0].PeakBytes, AvailableBytes: 1 << 40}, {FilesystemID: "temp-fs", PeakBytes: request.Paths[1].PeakBytes, AvailableBytes: 1 << 40}}}}
	f.leases = append(f.leases, l)
	return l, nil
}

type fakeDiskLease struct {
	reservation sharedcapacity.DiskJobReceipt
	released    int
}

func (l *fakeDiskLease) Reservation() sharedcapacity.DiskJobReceipt { return l.reservation }
func (*fakeDiskLease) Heartbeat(context.Context) error              { return nil }
func (l *fakeDiskLease) SamplePath(_ context.Context, _ string, _, _ int64) (sharedcapacity.DiskJobReceipt, error) {
	return l.reservation, nil
}
func (*fakeDiskLease) Observe(context.Context) (sharedcapacity.DiskJobHeadroom, error) {
	return sharedcapacity.DiskJobHeadroom{}, nil
}
func (l *fakeDiskLease) Release(ctx context.Context) error { _, err := l.Complete(ctx); return err }
func (l *fakeDiskLease) Complete(context.Context) (sharedcapacity.DiskJobReceipt, error) {
	l.released++
	l.reservation.Status = "completed"
	return l.reservation, nil
}
func (l *fakeDiskLease) Extend(context.Context, []sharedcapacity.DiskJobPath) (sharedcapacity.DiskJobReceipt, error) {
	return l.reservation, nil
}

func TestEstimateDiskPeakBoundsGenerationAndInputs(t *testing.T) {
	output, temp, err := estimateDiskPeak(Payload{Params: map[string]string{"width": "512", "height": "512"}})
	if err != nil {
		t.Fatal(err)
	}
	if output != 512*512*4*5/4 || temp != output*2+(128<<20) {
		t.Fatalf("text-to-image estimate output=%d temp=%d", output, temp)
	}
	output, temp, err = estimateDiskPeak(Payload{InputKey: "input", MaskKey: "mask", Variations: 2})
	if err != nil {
		t.Fatal(err)
	}
	if output != 128_000_000*4*2*5/4 || temp <= output*2 {
		t.Fatalf("bounded image-edit estimate output=%d temp=%d", output, temp)
	}
}

func TestEstimateDiskPeakRejectsUnboundedDimensions(t *testing.T) {
	for _, params := range []map[string]string{
		{"width": "30000", "height": "30000"},
		{"width": "-1", "height": "512"},
		{"width": "9223372036854775807", "height": "2"},
	} {
		if _, _, err := estimateDiskPeak(Payload{Params: params}); err == nil {
			t.Fatalf("accepted unbounded disk estimate params=%v", params)
		}
	}
}

func TestEstimateDiskPeakEnforcesOwnerPlanningCeiling(t *testing.T) {
	_, _, err := estimateDiskPeak(Payload{
		Variations: 15,
		Params:     map[string]string{"width": "128000", "height": "1000"},
	})
	if err == nil {
		t.Fatal("accepted generated-output peak above the 25 GiB owner ceiling")
	}
}

func TestAdmissionAtomicallyReservesAllDiskPathsAndCompletesLease(t *testing.T) {
	engine, modelID := newCapacityTestEngine(t, "text_to_image", &fakeProvider{}, &fakeBroker{})
	disk := &fakeDiskAdmission{}
	engine.deps.Disk = disk
	raw, err := marshalPinnedPayload(t, engine, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID, Params: map[string]string{"width": "512", "height": "512"}})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := engine.Admit(context.Background(), jobs.Job{ID: "disk-admitted", Operation: "text_to_image", Payload: raw})
	if err != nil {
		t.Fatal(err)
	}
	if len(disk.leases) != 1 || len(disk.request.Paths) != 2 || disk.request.Paths[0].PeakBytes <= 0 || disk.request.Paths[1].PeakBytes <= disk.request.Paths[0].PeakBytes {
		t.Fatalf("disk request leases=%d request=%+v", len(disk.leases), disk.request)
	}
	if admission.Meta["disk_job_reservation_id"] == "" || admission.Meta["disk_job_filesystems"] != "2" {
		t.Fatalf("disk receipts missing from job metadata: %+v", admission.Meta)
	}
	if err := admission.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if disk.leases[0].released != 1 || disk.leases[0].reservation.Status != "completed" {
		t.Fatalf("disk release=%d status=%s", disk.leases[0].released, disk.leases[0].reservation.Status)
	}
}

func TestAdmissionDoesNotAcquirePartialDiskLeasesWhenAtomicJobRefuses(t *testing.T) {
	engine, modelID := newCapacityTestEngine(t, "text_to_image", &fakeProvider{}, &fakeBroker{})
	disk := &fakeDiskAdmission{fail: true}
	engine.deps.Disk = disk
	raw, err := marshalPinnedPayload(t, engine, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := engine.Admit(context.Background(), jobs.Job{ID: "disk-partial", Operation: "text_to_image", Payload: raw})
	if err == nil {
		t.Fatal("admission succeeded after temporary filesystem refusal")
	}
	if admission.Release == nil {
		t.Fatal("failed admission omitted cleanup")
	}
	if err := admission.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(disk.leases) != 0 {
		t.Fatalf("partial reservation created despite atomic refusal: %+v", disk.leases)
	}
}

func TestAdmissionRefusesOutputGenerationWithoutSharedDiskBroker(t *testing.T) {
	provider := &fakeProvider{}
	engine, _, modelID := newTestEngine(t, "text_to_image", provider)
	engine.deps.Disk = nil
	raw, err := marshalPinnedPayload(t, engine, "text_to_image", Payload{Operation: "text_to_image", ModelID: modelID})
	if err != nil {
		t.Fatal(err)
	}
	admission, err := engine.Admit(context.Background(), jobs.Job{ID: "disk-broker-missing", Operation: "text_to_image", Payload: raw})
	if err == nil || !strings.Contains(err.Error(), "shared host disk admission is unavailable") {
		t.Fatalf("missing disk broker admission error=%v", err)
	}
	if admission.Release == nil {
		t.Fatal("failed admission omitted release for any partial resources")
	}
	if err := admission.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if provider.execN != 0 {
		t.Fatalf("provider executed %d times without disk admission", provider.execN)
	}
}

type diskAdmissionReleaseCounter struct{ released int }

func (l *diskAdmissionReleaseCounter) Heartbeat(context.Context) error { return nil }
func (l *diskAdmissionReleaseCounter) Release(context.Context) error {
	l.released++
	return nil
}

func TestLeaseGroupRetainsDiskLeaseUntilOutputCleanupIsVerified(t *testing.T) {
	ctx := context.Background()
	disk := &fakeDiskLease{reservation: sharedcapacity.DiskJobReceipt{ID: "disk-retained", Status: "active"}}
	ram := &diskAdmissionReleaseCounter{}
	group := newLeaseGroup(ctx)
	group.renew("disk job", disk)
	group.renew("RAM", ram)
	group.markDiskCleanupVerified(false)

	if err := group.releaseInference(ctx); err != nil {
		t.Fatalf("release inference-only resources: %v", err)
	}
	if ram.released != 1 || disk.released != 0 {
		t.Fatalf("inference release RAM=%d disk=%d, want RAM released and disk retained", ram.released, disk.released)
	}
	if err := group.release(ctx); !errors.Is(err, ErrDiskOutputCleanupUnverified) {
		t.Fatalf("unverified output cleanup release error=%v", err)
	}
	if disk.released != 0 {
		t.Fatalf("disk lease released before cleanup verification: %d", disk.released)
	}
	group.markDiskCleanupVerified(true)
	if err := group.release(ctx); err != nil {
		t.Fatalf("release after verified cleanup: %v", err)
	}
	if disk.released != 1 || ram.released != 1 {
		t.Fatalf("final releases RAM=%d disk=%d", ram.released, disk.released)
	}
}

func TestOutputTempCleanupFailureIsUnverified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-output-temp")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	cleanupErr := errors.New("simulated remove failure")
	err := removeVerifiedAITempDir(path, func(string) error { return cleanupErr })
	if !errors.Is(err, ErrDiskOutputCleanupUnverified) || !errors.Is(err, cleanupErr) {
		t.Fatalf("cleanup error=%v, want unverified cleanup wrapping remove failure", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test fixture directory disappeared: %v", err)
	}
}
