// Package renderjobs coordinates Device Studio's durable job ledger, immutable
// shot/media owners, private PNG worker, routed storage, and native operation
// owner. It is deliberately caller-driven: it does not own a queue or ticker.
package renderjobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	platform "github.com/vrooli/platform-go"
)

const (
	OwnerID                                 = "device-studio"
	OutputProfilePNG                        = "png-native-v1"
	RendererVersion                         = "go-still-homography-v1"
	workerBlobPrefix                        = "renderjobs/workers/"
	intentBlobPrefix                        = "renderjobs/intents/"
	operationEvidenceBlobPrefix             = "renderjobs/operation-evidence/"
	outputBlobPrefix                        = "renderjobs/outputs/"
	receiptBlobPrefix                       = "renderjobs/receipts/"
	quiescenceBlobPrefix                    = "renderjobs/quiescence/"
	workspaceRelativeRoot                   = "device-studio/renderjobs"
	workerFileName                          = "worker"
	manifestMode                os.FileMode = 0o600
	workerMode                  os.FileMode = 0o700
	maxWorkerBytes                          = 256 << 20
	maxManifestBytes                        = 64 << 10
	maxReceiptBytes                         = 64 << 10
	maxOutputBytes                          = 160 << 20
	maxOutputPixels                         = 16_000_000
	maxWorkerInputBytes                     = 64 << 20
	stageTimeout                            = 30 * time.Second
	ownerCallTimeout                        = 15 * time.Second
	operationWallTime                       = 10 * time.Minute
)

var (
	ErrInvalidRequest = errors.New("renderjobs: invalid request")
	ErrRouteMismatch  = errors.New("renderjobs: stored job belongs to a different storage route")
	ErrUnknownOutcome = errors.New("renderjobs: operation outcome is unknown; automatic redispatch is forbidden")
	ErrNotEligible    = errors.New("renderjobs: shot revision is not eligible for this still profile")
	jobIDPattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type ShotResolver interface {
	ResolveForRender(context.Context, string, int64) (shots.ResolvedShot, error)
}

// OperationOwner is the narrow part of platform-go used by the coordinator.
// The production adapter delegates every method directly to platform-go.
type OperationOwner interface {
	NewIntent(string, string, platform.OperationSpec) (platform.OperationIntent, error)
	Start(context.Context, platform.OperationIntent, platform.OperationSpec) (*platform.OwnedOperation, error)
	Lookup(context.Context, platform.OperationIntent) (platform.OperationRef, error)
	Inspect(context.Context, platform.OperationRef) (platform.OperationObservation, error)
	Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error)
	Release(context.Context, platform.OperationRef) error
}

type platformOperations struct{}

func (platformOperations) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (platformOperations) Start(ctx context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	return platform.StartOwnedOperation(ctx, intent, spec)
}
func (platformOperations) Lookup(ctx context.Context, intent platform.OperationIntent) (platform.OperationRef, error) {
	return platform.LookupOwnedOperation(ctx, intent)
}
func (platformOperations) Inspect(ctx context.Context, ref platform.OperationRef) (platform.OperationObservation, error) {
	return platform.InspectOwnedOperation(ctx, ref)
}
func (platformOperations) Stop(ctx context.Context, ref platform.OperationRef) (platform.OperationStopReceipt, error) {
	return platform.StopOwnedOperation(ctx, ref)
}
func (platformOperations) Release(ctx context.Context, ref platform.OperationRef) error {
	return platform.ReleaseOwnedOperation(ctx, ref)
}

type Dependencies struct {
	Shots ShotResolver
	Media projects.SourceMediaService
	Jobs  jobs.Service
	Blobs blobstore.BlobStore
	Roots *filerouting.RoutedRoots
	Ops   OperationOwner
	Exe   func() (string, error)
	Now   func() time.Time
}

type Coordinator struct {
	shots ShotResolver
	media projects.SourceMediaService
	jobs  jobs.Service
	blobs blobstore.BlobStore
	roots *filerouting.RoutedRoots
	ops   OperationOwner
	exe   func() (string, error)
	now   func() time.Time
}

type SubmitRequest struct {
	ProjectID      string
	ShotID         string
	Revision       int64
	IdempotencyKey string
	OutputProfile  string
}

