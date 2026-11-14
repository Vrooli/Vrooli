package render

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

func TestBuildPlaywrightCaptureInstructions_FrameCount(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary: &exportsv1.ReplaySummary{
			TotalDurationMs: 5000,
		},
		Playback: &exportsv1.ReplayPlayback{
			FrameIntervalMs: 500,
		},
		Frames: []*exportsv1.ReplayFrame{{DurationMs: 500}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// navigate + stabilize-viewport + inject + (ceil(5000/500)+1)=11 waits + 11 screenshots
	if len(instr) != 3+(11*2) {
		t.Fatalf("unexpected instruction count: %d", len(instr))
	}
}

func TestBuildPlaywrightCaptureInstructions_RejectsBudgetWithoutTruncating(t *testing.T) {
	spec := &ReplayMovieSpec{Summary: &exportsv1.ReplaySummary{TotalDurationMs: 999999}, Frames: []*exportsv1.ReplayFrame{{DurationMs: 10}}}
	_, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 10)
	if err == nil || !strings.Contains(err.Error(), "timeline will not be truncated") {
		t.Fatalf("expected actionable frame budget refusal, got %v", err)
	}
}

func TestBuildPlaywrightCaptureInstructions_CoversEndBeyondOld120FrameLimit(t *testing.T) {
	spec := &ReplayMovieSpec{Summary: &exportsv1.ReplaySummary{TotalDurationMs: 8446}, Frames: []*exportsv1.ReplayFrame{{DurationMs: 8446}}}
	instructions, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(instructions) != 3+213*2 {
		t.Fatalf("incomplete timeline: %d instructions", len(instructions))
	}
	expression := instructions[len(instructions)-2].Action.GetEvaluate().GetExpression()
	if !strings.Contains(expression, "seekTo(8446)") {
		t.Fatalf("missing timeline end: %s", expression)
	}
}

func TestBuildPlaywrightCaptureInstructions_ScreenshotIsViewportOnly(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 2000},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 1000}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find all screenshot instructions and verify fullPage is false.
	// Using fullPage:true causes inconsistent frame dimensions when the
	// export page has variable-height content, leading to bottom-of-frame
	// flickering in assembled videos.
	screenshotCount := 0
	for _, inst := range instr {
		if !strings.HasPrefix(inst.NodeID, "screenshot-") {
			continue
		}
		screenshotCount++

		if inst.Action == nil {
			t.Fatalf("screenshot instruction %q has nil action", inst.NodeID)
		}
		if inst.Action.GetType() != basactions.ActionType_ACTION_TYPE_SCREENSHOT {
			t.Fatalf("screenshot instruction %q has wrong action type: %v", inst.NodeID, inst.Action.GetType())
		}

		params := inst.Action.GetScreenshot()
		if params == nil {
			t.Fatalf("screenshot instruction %q has nil screenshot params", inst.NodeID)
		}
		if params.FullPage == nil {
			t.Fatalf("screenshot instruction %q has nil FullPage (must be explicitly false)", inst.NodeID)
		}
		if *params.FullPage {
			t.Fatalf("screenshot instruction %q has fullPage=true; must be false to prevent inconsistent frame dimensions", inst.NodeID)
		}
	}

	if screenshotCount == 0 {
		t.Fatal("no screenshot instructions found")
	}
}

func TestBuildPlaywrightCaptureInstructions_InitialInstructions(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 1000},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 1000}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://localhost:3000/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First instruction must be navigate
	if instr[0].NodeID != "navigate" {
		t.Fatalf("first instruction should be navigate, got %q", instr[0].NodeID)
	}
	if instr[0].Action.GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		t.Fatalf("first instruction should be navigate type, got %v", instr[0].Action.GetType())
	}

	// Second instruction must be stabilize-viewport (CSS injection to prevent
	// scrollbar-induced layout shifts that cause bottom-of-frame flickering)
	if instr[1].NodeID != "stabilize-viewport" {
		t.Fatalf("second instruction should be stabilize-viewport, got %q", instr[1].NodeID)
	}
	if instr[1].Action.GetType() != basactions.ActionType_ACTION_TYPE_EVALUATE {
		t.Fatalf("stabilize-viewport should be evaluate type, got %v", instr[1].Action.GetType())
	}

	// Third instruction must be inject (render spec)
	if instr[2].NodeID != "inject" {
		t.Fatalf("third instruction should be inject, got %q", instr[2].NodeID)
	}
	if instr[2].Action.GetType() != basactions.ActionType_ACTION_TYPE_EVALUATE {
		t.Fatalf("inject should be evaluate type, got %v", instr[2].Action.GetType())
	}
}

