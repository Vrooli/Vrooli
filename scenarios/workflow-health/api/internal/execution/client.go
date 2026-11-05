package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"workflow-health/internal/workflows"

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
		if rendererURL := strings.TrimSpace(target.RendererURL); rendererURL != "" {
			// The Electron renderer is the trusted start URL for this bound
			// execution. This lets BAS navigate the attached renderer without
			// asking the desktop provider to rediscover a Tier 1 scenario port.
			basReq.Parameters.StartUrl = proto.String(rendererURL)
		}
		// The public BAS contract generalized the legacy electron_target field
		// to app_target so Android WebView and future target adapters can share
		// the same validation seam. Keep workflow-health's internal name for
		// compatibility with its desktop validation API, but emit the current
		// typed contract at the provider boundary.
		basReq.Options.AppTarget = &basexecution.AppTarget{
			TargetKind: "electron",
			TargetId:   target.TargetID, CdpEndpoint: target.CDPEndpoint,
			RendererId: target.RendererID, RendererUrl: target.RendererURL,
			RendererTitle: target.RendererTitle, ScenarioName: target.ScenarioName,
			ArtifactDigest: target.ArtifactDigest, ContextId: target.ContextID,
			CdpTransport:     target.CDPTransport,
			DownloadPath:     target.DownloadRoot,
			TestDatabaseRoot: target.TestDatabaseRoot,
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
	// ExtraHeaders are primarily browser-profile headers for the target app.
	// Do not send the target's routed-test marker to BAS itself: BAS is the
	// durable execution provider, and its status RPCs must remain in the
	// provider's control-plane database while the browser carries the marker
	// to the isolated target scenario.
	applyRequestHeaders(request.Header(), controlPlaneHeaders(req.Parameters.ExtraHeaders))
	resp, err := c.workflows.ExecuteAdhocWorkflow(ctx, request)
	if err != nil {
		return nil, err
	}
	executionID := strings.TrimSpace(resp.Msg.GetExecutionId())
	if executionID == "" {
		return nil, fmt.Errorf("BAS started an adhoc workflow without an execution id")
	}
	return c.waitForExecution(ctx, executionID, controlPlaneHeaders(req.Parameters.ExtraHeaders), workflowTimeout(req.Definition, c.timeout))
}

func (c *ConnectClient) waitForExecution(ctx context.Context, executionID string, headers map[string]string, timeout time.Duration) (*ExecuteResult, error) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	warned := false
	for {
		if ctx.Err() != nil {
			return c.settleCancelledExecution(ctx, executionID, headers, timeout)
		}
		// The timeout bounds one observation, not the durable browser operation.
		// Renewing an attachment must never start the next case over a live child.
		observeCtx, cancel := context.WithTimeout(ctx, timeout)
		request := connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: executionID})
		applyRequestHeaders(request.Header(), headers)
		resp, err := c.executions.GetExecution(observeCtx, request)
		cancel()
		if ctx.Err() != nil {
			return c.settleCancelledExecution(ctx, executionID, headers, timeout)
		}
		if err == nil && resp != nil && resp.Msg != nil && resp.Msg.GetExecution() != nil {
			execution := resp.Msg.GetExecution()
			if execution.GetCompletedAt() != nil {
				return executionResult(executionID, execution), nil
			}
		} else if !warned {
			slog.Warn("BAS observation unavailable; retaining the same durable execution", "execution_id", executionID, "error", err)
			warned = true
		}
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}

func executionResult(id string, execution *basexecution.Execution) *ExecuteResult {
	return &ExecuteResult{ExecutionID: id, Status: execution.GetStatus(), Error: strings.TrimSpace(execution.GetError()), Execution: execution}
}

// An actual owner abort joins the BAS cancellation owner before the caller may
// release its fixture. A lost acknowledgement is not proof that cleanup ended.
func (c *ConnectClient) settleCancelledExecution(parent context.Context, id string, headers map[string]string, timeout time.Duration) (*ExecuteResult, error) {
	cleanup := context.WithoutCancel(parent)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	stopNeeded, stopAcknowledged := true, false
	for {
		if stopNeeded {
			stopCtx, cancel := context.WithTimeout(cleanup, timeout)
			req := connect.NewRequest(&basapi.StopExecutionRequest{ExecutionId: id})
			applyRequestHeaders(req.Header(), headers)
			_, err := c.executions.StopExecution(stopCtx, req)
			cancel()
			if err == nil {
				stopNeeded, stopAcknowledged = false, true
			} else {
				slog.Warn("BAS cancellation not yet settled; retaining fixture ownership", "execution_id", id, "error", err)
				// Do not repeat an authority denial. The owner must reconcile it; reads
				// cannot prove cleanup completion; ownership stays held for reconciliation.
				if code := connect.CodeOf(err); code == connect.CodePermissionDenied || code == connect.CodeUnauthenticated {
					stopNeeded = false
				}
			}
		}
		observeCtx, cancel := context.WithTimeout(cleanup, timeout)
		req := connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: id})
		applyRequestHeaders(req.Header(), headers)
		resp, err := c.executions.GetExecution(observeCtx, req)
		cancel()
		if stopAcknowledged && err == nil && resp != nil && resp.Msg.GetExecution() != nil && resp.Msg.GetExecution().GetCompletedAt() != nil {
			return executionResult(id, resp.Msg.GetExecution()), fmt.Errorf("BAS execution %s settled after owner cancellation: %w", id, parent.Err())
		}
		<-ticker.C
	}
}

// workflowTimeout bounds an individual status/cleanup attachment. The BAS
// execution owner enforces authored workflow deadlines; an observation timeout
// does not end its operation or permit another case to start.
func workflowTimeout(definition map[string]any, fallback time.Duration) time.Duration {
	settings, ok := definition["settings"].(map[string]any)
	if !ok {
		return fallback
	}
	raw, ok := settings["timeout_ms"]
	if !ok {
		return fallback
	}
	var millis int64
	switch value := raw.(type) {
	case float64:
		millis = int64(value)
	case int:
		millis = int64(value)
	case int64:
		millis = value
	}
	if millis <= 0 {
		return fallback
	}
	timeout := time.Duration(millis) * time.Millisecond
	if timeout > 2*time.Hour {
		return 2 * time.Hour
	}
	if timeout > fallback {
		return timeout
	}
	return fallback
}

func applyRequestHeaders(dst http.Header, headers map[string]string) {
	for key, value := range headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			dst.Set(key, value)
		}
	}
}

func controlPlaneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		// These headers belong to the browser target, not BAS's control plane.
		// In particular, never let a target's LocalSession overwrite or leak
		// into the BAS client's own authentication transport.
		if strings.EqualFold(strings.TrimSpace(key), "X-Vrooli-Test-Mode") || strings.EqualFold(strings.TrimSpace(key), "X-Vrooli-Test-Lease-ID") || strings.EqualFold(strings.TrimSpace(key), "Authorization") {
			continue
		}
		out[key] = value
	}
	return out
}

func (c *ConnectClient) Timeline(ctx context.Context, executionID string, headers map[string]string) (*bastimeline.ExecutionTimeline, error) {
	request := connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: executionID})
	applyRequestHeaders(request.Header(), controlPlaneHeaders(headers))
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
