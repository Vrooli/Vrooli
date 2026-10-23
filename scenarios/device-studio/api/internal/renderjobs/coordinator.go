// Package renderjobs coordinates Device Studio's durable job ledger, immutable
// shot/media owners, private PNG worker, routed storage, and native operation
// owner. It is deliberately caller-driven: it does not own a queue or ticker.
package renderjobs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"device-studio/internal/geometry"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	platform "github.com/vrooli/platform-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

const (
	OwnerID          = "device-studio"
	OutputProfilePNG = "png-native-v1"
	// OutputProfileMP4 is the caller-selected media format. Physical motion
	// jobs store a more specific immutable profile chosen by the shot revision.
	OutputProfileMP4                                     = "mp4-h264-sdr-v1"
	OutputProfilePhysicalMotionMP4                       = "physical-motion-mp4-h264-sdr-v1"
	OutputProfilePhysicalMotionMP4V2                     = "physical-motion-mp4-h264-sdr-v2"
	OutputProfilePhysicalRecordingMP4                    = "physical-recording-mp4-h264-sdr-v1"
	OutputProfilePhysicalPreviewPNG                      = "physical-preview-png-v1"
	OutputProfilePhysicalDiagnosticPNG                   = "physical-diagnostic-png-v1"
	OutputProfilePhysicalDiagnosticMotionMP4             = "physical-diagnostic-motion-mp4-v1"
	RendererVersion                                      = "go-still-homography-v1"
	workerBlobPrefix                                     = "renderjobs/workers/"
	intentBlobPrefix                                     = "renderjobs/intents/"
	closureBlobPrefix                                    = "renderjobs/closures/"
	inputBlobPrefix                                      = "renderjobs/inputs/"
	operationEvidenceBlobPrefix                          = "renderjobs/operation-evidence/"
	outputBlobPrefix                                     = "renderjobs/outputs/"
	receiptBlobPrefix                                    = "renderjobs/receipts/"
	quiescenceBlobPrefix                                 = "renderjobs/quiescence/"
	workspaceRelativeRoot                                = "device-studio/renderjobs"
	workerFileName                                       = "worker"
	manifestMode                             os.FileMode = 0o600
	workerMode                               os.FileMode = 0o700
	maxWorkerBytes                                       = 256 << 20
	maxManifestBytes                                     = 64 << 10
	maxReceiptBytes                                      = 64 << 10
	maxOutputBytes                                       = 160 << 20
	maxOutputPixels                                      = 16_000_000
	maxWorkerInputBytes                      int64       = 64 << 20
	maxInspectionInputBytes                  int64       = 512 << 20
	maxInspectionSidecarBytes                int64       = 64 << 10
	maxPhysicalManifestBytes                 int64       = 4 << 20
	maxPhysicalClosureBytes                  int64       = 5 << 20
	stageTimeout                                         = 30 * time.Second
	ownerCallTimeout                                     = 15 * time.Second
	operationWallTime                                    = 10 * time.Minute
)

var (
	ErrInvalidRequest    = errors.New("renderjobs: invalid request")
	ErrRouteMismatch     = errors.New("renderjobs: stored job belongs to a different storage route")
	ErrUnknownOutcome    = errors.New("renderjobs: operation outcome is unknown; automatic redispatch is forbidden")
	ErrNotEligible       = errors.New("renderjobs: shot revision is not eligible for this still profile")
	ErrOutputUnavailable = errors.New("renderjobs: output is not available yet")
	jobIDPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	stageFilePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

type ShotResolver interface {
	ResolveForRender(context.Context, string, int64) (shots.ResolvedShot, error)
}

type PhysicalArtifactResolver interface {
	Resolve(context.Context, physical.ArtifactRef, library.Use) (library.ResolvedArtifact, error)
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
	Shots             ShotResolver
	Media             projects.SourceMediaService
	Jobs              jobs.Service
	Scopes            jobs.RunScopeService
	Blobs             blobstore.BlobStore
	Roots             *filerouting.RoutedRoots
	Ops               OperationOwner
	Exe               func() (string, error)
	InspectionProfile func(context.Context) (*cliv1.RendererToolProfile, error)
	PhysicalArtifacts PhysicalArtifactResolver
	PhysicalProfile   func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error)
	PhysicalAdapter   func() []byte
	GeneratedIntake   interface {
		PersistGenerated(context.Context, library.IntakeClaims, string, library.CreationLineage, []byte) (library.IntakeCandidate, error)
	}
	Now func() time.Time
}

type Coordinator struct {
	shots             ShotResolver
	media             projects.SourceMediaService
	jobs              jobs.Service
	scopes            jobs.RunScopeService
	blobs             blobstore.BlobStore
	roots             *filerouting.RoutedRoots
	ops               OperationOwner
	exe               func() (string, error)
	now               func() time.Time
	adapters          map[jobs.SubjectKind]subjectAdapter
	inspectionProfile func(context.Context) (*cliv1.RendererToolProfile, error)
	physicalArtifacts PhysicalArtifactResolver
	physicalProfile   func(context.Context) ([]byte, *cliv1.BlenderToolProfile, error)
	physicalAdapter   func() []byte
	generatedIntake   interface {
		PersistGenerated(context.Context, library.IntakeClaims, string, library.CreationLineage, []byte) (library.IntakeCandidate, error)
	}
}

type operationMode string

const (
	operationStillRender       operationMode = "still-render-v1"
	operationSourceInspect     operationMode = "source-inspect-v1"
	operationVideoRender       operationMode = "video-render-v1"
	operationPhysicalRender    operationMode = "physical-render-v1"
	operationPhysicalMotion    operationMode = "physical-motion-render-v2"
	operationPhysicalMotionV2  operationMode = "physical-motion-render-v3"
	operationPhysicalPreview   operationMode = "physical-preview-render-v1"
	operationPhysicalRecording operationMode = "physical-recording-render-v1"
	operationGeometryCreate    operationMode = "geometry-create-local-v1"
)

type stagedFile struct {
	Name     string
	Data     []byte
	BlobKey  string
	SHA256   string
	Size     int64
	Mode     os.FileMode
	Open     func(context.Context) (io.ReadCloser, error)
	MaxBytes int64
}

// preparedOperation separates subject payloads from the one execution
// lifecycle. Large immutable inputs can stay in routed blob storage and are
// streamed into the private workspace with size and digest verification.
type preparedOperation struct {
	Mode          operationMode
	ManifestBytes []byte
	Files         []stagedFile
	MaxInputBytes int64
}