func TestBuildPlaywrightCaptureInstructions_WaitScreenshotAlternation(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 3000},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 1000}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After the 3 initial instructions, pairs should alternate: wait, screenshot
	for i := 3; i < len(instr); i += 2 {
		if !strings.HasPrefix(instr[i].NodeID, "wait-") {
			t.Fatalf("instruction %d should be wait, got %q", i, instr[i].NodeID)
		}
		if i+1 < len(instr) && !strings.HasPrefix(instr[i+1].NodeID, "screenshot-") {
			t.Fatalf("instruction %d should be screenshot, got %q", i+1, instr[i+1].NodeID)
		}
	}
}

func TestBuildPlaywrightCaptureInstructions_DefaultDuration(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 0},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 0},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 0}},
	}
	// With zero duration and zero interval, should use defaults and not panic
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instr) < 5 {
		t.Fatalf("expected at least navigate + stabilize + inject + 1 wait/screenshot pair, got %d instructions", len(instr))
	}
}

func TestBuildPlaywrightCaptureInstructions_NegativeCaptureInterval(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 5000},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 100},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 100}},
	}
	// Negative interval should be treated as default (1000ms), not panic or error
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, -500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instr) < 5 {
		t.Fatalf("expected at least navigate + stabilize + inject + 1 wait/screenshot pair, got %d instructions", len(instr))
	}
}

func TestBuildPlaywrightCaptureInstructions_FallbackDuration(t *testing.T) {
	// When TotalDurationMs is 0 but DurationMs is set, should use DurationMs
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 0},
		Playback: &exportsv1.ReplayPlayback{DurationMs: 3000, FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 1000}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ceil(3000/1000)+1 = 4 screenshot pairs + 3 initial = 11
	if len(instr) != 11 {
		t.Fatalf("expected 11 instructions with DurationMs fallback, got %d", len(instr))
	}
}

// TestBuildPlaywrightCaptureInstructions_ViewportStabilization verifies that
// the capture instructions inject CSS to force a permanent scrollbar BEFORE
// the render spec is injected. This prevents scrollbar appearance/disappearance
// during playback from changing the viewport width, which causes content shifts
// that manifest as bottom-of-frame flickering in assembled videos.
func TestBuildPlaywrightCaptureInstructions_ViewportStabilization(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 1000},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 1000}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the stabilize-viewport instruction
	found := false
	for _, inst := range instr {
		if inst.NodeID != "stabilize-viewport" {
			continue
		}
		found = true

		if inst.Action == nil {
			t.Fatal("stabilize-viewport has nil action")
		}
		params := inst.Action.GetEvaluate()
		if params == nil {
			t.Fatal("stabilize-viewport has nil evaluate params")
		}

		script := params.GetExpression()

		// The export is a fixed canvas: neither scrollbar painting nor root
		// reflow belongs in the encoded result.
		if !strings.Contains(script, "html,body{overflow:hidden") || strings.Contains(script, "overflow-y:scroll") {
			t.Fatalf("fixed export canvas must hide root overflow: %s", script)
		}

		break
	}

	if !found {
		t.Fatal("no stabilize-viewport instruction found")
	}

	// Stabilize must come BEFORE inject (render spec)
	stabilizeIdx := -1
	injectIdx := -1
	for i, inst := range instr {
		if inst.NodeID == "stabilize-viewport" {
			stabilizeIdx = i
		}
		if inst.NodeID == "inject" {
			injectIdx = i
		}
	}
	if stabilizeIdx >= injectIdx {
		t.Fatalf("stabilize-viewport (idx %d) must come before inject (idx %d)", stabilizeIdx, injectIdx)
	}
}

