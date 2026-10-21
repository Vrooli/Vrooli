package projects

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"

	"connectrpc.com/connect"
	"device-studio/internal/jobs"

	domain "device-studio/internal/projects"
	renderjobsDomain "device-studio/internal/renderjobs"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"

	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
)

type Deps struct {
	Service      domain.Service
	MediaService domain.SourceMediaService
	Inspection   SourceInspectionCoordinator
	Activator    RouteActivator
	DB           *database.RoutedDB
	Roots        *filerouting.RoutedRoots
	Logger       *log.Logger
}
type handler struct{ deps Deps }

type SourceInspectionCoordinator interface {
	EnsureSourceInspection(context.Context, string, string) (jobs.Job, bool, error)
	GetSourceInspectionStatus(context.Context, string, string) (jobs.Job, error)
	RetrySourceInspection(context.Context, string, string, int64) (jobs.Job, error)
	CancelSourceInspection(context.Context, string, string, int64) (jobs.Job, error)
}

type RouteActivator interface {
	ActivateTestRoute(context.Context) error
}

func NewConnectHandler(d Deps) *handler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &handler{deps: d}
}

func (h *handler) ListProjects(ctx context.Context, req *connect.Request[projectsv1.ListProjectsRequest]) (*connect.Response[projectsv1.ListProjectsResponse], error) {
	rows, next, err := h.deps.Service.List(ctx, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	out := &projectsv1.ListProjectsResponse{Projects: make([]*projectsv1.Project, 0, len(rows)), NextPageToken: next}
	for _, p := range rows {
		out.Projects = append(out.Projects, toProto(p))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) CreateProject(ctx context.Context, req *connect.Request[projectsv1.CreateProjectRequest]) (*connect.Response[projectsv1.CreateProjectResponse], error) {
	p, err := h.deps.Service.Create(ctx, domain.CreateInput{Name: req.Msg.Name})
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.CreateProjectResponse{Project: toProto(p)}), nil
}

func (h *handler) GetProject(ctx context.Context, req *connect.Request[projectsv1.GetProjectRequest]) (*connect.Response[projectsv1.GetProjectResponse], error) {
	p, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.GetProjectResponse{Project: toProto(p)}), nil
}

func (h *handler) RenameProject(ctx context.Context, req *connect.Request[projectsv1.RenameProjectRequest]) (*connect.Response[projectsv1.RenameProjectResponse], error) {
	p, err := h.deps.Service.Rename(ctx, domain.RenameInput{ID: req.Msg.Id, Name: req.Msg.Name, ExpectedRevision: req.Msg.ExpectedRevision})
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.RenameProjectResponse{Project: toProto(p)}), nil
}

func (h *handler) ListSourceMedia(ctx context.Context, req *connect.Request[projectsv1.ListSourceMediaRequest]) (*connect.Response[projectsv1.ListSourceMediaResponse], error) {
	if h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source media is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source media storage is unavailable"))
	}
	rows, next, err := h.deps.MediaService.List(ctx, req.Msg.ProjectId, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	out := &projectsv1.ListSourceMediaResponse{SourceMedia: make([]*projectsv1.SourceMedia, 0, len(rows)), NextPageToken: next}
	for _, row := range rows {
		media, err := h.sourceMediaWithInspection(ctx, row, nil)
		if err != nil {
			return nil, inspectionConnectError(err)
		}
		out.SourceMedia = append(out.SourceMedia, media)
	}
	return connect.NewResponse(out), nil
}

func (h *handler) GetSourceMedia(ctx context.Context, req *connect.Request[projectsv1.GetSourceMediaRequest]) (*connect.Response[projectsv1.GetSourceMediaResponse], error) {
	if h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source media is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source media storage is unavailable"))
	}
	row, err := h.deps.MediaService.Get(ctx, req.Msg.ProjectId, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	media, err := h.sourceMediaWithInspection(ctx, row, nil)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	return connect.NewResponse(&projectsv1.GetSourceMediaResponse{SourceMedia: media}), nil
}

type clockPageCursor struct {
	ClockSHA256 string `json:"clock_sha256"`
	Offset      uint32 `json:"offset"`
}

