package journeys

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

const DefaultScenarioBASFlowPath = "bas/cases/mobile/conformance-flow.json"

// HTTPBASClient submits the existing scenario BAS flow unchanged through the
// WorkflowsService adhoc RPC. BAS receives the already-running WebView target
// and owns only compilation/execution of web-content steps.
type HTTPBASClient struct {
	BaseURL  string
	FlowRoot string
	Client   connect.HTTPClient
	HTTP     *http.Client
}

// ValidateFlow checks a scenario-owned flow file and its BAS registry entry
// before a scenario-app journey acquires a device lease or installs an artifact.
func (c HTTPBASClient) ValidateFlow(ctx context.Context, request BASRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("BAS URL is not configured")
	}
	expectedID := strings.TrimSpace(request.Arguments["flow_reference"])
	flowPath := strings.TrimSpace(request.FlowPath)
	if flowPath == "" {
		return errors.New("scenario BAS flow path is required")
	}
	root := strings.TrimSpace(c.FlowRoot)
	if root == "" {
		root = "."
	}
	path, err := resolveScenarioBASFlow(root, request.Scenario, flowPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read scenario BAS flow %q: %w", path, err)
	}
	data, err = normalizeFlowJSON(data)
	if err != nil {
		return fmt.Errorf("normalize scenario BAS flow %q: %w", path, err)
	}
	flow := &basworkflows.WorkflowDefinitionV2{}
	if err := protojson.Unmarshal(data, flow); err != nil {
		return fmt.Errorf("decode scenario BAS flow %q: %w", path, err)
	}
	if expectedID != "" && (flow.GetMetadata().GetName() != expectedID || flow.GetMetadata().GetLabels()["flow_id"] != expectedID) {
		return fmt.Errorf("scenario BAS flow %q must declare name and flow_id %q", flowPath, expectedID)
	}
	if expectedID == UpdatePreservationFlowID {
		if flow.GetMetadata().GetExecutionMode() != basworkflows.ExecutionMode_EXECUTION_MODE_OBSERVER || flow.GetMetadata().GetLabels()["reset"] != "none" {
			return fmt.Errorf("scenario BAS preservation flow %q must declare observer execution and reset=none", flowPath)
		}
		for _, node := range flow.GetNodes() {
			if node == nil || node.GetAction() == nil {
				return fmt.Errorf("scenario BAS preservation flow %q contains a node without an action", flowPath)
			}
			switch node.GetAction().GetType() {
			case basactions.ActionType_ACTION_TYPE_NAVIGATE, basactions.ActionType_ACTION_TYPE_WAIT, basactions.ActionType_ACTION_TYPE_ASSERT, basactions.ActionType_ACTION_TYPE_SCREENSHOT, basactions.ActionType_ACTION_TYPE_EXTRACT:
			default:
				return fmt.Errorf("scenario BAS preservation flow %q contains non-observer action %s", flowPath, node.GetAction().GetType().String())
			}
		}
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root for BAS registry: %w", err)
	}
	scenarioRoot := filepath.Join(rootPath, "scenarios", request.Scenario)
	registryPath := filepath.Join(scenarioRoot, "bas", "registry.json")
	registryData, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read scenario BAS registry %q: %w", registryPath, err)
	}
	var registry struct {
		Playbooks []struct {
			File string `json:"file"`
		} `json:"playbooks"`
	}
	if err := json.Unmarshal(registryData, &registry); err != nil {
		return fmt.Errorf("decode scenario BAS registry %q: %w", registryPath, err)
	}
	registered := false
	for _, playbook := range registry.Playbooks {
		if filepath.ToSlash(filepath.Clean(playbook.File)) == filepath.ToSlash(filepath.Clean(flowPath)) {
			registered = true
			break
		}
	}
	if !registered {
		return fmt.Errorf("scenario BAS flow %q is not registered in the scenario BAS registry", flowPath)
	}
	return nil
}