func TestBuildPlaywrightCaptureInstructions_SingleFrame(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  &exportsv1.ReplaySummary{TotalDurationMs: 100},
		Playback: &exportsv1.ReplayPlayback{FrameIntervalMs: 1000},
		Frames:   []*exportsv1.ReplayFrame{{DurationMs: 100}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Very short duration: ceil(100/1000)+1 = 2 screenshot pairs + 3 initial = 7
	if len(instr) < 5 {
		t.Fatalf("expected at least 5 instructions for minimal capture, got %d", len(instr))
	}
}

// The export page accepts bas:spec:set and exposes basExport.ready/seekTo.
// A video of its error screen must never count as a successful capture.
func TestBuildPlaywrightCaptureInstructions_ComposerProtocol(t *testing.T) {
	spec := &ReplayMovieSpec{Summary: &exportsv1.ReplaySummary{TotalDurationMs: 200}, Frames: []*exportsv1.ReplayFrame{{DurationMs: 200}}}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(instr[0].Action.GetNavigate().GetUrl(), "mode=capture") {
		t.Fatal("composer controller is only exposed in export mode")
	}
	inject := instr[2].Action.GetEvaluate().GetExpression()
	for _, contract := range []string{"bas:spec:set", "basExport.ready", "basExport.error"} {
		if !strings.Contains(inject, contract) {
			t.Errorf("capture must honor composer contract %s", contract)
		}
	}
	for i := 3; i < len(instr); i += 2 {
		if instr[i].Action.GetType() != basactions.ActionType_ACTION_TYPE_EVALUATE || !strings.Contains(instr[i].Action.GetEvaluate().GetExpression(), "basExport.seekTo(") {
			t.Fatal("each captured frame must await a deterministic composer seek")
		}
	}
}

func TestCaptureRecorderRefusesMemoryOverflowBeforeRetainingFrame(t *testing.T) {
	recorder := &inMemoryCaptureRecorder{capturedBytes: 128 * 1024 * 1024}
	_, err := recorder.RecordStepOutcome(context.Background(), autocontracts.ExecutionPlan{}, autocontracts.StepOutcome{Screenshot: &autocontracts.Screenshot{Data: []byte{1}}})
	if err == nil || len(recorder.outcomes) != 0 {
		t.Fatal("over-budget frame retained")
	}
}

// Execute the emitted browser script against a deterministic public-API fake.
// This validates message-delivery ordering without starting a browser or renderer.
func TestCaptureComposerWaitsForRequestedReplayIdentity(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required to evaluate the emitted JavaScript contract")
	}
	spec := &ReplayMovieSpec{
		Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: "requested-execution"},
		Summary:   &exportsv1.ReplaySummary{TotalDurationMs: 200},
		Frames:    []*exportsv1.ReplayFrame{{DurationMs: 200}},
	}
	instructions, err := buildPlaywrightCaptureInstructions("http://example.test/export", spec, 100)
	if err != nil {
		t.Fatal(err)
	}
	expression := instructions[2].Action.GetEvaluate().GetExpression()
	nextInstructions, err := buildPlaywrightCaptureInstructions("http://example.test/export", spec, 100)
	if err != nil {
		t.Fatal(err)
	}
	nextExpression := nextInstructions[2].Action.GetEvaluate().GetExpression()
	for _, test := range []struct {
		name, wantError string
		minimumTicks    int
	}{
		{name: "old-ready", minimumTicks: 1},
		{name: "same-execution", minimumTicks: 1},
		{name: "consecutive-same-execution", minimumTicks: 2},
		{name: "stale-token", minimumTicks: 2},
		{name: "wrong-token", wantError: "Replay composer did not become ready", minimumTicks: 15},
		{name: "matching-but-not-ready", minimumTicks: 2},
		{name: "wrong-identity", wantError: "Replay composer did not become ready", minimumTicks: 15},
		{name: "current-error", wantError: "current composer failure", minimumTicks: 1},
		{name: "old-error", minimumTicks: 1},
		{name: "stale-request-error", minimumTicks: 2},
		{name: "current-request-error", wantError: "Invalid replay spec", minimumTicks: 1},
		{name: "unavailable", wantError: "Replay composer unavailable", minimumTicks: 15},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, err := json.Marshal(map[string]string{"expression": expression, "nextExpression": nextExpression, "mode": test.name})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, node, "-e", composerIdentityHarness)
			command.Stdin = bytes.NewReader(input)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("JavaScript harness: %v: %s", err, output)
			}
			var got struct {
				Error  string `json:"error"`
				Ticks  int    `json:"ticks"`
				Pauses int    `json:"pauses"`
				Seeks  int    `json:"seeks"`
				Ready  bool   `json:"ready"`
			}
			if err := json.Unmarshal(output, &got); err != nil {
				t.Fatalf("harness witness: %v: %s", err, output)
			}
			if got.Error != test.wantError || got.Ticks < test.minimumTicks {
				t.Fatalf("captured stale/unready replay or lost bounded failure: %+v", got)
			}
			if test.wantError == "" {
				expectedCalls := 1
				if test.name == "consecutive-same-execution" {
					expectedCalls = 2
				}
				if !got.Ready || got.Pauses != expectedCalls || got.Seeks != expectedCalls {
					t.Fatalf("matching ready replay was not initialized: %+v", got)
				}
			} else if got.Ready || got.Pauses != 0 || got.Seeks != 0 {
				t.Fatalf("failed composer was captured: %+v", got)
			}
		})
	}
}