type SubmitRequest struct {
	ProjectID          string
	ShotID             string
	Revision           int64
	IdempotencyKey     string
	OutputProfile      string
	VideoMode          rendering.VideoMode
	ReuseCachedSuccess bool
}

// CacheReuseReceipt identifies the exact prior succeeded output returned for
// an opt-in cache hit. It never represents a new render attempt.
type CacheReuseReceipt struct {
	SourceRenderID  string
	SourceAttemptID string
	ArtifactID      string
	SHA256          string
	OutputKind      string
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
	if d.PhysicalProfile == nil {
		d.PhysicalProfile = rendering.ResolvePhysicalBlenderProfile
	}
	if d.PhysicalAdapter == nil {
		d.PhysicalAdapter = rendering.PhysicalAdapterBytes
	}
	c := &Coordinator{shots: d.Shots, media: d.Media, jobs: d.Jobs, scopes: d.Scopes, blobs: d.Blobs, roots: d.Roots, ops: d.Ops, exe: d.Exe, now: d.Now, inspectionProfile: d.InspectionProfile, physicalArtifacts: d.PhysicalArtifacts, physicalProfile: d.PhysicalProfile, physicalAdapter: d.PhysicalAdapter, generatedIntake: d.GeneratedIntake}
	c.adapters = map[jobs.SubjectKind]subjectAdapter{jobs.SubjectStillRender: stillSubjectAdapter{}, jobs.SubjectSourceInspection: sourceInspectionAdapter{}, jobs.SubjectVideoRender: videoSubjectAdapter{}, jobs.SubjectPhysicalRender: physicalSubjectAdapter{}, jobs.SubjectGeometryCreate: geometryCreationAdapter{}}
	return c, nil
}

// Submit creates an immutable still job. It pins the exact running executable
// bytes before the jobs ledger records the request; later execution never
// reopens os.Executable or trusts a mutable developer build path.
func (c *Coordinator) Submit(ctx context.Context, req SubmitRequest) (jobs.Job, bool, error) {
	job, created, _, err := c.SubmitWithReceipt(ctx, req)
	return job, created, err
}

// SubmitWithReceipt behaves like Submit, with an optional exact-input reuse
// receipt. The ordinary Submit API preserves the historical return shape.
func (c *Coordinator) SubmitWithReceipt(ctx context.Context, req SubmitRequest) (jobs.Job, bool, *CacheReuseReceipt, error) {
	ctx, cancel := boundedDetached(ctx, stageTimeout)
	defer cancel()
	if req.OutputProfile == "" && req.VideoMode == "" {
		req.OutputProfile = OutputProfilePNG
	}
	if req.OutputProfile == "" && req.VideoMode == rendering.VideoModeSample {
		req.OutputProfile = videoSampleOutputProfile
	}
	if req.OutputProfile == "" && req.VideoMode == rendering.VideoModeFinal {
		req.OutputProfile = videoFinalOutputProfile
	}
	if req.VideoMode != "" {
		if req.ReuseCachedSuccess {
			return jobs.Job{}, false, nil, fmt.Errorf("%w: cached reuse is not enabled for source-video renders", ErrInvalidRequest)
		}
		job, created, err := c.submitVideo(ctx, req)
		return job, created, nil, err
	}
	if err := validateSubmitRequest(req); err != nil {
		return jobs.Job{}, false, nil, err
	}
	spanStarted := time.Now()
	route, err := c.route(ctx)
	submitTimingSpan(ctx, "route", spanStarted)
	if err != nil {
		return jobs.Job{}, false, nil, err
	}
	spanStarted = time.Now()
	existing, found, lookupErr := c.findIdempotentSubmit(ctx, route, req)
	submitTimingSpan(ctx, "idempotency", spanStarted)
	if lookupErr != nil || found {
		return existing, false, nil, lookupErr
	}
	spanStarted = time.Now()
	resolved, err := c.shots.ResolveForRender(ctx, req.ShotID, req.Revision)
	submitTimingSpan(ctx, "shot_resolve", spanStarted)
	if err != nil {
		return jobs.Job{}, false, nil, fmt.Errorf("resolve immutable shot: %w", err)
	}
	if err := validateResolved(req.ProjectID, req.ShotID, req.Revision, resolved); err != nil {
		return jobs.Job{}, false, nil, err
	}
	if !resolved.Revision.RenderEligible {
		return jobs.Job{}, false, nil, fmt.Errorf("%w: %v", ErrNotEligible, resolved.Revision.Issues)
	}
	if resolved.Revision.Physical != nil {
		return c.submitPhysicalWithReceipt(ctx, route, req, resolved)
	}
	if req.OutputProfile != OutputProfilePNG {
		return jobs.Job{}, false, nil, fmt.Errorf("%w: this output profile requires an immutable physical motion shot", ErrInvalidRequest)
	}
	if err := validateStillProfile(resolved.Revision); err != nil {
		return jobs.Job{}, false, nil, err
	}
	workerBytes, _, workerDigest, err := c.readCurrentExecutable()
	if err != nil {
		return jobs.Job{}, false, nil, err
	}
	workerKey := workerBlobPrefix + workerDigest + "/device-studio-api"
	if err := c.putVerified(ctx, route, workerKey, workerBytes, "application/octet-stream", workerDigest); err != nil {
		return jobs.Job{}, false, nil, fmt.Errorf("pin private worker bytes: %w", err)
	}
	profileDigest, err := toolProfileDigest(workerDigest, req.OutputProfile, resolved.Revision.CompositorProfile)
	if err != nil {
		return jobs.Job{}, false, nil, err
	}
	closure, closureBytes, closureKey, err := c.freezeClosure(ctx, route, resolved, workerKey, workerDigest, workerBytes, req.OutputProfile)
	if err != nil {
		return jobs.Job{}, false, nil, fmt.Errorf("freeze immutable render closure: %w", err)
	}
	bindings, err := bindingsForShot(resolved, route, workerKey, workerDigest)
	if err != nil {
		return jobs.Job{}, false, nil, err
	}
	closureSHA := digestBytes(closureBytes)
	bindings = append(bindings, closureBindings(closure, closureKey, closureSHA)...)
	input := jobs.Input{
		ProjectID:      req.ProjectID,
		ShotRevisionID: shotRevisionID(req.ShotID, req.Revision), ShotRevision: req.Revision,
		Bindings: bindings, SourceClockDigest: stillClockDigest(), ToolProfileDigest: profileDigest,
		RendererVersion: closure.RendererVersion, OutputProfile: req.OutputProfile,
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, false, nil, err
	}
	return c.submitInputWithCache(ctx, route, req, input)
}

