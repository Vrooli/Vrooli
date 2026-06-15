package browsercapture

import (
	"context"
	"errors"
	"testing"

	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/execution"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// handshakeFrame builds the handshake assert frame with the given pass state.
func handshakeFrame(passed bool, errMsg string) execution.ParsedFrame {
	return execution.ParsedFrame{
		NodeID:   nodeHandshake,
		StepType: "assert",
		Status:   stepStatus(passed),
		Success:  passed,
		Assertion: &execution.ParsedAssertion{
			NodeID: nodeHandshake,
			Passed: passed,
			Error:  errMsg,
		},
	}
}

func stepStatus(passed bool) string {
	if passed {
		return "completed"
	}
	return "failed"
}

// screenshotFrame builds the screenshot frame carrying a screenshot ref.
func screenshotFrame(url string) execution.ParsedFrame {
	return execution.ParsedFrame{
		NodeID:     nodeScreens,
		StepType:   "screenshot",
		Status:     "completed",
		Success:    true,
		Screenshot: &execution.FrameScreenshot{URL: url, ArtifactID: "shot-1"},
	}
}

// networkEntry builds a timeline entry whose aggregates carry one network-event
// artifact with the given payload values.
func networkEntry(url string, status *int, failure string) *bastimeline.TimelineEntry {
	payload := map[string]*commonv1.JsonValue{
		"url": {Kind: &commonv1.JsonValue_StringValue{StringValue: url}},
	}
	if status != nil {
		payload["status"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(*status)}}
	}
	if failure != "" {
		payload["failure"] = &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: failure}}
	}
	return &bastimeline.TimelineEntry{
		Aggregates: &bastimeline.TimelineEntryAggregates{
			Artifacts: []*bastimeline.TimelineArtifact{
				{Type: basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT, Payload: payload},
			},
		},
	}
}

func intPtr(n int) *int { return &n }

func TestCapture_HandshakeSignaledPasses(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Frames: []execution.ParsedFrame{handshakeFrame(true, ""), screenshotFrame("http://bas/shot.png")},
		Proto:  &bastimeline.ExecutionTimeline{},
	}
	cap := New(&FakeWorkflowClient{Timeline: tl, Asset: []byte("PNGBYTES")})

	res, err := cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})
	ev := res.Evidence
	if err != nil {
		t.Fatalf("Capture returned error: %v", err)
	}
	if !ev.Loaded {
		t.Fatalf("expected Loaded=true")
	}
	if !ev.Handshake.Signaled {
		t.Fatalf("expected handshake signaled")
	}
	if ev.ScreenshotRef != "http://bas/shot.png" {
		t.Fatalf("screenshot ref = %q", ev.ScreenshotRef)
	}
	if string(res.Screenshot) != "PNGBYTES" {
		t.Fatalf("expected screenshot bytes to be downloaded, got %q", res.Screenshot)
	}
	if v := evidence.Analyze(ev); !v.Passed() {
		t.Fatalf("expected pass verdict, got %q: %s", v.Status, v.Message)
	}
}

func TestCapture_HandshakeTimeoutFails(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Frames: []execution.ParsedFrame{handshakeFrame(false, "selector not found within 15000ms")},
		Proto:  &bastimeline.ExecutionTimeline{},
	}
	cap := New(&FakeWorkflowClient{Timeline: tl, WaitErr: errors.New("workflow failed")})

	res, err := cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})
	ev := res.Evidence
	if err != nil {
		t.Fatalf("Capture should not surface the (expected) wait error: %v", err)
	}
	if ev.Handshake.Signaled {
		t.Fatalf("expected handshake NOT signaled")
	}
	if !ev.Handshake.TimedOut {
		t.Fatalf("expected handshake timed out")
	}
	v := evidence.Analyze(ev)
	if v.Passed() {
		t.Fatalf("expected fail verdict for handshake timeout")
	}
}