// GetSourceMediaClockPage exposes bounded, digest-bound PTS pages. Clock bytes
// and storage keys remain private to the Projects owner.
func (h *handler) GetSourceMediaClockPage(ctx context.Context, req *connect.Request[projectsv1.GetSourceMediaClockPageRequest]) (*connect.Response[projectsv1.GetSourceMediaClockPageResponse], error) {
	if h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source media is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source media storage is unavailable"))
	}
	row, err := h.deps.MediaService.Get(ctx, req.Msg.ProjectId, req.Msg.MediaId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	r := row.InspectionReceipt
	if row.Kind != domain.MediaKindVideo || row.Readiness != domain.ReadinessReady || r == nil || r.ClockSHA256 != req.Msg.ExpectedClockSha256 || r.ClockSHA256 == "" || row.InspectionJobID == "" || row.InspectionAttemptID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("source clock is not available for this inspected media"))
	}
	offset := uint32(0)
	if req.Msg.PageToken != "" {
		if len(req.Msg.PageToken) > 512 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid source clock page token"))
		}
		data, decodeErr := base64.RawURLEncoding.DecodeString(req.Msg.PageToken)
		var cursor clockPageCursor
		if decodeErr != nil || json.Unmarshal(data, &cursor) != nil || cursor.ClockSHA256 != r.ClockSHA256 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid source clock page token"))
		}
		offset = cursor.Offset
	}
	limit := uint32(req.Msg.PageSize)
	if limit == 0 {
		limit = 128
	}
	if limit > 256 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source clock page size must not exceed 256"))
	}
	_, clock, err := h.deps.MediaService.OpenInspectionArtifacts(ctx, row.ProjectID, row.ID, row.InspectionJobID, row.InspectionAttemptID, r.SHA256, r.ClockSHA256)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	if len(clock) < 12 || string(clock[:8]) != "DSCLK001" {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("stored source clock failed validation"))
	}
	count := binary.BigEndian.Uint32(clock[8:12])
	if count == 0 || count > 36000 || len(clock) != 12+int(count)*12 || offset > count {
		return nil, connect.NewError(connect.CodeDataLoss, errors.New("stored source clock failed validation"))
	}
	end := offset + limit
	if end > count {
		end = count
	}
	points := make([]*projectsv1.SourceClockPoint, 0, end-offset)
	for i := offset; i < end; i++ {
		pos := 12 + int(i)*12
		ordinal := binary.BigEndian.Uint32(clock[pos : pos+4])
		pts := int64(binary.BigEndian.Uint64(clock[pos+4 : pos+12]))
		if ordinal != i || (i > 0 && pts <= int64(binary.BigEndian.Uint64(clock[pos-8:pos]))) {
			return nil, connect.NewError(connect.CodeDataLoss, errors.New("stored source clock failed validation"))
		}
		points = append(points, &projectsv1.SourceClockPoint{Ordinal: ordinal, Pts: pts})
	}
	if err := requireProjectRoute(ctx, h.deps.DB, h.deps.Roots); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source media storage is unavailable"))
	}
	out := &projectsv1.GetSourceMediaClockPageResponse{ClockSha256: r.ClockSHA256, TimeBaseNumerator: r.TimeBaseNumerator, TimeBaseDenominator: r.TimeBaseDenominator, Points: points}
	if end < count {
		data, _ := json.Marshal(clockPageCursor{ClockSHA256: r.ClockSHA256, Offset: end})
		out.NextPageToken = base64.RawURLEncoding.EncodeToString(data)
	}
	return connect.NewResponse(out), nil
}

func requireProjectRoute(ctx context.Context, db *database.RoutedDB, roots *filerouting.RoutedRoots) error {
	if !database.IsTestMode(ctx) {
		return nil
	}
	if db == nil || roots == nil {
		return errors.New("test source route is not configured")
	}
	leaseID := db.ActiveLeaseID()
	if leaseID == "" {
		return errors.New("test source database lease is unavailable")
	}
	pinned, err := db.PinTestPool(ctx, leaseID)
	if err != nil {
		return err
	}
	_, err = roots.PinTestRoots(pinned, leaseID)
	return err
}

func (h *handler) EnsureSourceMediaInspection(ctx context.Context, req *connect.Request[projectsv1.EnsureSourceMediaInspectionRequest]) (*connect.Response[projectsv1.EnsureSourceMediaInspectionResponse], error) {
	if h.deps.Inspection == nil || h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source inspection is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source inspection storage is unavailable"))
	}
	job, _, err := h.deps.Inspection.EnsureSourceInspection(ctx, req.Msg.ProjectId, req.Msg.MediaId)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	if err := h.activateInspectionRoute(ctx, job); err != nil {
		return nil, err
	}
	row, err := h.deps.MediaService.Get(ctx, req.Msg.ProjectId, req.Msg.MediaId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	media, err := h.sourceMediaWithInspection(ctx, row, &job)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	return connect.NewResponse(&projectsv1.EnsureSourceMediaInspectionResponse{SourceMedia: media, Job: inspectionJobToProto(job)}), nil
}

