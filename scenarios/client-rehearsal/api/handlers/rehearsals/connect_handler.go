package rehearsals

import (
	"context"
	"errors"
	"log"

	internal "client-rehearsal/internal/rehearsals"
	"connectrpc.com/connect"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectHandler struct {
	service internal.Service
	logger  *log.Logger
}

func NewConnectHandler(service internal.Service, logger *log.Logger) *ConnectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &ConnectHandler{service: service, logger: logger}
}

func (h *ConnectHandler) CreateAndRunRehearsal(ctx context.Context, req *connect.Request[v1.CreateAndRunRehearsalRequest]) (*connect.Response[v1.CreateAndRunRehearsalResponse], error) {
	item, err := h.service.CreateAndRun(ctx, internal.CreateInput{Title: req.Msg.Title, WorkflowJSON: req.Msg.WorkflowJson, TriggerNodeID: req.Msg.TriggerNodeId, SimulatorProfile: req.Msg.SimulatorProfile, SimulatorSourceJSON: req.Msg.SimulatorSourceJson, InjectSeededFault: req.Msg.InjectSeededFault, RequiredOutputPaths: req.Msg.RequiredOutputPaths, RepairOfID: req.Msg.RepairOfId})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.CreateAndRunRehearsalResponse{Rehearsal: toProto(item)}), nil
}

func (h *ConnectHandler) ListRehearsals(ctx context.Context, req *connect.Request[v1.ListRehearsalsRequest]) (*connect.Response[v1.ListRehearsalsResponse], error) {
	items, err := h.service.List(ctx, int(req.Msg.Limit))
	if err != nil {
		h.logger.Printf("rehearsals.ListRehearsals: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not list rehearsals"))
	}
	out := &v1.ListRehearsalsResponse{Rehearsals: make([]*v1.Rehearsal, 0, len(items))}
	for _, item := range items {
		out.Rehearsals = append(out.Rehearsals, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *ConnectHandler) GetRehearsal(ctx context.Context, req *connect.Request[v1.GetRehearsalRequest]) (*connect.Response[v1.GetRehearsalResponse], error) {
	item, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.GetRehearsalResponse{Rehearsal: toProto(item)}), nil
}

func (h *ConnectHandler) RecordReview(ctx context.Context, req *connect.Request[v1.RecordReviewRequest]) (*connect.Response[v1.RecordReviewResponse], error) {
	decision := internal.Decision("")
	switch req.Msg.Decision {
	case v1.ReviewDecision_REVIEW_DECISION_APPROVE:
		decision = internal.DecisionApprove
	case v1.ReviewDecision_REVIEW_DECISION_REQUEST_CHANGES:
		decision = internal.DecisionRequestChanges
	}
	item, err := h.service.Review(ctx, internal.ReviewInput{ID: req.Msg.Id, Decision: decision, Note: req.Msg.Note})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.RecordReviewResponse{Rehearsal: toProto(item)}), nil
}

func toConnectError(err error) error {
	var invalid internal.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var missing internal.ErrNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var conflict internal.ErrConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("rehearsal operation failed"))
}

func toProto(item internal.Rehearsal) *v1.Rehearsal {
	state := v1.RehearsalState_REHEARSAL_STATE_UNSPECIFIED
	switch item.State {
	case internal.StateNeedsReview:
		state = v1.RehearsalState_REHEARSAL_STATE_NEEDS_REVIEW
	case internal.StateApproved:
		state = v1.RehearsalState_REHEARSAL_STATE_APPROVED
	case internal.StateChangesRequested:
		state = v1.RehearsalState_REHEARSAL_STATE_CHANGES_REQUESTED
	case internal.StateFailed:
		state = v1.RehearsalState_REHEARSAL_STATE_FAILED
	}
	decision := v1.ReviewDecision_REVIEW_DECISION_UNSPECIFIED
	switch item.ReviewDecision {
	case internal.DecisionApprove:
		decision = v1.ReviewDecision_REVIEW_DECISION_APPROVE
	case internal.DecisionRequestChanges:
		decision = v1.ReviewDecision_REVIEW_DECISION_REQUEST_CHANGES
	}
	out := &v1.Rehearsal{Id: item.ID, Title: item.Title, SourceFormat: item.SourceFormat, SourceSha256: item.SourceSHA256, CredentialReplacements: item.CredentialReplacements, SimulatorProfile: item.SimulatorProfile, SimulatorSourceSha256: item.SimulatorSourceSHA256, State: state, FaultSummary: item.FaultSummary, RepairSummary: item.RepairSummary, HandoffMarkdown: item.HandoffMarkdown, ReviewDecision: decision, ReviewNote: item.ReviewNote, CreatedAt: timestamppb.New(item.CreatedAt), Trace: make([]*v1.TraceEvent, 0, len(item.Trace))}
	if item.ReviewedAt != nil {
		out.ReviewedAt = timestamppb.New(*item.ReviewedAt)
	}
	for _, event := range item.Trace {
		out.Trace = append(out.Trace, &v1.TraceEvent{Sequence: event.Sequence, Operation: event.Operation, RequestSummary: event.RequestSummary, ResponseSummary: event.ResponseSummary, Status: event.Status, OccurredAt: timestamppb.New(event.OccurredAt)})
	}
	return out
}
