package motions

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/motions"
	motionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions"
)

type Service interface {
	SaveCandidate(context.Context, domain.MotionCandidate, uint32) (domain.Revision, error)
	GetRevision(context.Context, string, uint32) (domain.Revision, error)
	GetHead(context.Context, string) (domain.Revision, error)
	ListRevisions(context.Context, string, uint32, int) ([]domain.Revision, error)
	ListHeads(context.Context, string, int) ([]domain.Revision, error)
}

type Handler struct{ service Service }

func NewConnectHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) SaveMotionCandidate(ctx context.Context, req *connect.Request[motionsv1.SaveMotionCandidateRequest]) (*connect.Response[motionsv1.SaveMotionCandidateResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if req.Msg.Candidate == nil {
		return nil, revisionapi.InvalidCandidate()
	}
	candidate, err := revisionapi.Decode[domain.MotionCandidate](req.Msg.Candidate)
	if err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	revision, err := h.service.SaveCandidate(ctx, candidate, req.Msg.ExpectedHead)
	if err != nil {
		return nil, mapError(err)
	}
	out := &motionsv1.MotionRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&motionsv1.SaveMotionCandidateResponse{Revision: out}), nil
}

func (h *Handler) GetMotionRevision(ctx context.Context, req *connect.Request[motionsv1.GetMotionRevisionRequest]) (*connect.Response[motionsv1.GetMotionRevisionResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetRevision(ctx, req.Msg.MotionId, req.Msg.Revision)
	if err != nil {
		return nil, mapError(err)
	}
	out := &motionsv1.MotionRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&motionsv1.GetMotionRevisionResponse{Revision: out}), nil
}

func (h *Handler) GetMotionHead(ctx context.Context, req *connect.Request[motionsv1.GetMotionHeadRequest]) (*connect.Response[motionsv1.GetMotionHeadResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetHead(ctx, req.Msg.MotionId)
	if err != nil {
		return nil, mapError(err)
	}
	out := &motionsv1.MotionRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&motionsv1.GetMotionHeadResponse{Revision: out}), nil
}

func (h *Handler) ListMotionRevisions(ctx context.Context, req *connect.Request[motionsv1.ListMotionRevisionsRequest]) (*connect.Response[motionsv1.ListMotionRevisionsResponse], error) {
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
	rows, err := h.service.ListRevisions(ctx, req.Msg.MotionId, req.Msg.AfterRevision, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &motionsv1.ListMotionRevisionsResponse{Revisions: make([]*motionsv1.MotionRevision, 0, len(rows))}
	for _, row := range rows {
		item := &motionsv1.MotionRevision{}
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

func (h *Handler) ListMotionCandidates(ctx context.Context, req *connect.Request[motionsv1.ListMotionCandidatesRequest]) (*connect.Response[motionsv1.ListMotionCandidatesResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterMotionId); err != nil {
		return nil, err
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListHeads(ctx, req.Msg.AfterMotionId, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &motionsv1.ListMotionCandidatesResponse{Candidates: make([]*motionsv1.MotionRevision, 0, len(rows))}
	for _, row := range rows {
		item := &motionsv1.MotionRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Candidates = append(out.Candidates, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterMotionId = rows[len(rows)-1].MotionID
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
