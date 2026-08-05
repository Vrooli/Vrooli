package smoketest

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"scenario-to-desktop-api/procmetrics"
)

func TestDesktopJourney_XdotoolAbsentIsDegradedAndNeverPasses(t *testing.T) {
	detector := procmetrics.NewXdotoolDetector(func(context.Context, []string, string, ...string) ([]byte, error) {
		return nil, errors.New("command not found")
	}, slog.Default())
	service := &DefaultService{windowDetector: detector}
	result := service.runDesktopJourney(context.Background(), "smoke-1", "demo", "linux", recordingState{
		captureID:     "recording-1",
		displayID:     ":99",
		windowManager: "openbox",
		titlebar:      true,
	})

	if result.Disposition != journeyDegraded {
		t.Fatalf("disposition = %q, want degraded", result.Disposition)
	}
	if result.DegradedReason != "xdotool_unavailable" {
		t.Fatalf("degraded reason = %q, want xdotool_unavailable", result.DegradedReason)
	}
	if result.Disposition == journeyPass {
		t.Fatal("degraded journey must never pass")
	}
	if !result.RecordingStartedBeforeLaunch {
		t.Fatal("journey should retain recording-before-launch provenance")
	}
}

func TestJourneyRequiresScreenshotBeforeAndAfterEveryInteraction(t *testing.T) {
	complete := []JourneyStep{{Action: "pointer_click", BeforeCaptureID: "before", AfterCaptureID: "after"}}
	if !journeyHasScreenshotPairs(complete) {
		t.Fatal("complete interaction screenshot pair should be accepted")
	}
	incomplete := []JourneyStep{{Action: "pointer_click", BeforeCaptureID: "before"}}
	if journeyHasScreenshotPairs(incomplete) {
		t.Fatal("missing after screenshot should be rejected")
	}
}