func (c *Coordinator) findIdempotentSubmit(ctx context.Context, route routePin, req SubmitRequest) (jobs.Job, bool, error) {
	existing, err := c.jobs.FindByIdempotencyKey(ctx, req.IdempotencyKey)
	if errors.Is(err, jobs.ErrNotFound) {
		return jobs.Job{}, false, nil
	}
	if err != nil {
		return jobs.Job{}, false, err
	}
	if existing.Input.ProjectID != req.ProjectID || existing.Input.ShotRevisionID != shotRevisionID(req.ShotID, req.Revision) || existing.Input.ShotRevision != req.Revision || !idempotentOutputProfileMatches(existing.Input, req.OutputProfile) {
		return jobs.Job{}, false, jobs.ErrIdempotencyConflict
	}
	if err := verifyJobRoute(existing, route); err != nil {
		return jobs.Job{}, false, err
	}
	return existing, true, nil
}

func idempotentOutputProfileMatches(input jobs.Input, requested string) bool {
	if input.OutputProfile == requested {
		return true
	}
	return requested == OutputProfileMP4 && input.SubjectKind == jobs.SubjectPhysicalRender &&
		(input.OutputProfile == OutputProfilePhysicalMotionMP4 || input.OutputProfile == OutputProfilePhysicalMotionMP4V2 || input.OutputProfile == OutputProfilePhysicalRecordingMP4 || input.OutputProfile == OutputProfilePhysicalDiagnosticMotionMP4)
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
	if err != nil {
		return jobs.Job{}, nil, err
	}
	j, err := c.jobs.Get(ctx, jobID)
	if err != nil {
		return jobs.Job{}, nil, err
	}
	if err := verifyJobRoute(j, route); err != nil {
		return jobs.Job{}, nil, err
	}
	if j.Input.ProjectID != projectID {
		return jobs.Job{}, nil, jobs.ErrNotFound
	}
	if !isPublicStillJob(j) {
		return jobs.Job{}, nil, jobs.ErrNotFound
	}
	if j.State != jobs.Succeeded || len(j.Attempts) == 0 {
		return jobs.Job{}, nil, ErrOutputUnavailable
	}
	a := j.Attempts[len(j.Attempts)-1]
	if a.Completion == nil {
		return jobs.Job{}, nil, ErrOutputUnavailable
	}
	var ref *jobs.OutputRef
	wantKind := "still_png"
	if j.Input.SubjectKind == jobs.SubjectPhysicalRender {
		wantKind = physicalOutputKind(j.Input.OutputProfile)
	}
	for i := range a.Completion.Outputs {
		if a.Completion.Outputs[i].Kind == wantKind {
			ref = &a.Completion.Outputs[i]
			break
		}
	}
	if ref == nil || ref.MIMEType != "image/png" || ref.SizeBytes <= 0 || !validDigest(ref.SHA256) || ref.Width <= 0 || ref.Height <= 0 || int64(ref.Width)*int64(ref.Height) > maxOutputPixels {
		return jobs.Job{}, nil, errors.New("renderjobs: output metadata is invalid")
	}
	data, err := c.readBlob(ctx, ref.ArtifactID, maxOutputBytes, ref.SHA256)
	if err != nil {
		return jobs.Job{}, nil, err
	}
	if int64(len(data)) != ref.SizeBytes {
		return jobs.Job{}, nil, errors.New("renderjobs: output byte count changed")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != ref.Width || config.Height != ref.Height {
		return jobs.Job{}, nil, errors.New("renderjobs: stored output PNG metadata failed verification")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return jobs.Job{}, nil, errors.New("renderjobs: stored output PNG failed full decode")
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return jobs.Job{}, nil, err
	}
	return j, data, nil
}

func physicalOutputKind(profile string) string {
	if profile == OutputProfilePhysicalDiagnosticPNG {
		return "physical_diagnostic_png"
	}
	if profile == OutputProfilePhysicalPreviewPNG {
		return "physical_preview_png"
	}
	if profile == OutputProfilePhysicalMotionMP4 || profile == OutputProfilePhysicalMotionMP4V2 || profile == OutputProfilePhysicalRecordingMP4 || profile == OutputProfilePhysicalDiagnosticMotionMP4 {
		return "physical_motion_mp4"
	}
	return "physical_still_png"
}

func (c *Coordinator) GetProjectJob(ctx context.Context, projectID, jobID string) (jobs.Job, error) {
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
	if j.Input.ProjectID != projectID {
		return jobs.Job{}, jobs.ErrNotFound
	}
	if !isPublicRenderJob(j) {
		return jobs.Job{}, jobs.ErrNotFound
	}
	return j, nil
}

func isPublicRenderJob(j jobs.Job) bool {
	return j.Input.SubjectKind == "" || j.Input.SubjectKind == jobs.SubjectStillRender || j.Input.SubjectKind == jobs.SubjectVideoRender || j.Input.SubjectKind == jobs.SubjectPhysicalRender
}

func isPublicStillJob(j jobs.Job) bool {
	return j.Input.SubjectKind == "" || j.Input.SubjectKind == jobs.SubjectStillRender || j.Input.SubjectKind == jobs.SubjectPhysicalRender
}

func (c *Coordinator) ListProjectJobs(ctx context.Context, projectID string, cursor jobs.Cursor, limit int) ([]jobs.Job, error) {
	if limit < 1 || limit > 101 {
		return nil, ErrInvalidRequest
	}
	ctx, cancel := boundedDetached(ctx, ownerCallTimeout)
	defer cancel()
	route, err := c.route(ctx)
	if err != nil {
		return nil, err
	}
	const scanPageSize = 100
	visible := make([]jobs.Job, 0, limit)
	for len(visible) < limit {
		rows, err := c.jobs.ListProject(ctx, projectID, cursor, scanPageSize)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if err := verifyJobRoute(row, route); err != nil {
				return nil, err
			}
			cursor = jobs.Cursor{CreatedAt: row.CreatedAt, ID: row.ID}
			if isPublicRenderJob(row) {
				visible = append(visible, row)
				if len(visible) == limit {
					return visible, nil
				}
			}
		}
		if len(rows) < scanPageSize {
			return visible, nil
		}
	}
	return visible, nil
}