// New rejects partial owner wiring so an incomplete route can never become a
// successful or partially dispatched job.
func New(d Dependencies) (*Coordinator, error) {
	if d.Shots == nil || d.Media == nil || d.Jobs == nil || d.Blobs == nil || d.Roots == nil {
		return nil, errors.New("renderjobs: shots, media, jobs, routed blobs, and roots are required")
	}
	if d.Ops == nil {
		d.Ops = platformOperations{}
	}
	if d.Exe == nil {
		d.Exe = os.Executable
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	return &Coordinator{shots: d.Shots, media: d.Media, jobs: d.Jobs, blobs: d.Blobs, roots: d.Roots, ops: d.Ops, exe: d.Exe, now: d.Now}, nil
}

// Submit creates an immutable still job. It pins the exact running executable
// bytes before the jobs ledger records the request; later execution never
// reopens os.Executable or trusts a mutable developer build path.
func (c *Coordinator) Submit(ctx context.Context, req SubmitRequest) (jobs.Job, bool, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	if req.OutputProfile == "" {
		req.OutputProfile = OutputProfilePNG
	}
	if err := validateSubmitRequest(req); err != nil {
		return jobs.Job{}, false, err
	}
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, false, err
	}
	resolved, err := c.shots.ResolveForRender(ctx, req.ShotID, req.Revision)
	if err != nil {
		return jobs.Job{}, false, fmt.Errorf("resolve immutable shot: %w", err)
	}
	if err := validateResolved(req.ProjectID, req.ShotID, req.Revision, resolved); err != nil {
		return jobs.Job{}, false, err
	}
	if !resolved.Revision.RenderEligible {
		return jobs.Job{}, false, fmt.Errorf("%w: %v", ErrNotEligible, resolved.Revision.Issues)
	}
	if err := validateStillProfile(resolved.Revision); err != nil {
		return jobs.Job{}, false, err
	}
	workerBytes, _, workerDigest, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, err
	}
	workerKey := workerBlobPrefix + workerDigest + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerDigest); err != nil {
		return jobs.Job{}, false, fmt.Errorf("pin private worker bytes: %w", err)
	}
	profileDigest, err := toolProfileDigest(workerDigest, req.OutputProfile)
	if err != nil {
		return jobs.Job{}, false, err
	}
	bindings, err := bindingsForShot(resolved, route, workerKey, workerDigest)
	if err != nil {
		return jobs.Job{}, false, err
	}
	input := jobs.Input{
		ProjectID:      req.ProjectID,
		ShotRevisionID: shotRevisionID(req.ShotID, req.Revision), ShotRevision: req.Revision,
		Bindings: bindings, SourceClockDigest: stillClockDigest(), ToolProfileDigest: profileDigest,
		RendererVersion: RendererVersion, OutputProfile: req.OutputProfile,
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, err
	}
	job, created, err := c.jobs.Submit(ctx, jobs.SubmitInput{IdempotencyKey: req.IdempotencyKey, Input: input})
	if err != nil {
		return jobs.Job{}, false, err
	}
	return job, created, nil
}

// Advance performs one caller-requested progress step. A queued job stages and
// dispatches once; a claimed job only reconciles its persisted operation and
// can never be dispatched a second time.
func (c *Coordinator) Advance(ctx context.Context, jobID string) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	j, err := c.jobs.Get(ctx, jobID)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := verifyJobRoute(j, route); err != nil {
		return jobs.Job{}, err
	}
	switch j.State {
	case jobs.Queued:
		return c.dispatch(ctx, route, j)
	case jobs.Running, jobs.CancelRequested:
		return c.reconcile(ctx, route, j)
	default:
		return c.releaseTerminal(ctx, route, j)
	}
}

// Reconcile only observes persisted state and an existing operation. It never
// stages or dispatches a new attempt.
func (c *Coordinator) Reconcile(ctx context.Context, jobID string) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	j, err := c.jobs.Get(ctx, jobID)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := verifyJobRoute(j, route); err != nil {
		return jobs.Job{}, err
	}
	if j.State == jobs.Running || j.State == jobs.CancelRequested {
		return c.reconcile(ctx, route, j)
	}
	return c.releaseTerminal(ctx, route, j)
}

// Cancel persists the cancellation transition before stopping native work.
// A queued job is terminalized by the jobs owner without launching anything.
func (c *Coordinator) Cancel(ctx context.Context, jobID string, expectedVersion int64) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	j, err := c.jobs.Get(ctx, jobID)
	if err != nil {
		return jobs.Job{}, err
	}
	if err := verifyJobRoute(j, route); err != nil {
		return jobs.Job{}, err
	}
	if j.State == jobs.CancelRequested {
		if j.Version != expectedVersion {
			return jobs.Job{}, jobs.ErrConcurrentTransition
		}
		return c.cancelRunning(ctx, route, j)
	}
	j, err = c.jobs.RequestCancel(ctx, jobID, expectedVersion)
	if err != nil {
		return jobs.Job{}, err
	}
	if j.State == jobs.CancelRequested {
		return c.cancelRunning(ctx, route, j)
	}
	return j, nil
}

// OpenOutput returns only a verified PNG owned by a succeeded project job.
// The caller receives bytes, never a blob-store key or host path.
func (c *Coordinator) OpenOutput(ctx context.Context, projectID, jobID string) (jobs.Job, []byte, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil { return jobs.Job{}, nil, err }
	j, err := c.jobs.Get(ctx, jobID)
	if err != nil { return jobs.Job{}, nil, err }
	if err := verifyJobRoute(j, route); err != nil { return jobs.Job{}, nil, err }
	if j.Input.ProjectID != projectID { return jobs.Job{}, nil, jobs.ErrNotFound }
	if j.State != jobs.Succeeded || len(j.Attempts) == 0 { return jobs.Job{}, nil, errors.New("renderjobs: output is not available") }
	a := j.Attempts[len(j.Attempts)-1]
	if a.Completion == nil { return jobs.Job{}, nil, errors.New("renderjobs: completed job has no output evidence") }
	var ref *jobs.OutputRef
	for i := range a.Completion.Outputs { if a.Completion.Outputs[i].Kind == "still_png" { ref = &a.Completion.Outputs[i]; break } }
	if ref == nil || ref.MIMEType != "image/png" || ref.SizeBytes <= 0 || !validDigest(ref.SHA256) || ref.Width <= 0 || ref.Height <= 0 || int64(ref.Width)*int64(ref.Height) > maxOutputPixels {
		return jobs.Job{}, nil, errors.New("renderjobs: output metadata is invalid")
	}
	data, err := c.readBlob(ctx, ref.ArtifactID, maxOutputBytes, ref.SHA256)
	if err != nil { return jobs.Job{}, nil, err }
	if int64(len(data)) != ref.SizeBytes { return jobs.Job{}, nil, errors.New("renderjobs: output byte count changed") }
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != ref.Width || config.Height != ref.Height { return jobs.Job{}, nil, errors.New("renderjobs: stored output PNG metadata failed verification") }
	if _, err := png.Decode(bytes.NewReader(data)); err != nil { return jobs.Job{}, nil, errors.New("renderjobs: stored output PNG failed full decode") }
	if err := c.requireRoute(ctx, route); err != nil { return jobs.Job{}, nil, err }
	return j, data, nil
}

