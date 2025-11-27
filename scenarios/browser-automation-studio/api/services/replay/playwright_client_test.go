package replay

import (
	"testing"

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
	instr := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 500)
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
	instr := buildPlaywrightCaptureInstructions("http://example.com/export", spec, 10)
	// clamp at 120 frames => 240 instructions + 2 initial
	if len(instr) != 242 {
		t.Fatalf("expected 242 instructions, got %d", len(instr))
	}
}