func (c *Coordinator) dispatch(ctx context.Context, route routePin, j jobs.Job) (jobs.Job, error) {
	a, ok := currentAttempt(j)
	if !ok || a.State != jobs.AttemptQueued || a.Execution != nil {
		return j, ErrUnknownOutcome
	}
	adapter, err := c.adapterFor(j)
	if err != nil {
		return j, err
	}
	workspace, err := attemptWorkspace(route.Root, j.ID, a.ID)
	if err != nil {
		return j, err
	}
	prepared, err := adapter.prepare(ctx, c, route, j, a)
	if err != nil {
		if errors.Is(err, errDeterministicPreparation) {
			return c.rejectQueuedPreparation(ctx, route, j, a, err)
		}
		return j, err
	}
	workerSHA := bindingDigest(j.Input, "worker_executable")
	manifestBytes := append([]byte(nil), prepared.ManifestBytes...)
	if len(manifestBytes) == 0 || !json.Valid(bytes.TrimSpace(manifestBytes)) {
		return j, errors.New("renderjobs: subject adapter returned an invalid manifest")
	}
	if manifestBytes[len(manifestBytes)-1] != '\n' {
		manifestBytes = append(manifestBytes, '\n')
	}
	manifestLimit := int64(maxManifestBytes)
	if modeForJob, modeErr := operationModeForJob(j.Input); modeErr == nil && isPhysicalOperation(modeForJob) {
		manifestLimit = maxPhysicalManifestBytes
	}
	if int64(len(manifestBytes)) > manifestLimit {
		return j, errors.New("renderjobs: manifest exceeds the worker limit")
	}
	manifestSHA := digestBytes(manifestBytes)
	mode := prepared.Mode
	expectedMode, err := operationModeForJob(j.Input)
	if err != nil {
		return j, err
	}
	if mode == "" {
		mode = expectedMode
	}
	if mode != expectedMode {
		return j, errors.New("renderjobs: subject adapter selected a mismatched operation mode")
	}
	manifestName := rendering.ManifestName
	if mode == operationSourceInspect {
		manifestName = "inspection-manifest.json"
	}
	if mode == operationVideoRender {
		manifestName = "video-manifest.json"
	}
	if isPhysicalOperation(mode) {
		manifestName = rendering.PhysicalManifestName
	}
	if mode == operationGeometryCreate {
		manifestName = rendering.GeometryManifestName
	}
	workerPath := filepath.Join(workspace, workerFileName)
	manifestPath := filepath.Join(workspace, manifestName)
	spec, err := operationSpecFor(mode, workerPath, manifestPath, workspace)
	if err != nil {
		return j, err
	}
	intent, err := c.ops.NewIntent(OwnerID, a.ID, spec)
	if err != nil {
		return j, fmt.Errorf("create operation intent: %w", err)
	}
	envelope := persistedIntent{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, Route: route, Intent: intent, Spec: spec, ManifestSHA256: manifestSHA, WorkerSHA256: workerSHA}
	if mode != operationStillRender {
		envelope.Mode = mode
	}
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
	if isPhysicalOperation(mode) {
		if err := c.createPhysicalSandbox(ctx, route, workspace); err != nil {
			return c.failBeforeNativeStart(ctx, route, started, a.ID, err)
		}
	}
	workerBinding := bindingByRole(j.Input, "worker_executable")
	if workerBinding.ArtifactID == "" || workerBinding.SHA256 != workerSHA {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, errors.New("subject operation lacks its pinned worker binding"))
	}
	workerBytes, err := c.readBlob(ctx, workerBinding.ArtifactID, maxWorkerBytes, workerSHA)
	if err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("read pinned worker executable: %w", err))
	}
	if err := c.writeWorkspaceFile(ctx, route, workspace, workerFileName, workerBytes, workerMode, true); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage worker after execution claim: %w", err))
	}
	maxInputBytes := maxWorkerInputBytes
	if mode == operationSourceInspect {
		maxInputBytes = maxInspectionInputBytes
	} else if mode == operationVideoRender {
		maxInputBytes = maxVideoInputBytes
	} else if isPhysicalOperation(mode) {
		maxInputBytes = physicalWorkspaceInputMax
	}
	if prepared.MaxInputBytes != 0 && prepared.MaxInputBytes != maxInputBytes {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, errors.New("subject input limit does not match its operation mode"))
	}
	var stagedBytes int64
	stagingLimit := maxInputBytes
	if mode == operationSourceInspect {
		stagingLimit += maxInspectionSidecarBytes
	} else if mode == operationVideoRender {
		stagingLimit = maxVideoWorkspaceInputBytes
	} else if isPhysicalOperation(mode) {
		stagingLimit = physicalWorkspaceInputMax + (1 << 20) + (64 << 10)
	}
	for _, file := range prepared.Files {
		size := file.Size
		if size == 0 {
			size = int64(len(file.Data))
		}
		stagedBytes += size
		if stagedBytes > stagingLimit {
			return c.failBeforeNativeStart(ctx, route, started, a.ID, errors.New("subject input staging exceeds aggregate limit"))
		}
		if err := c.stagePreparedFile(ctx, route, workspace, j, file, maxInputBytes); err != nil {
			return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage %s after execution claim: %w", file.Name, err))
		}
	}
	if err := c.writeWorkspaceFile(ctx, route, workspace, manifestName, manifestBytes, manifestMode, true); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage manifest after execution claim: %w", err))
	}
	if err := c.writeWorkspaceFile(ctx, route, workspace, "intent.json", intentBytes, manifestMode, true); err != nil {
		return c.failBeforeNativeStart(ctx, route, started, a.ID, fmt.Errorf("stage intent after execution claim: %w", err))
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

func (c *Coordinator) rejectQueuedPreparation(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, cause error) (jobs.Job, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return j, errors.Join(cause, err)
	}
	reason := boundedFailure("invalid immutable render manifest; revise the saved shot or regenerate its source inspection", 0)
	failed, err := c.jobs.FailBeforeDispatch(ctx, j.ID, a.ID, reason, j.Version)
	if err != nil {
		latest, getErr := c.jobs.Get(ctx, j.ID)
		if getErr == nil {
			return latest, errors.Join(cause, err)
		}
		return j, errors.Join(cause, err, getErr)
	}
	return failed, cause
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
	if proof, proofKey, proofSHA, proofErr := c.loadNotDispatched(ctx, route, j, a, envelope, intentKey); proofErr == nil {
		failure := proof.Failure
		if failure == "" {
			failure = "pre-dispatch staging failed"
		}
		if j.State == jobs.CancelRequested {
			cancelled, cancelErr := c.jobs.ConfirmCancelled(ctx, j.ID, a.ID, jobs.QuiescenceProof{JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ExecutionClaimID: a.Execution.ID, OwnerID: OwnerID, ScopeID: envelope.Intent.InvocationID, ProofArtifactID: proofKey, ProofSHA256: proofSHA, ObservedAt: proof.ObservedAt, NoOwnedExecution: true, Reason: "persisted dispatch call proved native execution was never invoked"}, j.Version)
			if cancelErr != nil {
				return j, cancelErr
			}
			return c.jobs.RecordReleased(ctx, cancelled.ID, a.ID, cancelled.Version, proof.ObservedAt)
		}
		failed, failErr := c.jobs.Fail(ctx, j.ID, a.ID, failure, j.Version)
		if failErr != nil {
			return j, failErr
		}
		return c.jobs.RecordReleased(ctx, failed.ID, a.ID, failed.Version, proof.ObservedAt)
	} else if !isMissingNotDispatchedProof(proofErr) {
		return j, proofErr
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
	adapter, adapterErr := c.adapterFor(j)
	if adapterErr != nil {
		return j, adapterErr
	}
	completion, err := adapter.validateCompletion(ctx, c, route, j, a, envelope, nativeKey, nativeSHA)
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
	if j.Input.SubjectKind == jobs.SubjectGeometryCreate {
		proof.InferenceOutcomeStatus = inferenceOutcomeNotBound
		if workspace, workspaceErr := attemptWorkspace(route.Root, j.ID, a.ID); workspaceErr == nil {
			if outcome, outcomeSHA, outcomeErr := readGeometryInferenceOutcome(workspace, j, a); outcomeErr == nil {
				proof.InferenceOutcomeStatus = inferenceOutcomeBound
				proof.InferenceOutcome = outcome
				proof.InferenceOutcomeSHA256 = outcomeSHA
			}
		}
	}
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
	a, ok := currentAttempt(latest)
	if !ok || a.ID != attemptID || a.Execution == nil {
		return latest, errors.Join(cause, ErrUnknownOutcome)
	}
	env, intentKey, loadErr := c.loadIntent(ctx, route, latest, a)
	if loadErr != nil {
		return latest, errors.Join(cause, loadErr)
	}
	failure := boundedFailure("pre-dispatch staging failed: "+cause.Error(), 0)
	_, proofKey, _, proofErr := c.persistNotDispatched(ctx, route, latest, a, env, intentKey, failure)
	if proofErr != nil {
		return latest, errors.Join(cause, proofErr)
	}
	failed, failErr := c.jobs.Fail(ctx, latest.ID, attemptID, failure, latest.Version)
	if failErr != nil {
		return failed, errors.Join(cause, failErr)
	}
	if err := c.requireRoute(ctx, route); err != nil {
		return failed, errors.Join(cause, err)
	}
	released, releaseErr := c.jobs.RecordReleased(ctx, failed.ID, a.ID, failed.Version, c.now().UTC())
	if releaseErr != nil {
		return failed, errors.Join(cause, fmt.Errorf("record not-dispatched attempt release (%s): %w", proofKey, releaseErr))
	}
	return released, cause
}

const notDispatchedSuffix = "not-dispatched.json"

var errNotDispatchedProofMissing = errors.New("renderjobs: no not-dispatched proof exists for attempt")

func notDispatchedKey(jobID, attemptID string) string {
	return quiescenceBlobPrefix + jobID + "/" + attemptID + "/" + notDispatchedSuffix
}

func (c *Coordinator) persistNotDispatched(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, intentKey, failure string) (persistedQuiescence, string, string, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return persistedQuiescence{}, "", "", err
	}
	if a.Execution == nil || env.JobID != j.ID || env.AttemptID != a.ID || env.InputFingerprint != j.InputFingerprint || env.Route != route || intentKey != a.Execution.ArtifactID {
		return persistedQuiescence{}, "", "", ErrUnknownOutcome
	}
	proof := persistedQuiescence{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, ClaimID: a.Execution.ID, IntentKey: intentKey, Route: route, NotDispatched: true, Failure: failure, ObservedAt: c.now().UTC()}
	proof.EvidenceSHA256 = quiescenceEvidenceDigest(proof)
	data, err := json.Marshal(proof)
	if err != nil {
		return persistedQuiescence{}, "", "", err
	}
	sha := digestBytes(data)
	key := notDispatchedKey(j.ID, a.ID)
	if err := c.putVerified(ctx, route, key, data, "application/json", sha); err != nil {
		return persistedQuiescence{}, "", "", err
	}
	return proof, key, sha, nil
}