func (c *Coordinator) GetProjectJob(ctx context.Context, projectID, jobID string) (jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout); defer cancel()
	route, err := c.route(ctx); if err != nil { return jobs.Job{}, err }
	j, err := c.jobs.Get(ctx, jobID); if err != nil { return jobs.Job{}, err }
	if err := verifyJobRoute(j, route); err != nil { return jobs.Job{}, err }
	if j.Input.ProjectID != projectID { return jobs.Job{}, jobs.ErrNotFound }
	return j, nil
}

func (c *Coordinator) ListProjectJobs(ctx context.Context, projectID string, cursor jobs.Cursor, limit int) ([]jobs.Job, error) {
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout); defer cancel()
	route, err := c.route(ctx); if err != nil { return nil, err }
	rows, err := c.jobs.ListProject(ctx, projectID, cursor, limit); if err != nil { return nil, err }
	for _, row := range rows { if err := verifyJobRoute(row, route); err != nil { return nil, err } }
	return rows, nil
}

func (c *Coordinator) dispatch(ctx context.Context, route routePin, j jobs.Job) (jobs.Job, error) {
	a, ok := currentAttempt(j)
	if !ok || a.State != jobs.AttemptQueued || a.Execution != nil {
		return j, ErrUnknownOutcome
	}
	resolved, err := c.resolveJobShot(ctx, j)
	if err != nil {
		return j, err
	}
	if err := validateStillProfile(resolved.Revision); err != nil {
		return j, err
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, a.ID)
	if err != nil {
		return j, err
	}
	prepared, err := c.prepareAttempt(ctx, route, j, a, resolved)
	if err != nil {
		return j, err
	}
	manifest := prepared.Manifest
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return j, err
	}
	manifestBytes = append(manifestBytes, '\n')
	if len(manifestBytes) > maxManifestBytes {
		return j, errors.New("renderjobs: manifest exceeds the worker limit")
	}
	manifestSHA := digestBytes(manifestBytes)
	workerPath := filepath.Join(workspace, workerFileName)
	manifestPath := filepath.Join(workspace, rendering.ManifestName)
	spec := operationSpec(workerPath, manifestPath, workspace)
	intent, err := c.ops.NewIntent(OwnerID, a.ID, spec)
	if err != nil {
		return j, fmt.Errorf("create operation intent: %w", err)
	}
	envelope := persistedIntent{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, Route: route, Intent: intent, Spec: spec, ManifestSHA256: manifestSHA, WorkerSHA256: bindingDigest(j.Input, "worker_executable")}
	intentBytes, err := json.Marshal(envelope)
	if err != nil {
		return j, err
	}
	intentSHA := digestBytes(intentBytes)
	intentKey := intentBlobPrefix + j.ID + "/" + a.ID + "/" + intentSHA + ".json"
	if err := c.putVerified(ctx, route, intentKey, intentBytes, "application/json", intentSHA); err != nil {
		return j, fmt.Errorf("persist operation intent: %w", err)
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	started, err := c.jobs.Start(ctx, j.ID, jobs.ExecutionClaimInput{AttemptID: a.ID, AttemptNumber: a.Number, OwnerID: OwnerID, ScopeID: intent.InvocationID, ArtifactID: intentKey, SHA256: intentSHA}, j.Version)
	if err != nil {
		return j, err
	}
	// Only the caller that won the durable queued-to-running CAS may mutate the
	// attempt workspace. A competing Advance can leave immutable intent blobs,
	// but it cannot race writes or delete the winner's staged files.
	if _, err := c.createAttemptWorkspace(ctx, route, j, a); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, err)
	}
	for _, name := range []string{"worker", "plate.image", "screen.image", "active-mask.png", "hardware-mask.png"} {
		data, ok := prepared.Files[name]
		if !ok {
			continue
		}
		mode := manifestMode
		if name == workerFileName {
			mode = workerMode
		}
		if err := c.writeWorkspaceFile(ctx, route, workspace, name, data, mode, true); err != nil {
			return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage %s after execution claim: %w", name, err))
		}
	}
	if err := c.writeWorkspaceFile(ctx, route, workspace, rendering.ManifestName, manifestBytes, manifestMode, true); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage manifest after execution claim: %w", err))
	}
	if err := c.writeWorkspaceFile(ctx, route, workspace, "intent.json", intentBytes, manifestMode, true); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage intent after execution claim: %w", err))
	}
	if digestBytes(prepared.Files[workerFileName]) != envelope.WorkerSHA256 {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, errors.New("pinned worker changed before dispatch"))
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return started, err
	}
	latest, err := c.jobs.Get(ctx, j.ID)
	if err != nil {
		return started, err
	}
	if latest.State == jobs.CancelRequested {
		latestAttempt, ok := currentAttempt(latest)
		if !ok || latestAttempt.ID != a.ID {
			return latest, ErrUnknownOutcome
		}
		return c.cancelBeforeNativeStart(ctx, route, latest, latestAttempt, envelope, intentKey)
	}
	if latest.State != jobs.Running {
		return latest, jobs.ErrInvalidTransition
	}
	if _, err := c.ops.Start(ctx, intent, spec); err != nil {
		// The claim and intent are durable. The native owner reports ambiguous
		// dispatch explicitly; future progress must reconcile this exact intent.
		return started, fmt.Errorf("start claimed render operation: %w", err)
	}
	latest, err = c.jobs.Get(ctx, j.ID)
	if err != nil {
		return started, err
	}
	if latest.State == jobs.CancelRequested {
		return c.cancelRunning(ctx, route, latest)
	}
	return started, nil
}

