package devices

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	domain "device-studio/internal/devices"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices"
)

type Service interface {
	SaveCandidate(context.Context, domain.DeviceCandidate, uint32) (domain.DeviceRevision, error)
	GetRevision(context.Context, string, uint32) (domain.DeviceRevision, error)
	GetHead(context.Context, string) (domain.DeviceRevision, error)
	ListRevisions(context.Context, string, uint32, int) ([]domain.DeviceRevision, error)
	ListHeads(context.Context, string, int) ([]domain.DeviceRevision, error)
}

type QualificationReviewer interface {
	ReviewCandidate(context.Context, domain.QualificationReviewRequest) (domain.QualificationReviewResult, error)
}

type QualificationPackageOwner interface {
	ExportReviewPackage(context.Context, string, string, uint32) ([]byte, error)
	ImportReviewPackage(context.Context, []byte, uint32) (domain.QualificationReviewResult, error)
}

type Handler struct {
	service  Service
	reviewer QualificationReviewer
	packages QualificationPackageOwner
}

func NewConnectHandler(service Service, reviewers ...QualificationReviewer) *Handler {
	var reviewer QualificationReviewer
	if len(reviewers) > 0 {
		reviewer = reviewers[0]
	}
	packages, _ := reviewer.(QualificationPackageOwner)
	return &Handler{service: service, reviewer: reviewer, packages: packages}
}

func (h *Handler) ExportQualificationPackage(ctx context.Context, req *connect.Request[devicesv1.ExportQualificationPackageRequest]) (*connect.Response[devicesv1.ExportQualificationPackageResponse], error) {
	if h.packages == nil || req.Msg.ProjectId == "" || req.Msg.DeviceId == "" || req.Msg.QualifiedRevision == 0 {
		return nil, revisionapi.InvalidCandidate()
	}
	data, err := h.packages.ExportReviewPackage(ctx, req.Msg.ProjectId, req.Msg.DeviceId, req.Msg.QualifiedRevision)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&devicesv1.ExportQualificationPackageResponse{PackageJson: data}), nil
}

func (h *Handler) ImportQualificationPackage(ctx context.Context, req *connect.Request[devicesv1.ImportQualificationPackageRequest]) (*connect.Response[devicesv1.ReviewDeviceCandidateResponse], error) {
	if h.packages == nil || len(req.Msg.PackageJson) == 0 {
		return nil, revisionapi.InvalidCandidate()
	}
	result, err := h.packages.ImportReviewPackage(ctx, req.Msg.PackageJson, req.Msg.ExpectedHead)
	if err != nil {
		return nil, mapError(err)
	}
	revision := &devicesv1.DeviceRevision{}
	if err := revisionapi.Encode(result.Revision, revision); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&devicesv1.ReviewDeviceCandidateResponse{Revision: revision, PresetId: result.PresetID, PresetRevision: result.PresetRevision, PresetSha256: result.PresetSHA256}), nil
}

func (h *Handler) ReviewDeviceCandidate(ctx context.Context, req *connect.Request[devicesv1.ReviewDeviceCandidateRequest]) (*connect.Response[devicesv1.ReviewDeviceCandidateResponse], error) {
	if h.service == nil || h.reviewer == nil {
		return nil, revisionapi.Internal()
	}
	if req.Msg.Review == nil {
		return nil, revisionapi.InvalidCandidate()
	}
	review, err := revisionapi.Decode[domain.QualificationReview](req.Msg.Review)
	if err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	result, err := h.reviewer.ReviewCandidate(ctx, domain.QualificationReviewRequest{ProjectID: req.Msg.ProjectId, DeviceID: req.Msg.DeviceId, CandidateRevision: req.Msg.CandidateRevision, CandidateSHA256: req.Msg.CandidateSha256, ExpectedHead: req.Msg.ExpectedHead, Review: review})
	if err != nil {
		return nil, mapError(err)
	}
	revision := &devicesv1.DeviceRevision{}
	if err := revisionapi.Encode(result.Revision, revision); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&devicesv1.ReviewDeviceCandidateResponse{Revision: revision, PresetId: result.PresetID, PresetRevision: result.PresetRevision, PresetSha256: result.PresetSHA256}), nil
}

