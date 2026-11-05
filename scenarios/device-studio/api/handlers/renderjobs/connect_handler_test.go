package renderjobs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"device-studio/internal/httpx"
	"device-studio/internal/jobs"
	"device-studio/internal/projects"
	"device-studio/internal/rendering"
	domain "device-studio/internal/renderjobs"
	"device-studio/internal/shots"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	_ "modernc.org/sqlite"
)

type submitBoundaryShotResolver struct{ called bool }

func (s *submitBoundaryShotResolver) ResolveForRender(context.Context, string, int64) (shots.ResolvedShot, error) {
	s.called = true
	return shots.ResolvedShot{}, errors.New("reached shot resolver")
}

type submitBoundaryMedia struct{}

func (submitBoundaryMedia) Import(context.Context, string, string, int64, io.Reader) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}
func (submitBoundaryMedia) BeginImport(context.Context, string, string, int64, io.Reader) (projects.StagedSourceMediaImport, error) {
	return nil, errors.New("unused")
}
func (submitBoundaryMedia) BindInspectionJob(context.Context, string, string, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}
func (submitBoundaryMedia) BindInspectionAttempt(context.Context, string, string, string, string, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}
func (submitBoundaryMedia) StoreInspectionArtifacts(context.Context, string, string, string, string, string, string, []byte, []byte) (string, string, error) {
	return "", "", errors.New("unused")
}
func (submitBoundaryMedia) OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error) {
	return nil, nil, errors.New("unused")
}
func (submitBoundaryMedia) ProjectInspection(context.Context, projects.InspectionProjection) error {
	return errors.New("unused")
}
func (submitBoundaryMedia) Get(context.Context, string, string) (projects.SourceMedia, error) {
	return projects.SourceMedia{}, errors.New("unused")
}
func (submitBoundaryMedia) Open(context.Context, string, string) (projects.SourceMedia, io.ReadCloser, error) {
	return projects.SourceMedia{}, nil, errors.New("unused")
}
func (submitBoundaryMedia) List(context.Context, string, int32, string) ([]projects.SourceMedia, string, error) {
	return nil, "", errors.New("unused")
}

func TestPhysicalStillSubmitConnectBoundaryUsesPNGOutputProfile(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "jobs.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(jobs.Schema)); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	shotResolver := &submitBoundaryShotResolver{}
	coordinator, err := domain.New(domain.Dependencies{
		Shots: shotResolver, Media: submitBoundaryMedia{}, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), nil),
		Blobs: blobstore.NewMemoryBlobStore(), Roots: roots,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(Deps{Coordinator: coordinator, DB: db, Roots: roots, Logger: log.New(io.Discard, "", 0)})
	_, err = h.SubmitRender(ctx, connect.NewRequest(&shotsv1.SubmitRenderRequest{
		ProjectId: "project", ShotId: "shot", Revision: 1, IdempotencyKey: "physical-still-connect-boundary",
		RenderMode: shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG,
	}))
	if !shotResolver.called {
		t.Fatalf("physical submit did not reach coordinator shot resolution; error=%v", err)
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("physical submit code=%v, want resolver's internal error after accepted PNG profile mapping", got)
	}
}

func TestPhysicalPreviewSubmitConnectBoundaryUsesDedicatedPreviewProfile(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "jobs.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(jobs.Schema)); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	shotResolver := &submitBoundaryShotResolver{}
	coordinator, err := domain.New(domain.Dependencies{
		Shots: shotResolver, Media: submitBoundaryMedia{}, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), nil),
		Blobs: blobstore.NewMemoryBlobStore(), Roots: roots,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(Deps{Coordinator: coordinator, DB: db, Roots: roots, Logger: log.New(io.Discard, "", 0)})
	_, err = h.SubmitRender(ctx, connect.NewRequest(&shotsv1.SubmitRenderRequest{
		ProjectId: "project", ShotId: "shot", Revision: 1, IdempotencyKey: "physical-preview-connect-boundary",
		RenderMode:    shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG,
		OutputProfile: shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1,
	}))
	if !shotResolver.called {
		t.Fatalf("physical preview submit did not reach coordinator resolution; error=%v", err)
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("physical preview submit code=%v, want resolver's internal error after accepted preview mapping", got)
	}
}

