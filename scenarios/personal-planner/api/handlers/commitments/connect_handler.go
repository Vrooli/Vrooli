package commitments

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/commitments"
	c "personal-planner/internal/commitments"
)

type (
	Deps struct {
		Service c.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListCommitments(ctx context.Context, _ *connect.Request[v.ListCommitmentsRequest]) (*connect.Response[v.ListCommitmentsResponse], error) {
	xs, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, c.ToConnectError(err)
	}
	out := &v.ListCommitmentsResponse{Commitments: make([]*v.Commitment, 0, len(xs))}
	for _, x := range xs {
		out.Commitments = append(out.Commitments, toProto(x))
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) CreateCommitment(ctx context.Context, r *connect.Request[v.CreateCommitmentRequest]) (*connect.Response[v.CreateCommitmentResponse], error) {
	x, err := h.deps.Service.Create(ctx, c.CreateInput{Result: r.Msg.Result, DefinitionOfDone: r.Msg.DefinitionOfDone, PromisedBoundary: r.Msg.PromisedBoundary, Timezone: r.Msg.Timezone, Beneficiary: r.Msg.Beneficiary, Assumptions: r.Msg.Assumptions, ScopeExclusions: r.Msg.ScopeExclusions, State: r.Msg.State})
	if err != nil {
		return nil, c.ToConnectError(err)
	}
	return connect.NewResponse(&v.CreateCommitmentResponse{Commitment: toProto(x)}), nil
}
func (h *connectHandler) UpdateCommitmentState(ctx context.Context, r *connect.Request[v.UpdateCommitmentStateRequest]) (*connect.Response[v.UpdateCommitmentStateResponse], error) {
	x, err := h.deps.Service.UpdateState(ctx, r.Msg.Id, r.Msg.State, r.Msg.ExpectedRevision)
	if err != nil {
		return nil, c.ToConnectError(err)
	}
	return connect.NewResponse(&v.UpdateCommitmentStateResponse{Commitment: toProto(x)}), nil
}
func (h *connectHandler) ReviseCommitment(ctx context.Context, r *connect.Request[v.ReviseCommitmentRequest]) (*connect.Response[v.ReviseCommitmentResponse], error) {
	x, revision, err := h.deps.Service.Revise(ctx, c.ReviseInput{ID: r.Msg.Id, PromisedBoundary: r.Msg.PromisedBoundary, Assumptions: r.Msg.Assumptions, ScopeExclusions: r.Msg.ScopeExclusions, Reason: r.Msg.Reason, AcknowledgmentStatus: r.Msg.AcknowledgmentStatus, ExpectedRevision: r.Msg.ExpectedRevision})
	if err != nil {
		return nil, c.ToConnectError(err)
	}
	return connect.NewResponse(&v.ReviseCommitmentResponse{Commitment: toProto(x), Revision: revisionToProto(revision)}), nil
}