func (c *Coordinator) reconcile(ctx context.Context, route routePin, j jobs.Job) (jobs.Job, error) {
	a, ok := currentAttempt(j)
	if !ok || a.Execution == nil {
		return j, ErrUnknownOutcome
	}
	envelope, intentKey, err := c.loadIntent(ctx, route, j, a)
	if err != nil {
		return j, err
	}
	ref, err := c.ops.Lookup(ctx, envelope.Intent)
	if err != nil {
		return j, fmt.Errorf("%w: lookup persisted operation: %v", ErrUnknownOutcome, err)
	}
	if ref.Intent != envelope.Intent {
		return j, platform.ErrOperationIdentity
	}
	if j.State == jobs.CancelRequested {
		return c.cancelWithRef(ctx, route, j, a, envelope, intentKey, ref)
	}
	obs, err := c.ops.Inspect(ctx, ref)
	if err != nil {
		return j, fmt.Errorf("%w: inspect operation: %v", ErrUnknownOutcome, err)
	}
	if obs.State == platform.OperationRunning {
		if a.ExecutionObserved {
			return j, nil
		}
		j, err = c.jobs.RecordObserved(ctx, j.ID, a.ID, j.Version)
		if err != nil {
			return j, err
		}
		a.ExecutionObserved = true
	}
	if !obs.ExitKnown || obs.Populated || obs.RootAlive || obs.State != platform.OperationStoppedOutcomeKnown {
		return j, nil
	}
	nativeKey, nativeSHA, err := c.persistNativeOutcome(ctx, route, j, a, envelope, ref, obs)
	if err != nil {
		return j, err
	}
	if obs.Result != "success" || obs.ExecMainStatus != 0 {
		reason := fmt.Sprintf("%s; native-evidence-sha256=%s", boundedFailure(obs.Result, obs.ExecMainStatus), nativeSHA)
		failed, failErr := c.jobs.Fail(ctx, j.ID, a.ID, reason, j.Version)
		if failErr != nil {
			return j, failErr
		}
		if _, releaseErr := c.releaseAttempt(ctx, route, failed, a, ref); releaseErr != nil {
			return failed, releaseErr
		}
		return failed, nil
	}
	completion, err := c.validateAndPublish(ctx, route, j, a, envelope, nativeKey, nativeSHA)
	if err != nil {
		reason := fmt.Sprintf("%s; native-evidence-sha256=%s", boundedFailure("verified output rejected: "+err.Error(), 0), nativeSHA)
		failed, failErr := c.jobs.Fail(ctx, j.ID, a.ID, reason, j.Version)
		if failErr != nil {
			return j, errors.Join(err, failErr)
		}
		if _, releaseErr := c.releaseAttempt(ctx, route, failed, a, ref); releaseErr != nil {
			return failed, errors.Join(err, releaseErr)
		}
		return failed, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	completed, err := c.jobs.Complete(ctx, j.ID, a.ID, completion, j.Version)
	if err != nil {
		// Cancellation may have serialized first. Keep the native unit retained
		// so its winning terminal path can persist quiescence and release it.
		return j, err
	}
	completed, err = c.releaseAttempt(ctx, route, completed, a, ref)
	if err != nil {
		return completed, err
	}
	return completed, nil
}

func (c *Coordinator) cancelRunning(ctx context.Context, route routePin, j jobs.Job) (jobs.Job, error) {
	a, ok := currentAttempt(j)
	if !ok || a.Execution == nil {
		return j, ErrUnknownOutcome
	}
	envelope, intentKey, err := c.loadIntent(ctx, route, j, a)
	if err != nil {
		return j, err
	}
	ref, err := c.ops.Lookup(ctx, envelope.Intent)
	if err != nil {
		return j, fmt.Errorf("%w: lookup operation for cancellation: %v", ErrUnknownOutcome, err)
	}
	return c.cancelWithRef(ctx, route, j, a, envelope, intentKey, ref)
}

func (c *Coordinator) cancelWithRef(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, intentKey string, ref platform.OperationRef) (jobs.Job, error) {
	if ref.Intent != env.Intent {
		return j, platform.ErrOperationIdentity
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	stop, err := c.ops.Stop(ctx, ref)
	if err != nil {
		return j, fmt.Errorf("stop exact operation tree: %w", err)
	}
	if !stop.NativeQuiescent || stop.Ref.Intent != env.Intent {
		return j, platform.ErrOperationNotQuiescent
	}
	proof := persistedQuiescence{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ClaimID: a.Execution.ID, IntentKey: intentKey, Ref: stop.Ref, Route: route, Stop: stop, ObservedAt: c.now().UTC()}
	data, err := json.Marshal(proof)
	if err != nil {
		return j, err
	}
	sha := digestBytes(data)
	key := quiescenceBlobPrefix + j.ID + "/" + a.ID + "/" + sha + ".json"
	if err := c.putVerified(ctx, route, key, data, "application/json", sha); err != nil {
		return j, fmt.Errorf("persist native quiescence proof: %w", err)
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	cancelled, err := c.jobs.ConfirmCancelled(ctx, j.ID, a.ID, jobs.QuiescenceProof{JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ExecutionClaimID: a.Execution.ID, OwnerID: OwnerID, ScopeID: env.Intent.InvocationID, ProofArtifactID: key, ProofSHA256: sha, ObservedAt: proof.ObservedAt, NoOwnedExecution: true, Reason: "manager confirmed native tree quiescence for the persisted operation intent"}, j.Version)
	if err != nil {
		return j, err
	}
	cancelled, err = c.releaseAttempt(ctx, route, cancelled, a, stop.Ref)
	if err != nil {
		return cancelled, err
	}
	return cancelled, nil
}

// cancelBeforeNativeStart is used only by the dispatch call that still owns
// the claim and has not invoked the native owner. It persists that fact before
// terminalizing cancellation; a later process cannot infer it from absence.
func (c *Coordinator) cancelBeforeNativeStart(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, intentKey string) (jobs.Job, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	if j.State != jobs.CancelRequested || a.Execution == nil || a.Execution.ScopeID != env.Intent.InvocationID {
		return j, ErrUnknownOutcome
	}
	proof := persistedQuiescence{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ClaimID: a.Execution.ID, IntentKey: intentKey, Route: route, NotDispatched: true, ObservedAt: c.now().UTC()}
	data, err := json.Marshal(proof)
	if err != nil {
		return j, err
	}
	sha := digestBytes(data)
	key := quiescenceBlobPrefix + j.ID + "/" + a.ID + "/" + sha + ".json"
	if err := c.putVerified(ctx, route, key, data, "application/json", sha); err != nil {
		return j, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return j, err
	}
	cancelled, err := c.jobs.ConfirmCancelled(ctx, j.ID, a.ID, jobs.QuiescenceProof{JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ExecutionClaimID: a.Execution.ID, OwnerID: OwnerID, ScopeID: env.Intent.InvocationID, ProofArtifactID: key, ProofSHA256: sha, ObservedAt: proof.ObservedAt, NoOwnedExecution: true, Reason: "dispatch call observed cancellation before invoking native owner"}, j.Version)
	if err != nil {
		return j, err
	}
	cancelled, err = c.jobs.RecordReleased(ctx, cancelled.ID, a.ID, cancelled.Version, proof.ObservedAt)
	if err != nil {
		return cancelled, err
	}
	return cancelled, nil
}

// failBeforeNativeStart terminalizes only a failure observed by the live
// dispatch call before it invoked the platform owner. If the leased route has
// expired or disappeared, it leaves the persisted claim unreconciled.
func (c *Coordinator) failBeforeNativeStart(ctx context.Context, route routePin, j jobs.Job, attemptID string, cause error) (jobs.Job, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return j, errors.Join(cause, err)
	}
	latest, err := c.jobs.Get(ctx, j.ID)
	if err != nil {
		return j, errors.Join(cause, err)
	}
	if latest.State == jobs.CancelRequested {
		a, ok := currentAttempt(latest)
		if !ok || a.ID != attemptID || a.Execution == nil {
			return latest, errors.Join(cause, ErrUnknownOutcome)
		}
		env, key, loadErr := c.loadIntent(ctx, route, latest, a)
		if loadErr != nil {
			return latest, errors.Join(cause, loadErr)
		}
		cancelled, cancelErr := c.cancelBeforeNativeStart(ctx, route, latest, a, env, key)
		return cancelled, errors.Join(cause, cancelErr)
	}
	if latest.State != jobs.Running {
		return latest, errors.Join(cause, jobs.ErrInvalidTransition)
	}
	failed, failErr := c.jobs.Fail(ctx, latest.ID, attemptID, boundedFailure("pre-dispatch staging failed: "+cause.Error(), 0), latest.Version)
	return failed, errors.Join(cause, failErr)
}

func (c *Coordinator) release(ctx context.Context, route routePin, ref platform.OperationRef) error {
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	return c.ops.Release(ctx, ref)
}

func (c *Coordinator) releaseAttempt(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, ref platform.OperationRef) (jobs.Job, error) {
	if err := c.release(ctx, route, ref); err != nil {
		return j, err
	}
	return c.jobs.RecordReleased(ctx, j.ID, a.ID, j.Version, c.now().UTC())
}

func (c *Coordinator) releaseTerminal(ctx context.Context, route routePin, j jobs.Job) (jobs.Job, error) {
	a, ok := currentAttempt(j)
	if !ok || a.Execution == nil {
		return j, nil
	}
	env, _, err := c.loadIntent(ctx, route, j, a)
	if err != nil {
		return j, err
	}
	ref, err := c.ops.Lookup(ctx, env.Intent)
	if errors.Is(err, platform.ErrOperationNotFound) {
		return c.jobs.RecordReleased(ctx, j.ID, a.ID, j.Version, c.now().UTC())
	}
	if err != nil {
		return j, err
	}
	return c.releaseAttempt(ctx, route, j, a, ref)
}

func (c *Coordinator) resolveJobShot(ctx context.Context, j jobs.Job) (shots.ResolvedShot, error) {
	shotID, revision, err := parseShotRevision(j.Input)
	if err != nil {
		return shots.ResolvedShot{}, err
	}
	resolved, err := c.shots.ResolveForRender(ctx, shotID, revision)
	if err != nil {
		return shots.ResolvedShot{}, fmt.Errorf("resolve pinned shot revision: %w", err)
	}
	if err := validateResolved(j.Input.ProjectID, shotID, revision, resolved); err != nil {
		return shots.ResolvedShot{}, err
	}
	if !resolved.Revision.RenderEligible {
		return shots.ResolvedShot{}, ErrNotEligible
	}
	if err := validateStillProfile(resolved.Revision); err != nil {
		return shots.ResolvedShot{}, err
	}
	bindings := bindingsByRole(j.Input)
	if bindings["shot_snapshot"].SHA256 != resolved.Revision.RecipeSHA256 {
		return shots.ResolvedShot{}, errors.New("renderjobs: pinned shot digest changed")
	}
	for role, media := range resolvedMedia(resolved.Revision) {
		binding, ok := bindings[role]
		if !ok || binding.ArtifactID != media.GetId() || binding.RevisionID != "sha256:"+media.GetSha256() || binding.SHA256 != media.GetSha256() {
			return shots.ResolvedShot{}, fmt.Errorf("renderjobs: pinned %s media binding changed", role)
		}
	}
	return resolved, nil
}

type preparedAttempt struct {
	Manifest rendering.Manifest
	Files    map[string][]byte
}

func (c *Coordinator) prepareAttempt(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, resolved shots.ResolvedShot) (preparedAttempt, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return preparedAttempt{}, err
	}
	workerBinding := bindingByRole(j.Input, "worker_executable")
	if workerBinding.ArtifactID == "" || !strings.HasPrefix(workerBinding.ArtifactID, workerBlobPrefix) {
		return preparedAttempt{}, errors.New("renderjobs: job does not bind a pinned worker executable")
	}
	workerBytes, err := c.readBlob(ctx, workerBinding.ArtifactID, maxWorkerBytes, workerBinding.SHA256)
	if err != nil {
		return preparedAttempt{}, fmt.Errorf("read pinned worker executable: %w", err)
	}

	mediaByRole := resolvedMedia(resolved.Revision)
	fileNames := map[string]string{"plate": "plate.image", "screen": "screen.image", "active_mask": "active-mask.png", "hardware_mask": "hardware-mask.png"}
	imageBindings := make(map[string]rendering.ImageBinding, len(mediaByRole))
	files := map[string][]byte{workerFileName: workerBytes}
	var totalBytes int64
	for _, role := range []string{"plate", "screen", "active_mask", "hardware_mask"} {
		media, ok := mediaByRole[role]
		if !ok {
			if role == "hardware_mask" {
				continue
			}
			return preparedAttempt{}, fmt.Errorf("renderjobs: resolved shot is missing %s media", role)
		}
		if err := c.requireRoute(ctx, route); err != nil {
			return preparedAttempt{}, err
		}
		opened, reader, err := c.media.Open(ctx, j.Input.ProjectID, media.GetId())
		if err != nil {
			return preparedAttempt{}, fmt.Errorf("open project %s media: %w", role, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, maxWorkerInputBytes+1))
		closeErr := reader.Close()
		if readErr != nil {
			return preparedAttempt{}, fmt.Errorf("read project %s media: %w", role, readErr)
		}
		if closeErr != nil {
			return preparedAttempt{}, fmt.Errorf("close project %s media: %w", role, closeErr)
		}
		if len(data) == 0 || int64(len(data)) > maxWorkerInputBytes || int64(len(data)) != opened.SizeBytes || opened.SizeBytes != media.GetSizeBytes() || digestBytes(data) != media.GetSha256() || opened.SHA256 != media.GetSha256() || opened.ID != media.GetId() || opened.ProjectID != j.Input.ProjectID || opened.MediaType != media.GetMediaType() || opened.Width != media.GetWidthPixels() || opened.Height != media.GetHeightPixels() {
			return preparedAttempt{}, fmt.Errorf("renderjobs: project %s media no longer matches its immutable binding", role)
		}
		totalBytes += int64(len(data))
		if totalBytes > maxWorkerInputBytes {
			return preparedAttempt{}, errors.New("renderjobs: aggregate source media exceeds staging limit")
		}
		name := fileNames[role]
		files[name] = data
		imageBindings[role] = rendering.ImageBinding{Name: name, SHA256: media.GetSha256(), SizeBytes: opened.SizeBytes}
	}

	if err := c.requireRoute(ctx, route); err != nil {
		return preparedAttempt{}, err
	}
	scene := resolved.Revision.Snapshot.GetScene()
	quad := scene.GetPlateQuad()
	points := [4]rendering.Corner{
		{X: quad.GetTopLeft().GetX(), Y: quad.GetTopLeft().GetY()},
		{X: quad.GetTopRight().GetX(), Y: quad.GetTopRight().GetY()},
		{X: quad.GetBottomRight().GetX(), Y: quad.GetBottomRight().GetY()},
		{X: quad.GetBottomLeft().GetX(), Y: quad.GetBottomLeft().GetY()},
	}
	manifest := rendering.Manifest{
		Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint,
		Profile: rendering.CompositorProfile, Plate: imageBindings["plate"], Screen: imageBindings["screen"],
		ActiveMask: imageBindings["active_mask"], HardwareMask: optionalImageBinding(imageBindings, "hardware_mask"),
		SurfaceWidth: int(scene.GetDisplayWidthPixels()), SurfaceHeight: int(scene.GetDisplayHeightPixels()), Quad: points, Samples: 2,
	}
	return preparedAttempt{Manifest: manifest, Files: files}, nil
}

func (c *Coordinator) validateAndPublish(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, nativeKey, nativeSHA string) (jobs.CompletionEvidence, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, a.ID)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := validateDirectoryTree(workspace); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	worker, err := readWorkspaceFile(workspace, workerFileName, maxWorkerBytes)
	if err != nil || digestBytes(worker) != env.WorkerSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: staged worker executable no longer matches pinned digest")
	}
	manifestBytes, err := readWorkspaceFile(workspace, rendering.ManifestName, maxManifestBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if digestBytes(manifestBytes) != env.ManifestSHA256 {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: staged manifest digest changed")
	}
	var manifest rendering.Manifest
	if err := decodeCanonical(manifestBytes, &manifest); err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("decode staged manifest: %w", err)
	}
	if manifest.JobID != j.ID || manifest.AttemptID != a.ID || manifest.InputFingerprint != j.InputFingerprint || manifest.Profile != rendering.CompositorProfile {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: staged manifest identity does not match job attempt")
	}
	if err := verifyManifestInputs(workspace, manifest); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	receiptBytes, err := readWorkspaceFile(workspace, rendering.ReceiptName, maxReceiptBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("read worker receipt: %w", err)
	}
	var receipt rendering.Receipt
	if err := decodeJSON(receiptBytes, &receipt); err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("decode worker receipt: %w", err)
	}
	if err := validateReceipt(receipt, manifest, digestBytes(manifestBytes), j, a); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	outputBytes, err := readWorkspaceFile(workspace, rendering.OutputName, maxOutputBytes)
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("read worker output: %w", err)
	}
	outputSHA := digestBytes(outputBytes)
	if outputSHA != receipt.OutputSHA256 || int64(len(outputBytes)) != receipt.OutputSizeBytes {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: output PNG digest or size does not match worker receipt")
	}
	config, err := png.DecodeConfig(bytes.NewReader(outputBytes))
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("read output PNG dimensions: %w", err)
	}
	if config.Width != receipt.OutputWidth || config.Height != receipt.OutputHeight || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxOutputPixels {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: PNG dimensions differ from receipt or exceed decode bounds")
	}
	decoded, err := png.Decode(bytes.NewReader(outputBytes))
	if err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("fully decode output PNG: %w", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != receipt.OutputWidth || bounds.Dy() != receipt.OutputHeight || bounds.Dx() <= 0 || bounds.Dy() <= 0 || int64(bounds.Dx())*int64(bounds.Dy()) > maxOutputPixels {
		return jobs.CompletionEvidence{}, errors.New("renderjobs: decoded output dimensions differ from receipt or exceed limits")
	}
	outputKey := outputBlobPrefix + j.ID + "/" + a.ID + "/" + outputSHA + ".png"
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.putVerified(ctx, route, outputKey, outputBytes, "image/png", outputSHA); err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("publish verified PNG: %w", err)
	}
	receiptSHA := digestBytes(receiptBytes)
	receiptKey := receiptBlobPrefix + j.ID + "/" + a.ID + "/" + receiptSHA + ".json"
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.putVerified(ctx, route, receiptKey, receiptBytes, "application/json", receiptSHA); err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("publish verified receipt: %w", err)
	}
	adapterReceipt := completedRenderEvidence{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ToolProfileDigest: j.Input.ToolProfileDigest, NativeEvidenceKey: nativeKey, NativeEvidenceSHA256: nativeSHA, WorkerReceiptKey: receiptKey, WorkerReceiptSHA256: receiptSHA, OutputSHA256: outputSHA, OutputSizeBytes: int64(len(outputBytes)), OutputWidth: bounds.Dx(), OutputHeight: bounds.Dy(), VerifiedAt: c.now().UTC()}
	adapterReceiptBytes, err := json.Marshal(adapterReceipt)
	if err != nil {
		return jobs.CompletionEvidence{}, err
	}
	adapterReceiptSHA := digestBytes(adapterReceiptBytes)
	adapterReceiptKey := operationEvidenceBlobPrefix + j.ID + "/" + a.ID + "/completed-" + adapterReceiptSHA + ".json"
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.CompletionEvidence{}, err
	}
	if err := c.putVerified(ctx, route, adapterReceiptKey, adapterReceiptBytes, "application/json", adapterReceiptSHA); err != nil {
		return jobs.CompletionEvidence{}, fmt.Errorf("persist completed render evidence: %w", err)
	}
	return jobs.CompletionEvidence{
		AttemptID: a.ID, ExecutionClaimID: a.Execution.ID, InputFingerprint: j.InputFingerprint,
		Outputs:           []jobs.OutputRef{{Kind: "still_png", ArtifactID: outputKey, SHA256: outputSHA, MIMEType: "image/png", SizeBytes: int64(len(outputBytes)), Width: bounds.Dx(), Height: bounds.Dy()}},
		ReceiptArtifactID: adapterReceiptKey, ReceiptSHA256: adapterReceiptSHA, CompletedAt: c.now().UTC(),
	}, nil
}

