package library

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/library"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
)

type Service interface {
	GetRevision(context.Context, string, uint32) (domain.BundleRevision, error)
	ListRevisions(context.Context, string, uint32, int) ([]domain.BundleRevision, error)
}

type IntakeService interface {
	Get(context.Context, string, uint32) (domain.IntakeCandidate, error)
	List(context.Context, string, int) ([]domain.IntakeCandidate, error)
}

type Handler struct {
	service Service
	intake  IntakeService
}

func NewConnectHandler(service Service) *Handler { return &Handler{service: service} }
func NewConnectHandlerWithIntake(service Service, intake IntakeService) *Handler {
	return &Handler{service: service, intake: intake}
}

func (h *Handler) GetBundleRevision(ctx context.Context, req *connect.Request[v1.GetBundleRevisionRequest]) (*connect.Response[v1.GetBundleRevisionResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("library catalog is unavailable"))
	}
	revision, err := h.service.GetRevision(ctx, req.Msg.BundleId, req.Msg.Revision)
	if err != nil {
		return nil, mapError(err)
	}
	metadata := &v1.BundleRevisionMetadata{}
	if err := revisionapi.Encode(revision, metadata); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("library metadata projection failed"))
	}
	return connect.NewResponse(&v1.GetBundleRevisionResponse{Revision: metadata}), nil
}

func (h *Handler) ListBundleRevisions(ctx context.Context, req *connect.Request[v1.ListBundleRevisionsRequest]) (*connect.Response[v1.ListBundleRevisionsResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("library catalog is unavailable"))
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	if req.Msg.AfterBundleId == "" && req.Msg.AfterRevision != 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("after_revision requires after_bundle_id"))
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterBundleId); err != nil {
		return nil, err
	}
	rows, err := h.service.ListRevisions(ctx, req.Msg.AfterBundleId, req.Msg.AfterRevision, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("library catalog read failed"))
	}
	out := &v1.ListBundleRevisionsResponse{Revisions: make([]*v1.BundleRevisionMetadata, 0, len(rows))}
	for _, row := range rows {
		metadata := &v1.BundleRevisionMetadata{}
		if err := revisionapi.Encode(row, metadata); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("library metadata projection failed"))
		}
		out.Revisions = append(out.Revisions, metadata)
	}
	if len(rows) == limit && len(rows) > 0 {
		last := rows[len(rows)-1]
		out.NextAfterBundleId, out.NextAfterRevision = last.Manifest.BundleID, last.Manifest.Revision
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetIntakeCandidate(ctx context.Context, req *connect.Request[v1.GetIntakeCandidateRequest]) (*connect.Response[v1.GetIntakeCandidateResponse], error) {
	if h.intake == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("local geometry intake is unavailable"))
	}
	if err := revisionapi.ValidateCursorID(req.Msg.CandidateId); err != nil || req.Msg.CandidateId == "" || req.Msg.Revision == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("candidate_id and positive revision are required"))
	}
	candidate, err := h.intake.Get(ctx, req.Msg.CandidateId, req.Msg.Revision)
	if err != nil {
		return nil, mapIntakeError(err)
	}
	projected := &v1.IntakeCandidateRevision{}
	if err := revisionapi.Encode(candidate, projected); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("intake metadata projection failed"))
	}
	return connect.NewResponse(&v1.GetIntakeCandidateResponse{Candidate: projected}), nil
}

func (h *Handler) ListIntakeCandidates(ctx context.Context, req *connect.Request[v1.ListIntakeCandidatesRequest]) (*connect.Response[v1.ListIntakeCandidatesResponse], error) {
	if h.intake == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("local geometry intake is unavailable"))
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterCandidateId); err != nil {
		return nil, err
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.intake.List(ctx, req.Msg.AfterCandidateId, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("intake candidate list failed"))
	}
	out := &v1.ListIntakeCandidatesResponse{Candidates: make([]*v1.IntakeCandidateRevision, 0, len(rows))}
	for _, row := range rows {
		projected := &v1.IntakeCandidateRevision{}
		if err := revisionapi.Encode(row, projected); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("intake metadata projection failed"))
		}
		out.Candidates = append(out.Candidates, projected)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterCandidateId = rows[len(rows)-1].ID
	}
	return connect.NewResponse(out), nil
}

func parseIntakeRevision(raw string) (uint32, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("revision must be a positive 32-bit integer")
	}
	return uint32(value), nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrRevisionNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("library bundle revision not found"))
	default:
		return connect.NewError(connect.CodeInternal, errors.New("library catalog read failed"))
	}
}

func mapIntakeError(err error) error {
	switch {
	case errors.Is(err, domain.ErrIntakeNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("intake candidate revision not found"))
	case errors.Is(err, domain.ErrInvalidIntake):
		return connect.NewError(connect.CodeInvalidArgument, errors.New("intake request is invalid"))
	default:
		return connect.NewError(connect.CodeInternal, errors.New("intake candidate read failed"))
	}
}
