package renderjobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"device-studio/internal/httpx"
	"device-studio/internal/jobs"
	projectsDomain "device-studio/internal/projects"
	"device-studio/internal/rendering"
	domain "device-studio/internal/renderjobs"
	shotsDomain "device-studio/internal/shots"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Coordinator *domain.Coordinator
	Jobs        jobs.Service
	Activator   RouteActivator
	DB          *database.RoutedDB
	Roots       *filerouting.RoutedRoots
	Logger      *log.Logger
}
type (
	RouteActivator interface{ ActivateTestRoute(context.Context) error }
	handler        struct{ deps Deps }
)

func NewConnectHandler(deps Deps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &handler{deps: deps}
}

func (h *handler) SubmitRender(ctx context.Context, req *connect.Request[shotsv1.SubmitRenderRequest]) (*connect.Response[shotsv1.SubmitRenderResponse], error) {
	var timing *domain.SubmitTiming
	if req.Msg.RenderMode == shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG {
		ctx, timing = domain.NewSubmitTiming(ctx)
	}
	spanStarted := time.Now()
	var err error
	ctx, err = pinRequestRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed render storage is unavailable"))
	}
	domain.RecordSubmitTiming(ctx, "route_pin", time.Since(spanStarted))
	profile := domain.OutputProfilePNG
	videoMode := rendering.VideoMode("")
	switch req.Msg.RenderMode {
	case shotsv1.RenderMode_RENDER_MODE_UNSPECIFIED, shotsv1.RenderMode_RENDER_MODE_STILL:
		if req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_UNSPECIFIED && req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported still output profile"))
		}
	case shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG:
		if req.Msg.SampleOutputIndex != 0 || req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_UNSPECIFIED && req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported video sample request"))
		}
		videoMode, profile = rendering.VideoModeSample, "video-sample-png-v1"
	case shotsv1.RenderMode_RENDER_MODE_VIDEO_FINAL_MP4:
		if req.Msg.SampleOutputIndex != 0 || req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_UNSPECIFIED && req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported video output profile"))
		}
		videoMode, profile = rendering.VideoModeFinal, "video-final-mp4-h264-sdr-v1"
	case shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG:
		if req.Msg.SampleOutputIndex != 0 || req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_UNSPECIFIED && req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PNG_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported physical still output profile"))
		}
		// Physical rendering is selected from the immutable shot subject. The
		// public output format remains the shared PNG profile; the renderer's
		// quality profile is pinned separately in the physical manifest.
	case shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG:
		if req.Msg.SampleOutputIndex != 0 || (req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_PREVIEW_PNG_V1 && req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_PNG_V1) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("physical preview requires a pinned preview or diagnostic PNG profile"))
		}
		if req.Msg.OutputProfile == shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_PNG_V1 {
			profile = domain.OutputProfilePhysicalDiagnosticPNG
		} else {
			profile = domain.OutputProfilePhysicalPreviewPNG
		}
	case shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4:
		if req.Msg.SampleOutputIndex != 0 || req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_MP4_H264_SDR_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("physical motion requires the pinned MP4 H.264 SDR output profile"))
		}
		profile = domain.OutputProfileMP4
	case shotsv1.RenderMode_RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4:
		if req.Msg.SampleOutputIndex != 0 || req.Msg.OutputProfile != shotsv1.RenderOutputProfile_RENDER_OUTPUT_PROFILE_PHYSICAL_DIAGNOSTIC_MOTION_MP4_V1 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("physical motion diagnostic requires its pinned low-sample MP4 profile"))
		}
		profile = domain.OutputProfilePhysicalDiagnosticMotionMP4
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported render mode"))
	}
	spanStarted = time.Now()
	job, created, cacheReceipt, err := h.deps.Coordinator.SubmitWithReceipt(ctx, domain.SubmitRequest{ProjectID: req.Msg.ProjectId, ShotID: req.Msg.ShotId, Revision: req.Msg.Revision, IdempotencyKey: req.Msg.IdempotencyKey, OutputProfile: profile, VideoMode: videoMode, ReuseCachedSuccess: req.Msg.ReuseCachedSuccess})
	domain.RecordSubmitTiming(ctx, "coordinator_submit", time.Since(spanStarted))
	if err != nil {
		return nil, h.connectError("submit render", err)
	}
	if h.deps.Activator != nil {
		spanStarted = time.Now()
		if err := h.deps.Activator.ActivateTestRoute(ctx); err != nil {
			h.deps.Logger.Printf("render test route activation failed after accepted submission %s: %v", job.ID, err)
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("render was accepted but its routed progress context could not be confirmed; retry with the same idempotency key"))
		}
		domain.RecordSubmitTiming(ctx, "pump_activate", time.Since(spanStarted))
	}
	response := connect.NewResponse(&shotsv1.SubmitRenderResponse{Job: jobToProto(job, ""), ReusedExisting: !created})
	if cacheReceipt != nil {
		response.Msg.CacheReuse = &shotsv1.CachedRenderReuse{
			SourceRenderId: cacheReceipt.SourceRenderID, SourceAttemptId: cacheReceipt.SourceAttemptID,
			ArtifactSha256: cacheReceipt.SHA256, OutputKind: cacheReceipt.OutputKind,
		}
	}
	domain.SetSubmitTimingHeader(response.Header(), timing)
	return response, nil
}

