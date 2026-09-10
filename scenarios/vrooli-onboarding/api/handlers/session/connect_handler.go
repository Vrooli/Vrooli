package session

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	sessionv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type connectHandler struct{ service session.Service }

func NewConnectHandler(service session.Service) sessionv1connect.SessionServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) GetSession(ctx context.Context, _ *connect.Request[sessionv1.GetSessionRequest]) (*connect.Response[sessionv1.GetSessionResponse], error) {
	value, err := h.service.GetSession(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProto(value)), nil
}

func (h *connectHandler) AdvanceSessionStep(ctx context.Context, req *connect.Request[sessionv1.AdvanceSessionStepRequest]) (*connect.Response[sessionv1.GetSessionResponse], error) {
	stepID := strings.TrimSpace(req.Msg.GetStepId())
	if stepID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("step_id is required"))
	}
	value, err := h.service.AdvanceSessionStep(ctx, stepID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProto(value)), nil
}

func (h *connectHandler) GetStepModel(ctx context.Context, _ *connect.Request[sessionv1.GetStepModelRequest]) (*connect.Response[sessionv1.GetStepModelResponse], error) {
	model, err := h.service.GetStepModel(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	steps := make([]*sessionv1.Step, 0, len(model.Steps))
	for _, step := range model.Steps {
		steps = append(steps, &sessionv1.Step{Id: step.ID, Ordinal: step.Ordinal, Title: step.Title, Route: step.Route, Deferred: step.Deferred})
	}
	return connect.NewResponse(&sessionv1.GetStepModelResponse{Steps: steps}), nil
}

func (h *connectHandler) GetDraft(ctx context.Context, req *connect.Request[sessionv1.GetDraftRequest]) (*connect.Response[sessionv1.GetDraftResponse], error) {
	actor, err := verifiedDraftActor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	value, err := h.service.GetDraft(ctx, strings.TrimSpace(req.Msg.GetTarget()), actor)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sessionv1.GetDraftResponse{Draft: draftToProto(value)}), nil
}

func (h *connectHandler) SaveDraft(ctx context.Context, req *connect.Request[sessionv1.SaveDraftRequest]) (*connect.Response[sessionv1.GetDraftResponse], error) {
	actor, err := verifiedDraftActor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	value, err := h.service.SaveDraft(ctx, strings.TrimSpace(req.Msg.GetTarget()), actor, strings.TrimSpace(req.Msg.GetExpectedRevision()), strings.TrimSpace(req.Msg.GetBaseRevision()), strings.TrimSpace(req.Msg.GetStepId()), req.Msg.GetChoices())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "conflict") {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&sessionv1.GetDraftResponse{Draft: draftToProto(value)}), nil
}

func (h *connectHandler) DiscardDraft(ctx context.Context, req *connect.Request[sessionv1.DiscardDraftRequest]) (*connect.Response[sessionv1.GetDraftResponse], error) {
	actor, err := verifiedDraftActor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	value, err := h.service.DiscardDraft(ctx, strings.TrimSpace(req.Msg.GetTarget()), actor)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sessionv1.GetDraftResponse{Draft: draftToProto(value)}), nil
}

// verifiedDraftActor is the sole ownership binding for durable drafts. The
// request actor field remains wire-compatible for older clients, but it is
// intentionally ignored so a caller cannot read or mutate another operator's
// draft by substituting an identifier in JSON.
func verifiedDraftActor(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.IsVerified() || strings.TrimSpace(principal.Subject) == "" || strings.TrimSpace(string(principal.Source)) == "" {
		return "", errors.New("verified draft actor required")
	}
	return strings.TrimSpace(string(principal.Source)) + ":" + strings.TrimSpace(principal.Subject), nil
}

func toProto(value session.Response) *sessionv1.GetSessionResponse {
	return &sessionv1.GetSessionResponse{Step: value.Step, StepId: value.StepID, FirstUnsatisfiedStep: value.FirstUnsatisfiedStep, Completion: value.Completion}
}

func draftToProto(value session.Draft) *sessionv1.Draft {
	result := &sessionv1.Draft{Target: value.Target, Actor: value.Actor, BaseRevision: value.BaseRevision, Revision: value.Revision, StepId: value.StepID, Choices: value.Choices}
	if parsed, err := time.Parse(time.RFC3339Nano, value.UpdatedAt); err == nil {
		result.UpdatedAt = timestamppb.New(parsed)
	}
	return result
}
