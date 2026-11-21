package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	sessionv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
	"google.golang.org/protobuf/types/known/structpb"
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

func (h *connectHandler) GetProfileSession(ctx context.Context, req *connect.Request[sessionv1.GetProfileSessionRequest]) (*connect.Response[sessionv1.GetProfileSessionResponse], error) {
	actor, err := verifiedDraftActor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	value, err := h.service.GetProfileSession(ctx, strings.TrimSpace(req.Msg.GetTarget()), actor)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewResponse(&sessionv1.GetProfileSessionResponse{Session: profileSessionToProto(value)}), nil
}

func (h *connectHandler) SaveProfileSession(ctx context.Context, req *connect.Request[sessionv1.SaveProfileSessionRequest]) (*connect.Response[sessionv1.GetProfileSessionResponse], error) {
	actor, err := verifiedDraftActor(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	wire := req.Msg.GetSession()
	if wire == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session is required"))
	}
	answers, err := rawAnswers(wire.GetAnswers())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("answers: %w", err))
	}
	targetContext, err := stringContext(wire.GetTargetContext())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target_context: %w", err))
	}
	value, err := h.service.SaveProfileSession(ctx, session.ProfileSession{
		Target: strings.TrimSpace(req.Msg.GetTarget()), Actor: actor,
		Mode: strings.TrimSpace(wire.GetMode()), ProfileID: strings.TrimSpace(wire.GetProfileId()),
		ProfileVersion: strings.TrimSpace(wire.GetProfileVersion()), CatalogRevision: strings.TrimSpace(wire.GetCatalogRevision()),
		ConsequenceDigest: strings.TrimSpace(wire.GetConsequenceDigest()),
		BaseRevision:      strings.TrimSpace(wire.GetBaseRevision()), Answers: answers,
		ManualDecisions: wire.GetManualDecisions(), TargetContext: targetContext,
	}, strings.TrimSpace(req.Msg.GetExpectedRevision()))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "conflict") {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&sessionv1.GetProfileSessionResponse{Session: profileSessionToProto(value)}), nil
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

func profileSessionToProto(value *session.ProfileSession) *sessionv1.ProfileSession {
	if value == nil {
		return nil
	}
	result := &sessionv1.ProfileSession{
		Target: value.Target, Actor: value.Actor, Mode: value.Mode, ProfileId: value.ProfileID,
		ProfileVersion: value.ProfileVersion, CatalogRevision: value.CatalogRevision,
		ConsequenceDigest: value.ConsequenceDigest, NextQuestionId: value.NextQuestionID, NextAction: value.NextAction,
		BaseRevision: value.BaseRevision, ManualDecisions: value.ManualDecisions,
		Revision:            value.Revision,
		ReconciliationState: value.ReconciliationState, CurrentProfileVersion: value.CurrentProfileVersion,
		ReconciliationReasons: value.ReconciliationReasons,
	}
	for _, change := range value.ReconciliationChanges {
		result.ReconciliationChanges = append(result.ReconciliationChanges, &sessionv1.ReconciliationChange{
			Kind: change.Kind, Field: change.Field, Before: change.Before, After: change.After,
			Impact: change.Impact, RequiresReview: change.RequiresReview,
		})
	}
	if value.Answers != nil {
		answers := make(map[string]any, len(value.Answers))
		for key, raw := range value.Answers {
			var decoded any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				continue
			}
			answers[key] = decoded
		}
		if encoded, err := structpb.NewStruct(answers); err == nil {
			result.Answers = encoded
		}
	}
	if context, err := structpb.NewStruct(stringMapAny(value.TargetContext)); err == nil {
		result.TargetContext = context
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value.UpdatedAt); err == nil {
		result.UpdatedAt = timestamppb.New(parsed)
	}
	return result
}

func rawAnswers(value *structpb.Struct) (map[string]json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	result := make(map[string]json.RawMessage, len(value.GetFields()))
	for key, item := range value.GetFields() {
		encoded, err := json.Marshal(item.AsInterface())
		if err != nil {
			return nil, err
		}
		result[key] = encoded
	}
	return result, nil
}

func stringContext(value *structpb.Struct) (map[string]string, error) {
	if value == nil {
		return nil, nil
	}
	result := make(map[string]string, len(value.GetFields()))
	for key, item := range value.GetFields() {
		text, ok := item.Kind.(*structpb.Value_StringValue)
		if !ok {
			return nil, fmt.Errorf("%q must be a string", key)
		}
		result[key] = text.StringValue
	}
	return result, nil
}

func stringMapAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return nil
	}
	result := make(map[string]any, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