func (c *Coordinator) persistNativeOutcome(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, ref platform.OperationRef, obs platform.OperationObservation) (string, string, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return "", "", err
	}
	if ref != obs.Ref || ref.Intent != env.Intent || !obs.ExitKnown || obs.Populated || obs.RootAlive || obs.State != platform.OperationStoppedOutcomeKnown {
		return "", "", platform.ErrOperationNotQuiescent
	}
	evidence := nativeOutcomeEvidence{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ExecutionClaimID: a.Execution.ID, Route: route, Intent: env.Intent, Ref: ref, Observation: obs, ObservedAt: c.now().UTC()}
	data, err := json.Marshal(evidence)
	if err != nil {
		return "", "", err
	}
	sha := digestBytes(data)
	key := operationEvidenceBlobPrefix + j.ID + "/" + a.ID + "/native-" + sha + ".json"
	if err := c.putVerified(ctx, route, key, data, "application/json", sha); err != nil {
		return "", "", err
	}
	return key, sha, nil
}

func (c *Coordinator) loadIntent(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt) (persistedIntent, string, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return persistedIntent{}, "", err
	}
	if a.Execution == nil || a.Execution.OwnerID != OwnerID || a.Execution.ScopeID == "" || a.Execution.ArtifactID == "" {
		return persistedIntent{}, "", ErrUnknownOutcome
	}
	data, err := c.readBlob(ctx, a.Execution.ArtifactID, maxReceiptBytes, a.Execution.SHA256)
	if err != nil {
		return persistedIntent{}, "", fmt.Errorf("load persisted operation intent: %w", err)
	}
	var env persistedIntent
	if err := decodeJSON(data, &env); err != nil {
		return persistedIntent{}, "", err
	}
	if env.Version != 1 || env.JobID != j.ID || env.AttemptID != a.ID || env.InputFingerprint != j.InputFingerprint || env.Route != route || env.Intent.Owner != OwnerID || env.Intent.AttemptID != a.ID || env.Intent.InvocationID != a.Execution.ScopeID || env.Intent.SpecDigest == "" || env.WorkerSHA256 != bindingDigest(j.Input, "worker_executable") {
		return persistedIntent{}, "", ErrRouteMismatch
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, a.ID)
	if err != nil {
		return persistedIntent{}, "", err
	}
	if err := validateOperationSpec(env, workspace); err != nil {
		return persistedIntent{}, "", err
	}
	return env, a.Execution.ArtifactID, nil
}