func (c *Coordinator) loadNotDispatched(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, env persistedIntent, intentKey string) (persistedQuiescence, string, string, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return persistedQuiescence{}, "", "", err
	}
	key := notDispatchedKey(j.ID, a.ID)
	r, _, err := c.blobs.Get(ctx, key)
	if err != nil {
		if isBlobNotFound(err) {
			return persistedQuiescence{}, "", "", errNotDispatchedProofMissing
		}
		return persistedQuiescence{}, "", "", fmt.Errorf("read not-dispatched proof: %w", err)
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxReceiptBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxReceiptBytes {
		return persistedQuiescence{}, "", "", errors.New("renderjobs: not-dispatched proof is unreadable")
	}
	var proof persistedQuiescence
	if err := decodeJSON(data, &proof); err != nil || !proof.NotDispatched || proof.Version != 1 || proof.JobID != j.ID || proof.AttemptID != a.ID || proof.InputFingerprint != j.InputFingerprint || a.Execution == nil || proof.ClaimID != a.Execution.ID || proof.IntentKey != intentKey || proof.Route != route || env.JobID != j.ID || env.AttemptID != a.ID || env.InputFingerprint != j.InputFingerprint || proof.Failure == "" || proof.ObservedAt.IsZero() || proof.EvidenceSHA256 == "" || quiescenceEvidenceDigest(proof) != proof.EvidenceSHA256 || (j.State == jobs.Failed && proof.Failure != j.Failure) {
		return persistedQuiescence{}, "", "", ErrUnknownOutcome
	}
	return proof, key, digestBytes(data), nil
}

