package codingjobs

import (
	"client-rehearsal/internal/codingjobs"
	"client-rehearsal/internal/logx"
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectHandler struct {
	service codingjobs.Service
	logger  logx.Logger
}

func NewConnectHandler(service codingjobs.Service, logger logx.Logger) *ConnectHandler {
	if logger == nil {
		panic("NewConnectHandler requires Logger")
	}
	return &ConnectHandler{service: service, logger: logger}
}

func (h *ConnectHandler) CreateCodingJob(ctx context.Context, req *connect.Request[v1.CreateCodingJobRequest]) (*connect.Response[v1.CreateCodingJobResponse], error) {
	item, err := h.service.Create(ctx, codingjobs.CreateInput{ReplayKey: req.Msg.ReplayKey, CaseID: req.Msg.CaseId, Project: req.Msg.Project, Title: req.Msg.Title, Task: req.Msg.Task, SourceSealRef: req.Msg.SourceSealRef})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.CreateCodingJobResponse{Job: toProto(item)}), nil
}

func (h *ConnectHandler) RecordCodingJobStage(ctx context.Context, req *connect.Request[v1.RecordCodingJobStageRequest]) (*connect.Response[v1.RecordCodingJobStageResponse], error) {
	item, err := h.service.RecordStage(ctx, codingjobs.StageInput{ID: req.Msg.Id, Stage: req.Msg.Stage, Actor: req.Msg.Actor, EvidenceRef: req.Msg.EvidenceRef, EvidenceSHA256: req.Msg.EvidenceSha256, Note: req.Msg.Note, ExpectedRevision: int64(req.Msg.ExpectedRevision)})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.RecordCodingJobStageResponse{Job: toProto(item)}), nil
}

func (h *ConnectHandler) RecordCodingJobEvaluation(ctx context.Context, req *connect.Request[v1.RecordCodingJobEvaluationRequest]) (*connect.Response[v1.RecordCodingJobEvaluationResponse], error) {
	item, err := h.service.RecordEvaluation(ctx, codingjobs.EvaluationInput{ID: req.Msg.Id, ReceiptRef: req.Msg.ReceiptRef, ReceiptSHA256: req.Msg.ReceiptSha256, Score: req.Msg.Score, ExpectedRevision: int64(req.Msg.ExpectedRevision)})
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.RecordCodingJobEvaluationResponse{Job: toProto(item)}), nil
}

func (h *ConnectHandler) GetCodingJob(ctx context.Context, req *connect.Request[v1.GetCodingJobRequest]) (*connect.Response[v1.GetCodingJobResponse], error) {
	item, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&v1.GetCodingJobResponse{Job: toProto(item)}), nil
}

func (h *ConnectHandler) ListCodingJobs(ctx context.Context, req *connect.Request[v1.ListCodingJobsRequest]) (*connect.Response[v1.ListCodingJobsResponse], error) {
	items, err := h.service.List(ctx, int(req.Msg.Limit))
	if err != nil {
		h.logger.Printf("codingjobs.ListCodingJobs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("could not list coding jobs"))
	}
	out := &v1.ListCodingJobsResponse{Jobs: make([]*v1.CodingJob, 0, len(items))}
	for _, item := range items {
		out.Jobs = append(out.Jobs, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func toConnectError(err error) error {
	var invalid codingjobs.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var missing codingjobs.ErrNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var conflict codingjobs.ErrConflict
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, errors.New("coding job operation failed"))
}

func toProto(item codingjobs.Job) *v1.CodingJob {
	state := v1.CodingJobState_CODING_JOB_STATE_UNSPECIFIED
	switch item.State {
	case codingjobs.StateReady:
		state = v1.CodingJobState_CODING_JOB_STATE_READY
	case codingjobs.StateSolving:
		state = v1.CodingJobState_CODING_JOB_STATE_SOLVING
	case codingjobs.StateScoring:
		state = v1.CodingJobState_CODING_JOB_STATE_SCORING
	case codingjobs.StateReviewable:
		state = v1.CodingJobState_CODING_JOB_STATE_REVIEWABLE
	case codingjobs.StateFailed:
		state = v1.CodingJobState_CODING_JOB_STATE_FAILED
	}
	out := &v1.CodingJob{Id: item.ID, ReplayKey: item.ReplayKey, CaseId: item.CaseID, Project: item.Project, Title: item.Title, Task: item.Task, State: state, SourceSealRef: item.SourceSealRef, SolverRunRef: item.SolverRunRef, EvaluatorReceiptRef: item.EvaluatorReceiptRef, EvaluatorReceiptSha256: item.EvaluatorReceiptSHA256, Score: item.Score, Revision: uint64(item.Revision), CreatedAt: timestamppb.New(item.CreatedAt), UpdatedAt: timestamppb.New(item.UpdatedAt)}
	for _, e := range item.Events {
		out.Events = append(out.Events, &v1.CodingJobEvent{Sequence: e.Sequence, Stage: e.Stage, Actor: e.Actor, EvidenceRef: e.EvidenceRef, EvidenceSha256: e.EvidenceSHA256, Note: e.Note, OccurredAt: timestamppb.New(e.OccurredAt)})
	}
	return out
}
