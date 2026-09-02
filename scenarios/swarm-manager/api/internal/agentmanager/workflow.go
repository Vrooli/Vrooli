package agentmanager

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

type WorkflowStart struct {
	ExecutionID      string
	RunID            string
	DefinitionDigest string
	Status           domainpb.WorkflowExecutionStatus
}

// WorkflowProgress is a read-through projection of Agent Manager's durable
// workflow trace. It intentionally contains no prompt or result payload.
type WorkflowProgress struct {
	CurrentNode    string           `json:"current_node"`
	SliceCount     int              `json:"slice_count"`
	Turns          int32            `json:"turns"`
	Tokens         int32            `json:"tokens"`
	CostUSD        float64          `json:"cost_usd"`
	EdgeTraversals map[string]int32 `json:"edge_traversals"`
	UpdatedAt      string           `json:"updated_at,omitempty"`
}

// WorkflowExecutionState is the small lifecycle projection used by Swarm's
// activity ledger. It deliberately omits workflow input, output, and journal
// data; callers that need live progress use WorkflowProgress instead.
type WorkflowExecutionState struct {
	Status           domainpb.WorkflowExecutionStatus
	UpdatedAt        string
	TerminalCode     string
	TerminalEvidence bool
}

// WorkflowActivity identifies the durable owner of a declared workflow. It is
// deliberately correlation-only: Agent Manager remains authoritative for live
// workflow state and Swarm stores no copied progress.
type WorkflowActivity struct {
	OwnerType   string
	OwnerKind   string
	OwnerName   string
	OwnerTitle  string
	Purpose     string
	WorkflowKey string
}

// WorkflowActivityRecorder records a workflow launch at the single transport
// chokepoint. Implementations must not call Agent Manager from this method.
type WorkflowActivityRecorder interface {
	RecordWorkflowStart(context.Context, WorkflowActivity, WorkflowStart) error
}

type WorkflowService struct {
	client           *HTTPClient
	startGuard       WorkflowStartGuard
	activityRecorder WorkflowActivityRecorder
}

func (s *WorkflowService) SetWorkflowActivityRecorder(recorder WorkflowActivityRecorder) {
	s.activityRecorder = recorder
}

// GetWorkflowProgress reads the current workflow trace without collecting or
// persisting a terminal result. It is safe for live operator polling.
func (s *WorkflowService) GetWorkflowProgress(ctx context.Context, executionID string) (WorkflowProgress, error) {
	trace, err := s.client.GetWorkflowExecutionTrace(ctx, executionID)
	if err != nil {
		return WorkflowProgress{}, err
	}
	if trace.Execution == nil {
		return WorkflowProgress{}, fmt.Errorf("%w: workflow trace omitted execution", ErrRequestFailed)
	}
	progress := WorkflowProgress{CurrentNode: trace.Execution.GetCurrentNodeId(), EdgeTraversals: maps.Clone(trace.Execution.GetEdgeTraversals())}
	for _, entry := range trace.Journal {
		if entry.GetNodeId() == "slice" {
			progress.SliceCount++
		}
	}
	if usage := trace.Execution.GetBudgetUsage(); usage != nil {
		progress.Turns, progress.Tokens, progress.CostUSD = usage.GetTurns(), usage.GetTokens(), usage.GetCostUsd()
	}
	if updated := trace.Execution.GetUpdatedAt(); updated != nil {
		progress.UpdatedAt = updated.AsTime().UTC().Format(time.RFC3339Nano)
	}
	return progress, nil
}

// GetWorkflowExecutionState reads the durable lifecycle state needed to
// reconcile a correlation-only activity record. Trace is the lifecycle
// authority but intentionally omits output, so a successful execution uses
// the explicitly-authorized result endpoint only to establish that terminal
// evidence exists.
func (s *WorkflowService) GetWorkflowExecutionState(ctx context.Context, executionID string) (WorkflowExecutionState, error) {
	return readWorkflowExecutionState(ctx, s.client, executionID)
}

// GetWorkflowExecutionState exposes the same read-only lifecycle projection
// through the long-lived AgentService used by activity reconciliation.
func (s *AgentService) GetWorkflowExecutionState(ctx context.Context, executionID string) (WorkflowExecutionState, error) {
	if !s.enabled {
		return WorkflowExecutionState{}, ErrNotAvailable
	}
	return readWorkflowExecutionState(ctx, s.client, executionID)
}

func readWorkflowExecutionState(ctx context.Context, client *HTTPClient, executionID string) (WorkflowExecutionState, error) {
	trace, err := client.GetWorkflowExecutionTrace(ctx, executionID)
	if err != nil {
		return WorkflowExecutionState{}, err
	}
	if trace.Execution == nil {
		return WorkflowExecutionState{}, fmt.Errorf("%w: workflow trace omitted execution", ErrRequestFailed)
	}
	state := WorkflowExecutionState{Status: trace.Execution.GetStatus()}
	if reason := trace.Execution.GetTerminalReason(); reason != nil {
		state.TerminalCode = reason.GetCode()
	}
	if updated := trace.Execution.GetUpdatedAt(); updated != nil {
		state.UpdatedAt = updated.AsTime().UTC().Format(time.RFC3339Nano)
	}
	if state.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		result, err := client.GetWorkflowExecutionResult(ctx, executionID)
		if err != nil {
			return WorkflowExecutionState{}, err
		}
		state.TerminalEvidence = result.Execution != nil && result.Execution.GetOutput() != nil
	}
	return state, nil
}

func NewWorkflowService() *WorkflowService { return &WorkflowService{client: NewHTTPClient()} }

func NewWorkflowServiceWithClient(client *HTTPClient) *WorkflowService {
	if client == nil {
		client = NewHTTPClient()
	}
	return &WorkflowService{client: client}
}

func (c *HTTPClient) StartWorkflowExecution(ctx context.Context, req *apipb.StartWorkflowExecutionRequest) (*apipb.WorkflowExecutionResponse, error) {
	return c.workflowPost(ctx, "/api/v1/workflow-executions", req)
}

// WaitWorkflowExecution long-polls agent-manager until the execution is
// terminal or timeoutSeconds elapses server-side. Canceling this call (ctx) or
// the wait timing out never cancels the execution.
func (c *HTTPClient) WaitWorkflowExecution(ctx context.Context, executionID string, timeoutSeconds int32) (*apipb.WaitWorkflowExecutionResponse, error) {
	req := &apipb.WaitWorkflowExecutionRequest{ExecutionId: executionID, TimeoutSeconds: timeoutSeconds}
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/workflow-executions/"+url.PathEscape(executionID)+"/wait", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}
	var result apipb.WaitWorkflowExecutionResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPClient) GetWorkflowExecutionResult(ctx context.Context, executionID string) (*apipb.WorkflowExecutionResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/workflow-executions/"+url.PathEscape(executionID)+"/result?explicitly_authorized=true", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}
	var result apipb.WorkflowExecutionResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPClient) GetWorkflowExecutionTrace(ctx context.Context, executionID string) (*apipb.GetWorkflowExecutionTraceResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/workflow-executions/"+url.PathEscape(executionID)+"/trace", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}
	var result apipb.GetWorkflowExecutionTraceResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *HTTPClient) workflowPost(ctx context.Context, path string, req proto.Message) (*apipb.WorkflowExecutionResponse, error) {
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, readErrorResponse(resp)
	}
	var result apipb.WorkflowExecutionResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
