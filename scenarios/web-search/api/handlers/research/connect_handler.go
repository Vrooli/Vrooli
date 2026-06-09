package research

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"

	internalresearch "web-search/internal/research"
	"web-search/internal/research/agentmanager"
)

// Deps wires the seams the Connect research handler needs. The service owns the
// fetcher / searcher / synthesizer / agent-manager / findings wiring; this
// handler only translates wire <-> domain.
type Deps struct {
	Service *internalresearch.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect research handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// RunL2 runs the synchronous L2 fetch -> read -> single-pass cited synthesis
// pipeline and projects the outcome onto the wire response.
func (h *connectHandler) RunL2(ctx context.Context, req *connect.Request[researchv1.RunL2Request]) (*connect.Response[researchv1.RunL2Response], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	out, err := h.deps.Service.RunL2(ctx, req.Msg.GetQuery(), int(req.Msg.GetTopN()), req.Msg.GetCapture())
	if err != nil {
		h.deps.Logger.Printf("research.RunL2: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&researchv1.RunL2Response{
		Brief:              briefToProto(out.Brief),
		Synthesis:          out.Brief.Summary,
		Abstained:          out.Abstained,
		CapturedFindingIds: out.CapturedFindingIDs,
	}), nil
}

// RunL3 starts an agent-manager run for the iterative research-and-reconcile
// loop and returns the run handle. When agent-manager is not wired/available the
// error surfaces as Unavailable so callers can degrade to L2.
func (h *connectHandler) RunL3(ctx context.Context, req *connect.Request[researchv1.RunL3Request]) (*connect.Response[researchv1.RunL3Response], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	result, err := h.deps.Service.RunL3(ctx, req.Msg.GetQuery())
	if err != nil {
		return nil, mapAgentError("research.RunL3", err, h.deps.Logger)
	}
	return connect.NewResponse(&researchv1.RunL3Response{
		RunId:  result.RunID,
		Status: result.Status,
	}), nil
}

// GetResearchStatus polls an L3 run by id.
func (h *connectHandler) GetResearchStatus(ctx context.Context, req *connect.Request[researchv1.GetResearchStatusRequest]) (*connect.Response[researchv1.GetResearchStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	state, err := h.deps.Service.GetResearchStatus(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, mapAgentError("research.GetResearchStatus", err, h.deps.Logger)
	}
	return connect.NewResponse(&researchv1.GetResearchStatusResponse{
		RunId:      state.RunID,
		Status:     state.Status,
		Summary:    state.Summary,
		StartedAt:  rfc3339ToProto(state.StartedAt),
		FinishedAt: rfc3339ToProto(state.FinishedAt),
		ErrorMsg:   state.ErrorMsg,
	}), nil
}

// mapAgentError maps an agent-manager-path error to a Connect code: an
// unavailable upstream becomes Unavailable (callers retry / degrade), anything
// else Internal.
func mapAgentError(op string, err error, logger *log.Logger) error {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	logger.Printf("%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}
