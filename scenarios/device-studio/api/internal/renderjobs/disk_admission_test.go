package renderjobs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/jobs"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

type fakeDiskJobAdmitter struct {
	request sharedcapacity.DiskJobRequest
	lease   *fakeDiskJobLease
	err     error
}

func (a *fakeDiskJobAdmitter) ReserveDiskJob(_ context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	a.request = request
	if a.err != nil {
		return nil, a.err
	}
	a.lease = &fakeDiskJobLease{receipt: sharedcapacity.DiskJobReceipt{
		ID: "disk-job-test", OwnerID: request.OwnerID, JobID: request.JobID, Status: "active",
		Paths:             []sharedcapacity.DiskJobPathReceipt{{Path: request.Paths[0].Path, Role: request.Paths[0].Role, PeakBytes: request.Paths[0].PeakBytes}},
		Filesystems:       []sharedcapacity.DiskJobFilesystem{{FilesystemID: "dev:test", PeakBytes: request.Paths[0].PeakBytes, FloorBytes: request.FloorBytes}},
		HeartbeatDeadline: time.Now().Add(time.Minute),
	}}
	return a.lease, nil
}

type fakeDiskJobLease struct {
	receipt     sharedcapacity.DiskJobReceipt
	heartbeats  int
	completions int
	err         error
}

func (l *fakeDiskJobLease) Heartbeat(context.Context) error {
	l.heartbeats++
	return l.err
}
func (l *fakeDiskJobLease) Observe(context.Context) (sharedcapacity.DiskJobHeadroom, error) {
	return sharedcapacity.DiskJobHeadroom{}, nil
}
func (l *fakeDiskJobLease) Extend(context.Context, []sharedcapacity.DiskJobPath) (sharedcapacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *fakeDiskJobLease) Complete(context.Context) (sharedcapacity.DiskJobReceipt, error) {
	l.completions++
	if l.err != nil {
		return sharedcapacity.DiskJobReceipt{}, l.err
	}
	l.receipt.Status = "completed"
	return l.receipt, nil
}
func (l *fakeDiskJobLease) Release(ctx context.Context) error {
	_, err := l.Complete(ctx)
	return err
}
func (l *fakeDiskJobLease) Reservation() sharedcapacity.DiskJobReceipt { return l.receipt }

func TestEstimateRenderPeakBytesCoversBoundedVideoAndPhysicalTransientWrites(t *testing.T) {
	video := jobs.Job{Input: jobs.Input{SubjectKind: jobs.SubjectVideoRender, OutputProfile: videoFinalOutputProfile}}
	got, err := EstimateRenderPeakBytes(video)
	want := maxWorkerBytes + maxVideoWorkspaceInputBytes + 2*maxVideoOutputBytes + (1 << 20)
	if err != nil || got != want {
		t.Fatalf("video peak=%d err=%v, want=%d", got, err, want)
	}
	physical := jobs.Job{Input: jobs.Input{SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: OutputProfilePhysicalMotionMP4}}
	got, err = EstimateRenderPeakBytes(physical)
	want = physicalWorkspaceInputMax + maxWorkerBytes + 2*maxOutputBytes + 6*(64<<20) + (8 << 20)
	if err != nil || got != want {
		t.Fatalf("physical motion peak=%d err=%v, want=%d", got, err, want)
	}
	if _, err := EstimateRenderPeakBytes(jobs.Job{Input: jobs.Input{SubjectKind: "unknown"}}); err == nil {
		t.Fatal("unknown render type was admitted without a bounded estimate")
	}
}

func TestRenderDiskAdmissionUsesSharedOwnerAndCompletesOnTerminalPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	admitter := &fakeDiskJobAdmitter{}
	coordinator := &Coordinator{diskJobs: admitter, diskLeases: make(map[string]diskLeaseEntry)}
	job := jobs.Job{ID: "video-job-1", Input: jobs.Input{SubjectKind: jobs.SubjectVideoRender, OutputProfile: videoFinalOutputProfile}}
	lease, created, err := coordinator.reserveDiskJob(context.Background(), routePin{Root: root}, job)
	if err != nil || !created || lease == nil {
		t.Fatalf("reserve disk job: lease=%v created=%v err=%v", lease, created, err)
	}
	if admitter.request.OwnerID != OwnerID || admitter.request.JobID != job.ID || admitter.request.FloorBytes != minimumRenderDiskFreeBytes || len(admitter.request.Paths) != 1 || admitter.request.Paths[0].Path != root || admitter.request.Paths[0].PeakBytes != mustRenderPeak(t, job) {
		t.Fatalf("disk owner request does not bind the whole video job: %+v", admitter.request)
	}
	if err := coordinator.heartbeatDiskJob(context.Background(), job.ID); err != nil || admitter.lease.heartbeats != 1 {
		t.Fatalf("heartbeat count=%d err=%v", admitter.lease.heartbeats, err)
	}
	if err := coordinator.completeDiskJob(context.Background(), job.ID); err != nil {
		t.Fatalf("terminal completion: %v", err)
	}
	if admitter.lease.completions != 1 || admitter.lease.receipt.Status != "completed" {
		t.Fatalf("disk lease was not completed exactly once: %+v", admitter.lease)
	}
	if _, ok := coordinator.diskLeases[job.ID]; ok {
		t.Fatal("terminal disk lease remained retained after completion")
	}
}

