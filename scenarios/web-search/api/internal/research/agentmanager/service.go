package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type Service interface {
	Spawn(context.Context, SpawnRequest) (RunResult, error)
	GetRunState(context.Context, string) (RunState, error)
}

type Canceller interface {
	Cancel(context.Context, string, string, string) (RunState, error)
}

// Waiting is a separate capability so unavailable transports cannot simulate polling.
type (
	Waiter interface {
		Wait(context.Context, string, int) (RunState, error)
	}
	SpawnRequest struct {
		Query, Title        string
		GatherCap, MaxLoops int
		ConfidenceGate      float64
		IdempotencyKey      string
		Policy              map[string]any
		Questions           []map[string]any
	}
	RunResult struct{ TaskID, RunID, Status string }
	RunState  struct {
		RunID, Status, Summary, StartedAt, FinishedAt, ErrorMsg string
		Result                                                  map[string]any
		TimedOut                                                bool
	}
	agentService struct {
		client Client
		now    func() time.Time
	}
)

func NewService(c Client) Service { return &agentService{client: c, now: time.Now} }
func (s *agentService) Spawn(ctx context.Context, r SpawnRequest) (RunResult, error) {
	key := r.IdempotencyKey
	if key == "" {
		key = uuid.NewString()
	}
	inputMap := map[string]any{"query": r.Query, "gather_cap": r.GatherCap, "confidence_gate": r.ConfidenceGate, "max_loops": r.MaxLoops}
	if r.Policy != nil {
		inputMap["evidence_policy"] = r.Policy
	}
	if r.Questions != nil {
		questions := make([]any, len(r.Questions))
		for i, question := range r.Questions {
			questions[i] = question
		}
		inputMap["questions"] = questions
	}
	input, err := structpb.NewValue(inputMap)
	if err != nil {
		return RunResult{}, err
	}
	e, err := s.client.Start(ctx, &apipb.StartWorkflowExecutionRequest{Owner: "web-search", WorkflowKey: "web-search/research", IdempotencyKey: key, Input: input})
	if err != nil {
		return RunResult{}, err
	}
	if err = owned(e); err != nil {
		return RunResult{}, err
	}
	return RunResult{RunID: e.Id, Status: workflowStatus(e.Status)}, nil
}

func (s *agentService) GetRunState(ctx context.Context, id string) (RunState, error) {
	e, err := s.client.Get(ctx, id)
	if err != nil {
		return RunState{}, err
	}
	return s.state(ctx, e, false)
}

func (s *agentService) Wait(ctx context.Context, id string, seconds int) (RunState, error) {
	if seconds < 1 || seconds > 90 {
		return RunState{}, fmt.Errorf("wait timeout must be within 1..90 seconds")
	}
	// Validate ownership before attaching to a potentially sensitive execution.
	e, err := s.client.Get(ctx, id)
	if err != nil {
		return RunState{}, err
	}
	if err = owned(e); err != nil {
		return RunState{}, err
	}
	e, timedOut, err := s.client.Wait(ctx, id, seconds)
	if err != nil {
		return RunState{}, err
	}
	return s.state(ctx, e, timedOut)
}

func (s *agentService) Cancel(ctx context.Context, id, reason, idempotencyKey string) (RunState, error) {
	e, err := s.client.Get(ctx, id)
	if err != nil {
		return RunState{}, err
	}
	if err = owned(e); err != nil {
		return RunState{}, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "cancel-" + id
	}
	canceller, ok := s.client.(CancelClient)
	if !ok {
		return RunState{}, fmt.Errorf("%w: owner cancellation is not configured", ErrNotAvailable)
	}
	e, err = canceller.Cancel(ctx, &apipb.WorkflowExecutionOperationRequest{
		ExecutionId: id, IdempotencyKey: idempotencyKey, Reason: strings.TrimSpace(reason),
	})
	if err != nil {
		return RunState{}, err
	}
	return s.state(ctx, e, false)
}

func owned(e *domainpb.WorkflowExecution) error {
	if e == nil || e.Id == "" || e.Owner != "web-search" || e.WorkflowKey != "web-search/research" {
		return fmt.Errorf("%w: execution is not owned by web-search research", ErrRequestFailed)
	}
	return nil
}

func (s *agentService) state(ctx context.Context, e *domainpb.WorkflowExecution, timedOut bool) (RunState, error) {
	if err := owned(e); err != nil {
		return RunState{}, err
	}
	out := RunState{RunID: e.Id, Status: workflowStatus(e.Status), TimedOut: timedOut, ErrorMsg: workflowError(e)}
	if e.CreatedAt != nil {
		out.StartedAt = e.CreatedAt.AsTime().UTC().Format(time.RFC3339)
	}
	if e.EndedAt != nil {
		out.FinishedAt = e.EndedAt.AsTime().UTC().Format(time.RFC3339)
		full, err := s.client.Result(ctx, e.Id)
		if err != nil {
			return RunState{}, err
		}
		if err = owned(full); err != nil {
			return RunState{}, err
		}
		out.Result, out.Summary = researchOutput(full.Output)
	}
	return s.finishState(out), nil
}