func quiescenceEvidenceDigest(proof persistedQuiescence) string {
	proof.EvidenceSHA256 = ""
	data, _ := json.Marshal(proof)
	return digestBytes(data)
}

func isMissingNotDispatchedProof(err error) bool {
	return errors.Is(err, errNotDispatchedProofMissing)
}

func isBlobNotFound(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if errors.Is(current, os.ErrNotExist) {
			return true
		}
		message := current.Error()
		if strings.HasPrefix(message, "blobstore: \"") && strings.HasSuffix(message, "\" not found") {
			return true
		}
	}
	return false
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
	var releaseErr error
	if ok && a.Execution != nil && a.ExecutionReleasedAt == nil {
		var env persistedIntent
		var intentKey string
		env, intentKey, releaseErr = c.loadIntent(ctx, route, j, a)
		if releaseErr == nil {
			if j.State == jobs.Failed {
				proof, _, _, proofErr := c.loadNotDispatched(ctx, route, j, a, env, intentKey)
				if proofErr == nil {
					j, releaseErr = c.jobs.RecordReleased(ctx, j.ID, a.ID, j.Version, proof.ObservedAt)
					if releaseErr != nil {
						return j, releaseErr
					}
					return j, c.finalizeTerminal(ctx, route, j)
				}
				if !isMissingNotDispatchedProof(proofErr) {
					return j, fmt.Errorf("failed execution lacks verified dispatch-owned no-launch proof: %w", proofErr)
				}
			}
			var ref platform.OperationRef
			ref, releaseErr = c.ops.Lookup(ctx, env.Intent)
			if errors.Is(releaseErr, platform.ErrOperationNotFound) {
				j, releaseErr = c.jobs.RecordReleased(ctx, j.ID, a.ID, j.Version, c.now().UTC())
			} else if releaseErr == nil {
				j, releaseErr = c.releaseAttempt(ctx, route, j, a, ref)
			}
		}
	}
	// Subject projection is an independent retryable obligation. A native
	// release failure must not prevent known terminal media state from settling.
	return j, errors.Join(releaseErr, c.finalizeTerminal(ctx, route, j))
}

type preparedAttempt = preparedOperation

func (c *Coordinator) loadClosure(ctx context.Context, route routePin, j jobs.Job) (immutableClosure, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return immutableClosure{}, err
	}
	binding := bindingByRole(j.Input, "render_closure")
	if binding.ArtifactID == "" || !strings.HasPrefix(binding.ArtifactID, closureBlobPrefix) || !validDigest(binding.SHA256) || binding.RevisionID != "sha256:"+binding.SHA256 {
		return immutableClosure{}, errors.New("renderjobs: job does not bind an immutable render closure")
	}
	data, err := c.readBlob(ctx, binding.ArtifactID, maxManifestBytes, binding.SHA256)
	if err != nil {
		return immutableClosure{}, fmt.Errorf("read immutable render closure: %w", err)
	}
	closure, err := decodeClosure(data)
	if err != nil {
		return immutableClosure{}, err
	}
	if closure.ProjectID != j.Input.ProjectID || closure.ShotID == "" || j.Input.ShotRevisionID != shotRevisionID(closure.ShotID, closure.Revision) || j.Input.ShotRevision != closure.Revision || closure.RecipeSHA256 != bindingDigest(j.Input, "shot_snapshot") || closure.RendererVersion != j.Input.RendererVersion || closure.OutputProfile != j.Input.OutputProfile {
		return immutableClosure{}, errors.New("renderjobs: closure identity differs from immutable job input")
	}
	workerBinding := bindingByRole(j.Input, "worker_executable")
	if workerBinding.ArtifactID != closure.WorkerKey || workerBinding.SHA256 != closure.WorkerSHA256 || workerBinding.RevisionID != "sha256:"+closure.WorkerSHA256 || !strings.HasPrefix(workerBinding.ArtifactID, workerBlobPrefix) {
		return immutableClosure{}, errors.New("renderjobs: closure worker binding differs from job input")
	}
	profileDigest, err := toolProfileDigest(closure.WorkerSHA256, closure.OutputProfile, closure.Profile)
	if err != nil || profileDigest != j.Input.ToolProfileDigest {
		return immutableClosure{}, errors.New("renderjobs: closure tool profile digest differs from job input")
	}
	for _, input := range closure.Inputs {
		binding := bindingByRole(j.Input, "source_"+input.Role)
		if binding.ArtifactID != input.BlobKey || binding.SHA256 != input.MediaSHA || binding.RevisionID != "sha256:"+input.MediaSHA || !validDigest(input.MediaSHA) || input.MediaSize <= 0 || input.MediaSize > maxWorkerInputBytes || input.BlobKey != inputBlobPrefix+input.MediaSHA+"/"+input.Role {
			return immutableClosure{}, fmt.Errorf("renderjobs: closure %s source binding differs from job input", input.Role)
		}
	}
	return closure, nil
}