func (c *Coordinator) createAttemptWorkspace(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt) (string, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return "", err
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, a.ID)
	if err != nil {
		return "", err
	}
	if err := c.ensureDirectories(ctx, route, filepath.Dir(workspace)); err != nil {
		return "", err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return "", err
	}
	if err := os.Mkdir(workspace, 0o700); err != nil {
		return "", fmt.Errorf("create immutable attempt workspace: %w", err)
	}
	c.roots.RecordWrite(ctx)
	return workspace, nil
}

func (c *Coordinator) ensureDirectories(ctx context.Context, route routePin, dir string) error {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return errors.New("renderjobs: directory path must be absolute and clean")
	}
	volume := filepath.VolumeName(dir)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(dir, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("renderjobs: directory path contains a symlink or non-directory component")
			}
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if err := c.requireRoute(ctx, route); err != nil {
			return err
		}
		if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
			return err
		} else if err == nil {
			c.roots.RecordWrite(ctx)
		}
	}
	return validateDirectoryTree(dir)
}

func (c *Coordinator) writeWorkspaceFile(ctx context.Context, route routePin, workspace, name string, data []byte, mode os.FileMode, exclusive bool) error {
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	if err := validateDirectoryTree(workspace); err != nil {
		return err
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return err
	}
	defer root.Close()
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if exclusive {
		flags |= os.O_EXCL
	}
	file, err := root.OpenFile(name, flags, mode)
	if err != nil {
		return err
	}
	mutated := true
	defer func() {
		if mutated {
			c.roots.RecordWrite(ctx)
		}
	}()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	mutated = false
	c.roots.RecordWrite(ctx)
	return nil
}

