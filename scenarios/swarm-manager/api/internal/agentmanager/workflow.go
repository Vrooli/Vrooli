package agentmanager

import (
	"context"
	"net/http"
	"net/url"

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
