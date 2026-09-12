package journeys

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// HTTPBASClient submits the existing scenario BAS flow unchanged through the
// WorkflowsService adhoc RPC. BAS receives the already-running WebView target
// and owns only compilation/execution of web-content steps.
type HTTPBASClient struct {
	BaseURL  string
	FlowRoot string
	Client   connect.HTTPClient
	HTTP     *http.Client
}

func (c HTTPBASClient) Execute(ctx context.Context, request BASRequest) (BASResult, error) {
	path := strings.TrimSpace(request.FlowPath)
	if path == "" {
		root := strings.TrimSpace(c.FlowRoot)
		if root == "" {
			root = "."
		}
		path = filepath.Join(root, "scenarios", request.Scenario, "bas", "cases", "mobile", "conformance-flow.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return BASResult{}, fmt.Errorf("read BAS flow %q: %w", path, err)
	}
	data, err = normalizeFlowJSON(data)
	if err != nil {
		return BASResult{}, fmt.Errorf("normalize BAS flow %q: %w", path, err)
	}
	flow := &basworkflows.WorkflowDefinitionV2{}
	if err := protojson.Unmarshal(data, flow); err != nil {
		return BASResult{}, fmt.Errorf("decode BAS flow %q: %w", path, err)
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return BASResult{}, fmt.Errorf("BAS URL is not configured")
	}
	if strings.TrimSpace(request.RendererURL) == "" {
		return BASResult{}, fmt.Errorf("Android WebView renderer URL is not configured")
	}
	httpClient := c.Client
	if httpClient == nil {
		client := c.HTTP
		if client == nil {
			client = http.DefaultClient
		}
		httpClient = client
	}
	client := basconnect.NewWorkflowsServiceClient(httpClient, strings.TrimRight(c.BaseURL, "/"))
	response, err := client.ExecuteAdhocWorkflow(ctx, connect.NewRequest(&basexecution.ExecuteAdhocRequest{
		FlowDefinition:    flow,
		WaitForCompletion: true,
		Metadata:          &basexecution.ExecutionMetadata{Name: request.StepID, Description: "scenario-to-android Android WebView conformance"},
		Options:           &basexecution.ExecuteWorkflowOptions{AppTarget: &basexecution.AppTarget{TargetKind: "android-webview", TargetId: request.TargetID, CdpEndpoint: request.CDPEndpoint, RendererId: request.RendererID, RendererUrl: request.RendererURL, ScenarioName: request.Scenario, ArtifactDigest: request.Artifact.ImmutableRef, ContextId: request.RunID, CdpTransport: "loopback-authenticated"}, ValidationContext: &basexecution.ValidationContext{ContextId: request.RunID, ScenarioName: request.Scenario, ArtifactDigest: request.Artifact.ImmutableRef, TargetId: request.TargetID, WorkflowId: request.RunID, IsolationLeaseId: request.IsolationLeaseID}},
	}))
	if err != nil {
		return BASResult{}, fmt.Errorf("execute BAS Android WebView flow: %w", err)
	}
	if response.Msg.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || strings.TrimSpace(response.Msg.GetError()) != "" {
		return BASResult{}, fmt.Errorf("BAS flow failed: status=%s error=%s", response.Msg.GetStatus().String(), response.Msg.GetError())
	}
	return BASResult{Completed: true}, nil
}

func normalizeFlowJSON(data []byte) ([]byte, error) {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if metadata, ok := document["metadata"].(map[string]any); ok {
		if mode, ok := metadata["execution_mode"].(string); ok {
			metadata["execution_mode"] = normalizedExecutionMode(mode)
		}
		if mode, ok := metadata["executionMode"].(string); ok {
			metadata["executionMode"] = normalizedExecutionMode(mode)
		}
		delete(metadata, "reset")
	}
	return json.Marshal(document)
}

func normalizedExecutionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "observer":
		return "EXECUTION_MODE_OBSERVER"
	case "mutating":
		return "EXECUTION_MODE_MUTATING"
	case "destructive":
		return "EXECUTION_MODE_DESTRUCTIVE"
	default:
		return mode
	}
}

var _ BASClient = HTTPBASClient{}