func TestRenderDiskAdmissionFailsClosedWhenOwnerDeniesPeak(t *testing.T) {
	root := t.TempDir()
	admitter := &fakeDiskJobAdmitter{err: errors.New("disk admission denied")}
	coordinator := &Coordinator{diskJobs: admitter}
	job := jobs.Job{ID: "video-job-denied", Input: jobs.Input{SubjectKind: jobs.SubjectVideoRender, OutputProfile: videoFinalOutputProfile}}
	if _, _, err := coordinator.reserveDiskJob(context.Background(), routePin{Root: root}, job); err == nil {
		t.Fatal("disk admission denial did not prevent the render job")
	}
}

func TestDispatchDiskDenialLeavesJobQueuedAndDoesNotStartWorker(t *testing.T) {
	ops := &integrationOps{}
	coordinator, ctx, _ := integrationCoordinator(t, ops, false)
	coordinator.diskJobs = &fakeDiskJobAdmitter{err: errors.New("fixture disk admission denied")}
	job := submitIntegrationJob(t, coordinator, ctx)
	advanced, err := coordinator.Advance(ctx, job.ID)
	if err == nil || advanced.State != jobs.Queued {
		t.Fatalf("disk denial should leave the job queued: state=%q err=%v", advanced.State, err)
	}
	ops.mu.Lock()
	starts := ops.starts
	ops.mu.Unlock()
	if starts != 0 {
		t.Fatalf("worker started despite disk owner denial: starts=%d", starts)
	}
	route, err := coordinator.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	attempt, ok := currentAttempt(advanced)
	if !ok {
		t.Fatal("queued job has no attempt")
	}
	workspace, err := attemptWorkspace(route.Root, advanced.ID, attempt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("workspace was created despite disk admission denial: stat err=%v", err)
	}
}

func TestVideoRenderHeartbeatAndCancellationCompleteSharedDiskLease(t *testing.T) {
	ops := &integrationOps{deferRun: true}
	coordinator, ctx, _ := integrationCoordinator(t, ops, false)
	admitter := &fakeDiskJobAdmitter{}
	coordinator.diskJobs = admitter
	job := submitIntegrationJob(t, coordinator, ctx)
	running, err := coordinator.Advance(ctx, job.ID)
	if err != nil || running.State != jobs.Running {
		t.Fatalf("video fixture did not enter running state: state=%q err=%v", running.State, err)
	}
	if admitter.lease == nil || admitter.lease.completions != 0 {
		t.Fatal("render lease was not held for the running operation")
	}
	if _, err := coordinator.Reconcile(ctx, job.ID); err != nil {
		t.Fatalf("renew active disk lease during operation reconciliation: %v", err)
	}
	if admitter.lease.heartbeats != 1 {
		t.Fatalf("active render did not heartbeat its disk lease: count=%d", admitter.lease.heartbeats)
	}
	latest, err := coordinator.jobs.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := coordinator.Cancel(ctx, job.ID, latest.Version)
	if err != nil || cancelled.State != jobs.Cancelled {
		t.Fatalf("cancel render: state=%q err=%v", cancelled.State, err)
	}
	if admitter.lease.completions != 1 || admitter.lease.receipt.Status != "completed" {
		t.Fatalf("cancellation did not complete shared disk receipt: %+v", admitter.lease)
	}
}

func mustRenderPeak(t *testing.T, job jobs.Job) int64 {
	t.Helper()
	peak, err := EstimateRenderPeakBytes(job)
	if err != nil {
		t.Fatal(err)
	}
	return peak
}
