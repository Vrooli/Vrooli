package shots

import (
	"context"
	"log"

	"connectrpc.com/connect"
	domain "device-studio/internal/shots"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

type Deps struct {
	Service domain.Service
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