func TestCapture_NetworkFailureFails(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Frames: []execution.ParsedFrame{handshakeFrame(true, ""), screenshotFrame("ref")},
		Proto: &bastimeline.ExecutionTimeline{
			Entries: []*bastimeline.TimelineEntry{
				networkEntry("http://localhost:3000/api/data", intPtr(500), ""),
				networkEntry("http://localhost:3000/img.png", nil, "net::ERR_CONNECTION_REFUSED"),
				networkEntry("http://localhost:3000/ok.js", intPtr(200), ""), // not a failure
			},
		},
	}
	cap := New(&FakeWorkflowClient{Timeline: tl})

	res, err := cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})
	ev := res.Evidence
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	if len(ev.Network) != 2 {
		t.Fatalf("expected 2 network failures, got %d (%+v)", len(ev.Network), ev.Network)
	}
	v := evidence.Analyze(ev)
	if v.Passed() {
		t.Fatalf("expected fail verdict for network failures")
	}
	if v.NetworkFailureCount != 2 {
		t.Fatalf("NetworkFailureCount = %d, want 2", v.NetworkFailureCount)
	}
}

func TestCapture_ConsoleErrorsCountedButPass(t *testing.T) {
	tl := &execution.ParsedTimeline{
		Frames: []execution.ParsedFrame{handshakeFrame(true, ""), screenshotFrame("ref")},
		Logs: []execution.ParsedLog{
			{Level: "error", Message: "handled error"},
			{Level: "warn", Message: "a warning"},
			{Level: "info", Message: "info line"},
		},
		Proto: &bastimeline.ExecutionTimeline{},
	}
	cap := New(&FakeWorkflowClient{Timeline: tl})

	res, err := cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})
	ev := res.Evidence
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	v := evidence.Analyze(ev)
	if !v.Passed() {
		t.Fatalf("console errors alone must not fail the verdict; got %q: %s", v.Status, v.Message)
	}
	if v.ConsoleErrorCount != 1 {
		t.Fatalf("ConsoleErrorCount = %d, want 1", v.ConsoleErrorCount)
	}
	if v.ConsoleWarningCount != 1 {
		t.Fatalf("ConsoleWarningCount = %d, want 1", v.ConsoleWarningCount)
	}
}

func TestCapture_ExecuteErrorIsNotLoaded(t *testing.T) {
	cap := New(&FakeWorkflowClient{ExecuteErr: errors.New("bas unreachable")})

	res, err := cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})
	ev := res.Evidence
	if err == nil {
		t.Fatalf("expected a transport error")
	}
	if ev.Loaded {
		t.Fatalf("expected Loaded=false on execute failure")
	}
	if evidence.Analyze(ev).Passed() {
		t.Fatalf("expected fail verdict when capture could not run")
	}
}

func TestBuildWorkflow_ShapeAndHandshakeGate(t *testing.T) {
	cap := New(&FakeWorkflowClient{Timeline: &execution.ParsedTimeline{Proto: &bastimeline.ExecutionTimeline{}}})
	_, _ = cap.Capture(context.Background(), Request{ScenarioURL: "http://localhost:3000"})

	def := cap.client.(*FakeWorkflowClient).LastDefinition
	nodes, ok := def["nodes"].([]any)
	if !ok || len(nodes) != 4 {
		t.Fatalf("expected 4 workflow nodes, got %#v", def["nodes"])
	}

	// The handshake node must be an assert on the readiness marker.
	var handshakeNode map[string]any
	for _, n := range nodes {
		nm := n.(map[string]any)
		if nm["id"] == nodeHandshake {
			handshakeNode = nm
		}
	}
	if handshakeNode == nil {
		t.Fatalf("handshake assert node missing")
	}
	action := handshakeNode["action"].(map[string]any)
	if action["type"] != "ACTION_TYPE_ASSERT" {
		t.Fatalf("handshake node is not an assert: %v", action["type"])
	}
	assert := action["assert"].(map[string]any)
	if assert["selector"] != bridgeReadyMarker {
		t.Fatalf("handshake assert selector = %v, want %v", assert["selector"], bridgeReadyMarker)
	}
	if assert["mode"] != "ASSERTION_MODE_EXISTS" {
		t.Fatalf("handshake assert mode = %v", assert["mode"])
	}
}

func TestSignalCheck_Shapes(t *testing.T) {
	cases := map[string]string{
		"IFRAME_BRIDGE_READY":           "w.IFRAME_BRIDGE_READY === true",
		"IframeBridge.ready":            "(w.IframeBridge && w.IframeBridge.ready === true)",
		"IframeBridge.getState().ready": "(w.IframeBridge && typeof w.IframeBridge.getState === 'function' && w.IframeBridge.getState().ready === true)",
	}
	for signal, want := range cases {
		if got := signalCheck(signal); got != want {
			t.Errorf("signalCheck(%q) = %q, want %q", signal, got, want)
		}
	}
}
