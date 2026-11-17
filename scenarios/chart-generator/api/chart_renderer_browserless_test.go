package main

import (
	"context"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

type fakePixelsClient struct {
	scenario, route string
	width, height   int
	pixels          []byte
	err             error
	ctx             context.Context
	cancel          context.CancelFunc
}

func (f *fakePixelsClient) CaptureScenarioPixels(ctx context.Context, scenario, route string, width, height int) ([]byte, error) {
	f.ctx = ctx
	f.scenario, f.route, f.width, f.height = scenario, route, width, height
	if f.cancel != nil {
		f.cancel()
		return nil, ctx.Err()
	}
	return f.pixels, f.err
}

func TestCapturePNGWithBASUsesVersionedViewportPixels(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	outputDir := t.TempDir()
	chartSub := filepath.Join(outputDir, "chart_123_output")
	if err := os.MkdirAll(chartSub, 0o755); err != nil {
		t.Fatal(err)
	}
	htmlPath := filepath.Join(chartSub, "chart_123.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body>chart</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantPNG := []byte("\x89PNG\r\n\x1a\nfake-bytes")
	fake := &fakePixelsClient{pixels: wantPNG}
	renderer := NewChartRenderer(outputDir)
	renderer.resolveCapture = func(context.Context) (captureClient, error) { return fake, nil }
	outputPath := filepath.Join(chartSub, "chart_123.png")
	if err := renderer.capturePNGWithBAS(context.Background(), outputPath, htmlPath, ChartGenerationProcessorRequest{Width: 640, Height: 480}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if fake.scenario != chartScenarioSlug || fake.route != renderedURLPrefix+"chart_123_output/chart_123.html" || fake.width != 640 || fake.height != 480 {
		t.Fatalf("capture args = %s %s %dx%d", fake.scenario, fake.route, fake.width, fake.height)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(wantPNG) {
		t.Fatalf("output = %q, want %q", got, wantPNG)
	}
}

func TestCapturePNGWithBASFailsWithoutVerifiedPixels(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	outputDir := t.TempDir()
	htmlPath := filepath.Join(outputDir, "chart.html")
	if err := os.WriteFile(htmlPath, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderer := NewChartRenderer(outputDir)
	renderer.resolveCapture = func(context.Context) (captureClient, error) {
		return &fakePixelsClient{err: errors.New("bundle unavailable")}, nil
	}
	if err := renderer.capturePNGWithBAS(context.Background(), filepath.Join(outputDir, "chart.png"), htmlPath, ChartGenerationProcessorRequest{}); err == nil {
		t.Fatal("expected bundle failure so caller can use its placeholder fallback")
	}
}

func TestRenderChart_CallerCancellationStopsBASCaptureWithoutPlaceholderFallback(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	outputDir := t.TempDir()
	renderer := NewChartRenderer(outputDir)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), struct{ key string }{"request"}, "chart-job"))
	fake := &fakePixelsClient{cancel: cancel}
	renderer.capture = fake

	_, err := renderer.RenderChart(ctx, "cancelled-chart", ChartGenerationProcessorRequest{
		ChartType: "bar", Data: generateTestChartData("bar", 5), ExportFormats: []string{"png"}, Width: 640, Height: 480,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RenderChart error = %v, want context.Canceled", err)
	}
	if fake.ctx == nil || fake.ctx.Value(struct{ key string }{"request"}) != "chart-job" {
		t.Fatal("BAS capture lost caller context values")
	}
	if _, err := os.Stat(filepath.Join(outputDir, "cancelled-chart_output", "cancelled-chart.png")); !os.IsNotExist(err) {
		t.Fatalf("canceled BAS capture wrote placeholder/output; stat err=%v", err)
	}
}

func TestRenderChart_BASFailureKeepsBrowserlessPNGFallback(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	outputDir := t.TempDir()
	renderer := NewChartRenderer(outputDir)
	renderer.capture = &fakePixelsClient{err: errors.New("BAS unavailable")}
	files, err := renderer.RenderChart(context.Background(), "fallback-chart", ChartGenerationProcessorRequest{
		ChartType: "bar", Data: generateTestChartData("bar", 5), ExportFormats: []string{"png"}, Width: 320, Height: 240,
	})
	if err != nil {
		t.Fatalf("browserless fallback returned error: %v", err)
	}
	path := files["png"]
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fallback PNG: %v", err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode fallback PNG: %v", err)
	}
	if config.Width != 320 || config.Height != 240 {
		t.Fatalf("fallback raster = %dx%d, want 320x240", config.Width, config.Height)
	}
}
