package planclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/stringsx"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"
	logconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log/log_v1connect"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

const scenarioName = "plan-manager"

// Resolver resolves the current plan-manager base URL. Scenario URLs can be
// short-lived, so the Connect client calls it for every operation.
type Resolver func(context.Context) (string, error)

// Client is swarm-manager's shared seam for plan-manager Connect calls.
type Client interface {
	ListPlans(ctx context.Context) ([]*sharedv1.Plan, error)
	GetPlan(ctx context.Context, id string) (*sharedv1.Plan, error)
	RenderMarkdown(ctx context.Context, id string, compact bool) (RenderMarkdownResult, error)
	ImportPlan(ctx context.Context, input ImportPlanInput) (*sharedv1.Plan, error)
	Start(ctx context.Context, req *executionv1.StartRequest) (*executionv1.StartResponse, error)
	Resume(ctx context.Context, req *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error)
	GetNext(ctx context.Context, req *executionv1.GetNextRequest) (*executionv1.GetNextResponse, error)
	GetStatus(ctx context.Context, req *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error)
	TransitionPhase(ctx context.Context, req *executionv1.TransitionPhaseRequest) (*executionv1.TransitionPhaseResponse, error)
	Complete(ctx context.Context, req *executionv1.CompleteRequest) (*executionv1.CompleteResponse, error)
	GetHandoff(ctx context.Context, req *executionv1.GetHandoffRequest) (*executionv1.GetHandoffResponse, error)
	AddDecision(ctx context.Context, req *logv1.AddDecisionRequest) (*logv1.AddEntryResponse, error)
	AddFinding(ctx context.Context, req *logv1.AddFindingRequest) (*logv1.AddEntryResponse, error)
	AddBug(ctx context.Context, req *logv1.AddBugRequest) (*logv1.AddEntryResponse, error)
	AddRecord(ctx context.Context, req *logv1.AddRecordRequest) (*logv1.AddEntryResponse, error)
	AddNote(ctx context.Context, req *logv1.AddNoteRequest) (*logv1.AddEntryResponse, error)
}

// PlanReader is the read subset used by planimport.
type PlanReader interface {
	ListPlans(ctx context.Context) ([]*sharedv1.Plan, error)
	GetPlan(ctx context.Context, id string) (*sharedv1.Plan, error)
}

// PlanAuditReader is the narrow, run-scoped audit seam consumed by Swarm
// Manager's evidence reconciler. Plan Manager retains fact authority.
type PlanAuditReader interface {
	ListAuditFacts(ctx context.Context, runID string) ([]PlanAuditFact, error)
}

type PlanAuditFact struct {
	EventID       string
	RunID         string
	TaskID        string
	Action        string
	PlanID        string
	ContentDigest string
	OccurredAt    time.Time
}

// PlanImporter is implemented by clients that can canonicalize external
// markdown through plan-manager before swarm-manager binds work to it.
type PlanImporter interface {
	ImportPlan(ctx context.Context, input ImportPlanInput) (*sharedv1.Plan, error)
}

// CandidateClient is the narrow Plan Manager authority used by workflows that
// propose a change to an existing canonical plan. It intentionally has no
// direct-update operation: Swarm may create and inspect a candidate, while an
// explicitly authorized operator applies it through the Plan Workshop flow.
type CandidateClient interface {
	CreateCandidateRevision(ctx context.Context, input CandidateRevisionInput) (*plansv1.CandidateRevision, error)
	PreviewCandidateRevision(ctx context.Context, candidateID string) (*plansv1.CandidateRevisionPreview, error)
	ApplyCandidateRevision(ctx context.Context, candidateID, expectedBaseContentHash string, acknowledgeQualityImpact bool) (*plansv1.ApplyCandidateRevisionResponse, error)
}

// CandidateDiscarder is kept separate so existing candidate canonicalizers do
// not gain an operator-only mutation requirement.
type CandidateDiscarder interface {
	DiscardCandidateRevision(ctx context.Context, candidateID, reason string) (*plansv1.CandidateRevision, error)
}

type CandidateRevisionInput struct {
	PlanID                  string
	ExpectedBaseContentHash string
	ProposalProvenance      string
	CandidatePlan           *sharedv1.Plan
}

// MarkdownRenderer is the render subset used by execution prompt resolution.
type MarkdownRenderer interface {
	RenderMarkdown(ctx context.Context, id string, compact bool) (RenderMarkdownResult, error)
}

type ImportPlanInput struct {
	SourcePath string
	Markdown   string
	Title      string
	Slug       string
	// Supersede preserves history when a workflow returns a repaired candidate.
	// Plan Manager creates and validates the new canonical plan; Swarm later
	// decides whether its domain reference may move to that new revision.
	Supersede string
}

