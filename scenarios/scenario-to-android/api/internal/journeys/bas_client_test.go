package journeys

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

func TestVerifyBASFlowCompletionRejectsPartialCompletedExecution(t *testing.T) {
	flow := conformanceFlowFixture()
	timeline := &bastimeline.ExecutionTimeline{
		Status:   basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Progress: 20,
		Entries:  []*bastimeline.TimelineEntry{{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(true)}}},
	}

	if err := verifyBASFlowCompletion(flow, timeline); err == nil {
		t.Fatal("expected partial execution to be rejected")
	}
}

func TestVerifyBASFlowCompletionRequiresSuccessfulAssertion(t *testing.T) {
	flow := conformanceFlowFixture()
	entries := make([]*bastimeline.TimelineEntry, 0, len(flow.GetNodes()))
	for _, node := range flow.GetNodes() {
		entry := &bastimeline.TimelineEntry{NodeId: stringPtr(node.GetId()), Context: &basbase.EventContext{Success: boolPtr(true)}}
		if node.GetId() == "assert-result" {
			entry.Context.Assertion = &basbase.AssertionResult{Success: false}
		}
		entries = append(entries, entry)
	}
	timeline := &bastimeline.ExecutionTimeline{Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, Progress: 100, Entries: entries}

	if err := verifyBASFlowCompletion(flow, timeline); err == nil {
		t.Fatal("expected unsuccessful assertion to be rejected")
	}
}

func TestVerifyBASFlowCompletionAcceptsAllSuccessfulStepsAndAssertion(t *testing.T) {
	flow := conformanceFlowFixture()
	entries := make([]*bastimeline.TimelineEntry, 0, len(flow.GetNodes()))
	for _, node := range flow.GetNodes() {
		entry := &bastimeline.TimelineEntry{NodeId: stringPtr(node.GetId()), Context: &basbase.EventContext{Success: boolPtr(true)}}
		if node.GetId() == "assert-result" {
			entry.Context.Assertion = &basbase.AssertionResult{Success: true}
		}
		entries = append(entries, entry)
	}
	timeline := &bastimeline.ExecutionTimeline{Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, Progress: 100, Entries: entries}

	if err := verifyBASFlowCompletion(flow, timeline); err != nil {
		t.Fatalf("expected complete conformance proof, got %v", err)
	}
}

func TestVerifyBASFlowCompletionAcceptsAssertionNodeWithScenarioOwnedID(t *testing.T) {
	flow := &basworkflows.WorkflowDefinitionV2{Nodes: []*basworkflows.WorkflowNodeV2{{Id: "open"}, {Id: "wait-state"}, {Id: "assert-state"}}}
	entries := []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(true)}},
		{NodeId: stringPtr("wait-state"), Context: &basbase.EventContext{Success: boolPtr(true)}},
		{NodeId: stringPtr("assert-state"), Context: &basbase.EventContext{Success: boolPtr(true), Assertion: &basbase.AssertionResult{Success: true}}},
	}
	timeline := &bastimeline.ExecutionTimeline{Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, Progress: 100, Entries: entries}
	require.NoError(t, verifyBASFlowCompletion(flow, timeline))
}

func TestAndroidWebViewExecutionParametersUseValidationArtifacts(t *testing.T) {
	parameters := androidWebViewExecutionParameters("/workspace/scenario")
	if parameters.GetProjectRoot() != "/workspace/scenario" {
		t.Fatalf("project root = %q, want /workspace/scenario", parameters.GetProjectRoot())
	}
	if parameters.GetArtifactConfig().GetProfile() != "validation" {
		t.Fatalf("artifact profile = %q, want validation", parameters.GetArtifactConfig().GetProfile())
	}
}

