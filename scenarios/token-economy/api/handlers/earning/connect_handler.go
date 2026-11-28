package earning

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/earning"

	earningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/earning"
)

type connectHandler struct {
	service domain.Service
	logger  *log.Logger
}

func NewConnectHandler(service domain.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

// SubmitEarning receives the adapter identity from the authenticated access
// edge. The wire payload cannot assert or replace that identity.
func (h *connectHandler) SubmitEarning(ctx context.Context, adapterIdentity string, req *connect.Request[earningv1.SubmitEarningRequest]) (*connect.Response[earningv1.SubmitEarningResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("earning service unavailable"))
	}
	submission, err := h.service.Submit(ctx, adapterIdentity, domain.Input{
		HolderID: req.Msg.HolderId, TokenTypeID: req.Msg.TokenTypeId,
		AmountMinor: req.Msg.AmountMinor, Reason: req.Msg.Reason, DedupKey: req.Msg.DedupKey,
	})
	if err != nil {
		return nil, h.mapError("SubmitEarning", err)
	}
	return connect.NewResponse(&earningv1.SubmitEarningResponse{Submission: submissionToProto(submission)}), nil
}

func (h *connectHandler) ListEarnings(ctx context.Context, _ *connect.Request[earningv1.ListEarningsRequest]) (*connect.Response[earningv1.ListEarningsResponse], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("earning service unavailable"))
	}
	values, err := h.service.List(ctx)
	if err != nil {
		return nil, h.mapError("ListEarnings", err)
	}
	out := &earningv1.ListEarningsResponse{Submissions: make([]*earningv1.EarningSubmission, 0, len(values))}
	for _, value := range values {
		out.Submissions = append(out.Submissions, submissionToProto(value))
	}
	return connect.NewResponse(out), nil
}

func submissionToProto(submission domain.Submission) *earningv1.EarningSubmission {
	return &earningv1.EarningSubmission{
		Id: submission.ID, HolderId: submission.HolderID, TokenTypeId: submission.TokenTypeID,
		AmountMinor: submission.AmountMinor, Reason: submission.Reason, DedupKey: submission.DedupKey,
		AdapterIdentity: submission.AdapterIdentity, ActorIdentity: submission.ActorIdentity,
		GrantId: submission.GrantID, SubmittedAt: timestamppb.New(submission.SubmittedAt),
		Replayed: submission.Replayed, PayloadSummary: submission.PayloadSummary,
	}
}

func (h *connectHandler) mapError(operation string, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidSubmission):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.logger.Printf("earning.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
