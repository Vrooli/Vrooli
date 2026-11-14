package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// decisionCaller identifies who raises a decision: the verified run of an
// identified agent caller, or an authenticated operator. A presented but
// unverified run token is refused, never treated as absence.
func (h *AgentManagerConnectHandler) decisionCaller(ctx context.Context, header http.Header) (string, error) {
	if token := strings.TrimSpace(header.Get(cliutil.HeaderAgentIdentityToken)); token != "" {
		if h.h == nil {
			return "", connect.NewError(connect.CodeUnavailable, errors.New("run identity verifier unavailable"))
		}
		verified, err := h.h.svc.VerifyIdentityToken(ctx, token)
		if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
			return "", connect.NewError(connect.CodeUnauthenticated, errors.New("run identity is invalid or unavailable"))
		}
		return verified.Claims.RunID.String(), nil
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	if _, err := h.effortActor(ctx, effortToken(header, authority), authority); err != nil {
		return "", err
	}
	return "", nil
}

func decisionConnectError(err error) error {
	switch {
	case errors.Is(err, supervision.ErrDecisionChanged), errors.Is(err, supervision.ErrDecisionAnswered):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, supervision.ErrDecisionNotAnswered):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return watchConnectError(err)
}

func (h *AgentManagerConnectHandler) RaiseEffortDecision(ctx context.Context, req *connect.Request[pb.RaiseEffortDecisionRequest]) (*connect.Response[pb.RaiseEffortDecisionResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	raisedBy, err := h.decisionCaller(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	out, err := s.RaiseDecision(ctx, req.Msg, raisedBy)
	if err != nil {
		return nil, decisionConnectError(err)
	}
	return connect.NewResponse(out), nil
}

// ListEffortDecisions is an unauthenticated read, like the board.
func (h *AgentManagerConnectHandler) ListEffortDecisions(ctx context.Context, req *connect.Request[pb.ListEffortDecisionsRequest]) (*connect.Response[pb.ListEffortDecisionsResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.ListDecisions(ctx, req.Msg)
	if err != nil {
		return nil, decisionConnectError(err)
	}
	return connect.NewResponse(out), nil
}

// AnswerEffortDecision is the operator's desktop answer.
func (h *AgentManagerConnectHandler) AnswerEffortDecision(ctx context.Context, req *connect.Request[pb.AnswerEffortDecisionRequest]) (*connect.Response[pb.EffortDecision], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	if strings.TrimSpace(req.Header().Get(cliutil.HeaderAgentIdentityToken)) != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("an agent run cannot answer an operator decision"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.AnswerDecision(ctx, req.Msg, actor)
	if err != nil {
		return nil, decisionConnectError(err)
	}
	return connect.NewResponse(out), nil
}

// MarkEffortDecisionRecorded is called by `effort decision sync` after it
// wrote FEEDBACK.md. It only fences on the answer time, so any caller that
// can write the goal home may mark it.
func (h *AgentManagerConnectHandler) MarkEffortDecisionRecorded(ctx context.Context, req *connect.Request[pb.MarkEffortDecisionRecordedRequest]) (*connect.Response[pb.EffortDecision], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.MarkDecisionRecorded(ctx, req.Msg)
	if err != nil {
		return nil, decisionConnectError(err)
	}
	return connect.NewResponse(out), nil
}
