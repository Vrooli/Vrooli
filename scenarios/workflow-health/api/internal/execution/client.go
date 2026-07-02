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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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
		FlowDefinition:    def,
		WaitForCompletion: true,
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
	resp, err := c.workflows.ExecuteAdhocWorkflow(ctx, connect.NewRequest(basReq))
	if err != nil {
		return nil, err
	}
	out := &ExecuteResult{
		ExecutionID: strings.TrimSpace(resp.Msg.GetExecutionId()),
		Status:      resp.Msg.GetStatus(),
	}
	if resp.Msg.Error != nil {
		out.Error = strings.TrimSpace(resp.Msg.GetError())
	}
	return out, nil
}

func (c *ConnectClient) Timeline(ctx context.Context, executionID string) (*bastimeline.ExecutionTimeline, error) {
	resp, err := c.executions.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: executionID}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func definitionToProto(definition map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	data, err := json.Marshal(normalizeWorkflowDefinitionAliases(definition))
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	var out basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode workflow definition: %w", err)
	}
	return &out, nil
}

func normalizeWorkflowDefinitionAliases(definition map[string]any) map[string]any {
	out := make(map[string]any, len(definition))
	for key, value := range definition {
		out[key] = value
	}
	metadata, ok := definition["metadata"].(map[string]any)
	if !ok {
		return out
	}
	metadataOut := make(map[string]any, len(metadata))
	for key, value := range metadata {
		metadataOut[key] = value
	}
	if mode, ok := metadataOut["execution_mode"].(string); ok {
		switch strings.ToLower(strings.TrimSpace(mode)) {
		case "observer":
			metadataOut["execution_mode"] = "EXECUTION_MODE_OBSERVER"
		case "mutating":
			metadataOut["execution_mode"] = "EXECUTION_MODE_MUTATING"
		case "destructive":
			metadataOut["execution_mode"] = "EXECUTION_MODE_DESTRUCTIVE"
		}
	}
	out["metadata"] = metadataOut
	return out
}

func parametersToProto(params Parameters) *basexecution.ExecutionParameters {
	out := &basexecution.ExecutionParameters{}
	populated := false
	if strings.TrimSpace(params.ProjectRoot) != "" {
		v := strings.TrimSpace(params.ProjectRoot)
		out.ProjectRoot = &v
		populated = true
	}
	if len(params.InitialParams) > 0 {
		out.InitialParams = anyMapToJSONValueMap(params.InitialParams)
		populated = true
	}
	if len(params.InitialStore) > 0 {
		out.InitialStore = anyMapToJSONValueMap(params.InitialStore)
		populated = true
	}
	if len(params.Env) > 0 {
		out.Env = anyMapToJSONValueMap(params.Env)
		populated = true
	}
	if len(params.ExtraHeaders) > 0 {
		out.BrowserProfile = &basbase.BrowserProfile{ExtraHeaders: params.ExtraHeaders}
		populated = true
	}
	if !populated {
		return nil
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
