package shots

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	domain "device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Service domain.Service
	Presets *domain.PhysicalPresetService
	Logger  *log.Logger
}
type handler struct{ deps Deps }

func NewConnectHandler(deps Deps) *handler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &handler{deps: deps}
}

func (h *handler) CreateShot(ctx context.Context, req *connect.Request[shotsv1.CreateShotRequest]) (*connect.Response[shotsv1.CreateShotResponse], error) {
	shot, revision, err := h.deps.Service.Create(ctx, req.Msg.ProjectId, req.Msg.Name, req.Msg.Snapshot)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.CreateShotResponse{Shot: shotToProto(shot), Revision: revisionToProto(revision)}), nil
}

func (h *handler) SaveShot(ctx context.Context, req *connect.Request[shotsv1.SaveShotRequest]) (*connect.Response[shotsv1.SaveShotResponse], error) {
	shot, revision, err := h.deps.Service.Save(ctx, req.Msg.ShotId, req.Msg.ExpectedRevision, req.Msg.Name, req.Msg.Snapshot)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.SaveShotResponse{Shot: shotToProto(shot), Revision: revisionToProto(revision)}), nil
}

func (h *handler) ListShots(ctx context.Context, req *connect.Request[shotsv1.ListShotsRequest]) (*connect.Response[shotsv1.ListShotsResponse], error) {
	rows, next, err := h.deps.Service.List(ctx, req.Msg.ProjectId, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	out := &shotsv1.ListShotsResponse{Shots: make([]*shotsv1.Shot, 0, len(rows)), NextPageToken: next}
	for _, row := range rows {
		out.Shots = append(out.Shots, shotToProto(row))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) GetShot(ctx context.Context, req *connect.Request[shotsv1.GetShotRequest]) (*connect.Response[shotsv1.GetShotResponse], error) {
	shot, revision, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.GetShotResponse{Shot: shotToProto(shot), Head: revisionToProto(revision)}), nil
}

func (h *handler) GetShotRevision(ctx context.Context, req *connect.Request[shotsv1.GetShotRevisionRequest]) (*connect.Response[shotsv1.GetShotRevisionResponse], error) {
	revision, err := h.deps.Service.GetRevision(ctx, req.Msg.ShotId, req.Msg.Revision)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.GetShotRevisionResponse{Revision: revisionToProto(revision)}), nil
}

func (h *handler) ListShotRevisions(ctx context.Context, req *connect.Request[shotsv1.ListShotRevisionsRequest]) (*connect.Response[shotsv1.ListShotRevisionsResponse], error) {
	rows, next, err := h.deps.Service.ListRevisions(ctx, req.Msg.ShotId, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	out := &shotsv1.ListShotRevisionsResponse{Revisions: make([]*shotsv1.ShotRevision, 0, len(rows)), NextPageToken: next}
	for _, row := range rows {
		out.Revisions = append(out.Revisions, revisionToProto(row))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) CompileShot(ctx context.Context, req *connect.Request[shotsv1.CompileShotRequest]) (*connect.Response[shotsv1.CompileShotResponse], error) {
	revision, err := h.deps.Service.Compile(ctx, req.Msg.ShotId, req.Msg.Revision)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	shot, _, err := h.deps.Service.Get(ctx, req.Msg.ShotId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.CompileShotResponse{Compiled: compiledToProto(shot.ProjectID, revision)}), nil
}

func (h *handler) ValidateShot(ctx context.Context, req *connect.Request[shotsv1.ValidateShotRequest]) (*connect.Response[shotsv1.ValidateShotResponse], error) {
	revision, err := h.deps.Service.Validate(ctx, req.Msg.ProjectId, req.Msg.Snapshot)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.ValidateShotResponse{RenderEligible: revision.RenderEligible, Issues: revision.Issues, Compiled: compiledToProto(req.Msg.ProjectId, revision)}), nil
}

func (h *handler) ReviewOutputReadability(ctx context.Context, req *connect.Request[shotsv1.ReviewOutputReadabilityRequest]) (*connect.Response[shotsv1.ReviewOutputReadabilityResponse], error) {
	review, err := h.deps.Service.ReviewOutputReadability(ctx, domain.ReviewReadabilityRequest{
		ProjectID: req.Msg.ProjectId, ShotID: req.Msg.ShotId, ShotRevision: req.Msg.ShotRevision,
		RenderID: req.Msg.RenderId, OutputSHA256: req.Msg.OutputSha256, Verdict: int32(req.Msg.Verdict),
		Note: req.Msg.Note, ExplicitlyInspectedFinalSize: req.Msg.ExplicitlyInspectedFinalSize,
	})
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.ReviewOutputReadabilityResponse{Review: readabilityReviewToProto(review)}), nil
}

func (h *handler) GetOutputReadabilityReview(ctx context.Context, req *connect.Request[shotsv1.GetOutputReadabilityReviewRequest]) (*connect.Response[shotsv1.GetOutputReadabilityReviewResponse], error) {
	review, err := h.deps.Service.GetOutputReadabilityReview(ctx, req.Msg.ProjectId, req.Msg.ShotId, req.Msg.ShotRevision, req.Msg.RenderId, req.Msg.OutputSha256)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.GetOutputReadabilityReviewResponse{Review: readabilityReviewToProto(review)}), nil
}