func (h *handler) GetVideoSampleBinding(ctx context.Context, req *connect.Request[shotsv1.GetVideoSampleBindingRequest]) (*connect.Response[shotsv1.GetVideoSampleBindingResponse], error) {
	ctx, err := pinRequestRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed render storage is unavailable"))
	}
	binding, err := h.deps.Coordinator.GetVideoSampleBinding(ctx, req.Msg.ProjectId, req.Msg.ShotId, req.Msg.RenderId)
	if err != nil {
		return nil, h.connectError("get video sample binding", err)
	}
	return connect.NewResponse(&shotsv1.GetVideoSampleBindingResponse{Binding: binding}), nil
}

func (h *handler) GetRender(ctx context.Context, req *connect.Request[shotsv1.GetRenderRequest]) (*connect.Response[shotsv1.GetRenderResponse], error) {
	var err error
	ctx, err = pinRequestRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed render storage is unavailable"))
	}
	job, err := h.deps.Coordinator.GetProjectJob(ctx, req.Msg.ProjectId, req.Msg.RenderId)
	if err != nil {
		return nil, h.connectError("get render", err)
	}
	return connect.NewResponse(&shotsv1.GetRenderResponse{Job: jobToProto(job, req.Msg.ProjectId)}), nil
}

func (h *handler) ListRenders(ctx context.Context, req *connect.Request[shotsv1.ListRendersRequest]) (*connect.Response[shotsv1.ListRendersResponse], error) {
	var err error
	ctx, err = pinRequestRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed render storage is unavailable"))
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page size must be between 1 and 100"))
	}
	cursor, err := decodeCursor(req.Msg.PageToken)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid render page token"))
	}
	rows, err := h.deps.Coordinator.ListProjectJobs(ctx, req.Msg.ProjectId, cursor, limit+1)
	if err != nil {
		return nil, h.connectError("list renders", err)
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	out := &shotsv1.ListRendersResponse{Jobs: make([]*shotsv1.RenderJob, 0, len(rows))}
	for _, job := range rows {
		out.Jobs = append(out.Jobs, jobToProto(job, req.Msg.ProjectId))
	}
	if more && len(rows) > 0 {
		out.NextPageToken, err = encodeCursor(jobs.Cursor{CreatedAt: rows[len(rows)-1].CreatedAt, ID: rows[len(rows)-1].ID})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("render page encoding failed"))
		}
	}
	return connect.NewResponse(out), nil
}