func TestHTTPBASClientValidatesRegisteredUpdatePreservationFlowBeforeDeviceMutation(t *testing.T) {
	root := t.TempDir()
	flowPath := filepath.Join(root, "scenarios", "hello-mobile", filepath.FromSlash(UpdatePreservationFlowPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(flowPath), 0o755))
	require.NoError(t, os.WriteFile(flowPath, []byte(`{"metadata":{"name":"scenario-app-update-preservation","labels":{"flow_id":"scenario-app-update-preservation","reset":"none"},"execution_mode":"observer"},"nodes":[{"id":"wait-state","action":{"type":"ACTION_TYPE_WAIT","wait":{"selector":"body","state":"WAIT_STATE_VISIBLE"}}}]}`), 0o600))
	registryPath := filepath.Join(root, "scenarios", "hello-mobile", "bas", "registry.json")
	require.NoError(t, os.WriteFile(registryPath, []byte(`{"playbooks":[{"file":"bas/cases/mobile/update-preservation-flow.json"}]}`), 0o600))
	client := HTTPBASClient{BaseURL: "http://bas.invalid", FlowRoot: root}
	request := BASRequest{Scenario: "hello-mobile", FlowPath: UpdatePreservationFlowPath, Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID}}
	require.NoError(t, client.ValidateFlow(context.Background(), request))

	require.NoError(t, os.WriteFile(registryPath, []byte(`{"playbooks":[]}`), 0o600))
	err := client.ValidateFlow(context.Background(), request)
	require.ErrorContains(t, err, "not registered")
}

func TestHTTPBASClientRejectsMutatingPreservationFlowBeforeDeviceMutation(t *testing.T) {
	root := t.TempDir()
	flowPath := filepath.Join(root, "scenarios", "hello-mobile", filepath.FromSlash(UpdatePreservationFlowPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(flowPath), 0o755))
	require.NoError(t, os.WriteFile(flowPath, []byte(`{"metadata":{"name":"scenario-app-update-preservation","labels":{"flow_id":"scenario-app-update-preservation","reset":"none"},"execution_mode":"observer"},"nodes":[{"id":"input","action":{"type":"ACTION_TYPE_INPUT","input":{"selector":"body","value":"would mutate"}}}]}`), 0o600))
	registryPath := filepath.Join(root, "scenarios", "hello-mobile", "bas", "registry.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0o755))
	require.NoError(t, os.WriteFile(registryPath, []byte(`{"playbooks":[{"file":"bas/cases/mobile/update-preservation-flow.json"}]}`), 0o600))
	client := HTTPBASClient{BaseURL: "http://bas.invalid", FlowRoot: root}
	request := BASRequest{Scenario: "hello-mobile", FlowPath: UpdatePreservationFlowPath, Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID}}
	require.ErrorContains(t, client.ValidateFlow(context.Background(), request), "non-observer action")
}

func TestHTTPBASClientRejectsWrongUpdatePreservationFlowIdentity(t *testing.T) {
	root := t.TempDir()
	flowPath := filepath.Join(root, "scenarios", "hello-mobile", filepath.FromSlash(UpdatePreservationFlowPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(flowPath), 0o755))
	require.NoError(t, os.WriteFile(flowPath, []byte(`{"metadata":{"name":"other-flow","labels":{"flow_id":"other-flow"}},"nodes":[]}`), 0o600))
	registryPath := filepath.Join(root, "scenarios", "hello-mobile", "bas", "registry.json")
	require.NoError(t, os.WriteFile(registryPath, []byte(`{"playbooks":[{"file":"bas/cases/mobile/update-preservation-flow.json"}]}`), 0o600))
	client := HTTPBASClient{BaseURL: "http://bas.invalid", FlowRoot: root}
	request := BASRequest{Scenario: "hello-mobile", FlowPath: UpdatePreservationFlowPath, Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID}}
	err := client.ValidateFlow(context.Background(), request)
	require.ErrorContains(t, err, "must declare name and flow_id")
}

func TestClosedWebViewFailureNodesReadRetryCauseWithoutExposingPageError(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("page.waitForLoadState: Target page, context or browser has been closed")}},
		{NodeId: stringPtr("wait-title"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("navigation required before wait-title")}},
	}}
	nodes := closedWebViewFailureNodes(timeline)
	if len(nodes) != 1 || nodes[0] != "open" {
		t.Fatalf("closed WebView failure nodes = %v, want [open]", nodes)
	}
	err := &transientWebViewClosureError{ExecutionID: "bas-run-1", NodeIDs: nodes}
	if !isTransientBASExecutionError(err) || !isTransientWebViewClosure(err) {
		t.Fatalf("typed closed-WebView failure was not classified as retryable: %v", err)
	}
	if strings.Contains(err.Error(), "page.waitForLoadState") {
		t.Fatalf("typed retry error exposed raw browser/page detail: %v", err)
	}
}