func (c *Coordinator) readCurrentExecutable() ([]byte, string, string, error) {
	path, err := c.exe()
	if err != nil {
		return nil, "", "", fmt.Errorf("resolve current worker executable: %w", err)
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, "", "", errors.New("renderjobs: executable path is not absolute and clean")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxWorkerBytes {
		return nil, "", "", errors.New("renderjobs: current executable is not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxWorkerBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxWorkerBytes {
		return nil, "", "", fmt.Errorf("read current executable bytes: %w", err)
	}
	return data, path, digestBytes(data), nil
}

func (c *Coordinator) putVerified(ctx context.Context, route routePin, key string, data []byte, mime, digest string) error {
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	if err := c.blobs.Put(ctx, key, bytes.NewReader(data), mime); err != nil {
		return err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return err
	}
	readback, _, err := c.blobs.Get(ctx, key)
	if err != nil {
		return err
	}
	defer readback.Close()
	verified, err := io.ReadAll(io.LimitReader(readback, int64(len(data))+1))
	if err != nil {
		return err
	}
	if len(verified) != len(data) || digestBytes(verified) != digest {
		return errors.New("renderjobs: routed blob readback does not match immutable digest")
	}
	return nil
}

func (c *Coordinator) readBlob(ctx context.Context, key string, limit int64, expectedDigest string) ([]byte, error) {
	route, err := c.route(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return nil, err
	}
	reader, _, err := c.blobs.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || int64(len(data)) > limit || digestBytes(data) != expectedDigest {
		return nil, errors.New("renderjobs: routed blob content failed size or digest verification")
	}
	return data, nil
}