func TestPhysicalPreviewSubmitRouteFailureReturnsUnavailableWithoutTimingPanic(t *testing.T) {
	h := NewConnectHandler(Deps{Logger: log.New(io.Discard, "", 0)})
	response, err := h.SubmitRender(database.WithTestMode(context.Background()), connect.NewRequest(&shotsv1.SubmitRenderRequest{
		ProjectId: "project", ShotId: "shot", Revision: 1, IdempotencyKey: "physical-preview-missing-route",
		RenderMode:    shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG,
		OutputProfile: shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1,
	}))
	if response != nil {
		t.Fatalf("route failure returned a response: %#v", response)
	}
	if got := connect.CodeOf(err); got != connect.CodeUnavailable {
		t.Fatalf("missing routed pool code=%v, want unavailable (error=%v)", got, err)
	}
}

func TestPhysicalMotionSubmitConnectBoundaryUsesPinnedMP4Profile(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "jobs.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(jobs.Schema)); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	shotResolver := &submitBoundaryShotResolver{}
	coordinator, err := domain.New(domain.Dependencies{
		Shots: shotResolver, Media: submitBoundaryMedia{}, Jobs: jobs.NewService(jobs.NewSQLiteRepository(db), nil),
		Blobs: blobstore.NewMemoryBlobStore(), Roots: roots,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(Deps{Coordinator: coordinator, DB: db, Roots: roots, Logger: log.New(io.Discard, "", 0)})
	_, err = h.SubmitRender(ctx, connect.NewRequest(&shotsv1.SubmitRenderRequest{
		ProjectId: "project", ShotId: "shot", Revision: 1, IdempotencyKey: "physical-motion-connect-boundary",
		RenderMode:    shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4,
		OutputProfile: shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1,
	}))
	if !shotResolver.called {
		t.Fatalf("physical motion submit did not reach coordinator resolution; error=%v", err)
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("physical motion submit code=%v, want resolver's internal error after accepted MP4 profile mapping", got)
	}
}

func TestConnectErrorMapsConflictAndRenderBlocked(t *testing.T) {
	h := &handler{deps: Deps{Logger: log.New(io.Discard, "", 0)}}
	blocked := shots.ErrRenderBlocked{ShotID: "shot", Revision: 2}
	if got := connect.CodeOf(h.connectError("submit", blocked)); got != connect.CodeFailedPrecondition {
		t.Fatalf("render blocked code = %v, want failed precondition", got)
	}
	if got := connect.CodeOf(h.connectError("submit", jobs.ErrIdempotencyConflict)); got != connect.CodeAlreadyExists {
		t.Fatalf("idempotency conflict code = %v, want already exists", got)
	}
}

func TestContentNotReadyIsConflictRatherThanInternalFailure(t *testing.T) {
	status, code, _ := contentErrorResponse(domain.ErrOutputUnavailable)
	if status != http.StatusConflict || code != httpx.CodeConflict {
		t.Fatalf("not-ready output mapped to %d/%q, want 409/conflict", status, code)
	}
}

func TestConnectErrorMapsUnavailableOutputAndSampleClaimToFailedPrecondition(t *testing.T) {
	h := &handler{deps: Deps{Logger: log.New(io.Discard, "", 0)}}
	for name, err := range map[string]error{
		"sentinel":         domain.ErrOutputUnavailable,
		"wrapped sentinel": errors.Join(errors.New("sample receipt is not eligible"), domain.ErrOutputUnavailable),
	} {
		t.Run(name, func(t *testing.T) {
			got := h.connectError("get video sample binding", err)
			if code := connect.CodeOf(got); code != connect.CodeFailedPrecondition {
				t.Fatalf("unavailable output code = %v, want failed precondition", code)
			}
			var ce *connect.Error
			if !errors.As(got, &ce) || ce.Message() != "render output or source-derived sample binding is not available" {
				t.Fatalf("unexpected public error: %v", got)
			}
		})
	}
	if code := connect.CodeOf(h.connectError("render", errors.New("unexpected storage failure"))); code != connect.CodeInternal {
		t.Fatalf("unrelated error code = %v, want internal", code)
	}
}

func TestKnownUnavailableManagedBlenderProfileMapsToSafeRepairablePrecondition(t *testing.T) {
	h := &handler{deps: Deps{Logger: log.New(io.Discard, "", 0)}}
	got := h.connectError("submit render", rendering.ErrPhysicalBlenderProfileUnavailable)
	if code := connect.CodeOf(got); code != connect.CodeFailedPrecondition {
		t.Fatalf("unavailable managed renderer code = %v, want failed precondition", code)
	}
	var connectErr *connect.Error
	if !errors.As(got, &connectErr) || connectErr.Message() != "managed Blender is unavailable for this render; inspect or repair the pinned tool through the control plane, then retry" {
		t.Fatalf("unexpected safe repair guidance: %v", got)
	}
	if code := connect.CodeOf(h.connectError("submit render", errors.New("unexpected renderer lookup failure"))); code != connect.CodeInternal {
		t.Fatalf("unexpected renderer failure was reclassified as %v, want internal", code)
	}
}

func TestVideoContentAbortsCorruptContentLengthTransfers(t *testing.T) {
	for _, size := range []int{8, 128 << 10} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			data := bytes.Repeat([]byte("v"), size)
			ref := jobs.OutputRef{MIMEType: "video/mp4", SizeBytes: int64(size), SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reader := &corruptTestOutputReader{remaining: data[:size-1]}
				if err := streamVerifiedVideoOutput(w, "job-id", ref, reader); err != nil {
					panic(http.ErrAbortHandler)
				}
			}))
			defer srv.Close()
			resp, err := http.Get(srv.URL)
			if err != nil {
				if size < 64<<10 {
					return // net/http buffered the whole attempted response and discarded it on abort.
				}
				t.Fatalf("GET: %v", err)
			}
			defer resp.Body.Close()
			got, readErr := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK || readErr == nil {
				t.Fatalf("corrupt transfer status=%d readErr=%v", resp.StatusCode, readErr)
			}
			if int64(len(got)) != int64(size-1) {
				t.Fatalf("corrupt response exposed %d bytes, want %d", len(got), size-1)
			}
		})
	}
}

