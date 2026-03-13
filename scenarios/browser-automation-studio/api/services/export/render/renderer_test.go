package render

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
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

// ---------------------------------------------------------------------------
// Render validation tests
// ---------------------------------------------------------------------------

func TestRender_NilSpec(t *testing.T) {
	r := newTestRenderer(t)
	_, err := r.Render(t.Context(), nil, RenderFormatMP4, "test.mp4")
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
}

func TestRender_EmptyFrames(t *testing.T) {
	r := newTestRenderer(t)
	spec := &ReplayMovieSpec{
		Frames: []export.ExportFrame{},
	}
	_, err := r.Render(t.Context(), spec, RenderFormatMP4, "test.mp4")
	if err == nil {
		t.Fatal("expected error for empty frames")
	}
}

func TestRender_UnsupportedFormat(t *testing.T) {
	r := newTestRenderer(t)
	spec := &ReplayMovieSpec{
		Frames: []export.ExportFrame{{DurationMs: 100}},
	}
	_, err := r.Render(t.Context(), spec, "webm", "test.webm")
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

// ---------------------------------------------------------------------------
// renderCapture: frame writing with gaps
// ---------------------------------------------------------------------------

// stubCaptureClient is a test double for replayCaptureClient that returns
// a pre-built captureResponse.
type stubCaptureClient struct {
	response *captureResponse
	err      error
}

func (s *stubCaptureClient) Capture(_ context.Context, _ *ReplayMovieSpec, _ int) (*captureResponse, error) {
	return s.response, s.err
}

// makeJPEGBase64 creates a minimal valid JPEG as base64.
func makeJPEGBase64(t *testing.T, w, h int) string {
	t.Helper()
	path := writeTestJPEG(t, w, h)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(data)
}

func TestRenderCapture_ContiguousFrameNumbering(t *testing.T) {
	// Simulate a capture where some frames have empty data (should be skipped).
	// The written files must form a contiguous sequence (no gaps) because
	// FFmpeg's image2 demuxer stops at the first missing index.
	validFrame := makeJPEGBase64(t, 320, 240)

	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: validFrame},
				{Index: 1, Data: ""}, // empty — should be skipped
				{Index: 2, Data: validFrame},
				{Index: 3, Data: "  "}, // whitespace-only — should be skipped
				{Index: 4, Data: validFrame},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 500},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	media, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer media.Cleanup()

	// Verify the mock encoder was called (meaning we got past frame writing)
	mock := r.videoEncoder.(*MockVideoEncoder)
	if len(mock.AssembleCalls) != 1 {
		t.Fatalf("expected 1 assemble call, got %d", len(mock.AssembleCalls))
	}

	// Verify the frame pattern directory has contiguous files
	pattern := mock.AssembleCalls[0].Pattern
	dir := filepath.Dir(pattern)

	for i := 0; i < 3; i++ {
		path := filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", i))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected contiguous frame-%05d.jpg to exist", i)
		}
	}

	// frame-00003 should NOT exist (only 3 valid frames: 0, 1, 2)
	path := filepath.Join(dir, "frame-00003.jpg")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("frame-00003.jpg should not exist — only 3 valid frames")
	}
}

func TestRenderCapture_AllEmptyFrames(t *testing.T) {
	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: ""},
				{Index: 1, Data: "  "},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 200},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error when all frames are empty")
	}
}

func TestRenderCapture_NilCaptureResponse(t *testing.T) {
	client := &stubCaptureClient{
		response: nil,
	}

	r := newTestRenderer(t)
	r.captureClient = client

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error for nil capture response")
	}
}

func TestRenderCapture_CaptureError(t *testing.T) {
	client := &stubCaptureClient{
		err: fmt.Errorf("browser crashed"),
	}

	r := newTestRenderer(t)
	r.captureClient = client

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error when capture fails")
	}
}

