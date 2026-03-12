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
	// navigate + evaluate + (ceil(5000/500)+1)=11 waits + 11 screenshots
	if len(instr) != 2+(11*2) {
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
	// clamp at 120 frames => 240 instructions + 2 initial
	if len(instr) != 242 {
		t.Fatalf("expected 242 instructions, got %d", len(instr))
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

	// Second instruction must be evaluate (script injection)
	if instr[1].NodeID != "inject" {
		t.Fatalf("second instruction should be inject, got %q", instr[1].NodeID)
	}
	if instr[1].Action.GetType() != basactions.ActionType_ACTION_TYPE_EVALUATE {
		t.Fatalf("second instruction should be evaluate type, got %v", instr[1].Action.GetType())
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

	// After the 2 initial instructions, pairs should alternate: wait, screenshot
	for i := 2; i < len(instr); i += 2 {
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
	if len(instr) < 4 {
		t.Fatalf("expected at least navigate + evaluate + 1 wait/screenshot pair, got %d instructions", len(instr))
	}
}