type corruptTestOutputReader struct{ remaining []byte }

func (r *corruptTestOutputReader) Read(p []byte) (int, error) {
	if len(r.remaining) == 0 {
		return 0, errors.New("digest mismatch")
	}
	n := copy(p, r.remaining)
	r.remaining = r.remaining[n:]
	return n, nil
}

func (r *corruptTestOutputReader) Close() error { return nil }

func TestJobToProtoKeepsClaimedUnobservedOutcomeUnresolved(t *testing.T) {
	job := jobs.Job{
		ID: "job-id", Input: jobs.Input{ProjectID: "project-id", ShotRevision: 2, Bindings: []jobs.ArtifactBinding{{Role: "shot_snapshot", ArtifactID: "shot:shot-id:2:revision"}}},
		State: jobs.Running, Version: 4, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0),
		Attempts: []jobs.Attempt{{ID: "attempt-id", State: jobs.AttemptRunning}},
	}
	proto := jobToProto(job, "")
	if proto.Status != shotsv1.RenderJobStatus_RENDER_JOB_STATUS_NEEDS_RECONCILIATION || proto.Issue == nil || proto.Issue.Code != shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED {
		t.Fatalf("unobserved claim must be unresolved: %#v", proto)
	}
	job.Attempts[0].ExecutionObserved = true
	proto = jobToProto(job, "")
	if proto.Status != shotsv1.RenderJobStatus_RENDER_JOB_STATUS_OBSERVED || proto.Issue != nil {
		t.Fatalf("confirmed execution should be explicitly observed: %#v", proto)
	}
	if proto.ShotId != "shot-id" || proto.ShotRevision != 2 {
		t.Fatalf("shot binding missing from job projection: %#v", proto)
	}
}