const composerIdentityHarness = `
const { expression, nextExpression, mode } = JSON.parse(require('node:fs').readFileSync(0, 'utf8'));
let now = 0, ticks = 0, pauses = 0, seeks = 0, posts = 0, posted;
const state = { id: mode === 'same-execution' ? 'requested-execution' : 'old-execution', token: 'previous-token', applied: 0, ready: true, error: null };
if (mode === 'old-error' || mode === 'stale-request-error') { state.ready = false; state.error = 'previous composer failure'; }
if (mode === 'stale-request-error') state.requestError = { requestToken: 'previous-token', message: 'previous composer failure' };
Date.now = () => now;
const assertCurrent = () => {
  if (!posted || state.applied !== posts || state.id !== posted.spec.execution.executionId || !posted.requestToken || state.token !== posted.requestToken || !state.ready) throw Error('captured stale replay');
};
global.window = {
  location: { origin: 'http://example.test' },
  basExport: {
    get ready() { return state.ready; },
    get error() { return state.error; },
    getMetadata: () => ({ specId: state.id, requestToken: state.token, requestError: state.requestError }),
    pause: () => { assertCurrent(); pauses++; },
    seekTo: async () => { assertCurrent(); seeks++; },
  },
  postMessage: message => { posted = message; posts++; },
};
if (mode === 'unavailable') window.basExport = undefined;
global.document = { fonts: { ready: Promise.resolve() } };
global.setTimeout = callback => {
  ticks++; now += 1000;
  if (posted) {
    state.applied = posts;
    if (mode === 'current-error') { state.error = 'current composer failure'; state.token = posted.requestToken; state.ready = false; }
    else if (mode === 'current-request-error') { state.error = 'Invalid replay spec'; state.requestError = { requestToken: posted.requestToken, message: state.error }; state.ready = false; }
    else if (mode === 'stale-request-error' && ticks < 2) { /* prior error is not a failure of the new request */ }
    else {
      state.error = null;
      state.requestError = undefined;
      state.id = mode === 'wrong-identity' ? 'old-execution' : posted.spec.execution.executionId;
      state.token = mode === 'wrong-token' || (mode === 'stale-token' && ticks < 2) ? 'stale-token' : posted.requestToken;
      state.ready = mode !== 'matching-but-not-ready' || ticks >= 2;
    }
  }
  callback();
};
(async () => {
  try { let result = await eval(expression); if (mode === 'consecutive-same-execution') result = await eval(nextExpression); console.log(JSON.stringify({ ready: result.ready, ticks, pauses, seeks, error: '' })); }
  catch (error) { console.log(JSON.stringify({ ready: false, ticks, pauses, seeks, error: error.message })); }
})();
`
