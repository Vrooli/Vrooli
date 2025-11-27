//go:build integration

package replay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/export"
)

// Requires a running Playwright driver at PLAYWRIGHT_DRIVER_URL and no Browserless URL configured.
func TestPlaywrightCaptureIntegration(t *testing.T) {
	if os.Getenv("PLAYWRIGHT_DRIVER_URL") == "" {
		t.Skip("PLAYWRIGHT_DRIVER_URL not set; skipping Playwright capture integration")
	}
	os.Unsetenv("BROWSERLESS_URL")

	// Minimal export page that listens for bas:render and advances a timer.
	exportPage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!doctype html>
<html>
<body>
<div id="app">export page</div>
<script>
window.addEventListener('bas:render', (ev) => {
  // simulate playback by updating DOM
  const frames = 5;
  let i = 0;
  const tick = () => {
    document.getElementById('app').textContent = 'frame-' + i;
    i++;
    if (i < frames) {
      setTimeout(tick, 50);
    }
  };
  tick();
});
</script>
</body>
</html>
		`))
	}))
	defer exportPage.Close()

	spec := &export.ReplayMovieSpec{
		Version: "test",
		Execution: export.ExportExecutionMetadata{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Status:      "completed",
			StartedAt:   time.Now(),
		},
		Playback: export.ExportPlayback{
			FrameIntervalMs: 100,
		},
		Frames: []export.ExportFrame{{DurationMs: 200}},
		Summary: export.ExportSummary{
			TotalDurationMs: 500,
		},
		Presentation: export.ExportPresentation{
			Canvas: export.ExportDimensions{
				Width:  1280,
				Height: 720,
			},
			Viewport: export.ExportDimensions{
				Width:  1280,
				Height: 720,
			},
		},
	}

	client := newPlaywrightCaptureClient(exportPage.URL)
	resp, err := client.Capture(context.Background(), spec, 100)
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}
	if resp == nil || len(resp.Frames) == 0 {
		t.Fatalf("expected frames from Playwright capture, got %+v", resp)
	}
}