func workflowError(e *domainpb.WorkflowExecution) string {
	reason := e.GetTerminalReason()
	message := reason.GetMessage()
	if message == "" {
		message = reason.GetCode()
	}
	if reason.GetBudgetName() != "" {
		message += ": " + reason.GetBudgetName()
	}
	return message
}

func researchOutput(output *structpb.Value) (map[string]any, string) {
	result := output.GetStructValue().GetFields()["result"]
	if result == nil {
		return nil, ""
	}
	fields, ok := result.AsInterface().(map[string]any)
	if !ok {
		return nil, ""
	}
	summary, _ := fields["summary"].(string)
	return fields, summary
}

func (s *agentService) finishState(out RunState) RunState {
	if out.Status != "complete" {
		return out
	}
	if out.Result == nil {
		out.ErrorMsg = "research_result_missing"
		out.Status = "failed"
		return out
	}
	if err := validateResearchResult(out.Result, s.now()); err != nil {
		out.Status = "failed"
		out.ErrorMsg = "research_result_invalid: " + err.Error()
		out.Result = nil
		out.Summary = ""
	}
	return out
}

type (
	resultCitation struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		RetrievedAt string `json:"retrieved_at"`
	}
	resultClaim struct {
		Text      string           `json:"text"`
		Citations []resultCitation `json:"citations"`
	}
	researchResult struct {
		Status         string        `json:"status"`
		Summary        string        `json:"summary"`
		Claims         []resultClaim `json:"claims"`
		Gaps           []string      `json:"gaps"`
		Contradictions []string      `json:"contradictions"`
		FindingIDs     []string      `json:"finding_ids"`
	}
)

// The declaration checks shape; this owner also checks answer sufficiency and
// citation identities. Semantic support still requires source review.
func validateResearchResult(fields map[string]any, now time.Time) error {
	result, err := decodeResearchResult(fields)
	if err != nil {
		return err
	}
	if err = validateResultCollections(fields); err != nil {
		return err
	}
	if !map[string]bool{"answered": true, "partial": true, "abstained": true}[result.Status] {
		return fmt.Errorf("unknown research status")
	}
	if err = validateAnsweredResult(result); err != nil {
		return err
	}
	for _, claim := range result.Claims {
		if err = validateResultClaim(claim, now); err != nil {
			return err
		}
	}
	return nil
}

func decodeResearchResult(fields map[string]any) (researchResult, error) {
	var result researchResult
	payload, err := json.Marshal(fields)
	if err != nil || len(payload) > 48000 {
		return result, fmt.Errorf("result exceeds bounded handoff")
	}
	if err = json.Unmarshal(payload, &result); err != nil {
		return result, fmt.Errorf("invalid research fields: %w", err)
	}
	return result, nil
}

func validateResultCollections(fields map[string]any) error {
	for _, key := range []string{"claims", "gaps", "contradictions", "finding_ids"} {
		values, ok := fields[key].([]any)
		if !ok || len(values) > 20 {
			return fmt.Errorf("%s must be a bounded array", key)
		}
	}
	return nil
}

func validateAnsweredResult(result researchResult) error {
	if result.Status != "answered" {
		return nil
	}
	if strings.TrimSpace(result.Summary) == "" || len(result.Claims) == 0 || len(result.Gaps) > 0 || len(result.Contradictions) > 0 {
		return fmt.Errorf("answered result has missing or unresolved evidence")
	}
	return nil
}

func validateResultClaim(claim resultClaim, now time.Time) error {
	if strings.TrimSpace(claim.Text) == "" {
		return fmt.Errorf("empty claim")
	}
	if len(claim.Citations) == 0 || len(claim.Citations) > 10 {
		return fmt.Errorf("claim lacks bounded citations")
	}
	for _, citation := range claim.Citations {
		if err := validateResultCitation(citation, now); err != nil {
			return err
		}
	}
	return nil
}

func validateResultCitation(citation resultCitation, now time.Time) error {
	if !validCitationURL(citation.URL) {
		return fmt.Errorf("invalid citation URL")
	}
	retrieved, err := time.Parse(time.RFC3339, citation.RetrievedAt)
	if err != nil || retrieved.After(now) {
		return fmt.Errorf("invalid citation retrieval date")
	}
	return nil
}

func validCitationURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Hostname() != "" && u.User == nil && (u.Scheme == "https" || u.Scheme == "http")
}

func workflowStatus(s domainpb.WorkflowExecutionStatus) string {
	if s == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		return "complete"
	}
	return strings.ToLower(strings.TrimPrefix(s.String(), "WORKFLOW_EXECUTION_STATUS_"))
}