type RenderMarkdownResult struct {
	Markdown        string
	Mirror          *sharedv1.RenderedPlanMirror
	Repaired        bool
	Plan            *sharedv1.Plan
	QualityStatus   string
	QualityFindings []string
}

// ConnectClient implements Client over plan-manager's Connect services.
type ConnectClient struct {
	http    *http.Client
	resolve Resolver
}

// NewConnectClient builds a plan-manager client. A nil httpClient gets a
// 30-second timeout, and a nil resolver discovers plan-manager per call.
func NewConnectClient(httpClient *http.Client, resolver Resolver) *ConnectClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if resolver == nil {
		resolver = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, scenarioName)
		}
	}
	return &ConnectClient{http: httpClient, resolve: resolver}
}

func (c *ConnectClient) GetPlan(ctx context.Context, id string) (*sharedv1.Plan, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetPlan(ctx, connect.NewRequest(&plansv1.GetPlanRequest{Id: id}))
	if err != nil {
		return nil, opError("GetPlan", id, err)
	}
	return resp.Msg.GetPlan(), nil
}

func (c *ConnectClient) ListPlans(ctx context.Context) ([]*sharedv1.Plan, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListPlans(ctx, connect.NewRequest(&plansv1.ListPlansRequest{}))
	if err != nil {
		return nil, opError("ListPlans", "", err)
	}
	return resp.Msg.GetPlans(), nil
}

func (c *ConnectClient) ListAuditFacts(ctx context.Context, runID string) ([]PlanAuditFact, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ListAuditFacts(ctx, connect.NewRequest(&plansv1.ListAuditFactsRequest{RunId: runID}))
	if err != nil {
		return nil, opError("ListAuditFacts", runID, err)
	}
	facts := make([]PlanAuditFact, 0, len(response.Msg.GetFacts()))
	for _, fact := range response.Msg.GetFacts() {
		occurredAt, err := time.Parse(time.RFC3339Nano, fact.GetOccurredAt())
		if err != nil {
			return nil, fmt.Errorf("ListAuditFacts %q: parse occurred_at: %w", runID, err)
		}
		facts = append(facts, PlanAuditFact{
			EventID:       fact.GetEventId(),
			RunID:         fact.GetRunId(),
			TaskID:        fact.GetTaskId(),
			Action:        fact.GetAction(),
			PlanID:        fact.GetPlanId(),
			ContentDigest: fact.GetContentDigest(),
			OccurredAt:    occurredAt.UTC(),
		})
	}
	return facts, nil
}

func (c *ConnectClient) RenderMarkdown(ctx context.Context, id string, compact bool) (RenderMarkdownResult, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return RenderMarkdownResult{}, err
	}
	resp, err := client.RenderMarkdown(ctx, connect.NewRequest(&plansv1.RenderMarkdownRequest{Id: id, Compact: compact}))
	if err != nil {
		return RenderMarkdownResult{}, opError("RenderMarkdown", id, err)
	}
	msg := resp.Msg
	return RenderMarkdownResult{
		Markdown:        msg.GetMarkdown(),
		Mirror:          msg.GetMirror(),
		Repaired:        msg.GetRepaired(),
		Plan:            msg.GetPlan(),
		QualityStatus:   msg.GetQualityStatus(),
		QualityFindings: append([]string(nil), msg.GetQualityFindings()...),
	}, nil
}

func (c *ConnectClient) ImportPlan(ctx context.Context, input ImportPlanInput) (*sharedv1.Plan, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ImportPlan(ctx, connect.NewRequest(&plansv1.ImportPlanRequest{
		SourcePath: input.SourcePath,
		Markdown:   input.Markdown,
		Title:      input.Title,
		Slug:       input.Slug,
		Supersede:  input.Supersede,
	}))
	if err != nil {
		return nil, opError("ImportPlan", stringsx.FirstNonEmpty(input.Slug, input.SourcePath, input.Title), err)
	}
	return resp.Msg.GetPlan(), nil
}

func (c *ConnectClient) CreateCandidateRevision(ctx context.Context, input CandidateRevisionInput) (*plansv1.CandidateRevision, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.CreateCandidateRevision(ctx, connect.NewRequest(&plansv1.CreateCandidateRevisionRequest{
		PlanId:                  input.PlanID,
		ExpectedBaseContentHash: input.ExpectedBaseContentHash,
		ProposalProvenance:      input.ProposalProvenance,
		CandidatePlan:           input.CandidatePlan,
	}))
	if err != nil {
		return nil, opError("CreateCandidateRevision", input.PlanID, err)
	}
	return response.Msg.GetCandidate(), nil
}

