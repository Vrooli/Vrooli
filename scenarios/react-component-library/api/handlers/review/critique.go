package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	"react-component-library/internal/designcritique"
)

func (h *Handler) GetCritiqueRubric(context.Context, *connect.Request[reviewv1.GetCritiqueRubricRequest]) (*connect.Response[reviewv1.CritiqueRubric], error) {
	return connect.NewResponse(&reviewv1.CritiqueRubric{RubricVersion: designcritique.RubricVersion, PolicyVersion: designcritique.PolicyVersion, Dimensions: designcritique.Dimensions(), Anchors: designcritique.Anchors(), Calibrated: false}), nil
}

func (h *Handler) RecordCritique(ctx context.Context, req *connect.Request[reviewv1.RecordCritiqueRequest]) (*connect.Response[reviewv1.CritiqueRecord], error) {
	if req.Msg.GetReview() == nil || len(req.Msg.GetIdempotencyKey()) < 1 || len(req.Msg.GetIdempotencyKey()) > 200 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("review and bounded idempotency key are required"))
	}
	raw, err := protojson.Marshal(req.Msg.GetReview())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var review designcritique.Review
	if err = json.Unmarshal(raw, &review); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err = designcritique.Validate(review); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.ports.CritiquesFor == nil || h.ports.CapturesFor == nil || h.ports.Screenshots == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique evidence services unavailable"))
	}
	repo, err := h.ports.CritiquesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	captures, err := h.ports.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	record, err := (designcritique.Service{Repository: repo, Verifier: designcritique.CaptureEvidenceVerifier{Captures: captures, Screenshots: h.ports.Screenshots}}).Record(ctx, req.Msg.GetIdempotencyKey(), review)
	if errors.Is(err, designcritique.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return critiqueResponse(record)
}

func (h *Handler) GetCritique(ctx context.Context, req *connect.Request[reviewv1.GetCritiqueRequest]) (*connect.Response[reviewv1.CritiqueRecord], error) {
	if h.ports.CritiquesFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique repository unavailable"))
	}
	repo, err := h.ports.CritiquesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	record, err := repo.Get(ctx, req.Msg.GetId())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return critiqueResponse(record)
}

func critiqueResponse(record designcritique.Record) (*connect.Response[reviewv1.CritiqueRecord], error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var out reviewv1.CritiqueRecord
	if err := protojson.Unmarshal(raw, &out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&out), nil
}

func (h *Handler) ListCritiques(ctx context.Context, req *connect.Request[reviewv1.ListCritiquesRequest]) (*connect.Response[reviewv1.ListCritiquesResponse], error) {
	if h.ports.CritiquesFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique repository unavailable"))
	}
	target := req.Msg.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exact critique target is required"))
	}
	repo, err := h.ports.CritiquesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	records, next, err := repo.List(ctx, designcritique.Target{Scenario: target.GetScenario(), DesignID: target.GetDesignId(), Revision: target.GetRevision(), RenderHash: target.GetRenderHash()}, req.Msg.GetBeforeId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	out := &reviewv1.ListCritiquesResponse{NextBeforeId: next}
	for _, record := range records {
		converted, err := critiqueResponse(record)
		if err != nil {
			return nil, err
		}
		out.Reviews = append(out.Reviews, &reviewv1.CritiqueSummary{Id: record.ID, Hash: record.Hash, Critic: converted.Msg.GetReview().GetCritic(), Assessment: converted.Msg.GetAssessment(), RecordedAt: record.RecordedAt})
	}
	return connect.NewResponse(out), nil
}
