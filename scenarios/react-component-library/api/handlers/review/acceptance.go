package review

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	sketchdomain "react-component-library/internal/sketch"
)

// CheckCandidateAcceptance reads fresh candidate, page, render and review facts.
// A successful check is preflight only; it never records acceptance.
func (h *Handler) CheckCandidateAcceptance(ctx context.Context, req *connect.Request[reviewv1.CheckCandidateAcceptanceRequest]) (*connect.Response[reviewv1.CandidateAcceptanceCheck], error) {
	if req == nil || req.Msg == nil || req.Msg.GetRender() == nil || req.Msg.GetRender().GetCandidate() == nil || req.Msg.GetExpectedRenderHash() == "" || len(req.Msg.GetCritiqueIds()) > 16 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("render, expected render hash, and at most sixteen critique IDs are required"))
	}
	if h.ports.AcceptanceEvaluator == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("candidate evaluation unavailable"))
	}
	result, err := h.ports.AcceptanceEvaluator.EvaluateAcceptance(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// AcceptCandidate stores the same prepared intent and invokes its evaluation
// callback inside the sketch store's idempotent transaction boundary.
func (h *Handler) AcceptCandidate(ctx context.Context, req *connect.Request[reviewv1.AcceptCandidateRequest]) (*connect.Response[reviewv1.AcceptanceDecision], error) {
	check := req.Msg.GetCheck()
	if check == nil || check.GetRender() == nil || check.GetRender().GetCandidate() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("acceptance check inputs are required"))
	}
	if h.ports.AcceptanceEvaluator == nil || h.ports.AcceptanceStoreFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("acceptance services unavailable"))
	}
	if err := h.ports.AcceptanceEvaluator.ValidateAcceptanceCandidate(ctx, check); err != nil {
		return nil, err
	}
	store, err := h.ports.AcceptanceStoreFor(ctx)
	if err != nil {
		return nil, acceptanceError(err)
	}
	render := check.GetRender()
	intent := sketchdomain.AcceptanceIntent{
		DesignID: render.GetCandidate().GetDesignId(), CandidateHash: render.GetCandidate().GetHash(),
		ExpectedRenderHash: check.GetExpectedRenderHash(), Actor: req.Msg.GetActor(), Kit: render.GetKit(),
		Theme: render.GetTheme(), Direction: render.GetDirection(), PreviewState: render.GetPreviewState(),
		MissingLabel: render.GetMissingLabel(), FailedLabel: render.GetFailedLabel(), CritiqueIDs: append([]string{}, check.GetCritiqueIds()...),
	}
	evaluated := false
	decision, err := store.RecordAcceptance(ctx, render.GetCandidate().GetScenario(), req.Msg.GetIdempotencyKey(), intent, func(ctx context.Context) (sketchdomain.AcceptanceFacts, error) {
		evaluated = true
		result, evalErr := h.ports.AcceptanceEvaluator.EvaluateAcceptance(ctx, check)
		if evalErr != nil {
			return sketchdomain.AcceptanceFacts{}, evalErr
		}
		facts := sketchdomain.AcceptanceFacts{PolicyVersion: result.GetPolicyVersion(), CandidateHash: render.GetCandidate().GetHash(), RenderHash: result.GetRenderHash(), InputHashes: result.GetInputHashes()}
		for _, row := range result.GetRequirements() {
			facts.Requirements = append(facts.Requirements, sketchdomain.AcceptanceRequirement{Code: row.GetCode(), Status: row.GetStatus(), Detail: row.GetDetail(), EvidenceIDs: row.GetEvidenceIds()})
		}
		return facts, nil
	})
	if err != nil {
		return nil, acceptanceError(err)
	}
	if evaluated && h.ports.AcceptanceWrite != nil {
		h.ports.AcceptanceWrite(ctx)
	}
	return acceptanceResponse(decision)
}

// GetAcceptance returns an immutable decision by its scenario/design identity.
func (h *Handler) GetAcceptance(ctx context.Context, req *connect.Request[reviewv1.GetAcceptanceRequest]) (*connect.Response[reviewv1.AcceptanceDecision], error) {
	if h.ports.AcceptanceStoreFor == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("acceptance persistence unavailable"))
	}
	store, err := h.ports.AcceptanceStoreFor(ctx)
	if err != nil {
		return nil, acceptanceError(err)
	}
	decision, err := store.ReadAcceptance(req.Msg.GetScenario(), req.Msg.GetDesignId(), req.Msg.GetId())
	if err != nil {
		return nil, acceptanceError(err)
	}
	return acceptanceResponse(decision)
}

func acceptanceResponse(decision sketchdomain.AcceptanceDecision) (*connect.Response[reviewv1.AcceptanceDecision], error) {
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var response reviewv1.AcceptanceDecision
	if err = protojson.Unmarshal(raw, &response); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&response), nil
}

func acceptanceError(err error) error {
	var coded *connect.Error
	if errors.As(err, &coded) {
		return coded
	}
	switch {
	case errors.Is(err, sketchdomain.ErrAcceptanceConflict):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, os.ErrNotExist):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
