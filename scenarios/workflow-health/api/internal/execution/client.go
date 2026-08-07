package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"workflow-health/internal/workflows"
)

type ConnectClient struct {
	workflows  apiconnect.WorkflowsServiceClient
	executions apiconnect.ExecutionsServiceClient
	timeout    time.Duration
}

func NewConnectClient(baseURL string, httpClient *http.Client) *ConnectClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL = strings.TrimRight(strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/api/v1"), "/")
	return &ConnectClient{
		workflows:  apiconnect.NewWorkflowsServiceClient(httpClient, baseURL),
		executions: apiconnect.NewExecutionsServiceClient(httpClient, baseURL),
		timeout:    5 * time.Minute,
	}
}

func (c *ConnectClient) ValidateResolved(ctx context.Context, definition map[string]any) (*ValidationResult, error) {
	def, err := definitionToProto(definition)
	if err != nil {
		return nil, err
	}
	resp, err := c.workflows.ValidateResolvedWorkflow(ctx, connect.NewRequest(&basapi.ValidateWorkflowRequest{Workflow: def}))
	if err != nil {
		return nil, err
	}
	result := resp.Msg.GetResult()
	if result == nil {
		return &ValidationResult{Valid: true}, nil
	}
	out := &ValidationResult{Valid: result.GetValid()}
	for _, issue := range result.GetErrors() {
		out.Errors = append(out.Errors, ValidationIssue{Code: issue.GetCode(), Message: issue.GetMessage()})
	}
	for _, issue := range result.GetWarnings() {
		out.Warnings = append(out.Warnings, ValidationIssue{Code: issue.GetCode(), Message: issue.GetMessage()})
	}
	return out, nil
}

func (c *ConnectClient) ExecuteAdhoc(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	def, err := definitionToProto(req.Definition)
	if err != nil {
		return nil, err
	}
	basReq := &basexecution.ExecuteAdhocRequest{
		FlowDefinition: def,
		// BAS's synchronous endpoint is intentionally protected by its regular
		// request timeout. A workflow suite can legitimately run longer than
		// that, so start asynchronously and make the durable provider own the
		// completion wait through the execution-status contract instead.
		WaitForCompletion: false,
		Metadata: &basexecution.ExecutionMetadata{
			Name:        strings.TrimSpace(req.Name),
			Description: strings.TrimSpace(req.Description),
		},
		Parameters: parametersToProto(req.Parameters),
		Options: &basexecution.ExecuteWorkflowOptions{
			RequiresVideo: req.Options.RequiresVideo,
			RequiresTrace: req.Options.RequiresTrace,
			RequiresHar:   req.Options.RequiresHAR,
		},
	}
	if req.Options.ElectronTarget != nil {
		target := req.Options.ElectronTarget
		basReq.Options.ElectronTarget = &basexecution.ElectronTarget{
			TargetId: target.TargetID, CdpEndpoint: target.CDPEndpoint,
			RendererId: target.RendererID, RendererUrl: target.RendererURL,
			RendererTitle: target.RendererTitle, ScenarioName: target.ScenarioName,
			ArtifactDigest: target.ArtifactDigest, ContextId: target.ContextID,
			CdpTransport: target.CDPTransport,
		}
	}
	if req.Options.ValidationContext != nil {
		context := req.Options.ValidationContext
		basReq.Options.ValidationContext = &basexecution.ValidationContext{
			ContextId: context.ContextID, ScenarioName: context.ScenarioName,
			ArtifactDigest: context.ArtifactDigest, TargetId: context.TargetID,
			WorkflowId: context.WorkflowID, ProfileId: context.ProfileID,
			IsolationLeaseId: context.IsolationLeaseID,
		}
	}
	request := connect.NewRequest(basReq)
	applyRequestHeaders(request.Header(), req.Parameters.ExtraHeaders)
	resp, err := c.workflows.ExecuteAdhocWorkflow(ctx, request)
	if err != nil {
		return nil, err
	}
	executionID := strings.TrimSpace(resp.Msg.GetExecutionId())
	if executionID == "" {
		return nil, fmt.Errorf("BAS started an adhoc workflow without an execution id")
	}
	return c.waitForExecution(ctx, executionID, req.Parameters.ExtraHeaders)
}

