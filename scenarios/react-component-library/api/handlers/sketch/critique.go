package sketch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"google.golang.org/protobuf/encoding/protojson"
	"react-component-library/internal/designcapture"
	"react-component-library/internal/designcritique"
)

func (h *connectHandler) GetCritiqueRubric(context.Context, *connect.Request[sketchv1.GetCritiqueRubricRequest]) (*connect.Response[sketchv1.CritiqueRubric], error) {
	return connect.NewResponse(&sketchv1.CritiqueRubric{RubricVersion: designcritique.RubricVersion, PolicyVersion: designcritique.PolicyVersion, Dimensions: designcritique.Dimensions(), Anchors: designcritique.Anchors(), Calibrated: false}), nil
}
func (h *connectHandler) RecordCritique(ctx context.Context, req *connect.Request[sketchv1.RecordCritiqueRequest]) (*connect.Response[sketchv1.CritiqueRecord], error) {
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
	resolver, ok := h.deps.CaptureDispatcher.(designcapture.ScreenshotResolver)
	if h.deps.CritiquesFor == nil || h.deps.CapturesFor == nil || !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique evidence services unavailable"))
	}
	repo, err := h.deps.CritiquesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	captures, err := h.deps.CapturesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	record, err := (designcritique.Service{Repository: repo, Verifier: designcritique.CaptureEvidenceVerifier{Captures: captures, Screenshots: resolver}}).Record(ctx, req.Msg.GetIdempotencyKey(), review)
	if errors.Is(err, designcritique.ErrConflict) {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return critiqueResponse(record)
}
func (h *connectHandler) GetCritique(ctx context.Context, req *connect.Request[sketchv1.GetCritiqueRequest]) (*connect.Response[sketchv1.CritiqueRecord], error) {
	if h.deps.CritiquesFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique repository unavailable"))
	}
	repo, err := h.deps.CritiquesFor(ctx)
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
func critiqueResponse(record designcritique.Record) (*connect.Response[sketchv1.CritiqueRecord], error) {
	raw, err := json.Marshal(record)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var output sketchv1.CritiqueRecord
	if err := protojson.Unmarshal(raw, &output); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&output), nil
}

func (h *connectHandler) ListCritiques(ctx context.Context, req *connect.Request[sketchv1.ListCritiquesRequest]) (*connect.Response[sketchv1.ListCritiquesResponse], error) {
	if h.deps.CritiquesFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("critique repository unavailable"))
	}
	target := req.Msg.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exact critique target is required"))
	}
	repo, err := h.deps.CritiquesFor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	records, next, err := repo.List(ctx, designcritique.Target{Scenario: target.GetScenario(), DesignID: target.GetDesignId(), Revision: target.GetRevision(), RenderHash: target.GetRenderHash()}, req.Msg.GetBeforeId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	output := &sketchv1.ListCritiquesResponse{NextBeforeId: next}
	for _, record := range records {
		converted, err := critiqueResponse(record)
		if err != nil {
			return nil, err
		}
		output.Reviews = append(output.Reviews, &sketchv1.CritiqueSummary{Id: record.ID, Hash: record.Hash, Critic: converted.Msg.Review.Critic, Assessment: converted.Msg.Assessment, RecordedAt: record.RecordedAt})
	}
	return connect.NewResponse(output), nil
}
