package workflows

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	ExtractWorkflowKey      = "react-component-library/extract-assist"
	AdoptWorkflowKey        = "react-component-library/adopt-assist"
	agentManagerHTTPTimeout = 5 * time.Minute
)

// AgentManagerDispatcher only starts and waits on RCL's declared workflow.
// RCL supplies typed input and consumes its typed result; it never creates
// Agent Manager tasks/runs or composes a consumer-owned prompt.
type AgentManagerDispatcher struct {
	Resolver *discovery.Resolver
	Client   *http.Client
}

func NewAgentManagerDispatcher() *AgentManagerDispatcher {
	return &AgentManagerDispatcher{Resolver: discovery.NewResolver(discovery.ResolverConfig{}), Client: newAgentManagerHTTPClient()}
}

func newAgentManagerHTTPClient() *http.Client {
	return &http.Client{Timeout: agentManagerHTTPTimeout}
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
	started, err := d.connectClient(base).StartWorkflowExecution(ctx, connect.NewRequest(&apipb.StartWorkflowExecutionRequest{Owner: "react-component-library", WorkflowKey: workflowKey, Input: input, IdempotencyKey: in.IdempotencyKey}))
	if err != nil {
		return DispatchResult{}, err
	}
	if started.Msg.Execution == nil || started.Msg.Execution.Id == "" {
		return DispatchResult{}, fmt.Errorf("agent-manager workflow response missing execution id")
	}
	return DispatchResult{ExecutionID: started.Msg.Execution.Id, Status: statusFromWorkflow(started.Msg.Execution.Status)}, nil
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
		input["assets"] = []any{map[string]any{
			"assetId":            in.AssetID,
			"targetScenario":     in.TargetScenario,
			"sourcePath":         in.SourcePath,
			"requestedVersion":   in.RequestedVersion,
			"confirmOverwrite":   in.ConfirmOverwrite,
			"overrideValidation": in.OverrideValidation,
		}}
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
	waited, err := d.connectClient(base).WaitWorkflowExecution(ctx, connect.NewRequest(&apipb.WaitWorkflowExecutionRequest{ExecutionId: executionID}))
	if err != nil {
		return DispatchResult{ExecutionID: executionID}, err
	}
	if waited.Msg.Execution == nil {
		return DispatchResult{ExecutionID: executionID}, fmt.Errorf("agent-manager wait response missing execution")
	}
	result := DispatchResult{ExecutionID: executionID, Status: statusFromWorkflow(waited.Msg.Execution.Status)}
	if !waited.Msg.TimedOut && waited.Msg.Execution.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		output, err := d.connectClient(base).GetWorkflowExecutionResult(ctx, connect.NewRequest(&apipb.GetWorkflowExecutionResultRequest{ExecutionId: executionID, ExplicitlyAuthorized: true}))
		if err != nil {
			return result, err
		}
		result.Status, result.Error = projectAssistantResult(output.Msg.Execution, result.Status)
		result.Summary = workflowSummary(output.Msg.Execution)
	}
	if result.Status == StatusFailed {
		// Preserve a typed assistant outcome (for example, blocked validation)
		// when the orchestration envelope itself completed without a terminal
		// reason. The catalog must not turn that outcome back into a blank error.
		if result.Error == "" {
			result.Error = waited.Msg.Execution.TerminalReason.String()
		}
	}
	return result, nil
}

// Refresh reads the authoritative terminal execution result without waiting.
// This closes the client-disconnect/restart gap: RCL can reconcile its durable
// workflow projection after Agent Manager has already completed the work.
func (d *AgentManagerDispatcher) Refresh(ctx context.Context, executionID string) (DispatchResult, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return DispatchResult{ExecutionID: executionID}, err
	}
	output, err := d.connectClient(base).GetWorkflowExecutionResult(ctx, connect.NewRequest(&apipb.GetWorkflowExecutionResultRequest{ExecutionId: executionID, ExplicitlyAuthorized: true}))
	if err != nil {
		return DispatchResult{ExecutionID: executionID}, err
	}
	if output.Msg.Execution == nil {
		return DispatchResult{ExecutionID: executionID}, fmt.Errorf("agent-manager refresh response missing execution")
	}
	status, resultError := projectAssistantResult(output.Msg.Execution, statusFromWorkflow(output.Msg.Execution.Status))
	return DispatchResult{
		ExecutionID: executionID,
		Status:      status,
		Summary:     workflowSummary(output.Msg.Execution),
		Error:       resultError,
	}, nil
}

func (d *AgentManagerDispatcher) Stop(ctx context.Context, executionID string) (RunSnapshot, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return RunSnapshot{}, err
	}
	out, err := d.connectClient(base).CancelWorkflowExecution(ctx, connect.NewRequest(&apipb.WorkflowExecutionOperationRequest{ExecutionId: executionID, IdempotencyKey: "rcl-cancel/" + executionID, Reason: "cancelled from react-component-library"}))
	if err != nil {
		return RunSnapshot{}, err
	}
	if out.Msg.Execution == nil {
		return RunSnapshot{}, fmt.Errorf("agent-manager cancel response missing execution")
	}
	return RunSnapshot{Status: statusFromWorkflow(out.Msg.Execution.Status), Error: out.Msg.Execution.TerminalReason.String()}, nil
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

func workflowError(execution *domainpb.WorkflowExecution) string {
	if execution == nil || statusFromWorkflow(execution.Status) != StatusFailed {
		return ""
	}
	return execution.TerminalReason.String()
}

// projectAssistantResult keeps the RCL projection honest when Agent Manager
// completed the orchestration envelope but the assistant's typed result says
// that the requested work was blocked or needs review. A successful engine
// transition is not equivalent to a successful catalog operation.
func projectAssistantResult(execution *domainpb.WorkflowExecution, status Status) (Status, string) {
	if execution == nil || status != StatusSucceeded || execution.Output == nil {
		return status, workflowError(execution)
	}
	output, ok := execution.Output.AsInterface().(map[string]any)
	if !ok {
		return status, ""
	}
	result, _ := output["result"].(map[string]any)
	outcome, _ := result["outcome"].(string)
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "blocked", "needs_review", "failed":
		return StatusFailed, fmt.Sprintf("assistant outcome: %s", outcome)
	case "completed":
		return status, ""
	default:
		return status, ""
	}
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

func (d *AgentManagerDispatcher) connectClient(base string) apiconnect.AgentManagerServiceClient {
	if d.Client == nil {
		d.Client = newAgentManagerHTTPClient()
	}
	return apiconnect.NewAgentManagerServiceClient(d.Client, base)
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
