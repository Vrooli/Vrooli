package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakePixelsClient struct {
	scenario, route string
	width, height   int
	pixels          []byte
	err             error
}

func (f *fakePixelsClient) CaptureScenarioPixels(_ context.Context, scenario, route string, width, height int) ([]byte, error) {
	f.scenario, f.route, f.width, f.height = scenario, route, width, height
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
	if err := renderer.capturePNGWithBAS(outputPath, htmlPath, ChartGenerationProcessorRequest{Width: 640, Height: 480}); err != nil {
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
	if err := renderer.capturePNGWithBAS(filepath.Join(outputDir, "chart.png"), htmlPath, ChartGenerationProcessorRequest{}); err == nil {
		t.Fatal("expected bundle failure so caller can use its placeholder fallback")
	}
}