func TestRenderCapture_GIFFormat(t *testing.T) {
	validFrame := makeJPEGBase64(t, 320, 240)

	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: validFrame},
				{Index: 1, Data: validFrame},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 200},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	media, err := r.renderCapture(t.Context(), spec, RenderFormatGIF, "test.gif", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer media.Cleanup()

	mock := r.videoEncoder.(*MockVideoEncoder)

	// Should call AssembleVideoFromSequence (for intermediate MP4) then ConvertToGIF
	if len(mock.AssembleCalls) != 1 {
		t.Fatalf("expected 1 assemble call, got %d", len(mock.AssembleCalls))
	}
	if len(mock.ConvertCalls) != 1 {
		t.Fatalf("expected 1 convert-to-gif call, got %d", len(mock.ConvertCalls))
	}
	if media.ContentType != "image/gif" {
		t.Fatalf("expected content type image/gif, got %s", media.ContentType)
	}
}

func TestRenderCapture_DefaultFPS(t *testing.T) {
	validFrame := makeJPEGBase64(t, 320, 240)

	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     0, // no FPS from capture → should use default
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: validFrame},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	media, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer media.Cleanup()

	mock := r.videoEncoder.(*MockVideoEncoder)
	if len(mock.AssembleCalls) != 1 {
		t.Fatalf("expected 1 assemble call, got %d", len(mock.AssembleCalls))
	}
	// FPS should be positive (defaulted)
	if mock.AssembleCalls[0].FPS <= 0 {
		t.Fatalf("expected positive default FPS, got %d", mock.AssembleCalls[0].FPS)
	}
}

func TestRenderCapture_InvalidBase64Frame(t *testing.T) {
	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: "not-valid-base64!!!"},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error for invalid base64 frame data")
	}
}

func TestRenderCapture_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // immediately cancel

	client := &stubCaptureClient{
		err: ctx.Err(),
	}

	r := newTestRenderer(t)
	r.captureClient = client

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(ctx, spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRenderCapture_EncoderError(t *testing.T) {
	validFrame := makeJPEGBase64(t, 320, 240)

	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: validFrame},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{AssembleVideoErr: fmt.Errorf("encoder failed")}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatMP4, "test.mp4", 100)
	if err == nil {
		t.Fatal("expected error when encoder fails")
	}
}

func TestRenderCapture_GIFConversionError(t *testing.T) {
	validFrame := makeJPEGBase64(t, 320, 240)

	client := &stubCaptureClient{
		response: &captureResponse{
			Success: true,
			FPS:     10,
			Width:   320,
			Height:  240,
			Frames: []captureFrame{
				{Index: 0, Data: validFrame},
			},
		},
	}

	r := newTestRenderer(t)
	r.captureClient = client
	r.videoEncoder = &MockVideoEncoder{ConvertGIFErr: fmt.Errorf("gif conversion failed")}

	spec := &ReplayMovieSpec{
		Playback: export.ExportPlayback{FrameIntervalMs: 100},
		Summary:  export.ExportSummary{TotalDurationMs: 100},
		Frames:   []export.ExportFrame{{DurationMs: 100}},
	}

	_, err := r.renderCapture(t.Context(), spec, RenderFormatGIF, "test.gif", 100)
	if err == nil {
		t.Fatal("expected error when GIF conversion fails")
	}
}

// ---------------------------------------------------------------------------
// RenderedMedia cleanup
// ---------------------------------------------------------------------------

func TestRenderedMedia_Cleanup_Nil(t *testing.T) {
	// Should not panic
	var m *RenderedMedia
	m.Cleanup()
}

func TestRenderedMedia_Cleanup_NoFunc(t *testing.T) {
	// Should not panic
	m := &RenderedMedia{}
	m.Cleanup()
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestRenderer(t *testing.T) *ReplayRenderer {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	return &ReplayRenderer{
		log:               log,
		recordingsRoot:    t.TempDir(),
		ffmpegPath:        "ffmpeg",
		exportPageURL:     "http://localhost:9999/export",
		captureIntervalMs: 100,
		apiBaseURL:        "http://localhost:9999",
	}
}
