package handoffs

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	handoffs_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/handoffs"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "persona/internal/handoffs"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) OpenHandoff(ctx context.Context, req *connect.Request[handoffs_v1.OpenHandoffRequest]) (*connect.Response[handoffs_v1.OpenHandoffResponse], error) {
	out, err := h.service.Open(ctx, domain.OpenInput{PersonaID: req.Msg.GetPersonaId(), Kind: req.Msg.GetKind(), Title: req.Msg.GetTitle(), HumanAction: req.Msg.GetHumanAction(), IdentityToken: req.Header().Get(cliutil.HeaderAgentIdentityToken), Checkpoint: fromCheckpoint(req.Msg.GetCheckpoint()), Deadline: timeFromProto(req.Msg.GetDeadline())})
	if err != nil {
		return nil, handoffError(err)
	}
	return connect.NewResponse(&handoffs_v1.OpenHandoffResponse{Handoff: toProto(out)}), nil
}

func (h *connectHandler) GetHandoff(ctx context.Context, req *connect.Request[handoffs_v1.GetHandoffRequest]) (*connect.Response[handoffs_v1.GetHandoffResponse], error) {
	out, err := h.service.Get(ctx, req.Msg.GetHandoffId())
	if err != nil {
		return nil, handoffError(err)
	}
	return connect.NewResponse(&handoffs_v1.GetHandoffResponse{Handoff: toProto(out)}), nil
}

func (h *connectHandler) ListHandoffs(ctx context.Context, req *connect.Request[handoffs_v1.ListHandoffsRequest]) (*connect.Response[handoffs_v1.ListHandoffsResponse], error) {
	items, err := h.service.List(ctx, req.Msg.GetPersonaId(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, handoffError(err)
	}
	out := &handoffs_v1.ListHandoffsResponse{Handoffs: make([]*handoffs_v1.Handoff, 0, len(items))}
	for _, item := range items {
		out.Handoffs = append(out.Handoffs, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CompleteHandoff(ctx context.Context, req *connect.Request[handoffs_v1.CompleteHandoffRequest]) (*connect.Response[handoffs_v1.CompleteHandoffResponse], error) {
	out, err := h.service.Complete(ctx, req.Msg.GetHandoffId(), req.Msg.GetCompletedBy())
	if err != nil {
		return nil, handoffError(err)
	}
	return connect.NewResponse(&handoffs_v1.CompleteHandoffResponse{Handoff: toProto(out)}), nil
}

func (h *connectHandler) CancelHandoff(ctx context.Context, req *connect.Request[handoffs_v1.CancelHandoffRequest]) (*connect.Response[handoffs_v1.CancelHandoffResponse], error) {
	out, err := h.service.Cancel(ctx, req.Msg.GetHandoffId(), req.Msg.GetCancelledBy())
	if err != nil {
		return nil, handoffError(err)
	}
	return connect.NewResponse(&handoffs_v1.CancelHandoffResponse{Handoff: toProto(out)}), nil
}

func (h *connectHandler) ResumeHandoff(ctx context.Context, req *connect.Request[handoffs_v1.ResumeHandoffRequest]) (*connect.Response[handoffs_v1.ResumeHandoffResponse], error) {
	out, err := h.service.Resume(ctx, req.Msg.GetHandoffId(), req.Msg.GetRunId())
	if err != nil {
		return nil, handoffError(err)
	}
	return connect.NewResponse(&handoffs_v1.ResumeHandoffResponse{Handoff: toProto(out)}), nil
}

func (h *connectHandler) PrepareEnrolment(ctx context.Context, req *connect.Request[handoffs_v1.PrepareEnrolmentRequest]) (*connect.Response[handoffs_v1.PrepareEnrolmentResponse], error) {
	fields, handoff, err := h.service.PrepareEnrolment(ctx, domain.EnrolmentInput{PersonaID: req.Msg.GetPersonaId(), Target: req.Msg.GetTarget(), IdentityToken: req.Header().Get(cliutil.HeaderAgentIdentityToken), RequiredFields: req.Msg.GetRequiredFields()})
	if err != nil {
		return nil, handoffError(err)
	}
	out := &handoffs_v1.PrepareEnrolmentResponse{HandoffId: handoff.ID, Fields: make([]*handoffs_v1.EnrolmentField, 0, len(fields))}
	for _, field := range fields {
		out.Fields = append(out.Fields, &handoffs_v1.EnrolmentField{Name: field.Name, Value: field.Value, HumanOnly: field.HumanOnly})
	}
	return connect.NewResponse(out), nil
}

func handoffError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, domain.ErrMissingPersona) || errors.Is(err, domain.ErrMissingHandoff) || errors.Is(err, domain.ErrInvalidHandoff) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, domain.ErrInvalidTransition) || errors.Is(err, domain.ErrExpired) {
		code = connect.CodeFailedPrecondition
	}
	if errors.Is(err, domain.ErrProposalDenied) {
		code = connect.CodePermissionDenied
	}
	return connect.NewError(code, err)
}

func timeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func fromCheckpoint(in *handoffs_v1.Checkpoint) domain.Checkpoint {
	if in == nil {
		return domain.Checkpoint{}
	}
	fields := make([]domain.Field, 0, len(in.GetCompletedFields()))
	for _, item := range in.GetCompletedFields() {
		fields = append(fields, domain.Field{Name: item.GetName(), Value: item.GetValue()})
	}
	return domain.Checkpoint{CompletedFields: fields, RequiredDocumentIDs: in.GetRequiredDocumentIds(), ResumeToken: in.GetResumeToken()}
}

func toProto(h domain.Handoff) *handoffs_v1.Handoff {
	fields := make([]*handoffs_v1.CheckpointField, 0, len(h.Checkpoint.CompletedFields))
	for _, field := range h.Checkpoint.CompletedFields {
		fields = append(fields, &handoffs_v1.CheckpointField{Name: field.Name, Value: field.Value})
	}
	return &handoffs_v1.Handoff{Id: h.ID, PersonaId: h.PersonaID, Kind: h.Kind, Title: h.Title, HumanAction: h.HumanAction, Checkpoint: &handoffs_v1.Checkpoint{CompletedFields: fields, RequiredDocumentIds: h.Checkpoint.RequiredDocumentIDs, ResumeToken: h.Checkpoint.ResumeToken}, State: toState(h.State), OpenedByRunId: h.OpenedByRunID, AuthorisingHuman: h.AuthorisingHuman, Deadline: timestamppb.New(h.Deadline), CreatedAt: timestamppb.New(h.CreatedAt), UpdatedAt: timestamppb.New(h.UpdatedAt), RelayState: h.RelayState}
}

func toState(state domain.State) handoffs_v1.HandoffState {
	switch state {
	case domain.StateOpen:
		return handoffs_v1.HandoffState_HANDOFF_STATE_OPEN
	case domain.StateDelivered:
		return handoffs_v1.HandoffState_HANDOFF_STATE_DELIVERED
	case domain.StateAwaitingHuman:
		return handoffs_v1.HandoffState_HANDOFF_STATE_AWAITING_HUMAN
	case domain.StateCompleted:
		return handoffs_v1.HandoffState_HANDOFF_STATE_COMPLETED
	case domain.StateExpired:
		return handoffs_v1.HandoffState_HANDOFF_STATE_EXPIRED
	case domain.StateCancelled:
		return handoffs_v1.HandoffState_HANDOFF_STATE_CANCELLED
	case domain.StateResumed:
		return handoffs_v1.HandoffState_HANDOFF_STATE_RESUMED
	default:
		return handoffs_v1.HandoffState_HANDOFF_STATE_UNSPECIFIED
	}
}
