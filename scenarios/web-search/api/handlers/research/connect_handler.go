package research

import (
	"context"
	"database/sql"
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

	"web-search/internal/evaluation"
	"web-search/internal/evidence"
	internalresearch "web-search/internal/research"
	"web-search/internal/research/agentmanager"
)

// Deps wires the seams the Connect research handler needs. The service owns the
// fetcher / searcher / synthesizer / agent-manager / findings wiring; this
// handler only translates wire <-> domain.
type Deps struct {
	Service  *internalresearch.Service
	Registry *evaluation.MethodRegistry
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

func (h *connectHandler) registry() *evaluation.MethodRegistry {
	if h.deps.Registry == nil {
		h.deps.Registry = &evaluation.MethodRegistry{}
	}
	return h.deps.Registry
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
	policy := policyFromProto(req.Msg.GetPolicy())
	if policy.TopN == 0 {
		policy.TopN = int(req.Msg.GetTopN())
	}
	if policy.TopN > internalresearch.MaxTopN {
		policy.TopN = internalresearch.MaxTopN
	}
	policy.Query = req.Msg.GetQuery()
	policy.Effort = "l2"
	policy.Capture = req.Msg.GetCapture()
	policy.ParentRunID = req.Msg.GetParentRunId()
	questions := questionsFromProto(req.Msg.GetQuestions())
	if err := internalresearch.ValidateContractBounds(req.Msg.GetQuery(), questions, policy); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out, err := h.deps.Service.RunL2WithPolicyAndParent(ctx, policy, req.Msg.GetParentRunId())
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
		EvidenceReceiptIds: out.EvidenceReceiptIDs,
	}
	for _, assessment := range out.Assessments {
		wireAssessment := &researchv1.ClaimAssessment{ClaimId: assessment.ClaimID, AssessmentId: assessment.AssessmentID, Disposition: assessmentDispositionToProto(assessment.Disposition), PolicyRevision: assessment.PolicyRevision, Reason: assessment.Reason}
		for _, ref := range assessment.Evidence {
			wireAssessment.Evidence = append(wireAssessment.Evidence, &researchv1.EvidencePassageRef{ReceiptId: ref.ReceiptID, PassageId: ref.PassageID, ContentHash: ref.ContentHash, ExtractionRevision: ref.ExtractionRevision})
		}
		resp.Assessments = append(resp.Assessments, wireAssessment)
	}
	for _, coverage := range out.Coverage {
		resp.Coverage = append(resp.Coverage, &researchv1.QuestionCoverage{QuestionId: coverage.QuestionID, Status: coverage.Status, ClaimIds: coverage.ClaimIDs, UnresolvedReason: coverage.UnresolvedReason})
	}
	for _, failure := range out.FetchFailures {
		resp.FetchFailures = append(resp.FetchFailures, &researchv1.FetchFailure{Url: failure.URL, Code: failure.Code, Message: failure.Message, Retryable: failure.Retryable, ReceiptId: failure.ReceiptID})
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
	policy := policyFromProto(req.Msg.GetPolicy())
	questions := questionsFromProto(req.Msg.GetQuestions())
	if err := internalresearch.ValidateContractBounds(req.Msg.GetQuery(), questions, policy); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	result, err := h.deps.Service.StartResearchWithContract(ctx, req.Msg.GetQuery(), req.Msg.GetIdempotencyKey(), policy, questions)
	if err != nil {
		return nil, mapAgentError("research.RunL3", err, h.deps.Logger)
	}
	return connect.NewResponse(&researchv1.RunL3Response{
		RunId:  result.RunID,
		Status: result.Status,
	}), nil
}

func policyFromProto(p *researchv1.EvidencePolicy) internalresearch.EvidencePolicy {
	if p == nil {
		return internalresearch.EvidencePolicy{}
	}
	return internalresearch.EvidencePolicy{MaxAge: time.Duration(p.GetMaxAgeSeconds()) * time.Second, SourceDomains: p.GetSourceDomains(), MinimumSources: int(p.GetMinimumSources()), TopN: int(p.GetTopN()), MaxEvidenceBytes: int(p.GetMaxEvidenceBytes())}
}

func questionsFromProto(in []*researchv1.ResearchQuestion) []internalresearch.ResearchQuestion {
	out := make([]internalresearch.ResearchQuestion, 0, len(in))
	for _, q := range in {
		if q != nil {
			out = append(out, internalresearch.ResearchQuestion{ID: q.GetId(), Prompt: q.GetPrompt(), Required: q.GetRequired()})
		}
	}
	return out
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
	p := internalresearch.EvidencePolicy{Query: r.Query, Effort: r.Effort, MaxAge: time.Duration(r.MaxAgeSeconds) * time.Second, SourceDomains: r.SourceDomains, MinimumSources: int(r.MinimumSources), TopN: int(r.TopN), Capture: r.Capture, FindingID: r.FindingId, Questions: questionsFromProto(r.Questions), MaxEvidenceBytes: int(r.GetPolicy().GetMaxEvidenceBytes()), ParentRunID: r.GetParentRunId()}
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
	for _, assessment := range out.Assessments {
		result.Assessments = append(result.Assessments, &researchv1.ClaimAssessment{ClaimId: assessment.ClaimID, AssessmentId: assessment.AssessmentID, Disposition: assessmentDispositionToProto(assessment.Disposition), PolicyRevision: assessment.PolicyRevision, Reason: assessment.Reason})
		for _, ref := range assessment.Evidence {
			result.Assessments[len(result.Assessments)-1].Evidence = append(result.Assessments[len(result.Assessments)-1].Evidence, &researchv1.EvidencePassageRef{ReceiptId: ref.ReceiptID, PassageId: ref.PassageID, ContentHash: ref.ContentHash, ExtractionRevision: ref.ExtractionRevision})
		}
	}
	for _, c := range out.Coverage {
		result.Coverage = append(result.Coverage, &researchv1.QuestionCoverage{QuestionId: c.QuestionID, Status: c.Status, ClaimIds: c.ClaimIDs, UnresolvedReason: c.UnresolvedReason})
	}
	for _, r := range out.Results {
		result.Results = append(result.Results, &sharedv1.SearchResult{Url: r.URL, Title: r.Title, Snippet: r.Snippet, Engine: r.Engine, Score: r.Score, Category: r.Category})
	}
	return connect.NewResponse(result), nil
}

func assessmentDispositionToProto(d internalresearch.AssessmentDisposition) researchv1.AssessmentDisposition {
	switch d {
	case internalresearch.AssessmentSupported:
		return researchv1.AssessmentDisposition_ASSESSMENT_SUPPORTED
	case internalresearch.AssessmentContradicted:
		return researchv1.AssessmentDisposition_ASSESSMENT_CONTRADICTED
	case internalresearch.AssessmentUnresolved:
		return researchv1.AssessmentDisposition_ASSESSMENT_UNRESOLVED
	case internalresearch.AssessmentUnknown:
		return researchv1.AssessmentDisposition_ASSESSMENT_UNKNOWN
	default:
		return researchv1.AssessmentDisposition_ASSESSMENT_DISPOSITION_UNSPECIFIED
	}
}

func (h *connectHandler) GetCaptureStatus(ctx context.Context, _ *connect.Request[researchv1.GetCaptureStatusRequest]) (*connect.Response[researchv1.GetCaptureStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	counts, err := h.deps.Service.CaptureStatus(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	status := "healthy"
	if counts.Pending > 0 || counts.Failed > 0 {
		status = "pending"
	}
	if counts.Delivered+counts.Pending+counts.Failed == 0 {
		status = "unknown"
	}
	response := &researchv1.GetCaptureStatusResponse{Pending: int32(counts.Pending), Delivered: int32(counts.Delivered), Failed: int32(counts.Failed), Status: status}
	if !counts.OldestPending.IsZero() {
		response.OldestPendingAt = counts.OldestPending.Format(time.RFC3339Nano)
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetEvidenceReceipt(ctx context.Context, req *connect.Request[researchv1.GetEvidenceReceiptRequest]) (*connect.Response[researchv1.GetEvidenceReceiptResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	r, err := h.deps.Service.EvidenceReceipt(ctx, req.Msg.GetReceiptId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(&researchv1.GetEvidenceReceiptResponse{ReceiptId: r.ReceiptID, ObservationId: r.ObservationID, ProducerExecutionId: r.ProducerExecutionID, OriginalUrl: r.URL, FinalUrl: r.FinalURL, RedirectUrls: r.RedirectURLs, RetrievedAt: r.RetrievedAt.UTC().Format(time.RFC3339Nano), ContentSha256: r.ContentHash, ArtifactId: r.ArtifactID, ExtractionRevision: r.ExtractionRevision, Retention: r.Retention, FailureCode: r.FailureCode}), nil
}

func (h *connectHandler) GetEvidencePassage(ctx context.Context, req *connect.Request[researchv1.GetEvidencePassageRequest]) (*connect.Response[researchv1.GetEvidencePassageResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	p, err := h.deps.Service.EvidencePassage(ctx, req.Msg.GetPassageId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if p.StartByte < 0 || p.EndByte < p.StartByte || len(p.Content) > evidence.MaxPassageBytes {
		return nil, connect.NewError(connect.CodeInternal, errors.New("stored evidence passage violates bounds"))
	}
	return connect.NewResponse(&researchv1.GetEvidencePassageResponse{PassageId: p.PassageID, ReceiptId: p.ReceiptID, StartByte: int32(p.StartByte), EndByte: int32(p.EndByte), Content: p.Content, ContentSha256: p.Hash}), nil
}

func (h *connectHandler) GetEvidenceAssessment(ctx context.Context, req *connect.Request[researchv1.GetEvidenceAssessmentRequest]) (*connect.Response[researchv1.GetEvidenceAssessmentResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	a, err := h.deps.Service.EvidenceAssessment(ctx, req.Msg.GetAssessmentId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if len(a.EvidenceJSON) > 64<<10 {
		return nil, connect.NewError(connect.CodeInternal, errors.New("stored evidence assessment exceeds bounds"))
	}
	return connect.NewResponse(&researchv1.GetEvidenceAssessmentResponse{AssessmentId: a.AssessmentID, ClaimId: a.ClaimID, Disposition: a.Disposition, PolicyRevision: a.PolicyRevision, Reason: a.Reason, EvidenceJson: a.EvidenceJSON}), nil
}

func releaseToProto(r evaluation.Release) *researchv1.MethodRelease {
	return &researchv1.MethodRelease{Revision: &researchv1.MethodRevision{Id: r.Revision.ID, ProgramHash: r.Revision.ProgramHash, ConfigHash: r.Revision.ConfigHash}, EvaluationId: r.EvaluationID, EvaluationHash: r.EvaluationHash, RollbackOf: r.RollbackOf, RevisionHash: evaluation.RevisionHash(r.Revision)}
}

func (h *connectHandler) GetMethodRelease(context.Context, *connect.Request[researchv1.GetMethodReleaseRequest]) (*connect.Response[researchv1.GetMethodReleaseResponse], error) {
	release, ok := h.registry().Current()
	var projected *researchv1.MethodRelease
	if ok {
		projected = releaseToProto(release)
	}
	return connect.NewResponse(&researchv1.GetMethodReleaseResponse{Release: projected, Found: ok}), nil
}

func (h *connectHandler) PromoteMethod(ctx context.Context, req *connect.Request[researchv1.PromoteMethodRequest]) (*connect.Response[researchv1.PromoteMethodResponse], error) {
	r, receipt, report := req.Msg.GetRevision(), req.Msg.GetReceipt(), req.Msg.GetReport()
	if r == nil || receipt == nil || report == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("revision, receipt, and report are required"))
	}
	err := h.registry().Promote(req.Msg.GetExpectedCurrentHash(), evaluation.MethodRevision{ID: r.GetId(), ProgramHash: r.GetProgramHash(), ConfigHash: r.GetConfigHash()}, evaluation.EvaluationReceipt{ID: receipt.GetId(), CandidateHash: receipt.GetCandidateHash(), ReportHash: receipt.GetReportHash(), Accepted: receipt.GetAccepted()}, evaluation.Report{Accepted: report.GetAccepted(), Reason: report.GetReason(), BaselineMethod: report.GetBaselineMethod(), CandidateMethod: report.GetCandidateMethod(), Population: int(report.GetPopulation()), MeanBaselineEffort: report.GetMeanBaselineEffort(), MeanCandidateEffort: report.GetMeanCandidateEffort()}, req.Msg.GetGrant())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	release, ok := h.registry().Current()
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("promotion produced no current release"))
	}
	return connect.NewResponse(&researchv1.PromoteMethodResponse{Release: releaseToProto(release)}), nil
}

func (h *connectHandler) RollbackMethod(ctx context.Context, req *connect.Request[researchv1.RollbackMethodRequest]) (*connect.Response[researchv1.RollbackMethodResponse], error) {
	if err := h.registry().Rollback(req.Msg.GetExpectedCurrentHash(), req.Msg.GetTargetHash()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	release, ok := h.registry().Current()
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("rollback produced no current release"))
	}
	return connect.NewResponse(&researchv1.RollbackMethodResponse{Release: releaseToProto(release)}), nil
}

func (h *connectHandler) SuspendMethod(ctx context.Context, req *connect.Request[researchv1.SuspendMethodRequest]) (*connect.Response[researchv1.SuspendMethodResponse], error) {
	if err := h.registry().Suspend(req.Msg.GetRevisionHash(), req.Msg.GetReason()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&researchv1.SuspendMethodResponse{RevisionHash: req.Msg.GetRevisionHash()}), nil
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

func (h *connectHandler) CancelResearch(ctx context.Context, req *connect.Request[researchv1.CancelResearchRequest]) (*connect.Response[researchv1.GetResearchStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("research service not configured"))
	}
	if strings.TrimSpace(req.Msg.GetRunId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	state, err := h.deps.Service.CancelResearch(ctx, req.Msg.GetRunId(), req.Msg.GetReason(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, mapAgentError("research.CancelResearch", err, h.deps.Logger)
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