func (h *Handler) SaveDeviceCandidate(ctx context.Context, req *connect.Request[devicesv1.SaveDeviceCandidateRequest]) (*connect.Response[devicesv1.SaveDeviceCandidateResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if req.Msg.Candidate == nil {
		return nil, revisionapi.InvalidCandidate()
	}
	candidate, err := revisionapi.Decode[domain.DeviceCandidate](req.Msg.Candidate)
	if err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	if _, _, _, err := domain.CanonicalizeDeviceCandidate(candidate); err != nil {
		return nil, revisionapi.InvalidCandidate()
	}
	revision, err := h.service.SaveCandidate(ctx, candidate, req.Msg.ExpectedHead)
	if err != nil {
		return nil, mapError(err)
	}
	out := &devicesv1.DeviceRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&devicesv1.SaveDeviceCandidateResponse{Revision: out}), nil
}

func (h *Handler) GetDeviceRevision(ctx context.Context, req *connect.Request[devicesv1.GetDeviceRevisionRequest]) (*connect.Response[devicesv1.GetDeviceRevisionResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetRevision(ctx, req.Msg.DeviceId, req.Msg.Revision)
	if err != nil {
		return nil, mapError(err)
	}
	out := &devicesv1.DeviceRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&devicesv1.GetDeviceRevisionResponse{Revision: out}), nil
}

func (h *Handler) GetDeviceHead(ctx context.Context, req *connect.Request[devicesv1.GetDeviceHeadRequest]) (*connect.Response[devicesv1.GetDeviceHeadResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	revision, err := h.service.GetHead(ctx, req.Msg.DeviceId)
	if err != nil {
		return nil, mapError(err)
	}
	out := &devicesv1.DeviceRevision{}
	if err := revisionapi.Encode(revision, out); err != nil {
		return nil, revisionapi.Internal()
	}
	return connect.NewResponse(&devicesv1.GetDeviceHeadResponse{Revision: out}), nil
}

func (h *Handler) ListDeviceRevisions(ctx context.Context, req *connect.Request[devicesv1.ListDeviceRevisionsRequest]) (*connect.Response[devicesv1.ListDeviceRevisionsResponse], error) {
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
	rows, err := h.service.ListRevisions(ctx, req.Msg.DeviceId, req.Msg.AfterRevision, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &devicesv1.ListDeviceRevisionsResponse{Revisions: make([]*devicesv1.DeviceRevision, 0, len(rows))}
	for _, row := range rows {
		item := &devicesv1.DeviceRevision{}
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

func (h *Handler) ListDeviceCandidates(ctx context.Context, req *connect.Request[devicesv1.ListDeviceCandidatesRequest]) (*connect.Response[devicesv1.ListDeviceCandidatesResponse], error) {
	if h.service == nil {
		return nil, revisionapi.Internal()
	}
	if err := revisionapi.ValidateCursorID(req.Msg.AfterDeviceId); err != nil {
		return nil, err
	}
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 50
	}
	if limit > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page_size must not exceed 100"))
	}
	rows, err := h.service.ListHeads(ctx, req.Msg.AfterDeviceId, limit)
	if err != nil {
		return nil, mapError(err)
	}
	out := &devicesv1.ListDeviceCandidatesResponse{Candidates: make([]*devicesv1.DeviceRevision, 0, len(rows))}
	for _, row := range rows {
		item := &devicesv1.DeviceRevision{}
		if err := revisionapi.Encode(row, item); err != nil {
			return nil, revisionapi.Internal()
		}
		out.Candidates = append(out.Candidates, item)
	}
	if len(rows) == limit && len(rows) > 0 {
		out.NextAfterDeviceId = rows[len(rows)-1].DeviceID
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