func TestJobToProtoProjectsImmutableRenderModeAcrossStates(t *testing.T) {
	tests := []struct {
		name    string
		kind    jobs.SubjectKind
		profile string
		state   jobs.State
		want    shotsv1.RenderMode
	}{
		{"still queued", jobs.SubjectStillRender, "png-v1", jobs.Queued, shotsv1.RenderMode_RENDER_MODE_STILL},
		{"sample queued", jobs.SubjectVideoRender, "video-sample-png-v1", jobs.Queued, shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG},
		{"sample succeeded", jobs.SubjectVideoRender, "video-sample-png-v1", jobs.Succeeded, shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG},
		{"final failed", jobs.SubjectVideoRender, "video-final-mp4-h264-sdr-v1", jobs.Failed, shotsv1.RenderMode_RENDER_MODE_VIDEO_FINAL_MP4},
		{"physical motion v1 queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalMotionMP4, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical motion v1 succeeded", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalMotionMP4, jobs.Succeeded, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical motion v2 queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalMotionMP4V2, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical motion v2 succeeded", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalMotionMP4V2, jobs.Succeeded, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical recording queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalRecordingMP4, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical recording succeeded", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalRecordingMP4, jobs.Succeeded, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4},
		{"physical diagnostic motion queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalDiagnosticMotionMP4, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4},
		{"physical still failed", jobs.SubjectPhysicalRender, domain.OutputProfilePNG, jobs.Failed, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG},
		{"physical preview queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalPreviewPNG, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG},
		{"physical preview succeeded", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalPreviewPNG, jobs.Succeeded, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG},
		{"physical diagnostic queued", jobs.SubjectPhysicalRender, domain.OutputProfilePhysicalDiagnosticPNG, jobs.Queued, shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := jobs.Job{ID: "job-id", Input: jobs.Input{ProjectID: "project-id", SubjectKind: tt.kind, OutputProfile: tt.profile}, State: tt.state}
			if got := jobToProto(job, "").RenderMode; got != tt.want {
				t.Fatalf("render mode = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobToProtoKeepsPhysicalPreviewOutputDistinctFromFinalStill(t *testing.T) {
	job := jobs.Job{
		ID: "preview-job", State: jobs.Succeeded,
		Input: jobs.Input{ProjectID: "project", SubjectID: "shot", SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: domain.OutputProfilePhysicalPreviewPNG},
		Attempts: []jobs.Attempt{{Completion: &jobs.CompletionEvidence{Outputs: []jobs.OutputRef{{
			Kind: "physical_preview_png", MIMEType: "image/png", SizeBytes: 1234, Width: 960, Height: 540,
			SHA256: strings.Repeat("a", 64), ArtifactID: "routed-output",
		}}}}},
	}
	got := jobToProto(job, "")
	if got.RenderMode != shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG || got.Output == nil {
		t.Fatalf("preview mode/output projection = %#v", got)
	}
	if got.Output.WidthPixels != 960 || got.Output.HeightPixels != 540 || got.Output.MediaType != "image/png" {
		t.Fatalf("preview output metadata = %#v", got.Output)
	}
}

func TestJobToProtoKeepsDiagnosticOutputDistinct(t *testing.T) {
	job := jobs.Job{
		ID: "diagnostic-job", State: jobs.Succeeded,
		Input: jobs.Input{ProjectID: "project", SubjectID: "shot", SubjectKind: jobs.SubjectPhysicalRender, OutputProfile: domain.OutputProfilePhysicalDiagnosticPNG},
		Attempts: []jobs.Attempt{{Completion: &jobs.CompletionEvidence{Outputs: []jobs.OutputRef{{
			Kind: "physical_diagnostic_png", MIMEType: "image/png", SizeBytes: 1234, Width: 960, Height: 540,
			SHA256: strings.Repeat("b", 64), ArtifactID: "routed-output",
		}}}}},
	}
	got := jobToProto(job, "")
	if got.RenderMode != shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG || got.Output == nil || got.Output.WidthPixels != 960 || got.Output.HeightPixels != 540 {
		t.Fatalf("diagnostic job projection = %+v", got)
	}
}

func TestPublicRenderProjectionExcludesInspectionJobsAndPrivateArtifacts(t *testing.T) {
	job := jobs.Job{ID: "inspection-id", Input: jobs.Input{ProjectID: "project-id", SubjectKind: jobs.SubjectSourceInspection, SubjectID: "media-id"}, State: jobs.Succeeded, Attempts: []jobs.Attempt{{Completion: &jobs.CompletionEvidence{Outputs: []jobs.OutputRef{{Kind: "inspection_receipt", ArtifactID: "private/key", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MIMEType: "application/json"}}}}}}
	proto := jobToProto(job, "")
	if proto.Output != nil || proto.ShotId != "" {
		t.Fatalf("inspection internals leaked through render DTO: %#v", proto)
	}
}

func TestCursorTokenRoundTripAndMalformedInput(t *testing.T) {
	want := jobs.Cursor{CreatedAt: time.Date(2026, 10, 20, 4, 30, 0, 123, time.UTC), ID: "job-123"}
	token, err := encodeCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeCursor(token)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Fatalf("cursor round trip = %#v, want %#v", got, want)
	}
	for _, invalid := range []string{"!", "e30", "eyJjcmVhdGVkX2F0IjowLCJpZCI6ImEifQ"} {
		if _, err := decodeCursor(invalid); err == nil {
			t.Errorf("decodeCursor(%q) succeeded", invalid)
		}
	}
}