func (h *handler) CancelRender(ctx context.Context, req *connect.Request[shotsv1.CancelRenderRequest]) (*connect.Response[shotsv1.CancelRenderResponse], error) {
	var err error
	ctx, err = pinRequestRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed render storage is unavailable"))
	}
	job, err := h.deps.Coordinator.GetProjectJob(ctx, req.Msg.ProjectId, req.Msg.RenderId)
	if err != nil {
		return nil, h.connectError("cancel render", err)
	}
	job, err = h.deps.Coordinator.Cancel(ctx, req.Msg.RenderId, req.Msg.ExpectedVersion)
	if err != nil {
		return nil, h.connectError("cancel render", err)
	}
	if h.deps.Activator != nil {
		if err := h.deps.Activator.ActivateTestRoute(ctx); err != nil {
			h.deps.Logger.Printf("render test route activation failed after cancel %s: %v", job.ID, err)
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("cancellation was recorded but routed reconciliation could not be confirmed"))
		}
	}
	return connect.NewResponse(&shotsv1.CancelRenderResponse{Job: jobToProto(job, req.Msg.ProjectId)}), nil
}

// pinRequestRoute binds every test-mode render request before it touches the
// ledger or blob roots. The returned context is also retained by Pump on
// accepted submit/cancel operations, so later work cannot rebind a new lease.
func pinRequestRoute(ctx context.Context, db *database.RoutedDB, roots *filerouting.RoutedRoots) (context.Context, error) {
	if !database.IsTestMode(ctx) {
		return ctx, nil
	}
	if db == nil || roots == nil {
		return nil, errors.New("test render route is not fully configured")
	}
	leaseID := db.ActiveLeaseID()
	if leaseID == "" {
		return nil, errors.New("test render database lease is unavailable")
	}
	pinned, err := db.PinTestPool(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	pinned, err = roots.PinTestRoots(pinned, leaseID)
	if err != nil {
		return nil, err
	}
	return pinned, nil
}

func (h *handler) connectError(op string, err error) error {
	var shotMissing shotsDomain.ErrShotNotFound
	var revisionMissing shotsDomain.ErrRevisionNotFound
	var projectMissing projectsDomain.ErrProjectNotFound
	var invalidShot shotsDomain.ErrInvalidShot
	var renderBlocked shotsDomain.ErrRenderBlocked
	var unsupportedPresentation shotsDomain.ErrUnsupportedPresentationVersion
	var sourceUnavailable shotsDomain.ErrSourceMediaUnavailable
	switch {
	case errors.Is(err, jobs.ErrNotFound), errors.As(err, &shotMissing), errors.As(err, &revisionMissing), errors.As(err, &projectMissing):
		return connect.NewError(connect.CodeNotFound, errors.New("render job not found"))
	case errors.Is(err, jobs.ErrConcurrentTransition):
		return connect.NewError(connect.CodeAborted, errors.New("render job changed; reload before retrying"))
	case errors.Is(err, domain.ErrNotEligible), errors.As(err, &renderBlocked):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("shot revision is not eligible for rendering; review its validation issues"))
	case errors.As(err, &unsupportedPresentation):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("this saved presentation version cannot be rendered by the current still worker"))
	case errors.As(err, &sourceUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("saved source media is unavailable for render validation"))
	case errors.Is(err, domain.ErrOutputUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("render output or source-derived sample binding is not available"))
	case errors.Is(err, rendering.ErrPhysicalBlenderProfileUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("managed Blender is unavailable for this render; inspect or repair the pinned tool through the control plane, then retry"))
	case errors.Is(err, jobs.ErrIdempotencyConflict):
		return connect.NewError(connect.CodeAlreadyExists, errors.New("idempotency key is already bound to a different render request"))
	case errors.Is(err, domain.ErrInvalidRequest), errors.Is(err, jobs.ErrInvalid), errors.As(err, &invalidShot):
		return connect.NewError(connect.CodeInvalidArgument, errors.New("invalid render request"))
	default:
		h.deps.Logger.Printf("%s failed: %v", op, err)
		return connect.NewError(connect.CodeInternal, errors.New("render operation failed"))
	}
}

type cursorToken struct {
	CreatedAt int64  `json:"created_at"`
	ID        string `json:"id"`
}