func (h *handler) RetrySourceMediaInspection(ctx context.Context, req *connect.Request[projectsv1.RetrySourceMediaInspectionRequest]) (*connect.Response[projectsv1.RetrySourceMediaInspectionResponse], error) {
	if h.deps.Inspection == nil || h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source inspection is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source inspection storage is unavailable"))
	}
	job, err := h.deps.Inspection.RetrySourceInspection(ctx, req.Msg.ProjectId, req.Msg.MediaId, req.Msg.ExpectedJobVersion)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	if err := h.activateInspectionRoute(ctx, job); err != nil {
		return nil, err
	}
	row, err := h.deps.MediaService.Get(ctx, req.Msg.ProjectId, req.Msg.MediaId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	media, err := h.sourceMediaWithInspection(ctx, row, &job)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	return connect.NewResponse(&projectsv1.RetrySourceMediaInspectionResponse{SourceMedia: media, Job: inspectionJobToProto(job)}), nil
}

func (h *handler) CancelSourceMediaInspection(ctx context.Context, req *connect.Request[projectsv1.CancelSourceMediaInspectionRequest]) (*connect.Response[projectsv1.CancelSourceMediaInspectionResponse], error) {
	if h.deps.Inspection == nil || h.deps.MediaService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("source inspection is unavailable"))
	}
	ctx, err := pinProjectRoute(ctx, h.deps.DB, h.deps.Roots)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("routed source inspection storage is unavailable"))
	}
	job, err := h.deps.Inspection.CancelSourceInspection(ctx, req.Msg.ProjectId, req.Msg.MediaId, req.Msg.ExpectedJobVersion)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	if err := h.activateInspectionRoute(ctx, job); err != nil {
		return nil, err
	}
	row, err := h.deps.MediaService.Get(ctx, req.Msg.ProjectId, req.Msg.MediaId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	media, err := h.sourceMediaWithInspection(ctx, row, &job)
	if err != nil {
		return nil, inspectionConnectError(err)
	}
	return connect.NewResponse(&projectsv1.CancelSourceMediaInspectionResponse{SourceMedia: media, Job: inspectionJobToProto(job)}), nil
}

func (h *handler) sourceMediaWithInspection(ctx context.Context, row domain.SourceMedia, known *jobs.Job) (*projectsv1.SourceMedia, error) {
	media := sourceMediaToProto(row)
	if h.deps.Inspection == nil || row.Kind != domain.MediaKindVideo || row.InspectionJobID == "" {
		return media, nil
	}
	job := known
	if job == nil {
		resolved, err := h.deps.Inspection.GetSourceInspectionStatus(ctx, row.ProjectID, row.ID)
		if err != nil {
			return nil, err
		}
		job = &resolved
	}
	media.InspectionJob = inspectionJobToProto(*job)
	return media, nil
}

func (h *handler) activateInspectionRoute(ctx context.Context, job jobs.Job) error {
	if h.deps.Activator == nil || job.ID == "" {
		return nil
	}
	if err := h.deps.Activator.ActivateTestRoute(ctx); err != nil {
		h.deps.Logger.Printf("inspection route activation failed after accepted job %s: %v", job.ID, err)
		return connect.NewError(connect.CodeUnavailable, errors.New("inspection job was accepted but routed progress could not be activated; retry with the same media id"))
	}
	return nil
}

func inspectionJobToProto(job jobs.Job) *projectsv1.InspectionJobStatus {
	status := &projectsv1.InspectionJobStatus{Id: job.ID, State: string(job.State), Version: job.Version}
	if n := len(job.Attempts); n > 0 {
		status.CurrentAttemptId = job.Attempts[n-1].ID
	}
	return status
}

func inspectionConnectError(err error) error {
	switch {
	case errors.Is(err, jobs.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("source media inspection target was not found"))
	case errors.Is(err, jobs.ErrConcurrentTransition), errors.Is(err, jobs.ErrInvalidTransition):
		return connect.NewError(connect.CodeAborted, errors.New("inspection changed; reload its current state and retry"))
	case errors.Is(err, jobs.ErrIdempotencyConflict), errors.Is(err, domain.ErrInspectionConflict):
		return connect.NewError(connect.CodeAlreadyExists, errors.New("source media is bound to a different inspection state"))
	case errors.Is(err, jobs.ErrInvalid), errors.Is(err, renderjobsDomain.ErrNotEligible):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("source media is not eligible for inspection"))
	default:
		return connect.NewError(connect.CodeInternal, errors.New("source media inspection failed"))
	}
}

func pinProjectRoute(ctx context.Context, db *database.RoutedDB, roots *filerouting.RoutedRoots) (context.Context, error) {
	if !database.IsTestMode(ctx) {
		return ctx, nil
	}
	if db == nil || roots == nil {
		return nil, errors.New("test source route is not configured")
	}
	leaseID := db.ActiveLeaseID()
	if leaseID == "" {
		return nil, errors.New("test source database lease is unavailable")
	}
	pinned, err := db.PinTestPool(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	return roots.PinTestRoots(pinned, leaseID)
}
