package triage

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	internal "signal-inbox/internal/triage"

	triagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/triage"
)

type connectHandler struct{ service *internal.Service }

func NewConnectHandler(service *internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) GetTriage(ctx context.Context, req *connect.Request[triagev1.GetTriageRequest]) (*connect.Response[triagev1.GetTriageResponse], error) {
	if req.Msg.SignalId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signal_id is required"))
	}
	disposition, annotations, err := h.service.Get(ctx, req.Msg.SignalId)
	if err != nil {
		return nil, toConnectError(err)
	}
	record := &triagev1.TriageRecord{Disposition: dispositionToProto(disposition), Annotations: make([]*triagev1.Annotation, 0, len(annotations))}
	for _, annotation := range annotations {
		record.Annotations = append(record.Annotations, annotationToProto(annotation))
	}
	return connect.NewResponse(&triagev1.GetTriageResponse{Triage: record}), nil
}

func (h *connectHandler) SetDisposition(ctx context.Context, req *connect.Request[triagev1.SetDispositionRequest]) (*connect.Response[triagev1.SetDispositionResponse], error) {
	if req.Msg.SignalId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signal_id is required"))
	}
	state, ok := stateFromProto(req.Msg.State)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid disposition state is required"))
	}
	var revisitAt *time.Time
	if req.Msg.RevisitAt != nil {
		if err := req.Msg.RevisitAt.CheckValid(); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		value := req.Msg.RevisitAt.AsTime().UTC()
		revisitAt = &value
	}
	disposition, err := h.service.Set(ctx, req.Msg.SignalId, state, revisitAt)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&triagev1.SetDispositionResponse{Disposition: dispositionToProto(disposition)}), nil
}

func (h *connectHandler) AddAnnotation(ctx context.Context, req *connect.Request[triagev1.AddAnnotationRequest]) (*connect.Response[triagev1.AddAnnotationResponse], error) {
	if req.Msg.SignalId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("signal_id is required"))
	}
	author, ok := authorFromProto(req.Msg.Author)
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("valid annotation author is required"))
	}
	outcome, err := outcomeFromProto(req.Msg.Outcome)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	annotation, err := h.service.Annotate(ctx, req.Msg.SignalId, author, req.Msg.Body, outcome)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&triagev1.AddAnnotationResponse{Annotation: annotationToProto(annotation)}), nil
}

func dispositionToProto(value internal.Disposition) *triagev1.Disposition {
	result := &triagev1.Disposition{SignalId: value.SignalID, State: stateToProto(value.State), UpdatedAt: timestamppb.New(value.UpdatedAt)}
	if value.RevisitAt != nil {
		result.RevisitAt = timestamppb.New(*value.RevisitAt)
	}
	return result
}

func annotationToProto(value internal.Annotation) *triagev1.Annotation {
	result := &triagev1.Annotation{Id: value.ID, SignalId: value.SignalID, Author: authorToProto(value.Author), Body: value.Body, CreatedAt: timestamppb.New(value.CreatedAt)}
	if value.Outcome != nil {
		result.Outcome = &triagev1.OutcomeLink{Kind: outcomeKindToProto(value.Outcome.Kind), TargetId: value.Outcome.TargetID}
	}
	return result
}

func stateToProto(value internal.State) triagev1.DispositionState {
	return map[internal.State]triagev1.DispositionState{internal.New: triagev1.DispositionState_DISPOSITION_STATE_NEW, internal.Triaged: triagev1.DispositionState_DISPOSITION_STATE_TRIAGED, internal.Routed: triagev1.DispositionState_DISPOSITION_STATE_ROUTED, internal.Done: triagev1.DispositionState_DISPOSITION_STATE_DONE, internal.Dropped: triagev1.DispositionState_DISPOSITION_STATE_DROPPED}[value]
}

func stateFromProto(value triagev1.DispositionState) (internal.State, bool) {
	state, ok := map[triagev1.DispositionState]internal.State{triagev1.DispositionState_DISPOSITION_STATE_NEW: internal.New, triagev1.DispositionState_DISPOSITION_STATE_TRIAGED: internal.Triaged, triagev1.DispositionState_DISPOSITION_STATE_ROUTED: internal.Routed, triagev1.DispositionState_DISPOSITION_STATE_DONE: internal.Done, triagev1.DispositionState_DISPOSITION_STATE_DROPPED: internal.Dropped}[value]
	return state, ok
}

func authorToProto(value internal.Author) triagev1.AnnotationAuthor {
	return map[internal.Author]triagev1.AnnotationAuthor{internal.Operator: triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_OPERATOR, internal.Agent: triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_AGENT, internal.System: triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_SYSTEM}[value]
}

func authorFromProto(value triagev1.AnnotationAuthor) (internal.Author, bool) {
	author, ok := map[triagev1.AnnotationAuthor]internal.Author{triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_OPERATOR: internal.Operator, triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_AGENT: internal.Agent, triagev1.AnnotationAuthor_ANNOTATION_AUTHOR_SYSTEM: internal.System}[value]
	return author, ok
}

func outcomeKindToProto(value internal.OutcomeKind) triagev1.OutcomeKind {
	return map[internal.OutcomeKind]triagev1.OutcomeKind{internal.OutcomeScenario: triagev1.OutcomeKind_OUTCOME_KIND_SCENARIO, internal.OutcomeBacklog: triagev1.OutcomeKind_OUTCOME_KIND_BACKLOG, internal.OutcomeIdeaPipeline: triagev1.OutcomeKind_OUTCOME_KIND_IDEA_PIPELINE, internal.OutcomeKnowledgeTopic: triagev1.OutcomeKind_OUTCOME_KIND_KNOWLEDGE_TOPIC}[value]
}

func outcomeFromProto(value *triagev1.OutcomeLink) (*internal.Outcome, error) {
	if value == nil {
		return nil, nil
	}
	kind, ok := map[triagev1.OutcomeKind]internal.OutcomeKind{triagev1.OutcomeKind_OUTCOME_KIND_SCENARIO: internal.OutcomeScenario, triagev1.OutcomeKind_OUTCOME_KIND_BACKLOG: internal.OutcomeBacklog, triagev1.OutcomeKind_OUTCOME_KIND_IDEA_PIPELINE: internal.OutcomeIdeaPipeline, triagev1.OutcomeKind_OUTCOME_KIND_KNOWLEDGE_TOPIC: internal.OutcomeKnowledgeTopic}[value.Kind]
	if !ok || value.TargetId == "" {
		return nil, errors.New("outcome kind and target_id are both required")
	}
	return &internal.Outcome{Kind: kind, TargetID: value.TargetId}, nil
}

func toConnectError(err error) error {
	var transition internal.ErrInvalidTransition
	var invalid internal.ErrInvalidTriage
	if errors.As(err, &transition) || errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