func encodeCursor(c jobs.Cursor) (string, error) {
	b, e := json.Marshal(cursorToken{CreatedAt: c.CreatedAt.UTC().UnixNano(), ID: c.ID})
	if e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(token string) (jobs.Cursor, error) {
	if token == "" {
		return jobs.Cursor{}, nil
	}
	b, e := base64.RawURLEncoding.DecodeString(token)
	if e != nil {
		return jobs.Cursor{}, e
	}
	var c cursorToken
	if e = json.Unmarshal(b, &c); e != nil || c.CreatedAt <= 0 || strings.TrimSpace(c.ID) == "" {
		return jobs.Cursor{}, fmt.Errorf("invalid cursor")
	}
	return jobs.Cursor{CreatedAt: time.Unix(0, c.CreatedAt).UTC(), ID: c.ID}, nil
}

func jobToProto(j jobs.Job, projectID string) *shotsv1.RenderJob {
	status := shotsv1.RenderJobStatus_RENDER_JOB_STATUS_NEEDS_RECONCILIATION
	var issue *shotsv1.RenderJobIssue
	summary := "Execution outcome requires reconciliation."
	switch j.State {
	case jobs.Queued:
		status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_QUEUED
		summary = "Queued for local rendering."
	case jobs.Running:
		if a, ok := lastAttempt(j); ok && a.ExecutionObserved {
			status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_OBSERVED
			summary = "The local renderer is running."
		} else {
			issue = &shotsv1.RenderJobIssue{Code: shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED, Summary: "Execution has been claimed but its running state is not yet confirmed."}
		}
	case jobs.CancelRequested:
		status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_CANCEL_REQUESTED
		summary = "Cancellation is requested; the native operation is being reconciled."
	case jobs.Succeeded:
		status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_SUCCEEDED
		summary = "The requested output is ready."
	case jobs.Failed:
		status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_FAILED
		summary = "The local render failed."
	case jobs.Cancelled:
		status = shotsv1.RenderJobStatus_RENDER_JOB_STATUS_CANCELLED
		summary = "The render was cancelled after native work stopped."
	default:
		issue = &shotsv1.RenderJobIssue{Code: shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTCOME_UNRESOLVED, Summary: "Execution outcome requires reconciliation."}
	}
	p := &shotsv1.RenderJob{Id: j.ID, ProjectId: j.Input.ProjectID, ShotId: shotIDFromInput(j.Input), ShotRevision: j.Input.ShotRevision, RenderMode: renderModeForJob(j), Status: status, Version: j.Version, CreatedAt: timestamppb.New(j.CreatedAt), UpdatedAt: timestamppb.New(j.UpdatedAt), Summary: summary, Issue: issue}
	if p.ProjectId == "" {
		p.ProjectId = projectID
	}
	if j.State == jobs.Failed {
		p.Issue = &shotsv1.RenderJobIssue{Code: shotsv1.RenderJobIssueCode_RENDER_JOB_ISSUE_CODE_OUTPUT_UNAVAILABLE, Summary: "The render failed; inspect the job and submit a corrected revision."}
	}
	if a, ok := lastAttempt(j); ok && a.Completion != nil {
		for _, o := range a.Completion.Outputs {
			if o.Kind == "still_png" || o.Kind == "physical_still_png" || o.Kind == "physical_preview_png" || o.Kind == "physical_diagnostic_png" || o.Kind == "video_sample_png" || o.Kind == "video_mp4" || o.Kind == "physical_motion_mp4" {
				p.Output = &shotsv1.RenderOutput{MediaType: o.MIMEType, SizeBytes: o.SizeBytes, WidthPixels: uint32(o.Width), HeightPixels: uint32(o.Height), Sha256: o.SHA256, ContentUrl: fmt.Sprintf("/api/v1/projects/%s/renders/%s/content", url.PathEscape(j.Input.ProjectID), url.PathEscape(j.ID))}
				break
			}
		}
	}
	return p
}

func renderModeForJob(j jobs.Job) shotsv1.RenderMode {
	switch j.Input.SubjectKind {
	case jobs.SubjectVideoRender:
		switch j.Input.OutputProfile {
		case "video-sample-png-v1":
			return shotsv1.RenderMode_RENDER_MODE_VIDEO_SAMPLE_PNG
		case "video-final-mp4-h264-sdr-v1":
			return shotsv1.RenderMode_RENDER_MODE_VIDEO_FINAL_MP4
		}
	case "", jobs.SubjectStillRender:
		return shotsv1.RenderMode_RENDER_MODE_STILL
	case jobs.SubjectPhysicalRender:
		if j.Input.OutputProfile == domain.OutputProfilePhysicalPreviewPNG || j.Input.OutputProfile == domain.OutputProfilePhysicalDiagnosticPNG {
			return shotsv1.RenderMode_RENDER_MODE_PHYSICAL_PREVIEW_PNG
		}
		if j.Input.OutputProfile == domain.OutputProfilePhysicalDiagnosticMotionMP4 {
			return shotsv1.RenderMode_RENDER_MODE_PHYSICAL_DIAGNOSTIC_MOTION_MP4
		}
		if j.Input.OutputProfile == domain.OutputProfilePhysicalMotionMP4 || j.Input.OutputProfile == domain.OutputProfilePhysicalMotionMP4V2 || j.Input.OutputProfile == domain.OutputProfilePhysicalRecordingMP4 || j.Input.OutputProfile == domain.OutputProfilePhysicalDiagnosticMotionMP4 {
			return shotsv1.RenderMode_RENDER_MODE_PHYSICAL_VIDEO_FINAL_MP4
		}
		return shotsv1.RenderMode_RENDER_MODE_PHYSICAL_STILL_PNG
	}
	return shotsv1.RenderMode_RENDER_MODE_UNSPECIFIED
}

func lastAttempt(j jobs.Job) (jobs.Attempt, bool) {
	if len(j.Attempts) == 0 {
		return jobs.Attempt{}, false
	}
	return j.Attempts[len(j.Attempts)-1], true
}

func shotIDFromInput(in jobs.Input) string {
	for _, b := range in.Bindings {
		if b.Role == "shot_snapshot" {
			parts := strings.Split(b.ArtifactID, ":")
			if len(parts) == 4 {
				return parts[1]
			}
		}
	}
	return ""
}

func newContentHandler(coordinator *domain.Coordinator, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, routeErr := pinRequestRoute(r.Context(), db, roots)
		if routeErr != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "routed render storage is unavailable")
			return
		}
		v := mux.Vars(r)
		videoJob, videoRef, videoReader, videoErr := coordinator.OpenVideoOutput(ctx, v["project_id"], v["render_id"])
		if videoErr == nil {
			if streamErr := streamVerifiedVideoOutput(w, videoJob.ID, videoRef, videoReader); streamErr != nil {
				logger.Printf("render output stream verification failed for %s: %v", videoJob.ID, streamErr)
				panic(http.ErrAbortHandler)
			}
			return
		}
		if !errors.Is(videoErr, jobs.ErrNotFound) {
			status, code, msg := contentErrorResponse(videoErr)
			if status == http.StatusInternalServerError {
				logger.Printf("video output read failed: %v", videoErr)
			}
			httpx.WriteError(w, status, code, msg)
			return
		}
		_, data, err := coordinator.OpenOutput(ctx, v["project_id"], v["render_id"])
		if err != nil {
			status, code, msg := contentErrorResponse(err)
			if status == http.StatusInternalServerError {
				logger.Printf("render output read failed: %v", err)
			}
			httpx.WriteError(w, status, code, msg)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}

func streamVerifiedVideoOutput(w http.ResponseWriter, jobID string, ref jobs.OutputRef, reader io.ReadCloser) error {
	w.Header().Set("Content-Type", ref.MIMEType)
	w.Header().Set("Content-Length", strconv.FormatInt(ref.SizeBytes, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	ext := ".mp4"
	if ref.MIMEType == "image/png" {
		ext = ".png"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"device-studio-%s%s\"", jobID, ext))
	w.WriteHeader(http.StatusOK)
	_, copyErr := io.Copy(w, reader)
	closeErr := reader.Close()
	return errors.Join(copyErr, closeErr)
}

func contentErrorResponse(err error) (int, string, string) {
	if errors.Is(err, jobs.ErrNotFound) {
		return http.StatusNotFound, httpx.CodeNotFound, "render output not found"
	}
	if errors.Is(err, domain.ErrOutputUnavailable) {
		return http.StatusConflict, httpx.CodeConflict, "render output is not ready"
	}
	return http.StatusInternalServerError, httpx.CodeInternal, "render output unavailable"
}
