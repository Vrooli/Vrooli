package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// fakeCaptureClient records the last CaptureRequest and returns a canned
// screenshot artifact, standing in for browser-automation-studio's
// CaptureService in tests.
type fakeCaptureClient struct {
	lastReq  *capturev1.CaptureRequest
	shotPath string
	err      error
}

func (f *fakeCaptureClient) Capture(_ context.Context, req *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error) {
	f.lastReq = req.Msg
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&capturev1.CaptureResponse{
		Artifacts: []*capturev1.CaptureArtifact{
			{
				Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
				Path: f.shotPath,
			},
		},
	}), nil
}

// TestCapturePNGWithBASScreenshotsServedChart asserts the renderer asks BAS to
// screenshot the chart HTML the API server serves (via the scenario= shorthand
// + renderedURLPrefix), and copies the returned artifact bytes to the output.
func TestCapturePNGWithBASScreenshotsServedChart(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	outputDir := t.TempDir()

	// The chart HTML the API server would be serving under renderedURLPrefix.
	chartSub := filepath.Join(outputDir, "chart_123_output")
	if err := os.MkdirAll(chartSub, 0o755); err != nil {
		t.Fatalf("mkdir chart output: %v", err)
	}
	htmlPath := filepath.Join(chartSub, "chart_123.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body>chart</body></html>"), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	// The screenshot BAS produced on its (shared) filesystem.
	shotPath := filepath.Join(t.TempDir(), "screenshot.png")
	wantPNG := []byte("\x89PNG\r\n\x1a\nfake-bytes")
	if err := os.WriteFile(shotPath, wantPNG, 0o644); err != nil {
		t.Fatalf("write fake screenshot: %v", err)
	}

	fake := &fakeCaptureClient{shotPath: shotPath}
	renderer := NewChartRenderer(outputDir)
	renderer.resolveCapture = func(context.Context) (captureClient, error) { return fake, nil }

	outputPath := filepath.Join(chartSub, "chart_123.png")
	req := ChartGenerationProcessorRequest{Width: 640, Height: 480}
	if err := renderer.capturePNGWithBAS(outputPath, htmlPath, req); err != nil {
		t.Fatalf("capturePNGWithBAS: %v", err)
	}

	// The capture must target this scenario's served chart path.
	wantURL := "scenario=chart-generator,path=" + renderedURLPrefix + "chart_123_output/chart_123.html"
	if got := fake.lastReq.GetUrl(); got != wantURL {
		t.Fatalf("capture URL = %q, want %q", got, wantURL)
	}
	if types := fake.lastReq.GetCaptures(); len(types) != 1 || types[0] != capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT {
		t.Fatalf("capture types = %v, want [SCREENSHOT]", fake.lastReq.GetCaptures())
	}
	if dim := fake.lastReq.GetDimensions(); dim.GetWidth() != 640 || dim.GetHeight() != 480 {
		t.Fatalf("capture dimensions = %dx%d, want 640x480", dim.GetWidth(), dim.GetHeight())
	}

	// The artifact bytes must be copied verbatim to the output path.
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(got) != string(wantPNG) {
		t.Fatalf("output bytes = %q, want %q", got, wantPNG)
	}
}

// TestCapturePNGWithBASNoArtifactErrors asserts a screenshot-less response is a
// failure the caller can fall back on (placeholder PNG), not a silent success.
func TestCapturePNGWithBASNoArtifactErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	outputDir := t.TempDir()
	htmlPath := filepath.Join(outputDir, "chart.html")
	if err := os.WriteFile(htmlPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	renderer := NewChartRenderer(outputDir)
	renderer.resolveCapture = func(context.Context) (captureClient, error) {
		return emptyArtifactClient{}, nil
	}

	err := renderer.capturePNGWithBAS(filepath.Join(outputDir, "chart.png"), htmlPath, ChartGenerationProcessorRequest{})
	if err == nil {
		t.Fatal("expected error when BAS returns no screenshot artifact, got nil")
	}
}

// emptyArtifactClient returns a successful capture with no screenshot artifact.
type emptyArtifactClient struct{}

func (emptyArtifactClient) Capture(context.Context, *connect.Request[capturev1.CaptureRequest]) (*connect.Response[capturev1.CaptureResponse], error) {
	return connect.NewResponse(&capturev1.CaptureResponse{}), nil
}
