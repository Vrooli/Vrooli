package renderjobs

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/rendering"
)

type failReleasedOnce struct {
	jobs.Service
	failed bool
	cancel bool
}

func (s *failReleasedOnce) RecordReleased(ctx context.Context, id, attempt string, version int64, at time.Time) (jobs.Job, error) {
	if !s.failed {
		s.failed = true
		return jobs.Job{}, errors.New("simulated release acknowledgement loss")
	}
	return s.Service.RecordReleased(ctx, id, attempt, version, at)
}

func (s *failReleasedOnce) Fail(ctx context.Context, id, attempt, reason string, version int64) (jobs.Job, error) {
	if s.cancel {
		return s.Service.RequestCancel(ctx, id, version)
	}
	return s.Service.Fail(ctx, id, attempt, reason, version)
}

func TestPreDispatchFailurePersistsTruthfulNoLaunchAndRecoversRelease(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, route, running, a, env, intentKey := newClaimedPreDispatchJob(t, ops, "pre-dispatch-failure")
	baseJobs := c.jobs
	failing := &failReleasedOnce{Service: baseJobs}
	c.jobs = failing
	failed, err := c.failBeforeNativeStart(ctx, route, running, a.ID, errors.New("reserved staged filename"))
	if err == nil || failed.State != jobs.Failed || failed.Attempts[0].ExecutionReleasedAt != nil {
		t.Fatalf("expected failed attempt with a pending release receipt, job=%+v err=%v", failed, err)
	}
	proof, _, _, proofErr := c.loadNotDispatched(ctx, route, failed, failed.Attempts[0], env, intentKey)
	if proofErr != nil || !proof.NotDispatched || proof.Failure == "" {
		t.Fatalf("dispatch-owned no-launch evidence missing: proof=%+v err=%v", proof, proofErr)
	}
	if ops.starts != 0 {
		t.Fatalf("native operation started %d times before dispatch failure", ops.starts)
	}
	c.jobs = baseJobs
	c.adapters[jobs.SubjectStillRender] = stillSubjectAdapter{}
	released, err := c.releaseTerminal(ctx, route, failed)
	if err != nil {
		t.Fatalf("recover release from explicit no-launch proof: %v", err)
	}
	if released.Attempts[0].ExecutionReleasedAt == nil {
		t.Fatalf("proof-backed release was not recorded: %+v", released.Attempts[0])
	}
	if ops.lookups != 0 {
		t.Fatalf("native owner was queried despite explicit not-dispatched evidence: lookups=%d", ops.lookups)
	}
}

func TestPreDispatchFailureMarkerRecoversConcurrentCancel(t *testing.T) {
	ops := &integrationOps{}
	c, ctx, route, running, a, _, _ := newClaimedPreDispatchJob(t, ops, "pre-dispatch-cancel-race")
	baseJobs := c.jobs
	c.jobs = &failReleasedOnce{Service: baseJobs, cancel: true}
	cancelRequested, err := c.failBeforeNativeStart(ctx, route, running, a.ID, errors.New("staging failure concurrent with cancel"))
	if err == nil || cancelRequested.State != jobs.CancelRequested {
		t.Fatalf("expected persisted no-launch proof to survive concurrent cancellation: state=%s err=%v", cancelRequested.State, err)
	}
	c.jobs = baseJobs
	settled, err := c.reconcile(ctx, route, cancelRequested)
	if err != nil {
		t.Fatalf("recover cancellation from dispatch-owned proof: %v", err)
	}
	if settled.State != jobs.Cancelled || settled.Attempts[0].ExecutionReleasedAt == nil || ops.lookups != 0 || ops.starts != 0 {
		t.Fatalf("cancel recovery was not proof-backed and settled: job=%+v lookups=%d starts=%d", settled, ops.lookups, ops.starts)
	}
}

func newClaimedPreDispatchJob(t *testing.T, ops *integrationOps, key string) (*Coordinator, context.Context, routePin, jobs.Job, jobs.Attempt, persistedIntent, string) {
	t.Helper()
	c, ctx, _ := integrationCoordinator(t, ops, false)
	route, err := c.route(ctx)
	if err != nil {
		t.Fatal(err)
	}
	digest := digestBytes([]byte("binding"))
	workerBytes := []byte("worker")
	workerSHA := digestBytes(workerBytes)
	workerKey := workerBlobPrefix + workerSHA + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerSHA); err != nil {
		t.Fatal(err)
	}
	job, _, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: key, Input: jobs.Input{
		ProjectID: "project", ShotRevisionID: "shot:one:revision:1", ShotRevision: 1,
		SourceClockDigest: digest, ToolProfileDigest: digest, RendererVersion: "still-v1", OutputProfile: OutputProfilePNG,
		Bindings: []jobs.ArtifactBinding{routeBinding(route), {Role: "worker_executable", ArtifactID: workerKey, RevisionID: "sha256:" + workerSHA, SHA256: workerSHA}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	a := job.Attempts[0]
	workspace, err := c.createAttemptWorkspace(ctx, route, job, a)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := operationSpecFor(operationStillRender, filepath.Join(workspace, workerFileName), filepath.Join(workspace, rendering.ManifestName), workspace)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := c.ops.NewIntent(OwnerID, a.ID, spec)
	if err != nil {
		t.Fatal(err)
	}
	intentKey := intentBlobPrefix + job.ID + "/" + a.ID + "/claimed-intent.json"
	env := persistedIntent{Version: 1, JobID: job.ID, AttemptID: a.ID, InputFingerprint: job.InputFingerprint, Route: route, Intent: intent, Spec: spec, ManifestSHA256: digest, WorkerSHA256: workerSHA}
	intentBytes, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	intentSHA := digestBytes(intentBytes)
	if err := c.putVerified(ctx, route, intentKey, intentBytes, "application/json", intentSHA); err != nil {
		t.Fatal(err)
	}
	running, err := c.jobs.Start(ctx, job.ID, jobs.ExecutionClaimInput{AttemptID: a.ID, AttemptNumber: a.Number, OwnerID: OwnerID, ScopeID: intent.InvocationID, ArtifactID: intentKey, SHA256: intentSHA}, job.Version)
	if err != nil {
		t.Fatal(err)
	}
	return c, ctx, route, running, a, env, intentKey
}