func (c HTTPBASClient) Execute(ctx context.Context, request BASRequest) (BASResult, error) {
	path := strings.TrimSpace(request.FlowPath)
	root := strings.TrimSpace(c.FlowRoot)
	if root == "" {
		root = "."
	}
	if path == "" {
		path = DefaultScenarioBASFlowPath
	}
	var err error
	path, err = resolveScenarioBASFlow(root, request.Scenario, path)
	if err != nil {
		return BASResult{}, err
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
	projectRoot, err := filepath.Abs(filepath.Join(filepath.Dir(path), "..", ".."))
	if err != nil {
		return BASResult{}, fmt.Errorf("resolve BAS project root for %q: %w", path, err)
	}
	httpClient := c.Client
	if httpClient == nil {
		client := c.HTTP
		if client == nil {
			client = http.DefaultClient
		}
		httpClient = client
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	client := basconnect.NewWorkflowsServiceClient(httpClient, baseURL)
	response, err := client.ExecuteAdhocWorkflow(ctx, connect.NewRequest(&basexecution.ExecuteAdhocRequest{
		FlowDefinition:    flow,
		WaitForCompletion: true,
		Metadata:          &basexecution.ExecutionMetadata{Name: request.StepID, Description: "scenario-to-android Android WebView conformance"},
		Parameters:        androidWebViewExecutionParameters(projectRoot),
		Options:           &basexecution.ExecuteWorkflowOptions{AppTarget: &basexecution.AppTarget{TargetKind: "android-webview", TargetId: request.TargetID, CdpEndpoint: request.CDPEndpoint, RendererId: request.RendererID, RendererUrl: request.RendererURL, ScenarioName: request.Scenario, ArtifactDigest: request.Artifact.Checksum, ContextId: request.RunID, CdpTransport: "loopback-authenticated"}, ValidationContext: &basexecution.ValidationContext{ContextId: request.RunID, ScenarioName: request.Scenario, ArtifactDigest: request.Artifact.Checksum, TargetId: request.TargetID, WorkflowId: request.RunID, IsolationLeaseId: request.IsolationLeaseID}},
	}))
	if err != nil {
		return BASResult{}, fmt.Errorf("execute BAS Android WebView flow: %w", err)
	}
	executionID := strings.TrimSpace(response.Msg.GetExecutionId())
	executions := basconnect.NewExecutionsServiceClient(httpClient, baseURL)
	if response.Msg.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || strings.TrimSpace(response.Msg.GetError()) != "" {
		if executionID != "" {
			timeline, timelineErr := executions.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: executionID}))
			if timelineErr == nil {
				if nodes := closedWebViewFailureNodes(timeline.Msg); len(nodes) > 0 {
					return BASResult{ExecutionID: executionID}, &transientWebViewClosureError{ExecutionID: executionID, NodeIDs: nodes}
				}
			}
		}
		return BASResult{ExecutionID: executionID}, fmt.Errorf("BAS flow failed: execution_id=%s status=%s error=%s", executionID, response.Msg.GetStatus().String(), response.Msg.GetError())
	}
	if executionID == "" {
		return BASResult{}, fmt.Errorf("BAS flow completed without an execution ID")
	}
	timeline, err := executions.GetExecutionTimeline(ctx, connect.NewRequest(&basapi.GetExecutionTimelineRequest{ExecutionId: executionID}))
	if err != nil {
		return BASResult{}, fmt.Errorf("read BAS execution timeline %q: %w", executionID, err)
	}
	if err := verifyBASFlowCompletion(flow, timeline.Msg); err != nil {
		return BASResult{}, fmt.Errorf("BAS execution %q did not prove complete conformance: %w", executionID, err)
	}
	evidence, err := basRuntimeSummaryEvidence(executionID, flow, timeline.Msg)
	if err != nil {
		return BASResult{}, fmt.Errorf("summarize BAS execution %q: %w", executionID, err)
	}
	return BASResult{Completed: true, ExecutionID: executionID, Evidence: []deliveryramp.EvidenceReference{evidence}}, nil
}

func resolveScenarioBASFlow(root, scenario, flowPath string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository root for BAS flow: %w", err)
	}
	requestedScenario := strings.TrimSpace(scenario)
	scenario = filepath.Clean(requestedScenario)
	flowPath = filepath.Clean(strings.TrimSpace(flowPath))
	if scenario == "" || scenario == "." || requestedScenario != scenario || filepath.IsAbs(scenario) || filepath.Base(scenario) != scenario || scenario == ".." {
		return "", fmt.Errorf("BAS scenario path is invalid")
	}
	if flowPath == "" || flowPath == "." || filepath.IsAbs(flowPath) || flowPath == ".." || strings.HasPrefix(flowPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("BAS flow path must stay within the scenario project")
	}
	scenarioRoot := filepath.Join(rootPath, "scenarios", scenario)
	path := filepath.Join(scenarioRoot, flowPath)
	relative, err := filepath.Rel(scenarioRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("BAS flow path must stay within the scenario project")
	}
	return path, nil
}

func androidWebViewExecutionParameters(projectRoot string) *basexecution.ExecutionParameters {
	validationProfile := "validation"
	return &basexecution.ExecutionParameters{
		ProjectRoot: &projectRoot,
		// Android WebView screenshots can take longer than an action's selector
		// timeout to return. Keep BAS assertion, navigation, and failure evidence,
		// while skipping automatic screenshots for successful wait/input steps.
		ArtifactConfig: &basexecution.ArtifactCollectionConfig{Profile: &validationProfile},
	}
}

// transientWebViewClosureError preserves the failed BAS execution identity
// while exposing only failed node IDs, not page content or raw browser errors.
type transientWebViewClosureError struct {
	ExecutionID string
	NodeIDs     []string
}