func readabilityReviewToProto(review domain.ReadabilityReview) *shotsv1.OutputReadabilityReview {
	return &shotsv1.OutputReadabilityReview{Id: review.ID, ProjectId: review.ProjectID, ShotId: review.ShotID,
		ShotRevision: review.ShotRevision, RenderId: review.RenderID, OutputSha256: review.OutputSHA256,
		ProofSha256: review.ProofSHA256, Verdict: shotsv1.ReadabilityVerdict(review.Verdict), Note: review.Note,
		OutputWidthPixels: review.OutputWidth, OutputHeightPixels: review.OutputHeight,
		RegionWidthPixels: review.RegionWidth, RegionHeightPixels: review.RegionHeight, ReviewedAt: timestamppb.New(review.ReviewedAt)}
}

func (h *handler) SavePhysicalShotPreset(ctx context.Context, req *connect.Request[shotsv1.SavePhysicalShotPresetRequest]) (*connect.Response[shotsv1.SavePhysicalShotPresetResponse], error) {
	if h.deps.Presets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("physical setup presets are unavailable"))
	}
	revision, err := h.deps.Presets.Save(ctx, req.Msg.PresetId, req.Msg.ExpectedRevision, req.Msg.Name, req.Msg.Setup)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.SavePhysicalShotPresetResponse{Revision: physicalPresetToProto(revision)}), nil
}

func (h *handler) GetPhysicalShotPreset(ctx context.Context, req *connect.Request[shotsv1.GetPhysicalShotPresetRequest]) (*connect.Response[shotsv1.GetPhysicalShotPresetResponse], error) {
	if h.deps.Presets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("physical setup presets are unavailable"))
	}
	revision, err := h.deps.Presets.Get(ctx, req.Msg.PresetId, req.Msg.Revision)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.GetPhysicalShotPresetResponse{Revision: physicalPresetToProto(revision)}), nil
}

func (h *handler) GetPhysicalShotPresetHead(ctx context.Context, req *connect.Request[shotsv1.GetPhysicalShotPresetHeadRequest]) (*connect.Response[shotsv1.GetPhysicalShotPresetHeadResponse], error) {
	if h.deps.Presets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("physical setup presets are unavailable"))
	}
	revision, err := h.deps.Presets.Head(ctx, req.Msg.PresetId)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&shotsv1.GetPhysicalShotPresetHeadResponse{Revision: physicalPresetToProto(revision)}), nil
}

func (h *handler) ListPhysicalShotPresets(ctx context.Context, req *connect.Request[shotsv1.ListPhysicalShotPresetsRequest]) (*connect.Response[shotsv1.ListPhysicalShotPresetsResponse], error) {
	if h.deps.Presets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("physical setup presets are unavailable"))
	}
	limit := presetPageSize(req.Msg.PageSize)
	rows, err := h.deps.Presets.ListHeads(ctx, req.Msg.AfterPresetId, limit+1)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = rows[len(rows)-1].PresetID
	}
	out := &shotsv1.ListPhysicalShotPresetsResponse{Presets: make([]*shotsv1.PhysicalShotPresetRevision, 0, len(rows)), NextAfterPresetId: next}
	for _, row := range rows {
		out.Presets = append(out.Presets, physicalPresetToProto(row))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) ListPhysicalShotPresetRevisions(ctx context.Context, req *connect.Request[shotsv1.ListPhysicalShotPresetRevisionsRequest]) (*connect.Response[shotsv1.ListPhysicalShotPresetRevisionsResponse], error) {
	if h.deps.Presets == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("physical setup presets are unavailable"))
	}
	limit := presetPageSize(req.Msg.PageSize)
	rows, err := h.deps.Presets.ListRevisions(ctx, req.Msg.PresetId, req.Msg.AfterRevision, limit+1)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	next := uint32(0)
	if len(rows) > limit {
		rows = rows[:limit]
		next = rows[len(rows)-1].Revision
	}
	out := &shotsv1.ListPhysicalShotPresetRevisionsResponse{Revisions: make([]*shotsv1.PhysicalShotPresetRevision, 0, len(rows)), NextAfterRevision: next}
	for _, row := range rows {
		out.Revisions = append(out.Revisions, physicalPresetToProto(row))
	}
	return connect.NewResponse(out), nil
}

func presetPageSize(value uint32) int {
	if value == 0 {
		return 50
	}
	if value > 99 {
		return 99
	}
	return int(value)
}

func physicalPresetToProto(revision domain.PhysicalPresetRevision) *shotsv1.PhysicalShotPresetRevision {
	return &shotsv1.PhysicalShotPresetRevision{PresetId: revision.PresetID, Revision: revision.Revision, Sha256: revision.SHA256, Name: revision.Name, Setup: revision.Setup, State: "candidate", CandidateOnly: true, CreatedAt: timestamppb.New(revision.CreatedAt)}
}
