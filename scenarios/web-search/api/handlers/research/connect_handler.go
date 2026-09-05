package research

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/shared"

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
	// Request schema (REQ-P1-001): query is required; top_n is clamped into
	// 1..10 by the service (non-positive → default, >10 → max).
	if strings.TrimSpace(req.Msg.GetQuery()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}
	out, err := h.deps.Service.RunL2(ctx, req.Msg.GetQuery(), int(req.Msg.GetTopN()), req.Msg.GetCapture())
	if err != nil {
		h.deps.Logger.Printf("research.RunL2: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &researchv1.RunL2Response{
		Brief:              briefToProto(out.Brief),
		Synthesis:          out.Brief.Summary,
		Abstained:          out.Abstained,
		AbstainReason:      string(out.AbstainReason),
		CapturedFindingIds: out.CapturedFindingIDs,
	}
	for _, e := range out.Excerpts {
		resp.Excerpts = append(resp.Excerpts, &researchv1.DocumentExcerpt{Url: e.URL, Title: e.Title, Excerpt: e.Excerpt})
	}
	for _, issue := range out.DegradedEngines {
		resp.DegradedEngines = append(resp.DegradedEngines, &sharedv1.EngineIssue{Engine: issue.Engine, Reason: issue.Reason})
	}
	return connect.NewResponse(resp), nil
}

// RunL3 starts an agent-manager run for the iterative research-and-reconcile
// loop and returns the run handle. When agent-manager is not wired/available the
// error surfaces as Unavailable so callers can degrade to L2.
func (h *connectHandler) RunL3(ctx context.Context, req *connect.Request[researchv1.RunL3Request]) (*connect.Response[researchv1.RunL3Response], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	result, err := h.deps.Service.StartResearch(ctx, req.Msg.GetQuery(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, mapAgentError("research.RunL3", err, h.deps.Logger)
	}
	return connect.NewResponse(&researchv1.RunL3Response{
		RunId:  result.RunID,
		Status: result.Status,
	}), nil
}

// GetResearchStatus reads a declared L3 execution by id.
func (h *connectHandler) GetResearchStatus(ctx context.Context, req *connect.Request[researchv1.GetResearchStatusRequest]) (*connect.Response[researchv1.GetResearchStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	state, err := h.deps.Service.GetResearchStatus(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, mapAgentError("research.GetResearchStatus", err, h.deps.Logger)
	}
	return connect.NewResponse(stateToProto(state)), nil
}

// GatherRelatedFindings runs the bounded GATHER step (OT-P1-003): it returns the
// findings semantically near the query, capped server-side. The applied cap is
// echoed so callers can see the sweep was bounded.
func (h *connectHandler) GatherRelatedFindings(ctx context.Context, req *connect.Request[researchv1.GatherRelatedFindingsRequest]) (*connect.Response[researchv1.GatherRelatedFindingsResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	gathered, capApplied, err := h.deps.Service.GatherRelatedFindings(ctx, req.Msg.GetQuery(), int(req.Msg.GetMax()))
	if err != nil {
		h.deps.Logger.Printf("research.GatherRelatedFindings: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*researchv1.GatheredFinding, 0, len(gathered))
	for _, g := range gathered {
		out = append(out, &researchv1.GatheredFinding{
			FindingId:  g.FindingID,
			Claim:      g.Claim,
			Confidence: g.Confidence,
			Status:     g.Status,
			Score:      g.Score,
		})
	}
	if capApplied < 0 || capApplied > math.MaxInt32 {
		return nil, connect.NewError(connect.CodeInternal, errors.New("invalid gather cap from research service"))
	}
	return connect.NewResponse(&researchv1.GatherRelatedFindingsResponse{
		Findings:   out,
		CapApplied: int32(capApplied),
	}), nil
}

// mapAgentError maps an agent-manager-path error to a Connect code: an
// unavailable upstream becomes Unavailable (callers retry / degrade), anything
// else Internal.
func mapAgentError(op string, err error, logger *log.Logger) error {
	var rpcError *connect.Error
	if errors.As(err, &rpcError) {
		return rpcError
	}
	if errors.Is(err, internalresearch.ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	logger.Printf("%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

// Answer translates the wire policy; research owns all evidence decisions.
func (h *connectHandler) Answer(ctx context.Context, req *connect.Request[researchv1.AnswerRequest]) (*connect.Response[researchv1.AnswerResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	r := req.Msg
	if r.MaxAgeSeconds < 0 || r.MaxAgeSeconds > 15552000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("max age exceeds 180 days"))
	}
	p := internalresearch.EvidencePolicy{Query: r.Query, Effort: r.Effort, MaxAge: time.Duration(r.MaxAgeSeconds) * time.Second, SourceDomains: r.SourceDomains, MinimumSources: int(r.MinimumSources), TopN: int(r.TopN), Capture: r.Capture, FindingID: r.FindingId}
	if err := p.Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out, err := h.deps.Service.Answer(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if out.LiveCalls < 0 || out.LiveCalls > math.MaxInt32 {
		return nil, connect.NewError(connect.CodeInternal, errors.New("invalid live call count from research service"))
	}
	result := &researchv1.AnswerResponse{Status: out.Status, AnswerKind: out.Kind, Brief: briefToProto(out.Brief), FindingIds: out.FindingIDs, Abstained: out.Abstained, Reason: out.Reason, CheckedAt: timestamppb.New(out.CheckedAt), LiveCalls: int32(out.LiveCalls), Cached: out.Cached, Gaps: out.Gaps, CapturedFindingIds: out.CapturedIDs}
	for _, r := range out.Results {
		result.Results = append(result.Results, &sharedv1.SearchResult{Url: r.URL, Title: r.Title, Snippet: r.Snippet, Engine: r.Engine, Score: r.Score, Category: r.Category})
	}
	return connect.NewResponse(result), nil
}

func (h *connectHandler) WaitResearch(ctx context.Context, req *connect.Request[researchv1.WaitResearchRequest]) (*connect.Response[researchv1.GetResearchStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	if req.Msg.TimeoutSeconds < 1 || req.Msg.TimeoutSeconds > 90 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timeout must be 1..90 seconds"))
	}
	state, err := h.deps.Service.WaitResearch(ctx, req.Msg.RunId, int(req.Msg.TimeoutSeconds))
	if err != nil {
		return nil, mapAgentError("research.WaitResearch", err, h.deps.Logger)
	}
	return connect.NewResponse(stateToProto(state)), nil
}

func stateToProto(state agentmanager.RunState) *researchv1.GetResearchStatusResponse {
	out := &researchv1.GetResearchStatusResponse{RunId: state.RunID, Status: state.Status, Summary: state.Summary, StartedAt: rfc3339ToProto(state.StartedAt), FinishedAt: rfc3339ToProto(state.FinishedAt), ErrorMsg: state.ErrorMsg, TimedOut: state.TimedOut}
	if state.Result != nil {
		out.Result, _ = structpb.NewStruct(state.Result)
	}
	return out
}
