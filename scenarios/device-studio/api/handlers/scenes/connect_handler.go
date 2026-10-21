package scenes

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/scenes"
	scenesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
)

type Service interface {
	SaveCandidate(context.Context, domain.SceneCandidate, uint32) (domain.Revision, error)
	GetRevision(context.Context, string, uint32) (domain.Revision, error)
	GetHead(context.Context, string) (domain.Revision, error)
	ListRevisions(context.Context, string, uint32, int) ([]domain.Revision, error)
	ListHeads(context.Context, string, int) ([]domain.Revision, error)
}

type Handler struct{ service Service }

func NewConnectHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) SaveSceneCandidate(ctx context.Context, req *connect.Request[scenesv1.SaveSceneCandidateRequest]) (*connect.Response[scenesv1.SaveSceneCandidateResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if req.Msg.Candidate == nil {
		return nil, revisionapi.InvalidCandidate()
	}
	candidate, err := revisionapi.Decode[domain.SceneCandidate](req.Msg.Candidate)
	if err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	revision, err := h.service.SaveCandidate(ctx, candidate, req.Msg.ExpectedHead)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.SaveSceneCandidateResponse{Revision: out}), nil
}

func (h *Handler) GetSceneRevision(ctx context.Context, req *connect.Request[scenesv1.GetSceneRevisionRequest]) (*connect.Response[scenesv1.GetSceneRevisionResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetRevision(ctx, req.Msg.SceneId, req.Msg.Revision)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.GetSceneRevisionResponse{Revision: out}), nil
}

func (h *Handler) GetSceneHead(ctx context.Context, req *connect.Request[scenesv1.GetSceneHeadRequest]) (*connect.Response[scenesv1.GetSceneHeadResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetHead(ctx, req.Msg.SceneId)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.SceneRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&scenesv1.GetSceneHeadResponse{Revision: out}), nil
}

func (h *Handler) ListSceneRevisions(ctx context.Context, req *connect.Request[scenesv1.ListSceneRevisionsRequest]) (*connect.Response[scenesv1.ListSceneRevisionsResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListRevisions(ctx, req.Msg.SceneId, req.Msg.AfterRevision, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.ListSceneRevisionsResponse{Revisions: make([]*scenesv1.SceneRevision, 0, len(rows))}
	for _, row := range rows {
		item := &scenesv1.SceneRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Revisions = append(out.Revisions, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterRevision = rows[len(rows)-1].Revision
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) ListSceneCandidates(ctx context.Context, req *connect.Request[scenesv1.ListSceneCandidatesRequest]) (*connect.Response[scenesv1.ListSceneCandidatesResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterSceneId); err != nil {
		return nil, err
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListHeads(ctx, req.Msg.AfterSceneId, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &scenesv1.ListSceneCandidatesResponse{Candidates: make([]*scenesv1.SceneRevision, 0, len(rows))}
	for _, row := range rows {
		item := &scenesv1.SceneRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Candidates = append(out.Candidates, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterSceneId = rows[len(rows)-1].SceneID
	}
	return connect.NewResponse(out), nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCandidate):
		return revisionapi.InvalidCandidate()
	case errors.Is(err, domain.ErrRevisionNotFound):
		return revisionapi.NotFound()
	case errors.Is(err, domain.ErrRevisionConflict):
		return revisionapi.Conflict()
	default:
		return revisionapi.Internal()
	}
}