func (c *ConnectClient) PreviewCandidateRevision(ctx context.Context, candidateID string) (*plansv1.CandidateRevisionPreview, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.PreviewCandidateRevision(ctx, connect.NewRequest(&plansv1.PreviewCandidateRevisionRequest{Id: candidateID}))
	if err != nil {
		return nil, opError("PreviewCandidateRevision", candidateID, err)
	}
	return response.Msg.GetPreview(), nil
}

func (c *ConnectClient) ApplyCandidateRevision(ctx context.Context, candidateID, expectedBaseContentHash string, acknowledgeQualityImpact bool) (*plansv1.ApplyCandidateRevisionResponse, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ApplyCandidateRevision(ctx, connect.NewRequest(&plansv1.ApplyCandidateRevisionRequest{
		Id: candidateID, ExpectedBaseContentHash: expectedBaseContentHash, AcknowledgeQualityImpact: acknowledgeQualityImpact,
	}))
	if err != nil {
		return nil, opError("ApplyCandidateRevision", candidateID, err)
	}
	return response.Msg, nil
}

func (c *ConnectClient) DiscardCandidateRevision(ctx context.Context, candidateID, reason string) (*plansv1.CandidateRevision, error) {
	client, err := c.plans(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.DiscardCandidateRevision(ctx, connect.NewRequest(&plansv1.DiscardCandidateRevisionRequest{Id: candidateID, Reason: reason}))
	if err != nil {
		return nil, opError("DiscardCandidateRevision", candidateID, err)
	}
	return response.Msg.GetCandidate(), nil
}

func (c *ConnectClient) Start(ctx context.Context, req *executionv1.StartRequest) (*executionv1.StartResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.Start(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("Start", req.GetPlanId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) Resume(ctx context.Context, req *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.Resume(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("Resume", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) GetNext(ctx context.Context, req *executionv1.GetNextRequest) (*executionv1.GetNextResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetNext(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("GetNext", req.GetExecutionId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) GetStatus(ctx context.Context, req *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetStatus(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("GetStatus", req.GetExecutionId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) TransitionPhase(ctx context.Context, req *executionv1.TransitionPhaseRequest) (*executionv1.TransitionPhaseResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.TransitionPhase(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("TransitionPhase", req.GetExecutionId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) Complete(ctx context.Context, req *executionv1.CompleteRequest) (*executionv1.CompleteResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.Complete(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("Complete", req.GetExecutionId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) GetHandoff(ctx context.Context, req *executionv1.GetHandoffRequest) (*executionv1.GetHandoffResponse, error) {
	client, err := c.execution(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetHandoff(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("GetHandoff", req.GetExecutionId(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) AddDecision(ctx context.Context, req *logv1.AddDecisionRequest) (*logv1.AddEntryResponse, error) {
	client, err := c.log(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddDecision(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("AddDecision", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) AddFinding(ctx context.Context, req *logv1.AddFindingRequest) (*logv1.AddEntryResponse, error) {
	client, err := c.log(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddFinding(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("AddFinding", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) AddBug(ctx context.Context, req *logv1.AddBugRequest) (*logv1.AddEntryResponse, error) {
	client, err := c.log(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddBug(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("AddBug", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) AddRecord(ctx context.Context, req *logv1.AddRecordRequest) (*logv1.AddEntryResponse, error) {
	client, err := c.log(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddRecord(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("AddRecord", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) AddNote(ctx context.Context, req *logv1.AddNoteRequest) (*logv1.AddEntryResponse, error) {
	client, err := c.log(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.AddNote(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, opError("AddNote", req.GetPlanOrExecution(), err)
	}
	return resp.Msg, nil
}

func (c *ConnectClient) plans(ctx context.Context) (plansconnect.PlansServiceClient, error) {
	baseURL, err := c.baseURL(ctx)
	if err != nil {
		return nil, err
	}
	return plansconnect.NewPlansServiceClient(c.http, baseURL), nil
}

func (c *ConnectClient) execution(ctx context.Context) (executionconnect.ExecutionServiceClient, error) {
	baseURL, err := c.baseURL(ctx)
	if err != nil {
		return nil, err
	}
	return executionconnect.NewExecutionServiceClient(c.http, baseURL), nil
}

func (c *ConnectClient) log(ctx context.Context) (logconnect.LogServiceClient, error) {
	baseURL, err := c.baseURL(ctx)
	if err != nil {
		return nil, err
	}
	return logconnect.NewLogServiceClient(c.http, baseURL), nil
}

func (c *ConnectClient) baseURL(ctx context.Context) (string, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return "", opError("resolve", scenarioName, err)
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func opError(op, target string, err error) error {
	if target == "" {
		return fmt.Errorf("plan-manager %s: %w", op, err)
	}
	return fmt.Errorf("plan-manager %s %q: %w", op, target, err)
}
