package render

import (
	"strings"
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"

	"github.com/vrooli/browser-automation-studio/services/export"
)

func TestBuildPlaywrightCaptureInstructions_FrameCount(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary: export.ExportSummary{
			TotalDurationMs: 5000,
		},
		Playback: export.ExportPlayback{
			FrameIntervalMs: 500,
		},
		Frames: []export.ExportFrame{{DurationMs: 500}},
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

func TestBuildPlaywrightCaptureInstructions_ClampCount(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary: export.ExportSummary{
			TotalDurationMs: 999999,
		},
		Playback: export.ExportPlayback{
			FrameIntervalMs: 10,
		},
		Frames: []export.ExportFrame{{DurationMs: 10}},
	}
	instr, err := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// clamp at 120 frames => 240 instructions + 3 initial
	if len(instr) != 243 {
		t.Fatalf("expected 243 instructions, got %d", len(instr))
	}
}

func TestBuildPlaywrightCaptureInstructions_ScreenshotIsViewportOnly(t *testing.T) {
	spec := &ReplayMovieSpec{
		Summary:  export.ExportSummary{TotalDurationMs: 2000},
		Playback: export.ExportPlayback{FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 1000}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 1000},
		Playback: export.ExportPlayback{FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 1000}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 3000},
		Playback: export.ExportPlayback{FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 1000}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 0},
		Playback: export.ExportPlayback{FrameIntervalMs: 0},
		Frames:   []export.ExportFrame{{DurationMs: 0}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 5000},
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 0},
		Playback: export.ExportPlayback{DurationMs: 3000, FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 1000}},
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
		Summary:  export.ExportSummary{TotalDurationMs: 1000},
		Playback: export.ExportPlayback{FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 1000}},
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

		// Must force permanent vertical scrollbar
		if !strings.Contains(script, "overflow-y:scroll") {
			t.Fatalf("stabilization script must force overflow-y:scroll, got: %s", script)
		}

		// Must hide horizontal overflow to prevent horizontal layout shifts
		if !strings.Contains(script, "overflow-x:hidden") {
			t.Fatalf("stabilization script must hide overflow-x, got: %s", script)
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
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Playback: export.ExportPlayback{FrameIntervalMs: 1000},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
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