func (e *transientWebViewClosureError) Error() string {
	return fmt.Sprintf("BAS execution %q lost its WebView context during node(s) %s", e.ExecutionID, strings.Join(e.NodeIDs, ", "))
}

func closedWebViewFailureNodes(timeline *bastimeline.ExecutionTimeline) []string {
	if timeline == nil {
		return nil
	}
	var nodes []string
	for _, entry := range timeline.GetEntries() {
		if entry == nil || entry.GetNodeId() == "" || entry.GetContext() == nil || entry.GetContext().GetSuccess() {
			continue
		}
		message := strings.ToLower(entry.GetContext().GetError())
		if strings.Contains(message, "target page, context or browser has been closed") || strings.Contains(message, "target page or context has been closed") || strings.Contains(message, "frame was detached") || strings.Contains(message, "no target with given id found") || strings.Contains(message, "no object with guid") {
			nodes = append(nodes, entry.GetNodeId())
		}
	}
	return nodes
}

func isTransientBASExecutionError(err error) bool {
	var closure *transientWebViewClosureError
	return errors.As(err, &closure)
}

func basRuntimeSummaryEvidence(executionID string, flow *basworkflows.WorkflowDefinitionV2, timeline *bastimeline.ExecutionTimeline) (deliveryramp.EvidenceReference, error) {
	type nodeResult struct {
		ID              string `json:"id"`
		Succeeded       bool   `json:"succeeded"`
		AssertionPassed bool   `json:"assertion_passed,omitempty"`
	}
	summary := struct {
		ExecutionID string       `json:"execution_id"`
		Status      string       `json:"status"`
		Progress    int32        `json:"progress"`
		Nodes       []nodeResult `json:"nodes"`
	}{ExecutionID: executionID, Status: timeline.GetStatus().String(), Progress: timeline.GetProgress(), Nodes: make([]nodeResult, 0, len(flow.GetNodes()))}
	entries := make(map[string]*bastimeline.TimelineEntry, len(timeline.GetEntries()))
	for _, entry := range timeline.GetEntries() {
		if entry != nil && entry.GetNodeId() != "" {
			entries[entry.GetNodeId()] = entry
		}
	}
	for _, node := range flow.GetNodes() {
		if node == nil {
			continue
		}
		entry := entries[node.GetId()]
		result := nodeResult{ID: node.GetId(), Succeeded: entry != nil && entry.GetContext() != nil && entry.GetContext().GetSuccess()}
		if entry != nil && entry.GetContext() != nil && entry.GetContext().GetAssertion() != nil {
			result.AssertionPassed = entry.GetContext().GetAssertion().GetSuccess()
		}
		summary.Nodes = append(summary.Nodes, result)
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return deliveryramp.EvidenceReference{}, err
	}
	digest := sha256.Sum256(encoded)
	return deliveryramp.EvidenceReference{
		ID: "bas-summary-" + executionID, Kind: "bas-runtime-summary",
		URI: "validation://bas/executions/" + executionID + "/summary", MediaType: "application/json",
		Checksum: "sha256:" + hex.EncodeToString(digest[:]), Redacted: true,
	}, nil
}

func verifyBASFlowCompletion(flow *basworkflows.WorkflowDefinitionV2, timeline *bastimeline.ExecutionTimeline) error {
	if flow == nil || timeline == nil {
		return fmt.Errorf("flow definition or execution timeline is missing")
	}
	if timeline.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
		return fmt.Errorf("timeline status is %s", timeline.GetStatus().String())
	}
	if timeline.GetProgress() != 100 {
		return fmt.Errorf("timeline progress is %d%%, want 100%%", timeline.GetProgress())
	}
	completed := make(map[string]bool, len(timeline.GetEntries()))
	assertionSeen := false
	assertionPassed := true
	for _, entry := range timeline.GetEntries() {
		if entry == nil {
			continue
		}
		nodeID := entry.GetNodeId()
		if nodeID == "" {
			continue
		}
		if ctx := entry.GetContext(); ctx != nil && ctx.GetSuccess() {
			completed[nodeID] = true
		}
		if entry.GetContext() != nil && entry.GetContext().GetAssertion() != nil {
			assertionSeen = true
			if !entry.GetContext().GetAssertion().GetSuccess() {
				assertionPassed = false
			}
		}
	}
	for _, node := range flow.GetNodes() {
		if node == nil || !completed[node.GetId()] {
			id := "<nil>"
			if node != nil {
				id = node.GetId()
			}
			return fmt.Errorf("node %q has no successful timeline entry", id)
		}
	}
	if !assertionSeen {
		return fmt.Errorf("flow has no assertion result in its execution timeline")
	}
	if !assertionPassed {
		return fmt.Errorf("flow contains an unsuccessful assertion result")
	}
	return nil
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