func TestClosedWebViewFailureNodesDoesNotRetryConfigurationErrors(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("wait-title"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("navigation required before step wait-title: no prior navigate step")}},
	}}
	if nodes := closedWebViewFailureNodes(timeline); len(nodes) != 0 {
		t.Fatalf("deterministic navigation configuration failure was classified as a WebView closure: %v", nodes)
	}
}

func TestClosedWebViewFailureNodesClassifiesDetachedAndroidFrameAsTransient(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("Frame not found, url=page.goto: net::ERR_ABORTED; maybe frame was detached?")}},
	}}
	if nodes := closedWebViewFailureNodes(timeline); len(nodes) != 1 || nodes[0] != "open" {
		t.Fatalf("detached WebView frame was not classified as transient: %#v", nodes)
	}
}

func TestClosedWebViewFailureNodesClassifiesMissingCDPTargetAsTransient(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("Instruction may have taken effect: browserContext.newCDPSession: Protocol error (Target.attachToTarget): No target with given id found")}},
	}}
	if nodes := closedWebViewFailureNodes(timeline); len(nodes) != 1 || nodes[0] != "open" {
		t.Fatalf("missing WebView CDP target was not classified as transient: %#v", nodes)
	}
}

func TestClosedWebViewFailureNodesClassifiesStaleCDPPageObjectAsTransient(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{
		{NodeId: stringPtr("open"), Context: &basbase.EventContext{Success: boolPtr(false), Error: stringPtr("Instruction may have taken effect: browserContext.newCDPSession: page: no object with guid page@e28b932b7e77336f0ad8c2ca116eeee9")}},
	}}
	if nodes := closedWebViewFailureNodes(timeline); len(nodes) != 1 || nodes[0] != "open" {
		t.Fatalf("stale WebView CDP page object was not classified as transient: %#v", nodes)
	}
}

func TestResolveScenarioBASFlowStaysWithinScenarioRoot(t *testing.T) {
	root := t.TempDir()
	path, err := resolveScenarioBASFlow(root, "hello-mobile", "bas/cases/mobile/update-preservation-flow.json")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "scenarios", "hello-mobile", "bas", "cases", "mobile", "update-preservation-flow.json")
	if path != want {
		t.Fatalf("resolved BAS flow path %q, want %q", path, want)
	}
	if _, err := resolveScenarioBASFlow(root, "hello-mobile", "../../device-control/private.json"); err == nil {
		t.Fatal("BAS flow paths must not escape the scenario project")
	}
	if _, err := resolveScenarioBASFlow(root, "../device-control", "bas/cases/mobile/conformance-flow.json"); err == nil {
		t.Fatal("scenario identity must not escape the scenario directory")
	}
	if _, err := resolveScenarioBASFlow(root, "hello-mobile/../device-control", "bas/cases/mobile/conformance-flow.json"); err == nil {
		t.Fatal("scenario identity containing path traversal must be rejected")
	}
}

func conformanceFlowFixture() *basworkflows.WorkflowDefinitionV2 {
	return &basworkflows.WorkflowDefinitionV2{Nodes: []*basworkflows.WorkflowNodeV2{
		{Id: "open"}, {Id: "wait-title"}, {Id: "input"}, {Id: "submit"}, {Id: "assert-result"},
	}}
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }

func TestHTTPBASClientExecutesLiveAndroidFlow(t *testing.T) {
	if os.Getenv("ANDROID_BAS_LIVE") != "1" {
		t.Skip("set ANDROID_BAS_LIVE=1 with BAS_URL, ANDROID_CDP_ENDPOINT, ANDROID_RENDERER_ID, and ANDROID_LEASE_ID for live Android WebView validation")
	}
	client := HTTPBASClient{BaseURL: os.Getenv("BAS_URL"), FlowRoot: os.Getenv("VROOLI_ROOT")}
	result, err := client.Execute(context.Background(), BASRequest{
		TargetID: os.Getenv("ANDROID_TARGET_ID"), Scenario: "hello-mobile", StepID: "hello-mobile-smoke",
		RunID: "android-live-bas", IsolationLeaseID: os.Getenv("ANDROID_LEASE_ID"), CDPEndpoint: os.Getenv("ANDROID_CDP_ENDPOINT"), RendererID: os.Getenv("ANDROID_RENDERER_ID"),
		Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:live", Checksum: "sha256:live"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed {
		t.Fatal("live BAS flow did not complete")
	}
}
