package workflows

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	ExtractWorkflowKey = "react-component-library/extract-assist"
	AdoptWorkflowKey   = "react-component-library/adopt-assist"
)

// AgentManagerDispatcher only starts and waits on RCL's declared workflow.
// RCL supplies typed input and consumes its typed result; it never creates
// Agent Manager tasks/runs or composes a consumer-owned prompt.
type AgentManagerDispatcher struct {
	Resolver *discovery.Resolver
	Client   *http.Client
}

func NewAgentManagerDispatcher() *AgentManagerDispatcher {
	return &AgentManagerDispatcher{Resolver: discovery.NewResolver(discovery.ResolverConfig{}), Client: &http.Client{}}
}

func (d *AgentManagerDispatcher) Start(ctx context.Context, in StartInput) (DispatchResult, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return DispatchResult{}, err
	}
	input, err := structpb.NewValue(workflowInput(in))
	if err != nil {
		return DispatchResult{}, fmt.Errorf("encode catalog workflow input: %w", err)
	}
	workflowKey, err := workflowKeyForKind(in.Kind)
	if err != nil {
		return DispatchResult{}, err
	}
	var started apipb.WorkflowExecutionResponse
	if err := d.post(ctx, base, "/api/v1/workflow-executions", &apipb.StartWorkflowExecutionRequest{Owner: "react-component-library", WorkflowKey: workflowKey, Input: input, IdempotencyKey: in.IdempotencyKey}, &started); err != nil {
		return DispatchResult{}, err
	}
	if started.Execution == nil || started.Execution.Id == "" {
		return DispatchResult{}, fmt.Errorf("agent-manager workflow response missing execution id")
	}
	return DispatchResult{ExecutionID: started.Execution.Id, Status: statusFromWorkflow(started.Execution.Status)}, nil
}

// workflowInput is intentionally kind-specific. The two declared workflows
// have strict schemas, so forwarding fields from the other operation would
// turn an otherwise valid extract or adopt request into an additional-property
// schema failure before Agent Manager can dispatch it.
func workflowInput(in StartInput) map[string]any {
	input := map[string]any{"kind": string(in.Kind)}
	switch in.Kind {
	case KindExtract:
		input["assetId"] = in.AssetID
		input["sourceScenario"] = in.SourceScenario
		input["sourcePath"] = in.SourcePath
		input["requestedVersion"] = in.RequestedVersion
	case KindAdopt:
		input["assetId"] = in.AssetID
		input["targetScenario"] = in.TargetScenario
		input["sourcePath"] = in.SourcePath
		input["requestedVersion"] = in.RequestedVersion
		input["confirmOverwrite"] = in.ConfirmOverwrite
		input["overrideValidation"] = in.OverrideValidation
	}
	return input
}

func workflowKeyForKind(kind Kind) (string, error) {
	switch kind {
	case KindExtract:
		return ExtractWorkflowKey, nil
	case KindAdopt:
		return AdoptWorkflowKey, nil
	default:
		return "", fmt.Errorf("unsupported RCL workflow kind %q", kind)
	}
}

func (d *AgentManagerDispatcher) Wait(ctx context.Context, executionID string) (DispatchResult, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return DispatchResult{ExecutionID: executionID}, err
	}
	var waited apipb.WaitWorkflowExecutionResponse
	if err := d.post(ctx, base, "/api/v1/workflow-executions/"+executionID+"/wait", &apipb.WaitWorkflowExecutionRequest{ExecutionId: executionID}, &waited); err != nil {
		return DispatchResult{ExecutionID: executionID}, err
	}
	if waited.Execution == nil {
		return DispatchResult{ExecutionID: executionID}, fmt.Errorf("agent-manager wait response missing execution")
	}
	result := DispatchResult{ExecutionID: executionID, Status: statusFromWorkflow(waited.Execution.Status)}
	if !waited.TimedOut && waited.Execution.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		var output apipb.WorkflowExecutionResponse
		if err := d.get(ctx, base, "/api/v1/workflow-executions/"+executionID+"/result?explicitly_authorized=true", &output); err != nil {
			return result, err
		}
		result.Summary = workflowSummary(output.Execution)
	}
	if result.Status == StatusFailed {
		result.Error = waited.Execution.TerminalReason.String()
	}
	return result, nil
}

func (d *AgentManagerDispatcher) Stop(ctx context.Context, executionID string) (RunSnapshot, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return RunSnapshot{}, err
	}
	var out apipb.WorkflowExecutionOperationResponse
	if err := d.post(ctx, base, "/api/v1/workflow-executions/"+executionID+"/cancel", &apipb.WorkflowExecutionOperationRequest{ExecutionId: executionID, IdempotencyKey: "rcl-cancel/" + executionID, Reason: "cancelled from react-component-library"}, &out); err != nil {
		return RunSnapshot{}, err
	}
	if out.Execution == nil {
		return RunSnapshot{}, fmt.Errorf("agent-manager cancel response missing execution")
	}
	return RunSnapshot{Status: statusFromWorkflow(out.Execution.Status), Error: out.Execution.TerminalReason.String()}, nil
}

func workflowSummary(execution *domainpb.WorkflowExecution) string {
	if execution == nil || execution.Output == nil {
		return ""
	}
	output, ok := execution.Output.AsInterface().(map[string]any)
	if !ok {
		return ""
	}
	result, _ := output["result"].(map[string]any)
	summary, _ := result["summary"].(string)
	return summary
}

func (d *AgentManagerDispatcher) baseURL(ctx context.Context) (string, error) {
	if d.Resolver == nil {
		d.Resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	base, err := d.Resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("agent-manager unavailable: %w", err)
	}
	return strings.TrimRight(base, "/"), nil
}

func (d *AgentManagerDispatcher) post(ctx context.Context, base, path string, in, out proto.Message) error {
	return d.request(ctx, http.MethodPost, base, path, in, out)
}

func (d *AgentManagerDispatcher) get(ctx context.Context, base, path string, out proto.Message) error {
	return d.request(ctx, http.MethodGet, base, path, nil, out)
}

func (d *AgentManagerDispatcher) request(ctx context.Context, method, base, path string, in, out proto.Message) error {
	if d.Client == nil {
		d.Client = &http.Client{}
	}
	var body io.Reader
	if in != nil {
		encoded, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := d.Client.Do(req)
	if err != nil {
		return fmt.Errorf("agent-manager request: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read agent-manager response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("agent-manager status %d: %s", response.StatusCode, string(raw))
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode agent-manager response: %w", err)
	}
	return nil
}

func statusFromWorkflow(s domainpb.WorkflowExecutionStatus) Status {
	switch s {
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_PENDING:
		return StatusQueued
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING, domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_WAITING:
		return StatusRunning
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED:
		return StatusSucceeded
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED, domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLING:
		return StatusStopped
	default:
		return StatusFailed
	}
}