func (c *ConnectClient) waitForExecution(ctx context.Context, executionID string, headers map[string]string) (*ExecuteResult, error) {
	waitCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		request := connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: executionID})
		applyRequestHeaders(request.Header(), headers)
		resp, err := c.executions.GetExecution(waitCtx, request)
		if err != nil {
			return nil, fmt.Errorf("read BAS execution %s: %w", executionID, err)
		}
		if resp == nil || resp.Msg == nil || resp.Msg.GetExecution() == nil {
			return nil, fmt.Errorf("BAS returned an empty execution status for %s", executionID)
		}
		execution := resp.Msg.GetExecution()
		if execution.GetCompletedAt() != nil {
			return &ExecuteResult{
				ExecutionID: executionID,
				Status:      execution.GetStatus(),
				Error:       strings.TrimSpace(execution.GetError()),
				Execution:   execution,
			}, nil
		}
		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("BAS execution %s did not complete within %s: %w", executionID, c.timeout, waitCtx.Err())
		case <-ticker.C:
		}
	}
}

func applyRequestHeaders(dst http.Header, headers map[string]string) {
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			dst.Set(key, value)
		}
	}
}

func (c *ConnectClient) Timeline(ctx context.Context, executionID string, headers map[string]string) (*bastimeline.ExecutionTimeline, error) {
	request := connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: executionID})
	applyRequestHeaders(request.Header(), headers)
	resp, err := c.executions.GetExecutionTimeline(ctx, request)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func definitionToProto(definition map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	return workflows.DecodeBASDefinition(definition)
}

// validationArtifactProfile asks BAS to collect what proves a case passed or
// failed and skip the rest. A validation run needs assertions, extracted data,
// and a frame at the point of failure; it does not need a full-viewport PNG of
// every variable assignment, which is where most of a suite's wall-clock goes.
//
// Product and replay executions do not set this, so they keep every frame.
const validationArtifactProfile = "validation"

func parametersToProto(params Parameters) *basexecution.ExecutionParameters {
	out := &basexecution.ExecutionParameters{
		ArtifactConfig: &basexecution.ArtifactCollectionConfig{
			Profile: proto.String(validationArtifactProfile),
		},
	}
	// The artifact profile above always populates the message, so the previous
	// "return nil when nothing was set" branch can no longer be reached.
	if strings.TrimSpace(params.ProjectRoot) != "" {
		v := strings.TrimSpace(params.ProjectRoot)
		out.ProjectRoot = &v
	}
	if len(params.InitialParams) > 0 {
		out.InitialParams = anyMapToJSONValueMap(params.InitialParams)
	}
	if len(params.InitialStore) > 0 {
		out.InitialStore = anyMapToJSONValueMap(params.InitialStore)
	}
	if len(params.Env) > 0 {
		out.Env = anyMapToJSONValueMap(params.Env)
	}
	if len(params.ExtraHeaders) > 0 {
		out.BrowserProfile = &basbase.BrowserProfile{ExtraHeaders: params.ExtraHeaders}
	}
	return out
}

func anyMapToJSONValueMap(input map[string]any) map[string]*commonv1.JsonValue {
	out := make(map[string]*commonv1.JsonValue, len(input))
	for k, v := range input {
		out[k] = anyToJSONValue(v)
	}
	return out
}

func anyToJSONValue(v any) *commonv1.JsonValue {
	switch x := v.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: x}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(x)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: x}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: x}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: x}}
	default:
		data, _ := json.Marshal(x)
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: string(data)}}
	}
}
