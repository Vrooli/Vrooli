package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const backlogWorkshopWorkflowKey = "swarm-manager/backlog-workshop-round"

// BacklogWorkshopSnapshot is the immutable consumer DTO sent to the generic
// Agent Manager workflow. The adapter intentionally carries values, not a
// filesystem path or mutable backlog handle.
type BacklogWorkshopSnapshot struct {
	Kind         string
	Name         string
	Version      string
	Title        string
	Description  string
	Status       string
	Priority     int
	Tags         []string
	OperatorNote string
}

type WorkflowStart struct {
	ExecutionID      string
	RunID            string
	DefinitionDigest string
	Status           domainpb.WorkflowExecutionStatus
}

type WorkshopWorkflowCompletion struct {
	ExecutionID      string
	DefinitionDigest string
	RunID            string
	ProfileIdentity  string
	EntityKind       string
	EntityName       string
	EntityVersion    string
	Result           json.RawMessage
}

// WorkshopWorkflowService is the narrow command-result seam consumed by the
// backlog package. It cannot create or continue runs directly.
type WorkshopWorkflowService interface {
	StartWorkshopRound(context.Context, BacklogWorkshopSnapshot, string) (WorkflowStart, error)
	CollectWorkshopRound(context.Context, string) (WorkshopWorkflowCompletion, error)
}

type WorkflowService struct {
	client     *HTTPClient
	startGuard WorkflowStartGuard
}

func NewWorkflowService() *WorkflowService { return &WorkflowService{client: NewHTTPClient()} }

func NewWorkflowServiceWithClient(client *HTTPClient) *WorkflowService {
	if client == nil {
		client = NewHTTPClient()
	}
	return &WorkflowService{client: client}
}

func (s *WorkflowService) StartWorkshopRound(ctx context.Context, snapshot BacklogWorkshopSnapshot, idempotencyKey string) (WorkflowStart, error) {
	// Workflow definitions are registered by agent-manager's startup declaration
	// sweep; the consumer no longer reconciles per start.
	tags := make([]any, len(snapshot.Tags))
	for i, tag := range snapshot.Tags {
		tags[i] = tag
	}
	input, err := structpb.NewValue(map[string]any{
		"entity": map[string]any{"kind": snapshot.Kind, "name": snapshot.Name, "version": snapshot.Version},
		"snapshot": map[string]any{
			"title": snapshot.Title, "description": snapshot.Description, "status": snapshot.Status,
			"priority": snapshot.Priority, "tags": tags,
		},
		"operatorNote": snapshot.OperatorNote,
	})
	if err != nil {
		return WorkflowStart{}, fmt.Errorf("encode workshop snapshot: %w", err)
	}
	return s.StartWorkflow(ctx, Invocation{Owner: "swarm-manager", WorkflowKey: backlogWorkshopWorkflowKey, Input: input, IdempotencyKey: idempotencyKey, FirstRunNodeID: "workshop"})
}

// workshopWaitTimeoutSeconds bounds each server-side blocking wait. It stays
// under the agent-manager HTTP client timeout (20s) so the long-poll returns
// cleanly; a wait that reaches this bound without a terminal execution surfaces
// as ErrWorkflowNotReady so the durable pending-file backstop re-arms the wait.
const workshopWaitTimeoutSeconds = 15

func (s *WorkflowService) CollectWorkshopRound(ctx context.Context, executionID string) (WorkshopWorkflowCompletion, error) {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: execution id is required", ErrRequestFailed)
	}
	invocation, err := s.CollectWorkflow(ctx, executionID)
	if err != nil {
		return WorkshopWorkflowCompletion{}, err
	}
	if invocation.Status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: workflow execution is not successfully terminal", ErrWorkflowNotReady)
	}
	if invocation.Input == nil || invocation.Output == nil {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: authorized workflow payload is incomplete", ErrRequestFailed)
	}
	input, ok := invocation.Input.AsInterface().(map[string]any)
	if !ok {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: workflow input is not an object", ErrRequestFailed)
	}
	entity, ok := input["entity"].(map[string]any)
	if !ok {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: workflow entity snapshot is missing", ErrRequestFailed)
	}
	output, ok := invocation.Output.AsInterface().(map[string]any)
	if !ok {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("%w: workflow output is not an object", ErrRequestFailed)
	}
	result, err := json.Marshal(output["result"])
	if err != nil {
		return WorkshopWorkflowCompletion{}, fmt.Errorf("encode workflow result: %w", err)
	}
	completion := WorkshopWorkflowCompletion{
		ExecutionID: executionID, DefinitionDigest: invocation.DefinitionDigest,
		EntityKind: stringValue(entity["kind"]), EntityName: stringValue(entity["name"]), EntityVersion: stringValue(entity["version"]),
		Result: result,
	}
	for _, attempt := range invocation.Attempts {
		if attempt.NodeId == "workshop" {
			completion.RunID, completion.ProfileIdentity = attempt.RunId, attempt.ProfileIdentity
			break
		}
	}
	return completion, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
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
