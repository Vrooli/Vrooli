//go:build integration

package render

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/internal/testutil/integration"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
)

// Requires a running Playwright driver at PLAYWRIGHT_DRIVER_URL and no Browserless URL configured.
func TestPlaywrightCaptureIntegration(t *testing.T) {
	integration.RequireEnv(t, "PLAYWRIGHT_DRIVER_URL", "Playwright capture integration")
	os.Unsetenv("BROWSERLESS_URL")

	// Minimal export page that listens for bas:render and advances a timer.
	exportPage := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		Execution: &exportsv1.ReplayExecutionMetadata{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Status:      "completed",
			StartedAt:   time.Now(),
		},
		Playback: &exportsv1.ReplayPlayback{
			FrameIntervalMs: 100,
		},
		Frames: []*exportsv1.ReplayFrame{{DurationMs: 200}},
		Summary: &exportsv1.ReplaySummary{
			TotalDurationMs: 500,
		},
		Presentation: &exportsv1.ReplayPresentation{
			Canvas: &exportsv1.ReplayDimensions{
				Width:  1280,
				Height: 720,
			},
			Viewport: &exportsv1.ReplayDimensions{
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
