package replay

import (
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/services/export"
)

func TestEstimateReplayRenderTimeoutBounds(t *testing.T) {
	smallSpec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{
			FrameIntervalMs: 40,
			TotalFrames:     5,
		},
		Summary: export.ExportSummary{TotalDurationMs: 200},
		Frames:  []export.ExportFrame{{Index: 0, DurationMs: 200}},
	}

	duration := EstimateReplayRenderTimeout(smallSpec)
	if duration < 3*time.Minute {
		t.Fatalf("expected minimum timeout of 3 minutes, got %s", duration)
	}
	if duration > 15*time.Minute {
		t.Fatalf("expected timeout to remain within 15 minutes, got %s", duration)
	}

	hugeSpec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{
			FrameIntervalMs: 20,
			TotalFrames:     120000,
		},
		Summary: export.ExportSummary{TotalDurationMs: 2400000},
		Frames:  make([]export.ExportFrame, 0),
	}

	bigDuration := EstimateReplayRenderTimeout(hugeSpec)
	if bigDuration > 15*time.Minute {
		t.Fatalf("expected timeout capped at 15 minutes, got %s", bigDuration)
	}
	if bigDuration < 3*time.Minute {
		t.Fatalf("expected timeout to exceed minimum for large specs, got %s", bigDuration)
	}
}

func ptr[T any](v T) *T {
	return &v
}