func (c *Coordinator) prepareAttemptFromClosure(ctx context.Context, route routePin, j jobs.Job, a jobs.Attempt, closure immutableClosure) (preparedAttempt, error) {
	if err := c.requireRoute(ctx, route); err != nil {
		return preparedAttempt{}, err
	}
	files := make([]stagedFile, 0, len(closure.Inputs))
	imageBindings := make(map[string]rendering.ImageBinding, len(closure.Inputs))
	var totalBytes int64
	for _, input := range closure.Inputs {
		if err := c.requireRoute(ctx, route); err != nil {
			return preparedAttempt{}, err
		}
		totalBytes += input.MediaSize
		if totalBytes > maxWorkerInputBytes {
			return preparedAttempt{}, errors.New("renderjobs: aggregate source media exceeds staging limit")
		}
		files = append(files, stagedFile{Name: input.Name, BlobKey: input.BlobKey, SHA256: input.MediaSHA, Size: input.MediaSize, Mode: manifestMode})
		imageBindings[input.Role] = rendering.ImageBinding{Name: input.Name, SHA256: input.MediaSHA, SizeBytes: input.MediaSize}
	}
	version := 1
	if closure.Presentation != nil {
		version = 2
	}
	manifest := rendering.Manifest{
		Version: version, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint,
		Profile: closure.Profile, Plate: imageBindings["plate"], Screen: imageBindings["screen"],
		ActiveMask: imageBindings["active_mask"], HardwareMask: optionalImageBinding(imageBindings, "hardware_mask"),
		SurfaceWidth: closure.SurfaceWidth, SurfaceHeight: closure.SurfaceHeight, Quad: closure.Quad, Samples: closure.Samples, Presentation: closure.Presentation, PresentationLayout: closure.PresentationLayout,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return preparedAttempt{}, err
	}
	return preparedAttempt{Mode: operationStillRender, ManifestBytes: append(manifestBytes, '\n'), Files: files}, nil
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
	if manifest.JobID != j.ID || manifest.AttemptID != a.ID || manifest.InputFingerprint != j.InputFingerprint || manifest.Profile != j.Input.RendererVersion {
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
	if j.Input.SubjectKind == jobs.SubjectGeometryCreate {
		workspace, workspaceErr := attemptWorkspace(route.Root, j.ID, a.ID)
		if workspaceErr != nil {
			return "", "", workspaceErr
		}
		outcome, outcomeSHA, readErr := readGeometryInferenceOutcome(workspace, j, a)
		if readErr == nil {
			evidence.InferenceOutcome = outcome
			evidence.InferenceOutcomeSHA256 = outcomeSHA
			if outcome.SemanticIssue != nil {
				evidence.RejectedRecipeStatus = "unavailable"
				if ref := c.persistRejectedGeometryRecipe(ctx, route, workspace, j, a, outcome, outcomeSHA); ref != nil {
					evidence.RejectedRecipe = ref
					evidence.RejectedRecipeStatus = "available"
				}
			}
		} else if !os.IsNotExist(readErr) {
			return "", "", fmt.Errorf("renderjobs: read geometry inference outcome: %w", readErr)
		}
	}
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

func (c *Coordinator) persistRejectedGeometryRecipe(ctx context.Context, route routePin, workspace string, j jobs.Job, a jobs.Attempt, outcome *rendering.GeometryInferenceOutcome, outcomeSHA string) *rejectedGeometryRecipeArtifact {
	if outcome == nil || outcome.SemanticIssue == nil || !validGeometrySemanticIssue(outcome.SemanticIssue) || outcome.ValueDisposition != "retained_exact" || outcome.ValueFile != rendering.GeometryResponseValueName || outcome.ValueBytes <= 0 || outcome.ValueBytes > geometry.MaxRecipeBytes || !validDigest(outcome.ValueSHA256) || !validDigest(outcomeSHA) {
		return nil
	}
	data, err := readWorkspaceFile(workspace, rendering.GeometryResponseValueName, geometry.MaxRecipeBytes)
	if err != nil || len(data) != outcome.ValueBytes || digestBytes(data) != outcome.ValueSHA256 || !json.Valid(data) {
		return nil
	}
	key := operationEvidenceBlobPrefix + j.ID + "/" + a.ID + "/rejected-recipe-" + outcome.ValueSHA256 + ".json"
	if err := c.putVerified(ctx, route, key, data, "application/json", outcome.ValueSHA256); err != nil {
		return nil
	}
	return &rejectedGeometryRecipeArtifact{Version: 1, JobID: j.ID, AttemptID: a.ID, InputFingerprint: j.InputFingerprint, OutcomeSHA256: outcomeSHA, OwnerID: OwnerID, ScopeID: geometryRejectedRecipeScope, ArtifactID: key, SHA256: outcome.ValueSHA256, SizeBytes: int64(len(data))}
}

func validGeometrySemanticIssue(issue *geometry.SemanticIssue) bool {
	if issue == nil || issue.PartID == "" || len(issue.PartID) > 48 {
		return false
	}
	for _, ch := range issue.PartID {
		if ch < 'a' || ch > 'z' {
			if ch < '0' || ch > '9' {
				if ch != '_' && ch != '-' {
					return false
				}
			}
		}
	}
	switch issue.Code {
	case "part_outside_envelope":
		return (issue.Axis == "x" || issue.Axis == "y" || issue.Axis == "z") && issue.BevelMM == 0 && issue.MinimumDimensionMM == 0 && issue.MaxBevelExclusiveMM == 0 && issue.Message == "part exceeds the locked envelope"
	case "rotated_part_outside_envelope":
		return (issue.Axis == "x" || issue.Axis == "y" || issue.Axis == "z") && issue.BevelMM == 0 && issue.MinimumDimensionMM == 0 && issue.MaxBevelExclusiveMM == 0 && issue.Message == "rotated part exceeds the locked envelope"
	case "bevel_exceeds_part_dimensions":
		return issue.Axis == "" && finitePositive(issue.BevelMM) && finitePositive(issue.MinimumDimensionMM) && finitePositive(issue.MaxBevelExclusiveMM) && issue.MaxBevelExclusiveMM == issue.MinimumDimensionMM/2 && issue.BevelMM >= issue.MaxBevelExclusiveMM && issue.Message == "bevel must be less than half the smallest part dimension"
	default:
		return false
	}
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func readGeometryInferenceOutcome(workspace string, j jobs.Job, a jobs.Attempt) (*rendering.GeometryInferenceOutcome, string, error) {
	outcomeBytes, err := readWorkspaceFile(workspace, rendering.GeometryInferenceOutcomeName, 64<<10)
	if err != nil {
		return nil, "", err
	}
	var outcome rendering.GeometryInferenceOutcome
	if err := decodeJSON(outcomeBytes, &outcome); err != nil {
		return nil, "", err
	}
	canonical, err := json.Marshal(outcome)
	if err != nil {
		return nil, "", err
	}
	usage := outcome.Usage
	if !bytes.Equal(outcomeBytes, append(canonical, '\n')) || outcome.Version != 1 || outcome.JobID != j.ID || outcome.AttemptID != a.ID || outcome.InputFingerprint != j.InputFingerprint || outcome.WorkerSHA256 != bindingDigest(j.Input, "worker_executable") || outcome.ProviderCalls < 0 || outcome.ProviderCalls > 1 || !validDigest(outcome.ValueSHA256) || outcome.ValueBytes < 0 || outcome.ReceivedAt.IsZero() || len(outcome.Provider) > 256 || len(outcome.Model) > 256 || usage == nil || usage.GetInputTokens() < 0 || usage.GetOutputTokens() < 0 || usage.GetCostMicros() < 0 || !validGeometryUsageProvenance(usage.GetInputTokensProvenance()) || !validGeometryUsageProvenance(usage.GetOutputTokensProvenance()) || !validGeometryUsageProvenance(usage.GetCostProvenance()) || outcome.SemanticIssue != nil && !validGeometrySemanticIssue(outcome.SemanticIssue) {
		return nil, "", errors.New("renderjobs: geometry inference outcome does not match immutable attempt")
	}
	return &outcome, digestBytes(outcomeBytes), nil
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
	wantMode, modeErr := operationModeForJob(j.Input)
	storedMode := env.Mode
	if storedMode == "" {
		storedMode = operationStillRender
	}
	if modeErr != nil || storedMode != wantMode || env.Version != 1 || env.JobID != j.ID || env.AttemptID != a.ID || env.InputFingerprint != j.InputFingerprint || env.Route != route || env.Intent.Owner != OwnerID || env.Intent.AttemptID != a.ID || env.Intent.InvocationID != a.Execution.ScopeID || env.Intent.SpecDigest == "" || env.WorkerSHA256 != bindingDigest(j.Input, "worker_executable") {
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

func (c *Coordinator) createPhysicalSandbox(ctx context.Context, route routePin, workspace string) error {
	for _, relative := range []string{"home", "tmp", "config", "cache", "config/blender", "scripts", "datafiles", "pycache"} {
		if err := c.requireRoute(ctx, route); err != nil {
			return err
		}
		path := filepath.Join(workspace, relative)
		if !strings.HasPrefix(path, workspace+string(filepath.Separator)) || filepath.Clean(path) != path {
			return errors.New("renderjobs: physical sandbox path escaped its attempt root")
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			if !os.IsExist(err) {
				return err
			}
			info, statErr := os.Lstat(path)
			if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("renderjobs: physical sandbox contains an unsafe directory")
			}
		} else {
			c.roots.RecordWrite(ctx)
		}
	}
	return validateDirectoryTree(workspace)
}

func (c *Coordinator) writeWorkspaceFile(ctx context.Context, route routePin, workspace, name string, data []byte, mode os.FileMode, exclusive bool) error {
	return c.writeWorkspaceReader(ctx, route, workspace, name, bytes.NewReader(data), int64(len(data)), digestBytes(data), mode, exclusive)
}

func (c *Coordinator) stagePreparedFile(ctx context.Context, route routePin, workspace string, job jobs.Job, file stagedFile, modeLimit int64) error {
	if !stageFilePattern.MatchString(file.Name) || file.Name == workerFileName || file.Name == "intent.json" || file.Name == rendering.ManifestName || file.Name == rendering.PhysicalManifestName || file.Name == rendering.PhysicalReceiptName || file.Name == "inspection-manifest.json" || file.Name == "video-manifest.json" || strings.Contains(file.Name, "..") {
		return errors.New("renderjobs: subject staged an unsafe filename")
	}
	if file.Size == 0 && len(file.Data) > 0 {
		file.Size = int64(len(file.Data))
	}
	limit := modeLimit
	if limit <= 0 {
		limit = maxWorkerInputBytes
	}
	if file.MaxBytes > 0 && file.MaxBytes < limit {
		limit = file.MaxBytes
	}
	if file.Size <= 0 || file.Size > limit {
		return errors.New("renderjobs: subject file size is outside bounds")
	}
	if file.Mode == 0 {
		file.Mode = manifestMode
	}
	if file.Open != nil {
		if file.BlobKey != "" || len(file.Data) != 0 || !validDigest(file.SHA256) {
			return errors.New("renderjobs: streamed input has conflicting source fields")
		}
		reader, err := file.Open(ctx)
		if err != nil {
			return err
		}
		writeErr := c.writeWorkspaceReader(ctx, route, workspace, file.Name, reader, file.Size, file.SHA256, file.Mode, true)
		closeErr := reader.Close()
		return errors.Join(writeErr, closeErr)
	}
	if file.BlobKey != "" {
		if len(file.Data) != 0 || !validDigest(file.SHA256) {
			return errors.New("renderjobs: blob input must have only a size and digest-bound reference")
		}
		bound := false
		for _, b := range job.Input.Bindings {
			if b.ArtifactID == file.BlobKey && b.SHA256 == file.SHA256 {
				bound = true
				break
			}
		}
		if !bound {
			return errors.New("renderjobs: staged blob is not bound by immutable job input")
		}
		if err := c.requireRoute(ctx, route); err != nil {
			return err
		}
		reader, _, err := c.blobs.Get(ctx, file.BlobKey)
		if err != nil {
			return err
		}
		defer reader.Close()
		return c.writeWorkspaceReader(ctx, route, workspace, file.Name, reader, file.Size, file.SHA256, file.Mode, true)
	}
	if file.SHA256 == "" {
		file.SHA256 = digestBytes(file.Data)
	}
	if int64(len(file.Data)) != file.Size || digestBytes(file.Data) != file.SHA256 {
		return errors.New("renderjobs: in-memory subject file differs from its size or digest")
	}
	return c.writeWorkspaceReader(ctx, route, workspace, file.Name, bytes.NewReader(file.Data), file.Size, file.SHA256, file.Mode, true)
}

func (c *Coordinator) writeWorkspaceReader(ctx context.Context, route routePin, workspace, name string, reader io.Reader, expectedSize int64, expectedSHA string, mode os.FileMode, exclusive bool) error {
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
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(reader, expectedSize+1))
	if err != nil {
		_ = file.Close()
		return err
	}
	if written != expectedSize || hex.EncodeToString(hasher.Sum(nil)) != expectedSHA {
		_ = file.Close()
		return errors.New("renderjobs: staged bytes differ from pinned size or digest")
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
	if !validDigest(digest) || digestBytes(data) != digest {
		return errors.New("renderjobs: blob payload differs from its immutable digest")
	}
	if contentAddressedInputKey(key, digest) {
		if reused, err := c.reuseVerifiedBlob(ctx, route, key, int64(len(data)), mime, digest); err != nil || reused {
			return err
		}
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
	return c.readBlobForRoute(ctx, route, key, limit, expectedDigest)
}

func (c *Coordinator) readBlobForRoute(ctx context.Context, route routePin, key string, limit int64, expectedDigest string) ([]byte, error) {
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
	if err := c.requireRoute(ctx, route); err != nil {
		return nil, err
	}
	return data, nil
}
