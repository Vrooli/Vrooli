package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// buildAssemblyFilterChain tests
// ---------------------------------------------------------------------------

func TestBuildAssemblyFilterChain_NoWatermark(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1920, 1080)

	// Must use scale with increase (not decrease) to fill target and prevent
	// black padding that causes bottom-of-frame flickering.
	if !strings.Contains(chain, "scale=1920:1080:force_original_aspect_ratio=increase") {
		t.Fatalf("filter chain missing scale=increase filter, got: %s", chain)
	}

	// Must use crop (not pad) to trim overflow instead of adding black bars.
	if !strings.Contains(chain, "crop=1920:1080:") {
		t.Fatalf("filter chain missing crop filter, got: %s", chain)
	}

	// Must end with yuv420p format conversion
	if !strings.HasSuffix(chain, "format=yuv420p") {
		t.Fatalf("filter chain must end with format=yuv420p, got: %s", chain)
	}

	// Must NOT contain drawtext
	if strings.Contains(chain, "drawtext") {
		t.Fatal("filter chain should not contain drawtext without watermark")
	}

	// Verify filter order: scale before crop before format
	scaleIdx := strings.Index(chain, "scale=")
	cropIdx := strings.Index(chain, "crop=")
	fmtIdx := strings.Index(chain, "format=")
	if scaleIdx >= cropIdx {
		t.Fatalf("scale must come before crop: scale@%d, crop@%d", scaleIdx, cropIdx)
	}
	if cropIdx >= fmtIdx {
		t.Fatalf("crop must come before format: crop@%d, format@%d", cropIdx, fmtIdx)
	}
}

func TestBuildAssemblyFilterChain_WithWatermark(t *testing.T) {
	chain := buildAssemblyFilterChain("BAS Preview", 1280, 720)

	if !strings.Contains(chain, "drawtext=text='BAS Preview'") {
		t.Fatalf("filter chain missing watermark text, got: %s", chain)
	}

	// Watermark must come after crop but before format
	cropIdx := strings.Index(chain, "crop=")
	drawIdx := strings.Index(chain, "drawtext=")
	fmtIdx := strings.Index(chain, "format=")

	if drawIdx <= cropIdx {
		t.Fatalf("drawtext must come after crop: crop@%d, drawtext@%d", cropIdx, drawIdx)
	}
	if drawIdx >= fmtIdx {
		t.Fatalf("drawtext must come before format: drawtext@%d, format@%d", drawIdx, fmtIdx)
	}
}

func TestBuildAssemblyFilterChain_WatermarkSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"colon", "time:12:34", `time\:12\:34`},
		{"apostrophe", "it's a test", `it\'s a test`},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"combined", `O'Brien: C:\Users`, `O\'Brien\: C\:\\Users`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chain := buildAssemblyFilterChain(tc.input, 1920, 1080)
			if !strings.Contains(chain, tc.expected) {
				t.Fatalf("expected escaped text %q in chain, got: %s", tc.expected, chain)
			}
		})
	}
}

func TestBuildAssemblyFilterChain_FilterCount(t *testing.T) {
	// Without watermark: scale, crop, format = 3 filters
	chain := buildAssemblyFilterChain("", 1920, 1080)
	parts := strings.Split(chain, ",")
	if len(parts) != 3 {
		t.Fatalf("expected 3 filters without watermark, got %d: %v", len(parts), parts)
	}

	// With watermark: scale, crop, drawtext, format = 4 filters
	chain = buildAssemblyFilterChain("test", 1920, 1080)
	parts = strings.Split(chain, ",")
	if len(parts) != 4 {
		t.Fatalf("expected 4 filters with watermark, got %d: %v", len(parts), parts)
	}
}

func TestBuildAssemblyFilterChain_CropCentersContent(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1920, 1080)

	// The crop filter must center the crop using (iw-W)/2 and (ih-H)/2
	if !strings.Contains(chain, "(iw-1920)/2:(ih-1080)/2") {
		t.Fatalf("crop filter must center content, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_UsesExplicitDimensions(t *testing.T) {
	tests := []struct {
		name  string
		w, h  int
		wantW string
		wantH string
	}{
		{"standard 1080p", 1920, 1080, "1920", "1080"},
		{"standard 720p", 1280, 720, "1280", "720"},
		{"small even", 640, 480, "640", "480"},
		{"4K", 3840, 2160, "3840", "2160"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chain := buildAssemblyFilterChain("", tc.w, tc.h)

			expectedScale := fmt.Sprintf("scale=%s:%s:force_original_aspect_ratio=increase", tc.wantW, tc.wantH)
			if !strings.Contains(chain, expectedScale) {
				t.Fatalf("expected scale=%s:%s:increase, got: %s", tc.wantW, tc.wantH, chain)
			}

			expectedCrop := fmt.Sprintf("crop=%s:%s:", tc.wantW, tc.wantH)
			if !strings.Contains(chain, expectedCrop) {
				t.Fatalf("expected crop=%s:%s, got: %s", tc.wantW, tc.wantH, chain)
			}
		})
	}
}

func TestBuildAssemblyFilterChain_NeverUsesIwIh(t *testing.T) {
	// The old bug: scale=iw:ih was a no-op because iw/ih refer to each
	// frame's own dimensions. Verify we never produce that pattern.
	chain := buildAssemblyFilterChain("", 1920, 1080)

	if strings.Contains(chain, "scale=iw:ih") {
		t.Fatalf("scale filter must NOT use iw:ih (it's a no-op), got: %s", chain)
	}
	// Must not use ceil expressions (old pad approach)
	if strings.Contains(chain, "ceil(iw") || strings.Contains(chain, "ceil(ih") {
		t.Fatalf("filter chain must use explicit dimensions, not ceil expressions, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_NeverUsesPad(t *testing.T) {
	// The old approach used pad=W:H which fills gaps with black, causing
	// flickering when frame dimensions vary. Verify we never produce pad.
	chain := buildAssemblyFilterChain("", 1280, 720)
	if strings.Contains(chain, "pad=") {
		t.Fatalf("filter chain must NOT use pad (causes flickering), got: %s", chain)
	}

	// Also verify with watermark
	chain = buildAssemblyFilterChain("test", 1280, 720)
	if strings.Contains(chain, "pad=") {
		t.Fatalf("filter chain with watermark must NOT use pad, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_NeverUsesDecrease(t *testing.T) {
	// force_original_aspect_ratio=decrease leaves shorter frames at original
	// size, causing black padding via the subsequent pad filter. Verify we
	// use increase (scale up to fill) instead.
	chain := buildAssemblyFilterChain("", 1280, 720)
	if strings.Contains(chain, "decrease") {
		t.Fatalf("filter chain must NOT use decrease (causes flickering), got: %s", chain)
	}
	if !strings.Contains(chain, "increase") {
		t.Fatalf("filter chain must use increase to fill target, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_CropUsesTargetDimensions(t *testing.T) {
	// Verify crop references the target dimensions, not iw/ih
	chain := buildAssemblyFilterChain("", 1280, 720)

	// Must contain crop=1280:720
	if !strings.Contains(chain, "crop=1280:720:") {
		t.Fatalf("crop must use target dimensions, got: %s", chain)
	}

	// Crop centering must reference the target dimensions in the offset calc
	if !strings.Contains(chain, "(iw-1280)/2:(ih-720)/2") {
		t.Fatalf("crop centering must reference target dims, got: %s", chain)
	}
}

// ---------------------------------------------------------------------------
// ceilEven tests
// ---------------------------------------------------------------------------

func TestCeilEven(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{1, 2},
		{2, 2},
		{3, 4},
		{4, 4},
		{719, 720},
		{720, 720},
		{721, 722},
		{1079, 1080},
		{1080, 1080},
		{1081, 1082},
		{1919, 1920},
		{1920, 1920},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.input), func(t *testing.T) {
			got := ceilEven(tc.input)
			if got != tc.expected {
				t.Fatalf("ceilEven(%d) = %d, want %d", tc.input, got, tc.expected)
			}
			if got%2 != 0 {
				t.Fatalf("ceilEven(%d) = %d, which is odd", tc.input, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// probeImageDimensions tests
// ---------------------------------------------------------------------------

func TestProbeImageDimensions_JPEG(t *testing.T) {
	path := writeTestJPEG(t, 1920, 1080)

	w, h, err := probeImageDimensions(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1920 || h != 1080 {
		t.Fatalf("expected 1920x1080, got %dx%d", w, h)
	}
}

func TestProbeImageDimensions_PNG(t *testing.T) {
	path := writeTestPNG(t, 1280, 721)

	w, h, err := probeImageDimensions(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 721 {
		t.Fatalf("expected 1280x721, got %dx%d", w, h)
	}
}

func TestProbeImageDimensions_OddDimensions(t *testing.T) {
	tests := []struct {
		name string
		w, h int
	}{
		{"both odd", 1919, 1079},
		{"odd width", 1919, 1080},
		{"odd height", 1920, 1079},
		{"small odd", 641, 479},
		{"single pixel", 1, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTestJPEG(t, tc.w, tc.h)
			w, h, err := probeImageDimensions(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w != tc.w || h != tc.h {
				t.Fatalf("expected %dx%d, got %dx%d", tc.w, tc.h, w, h)
			}
		})
	}
}

func TestProbeImageDimensions_MissingFile(t *testing.T) {
	_, _, err := probeImageDimensions(filepath.Join(t.TempDir(), "nonexistent.jpg"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestProbeImageDimensions_InvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.jpg")
	if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := probeImageDimensions(path)
	if err == nil {
		t.Fatal("expected error for invalid image data")
	}
}

// ---------------------------------------------------------------------------
// escapeFFmpegText tests
// ---------------------------------------------------------------------------

func TestEscapeFFmpegText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"with:colon", `with\:colon`},
		{"with'quote", `with\'quote`},
		{`with\backslash`, `with\\backslash`},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeFFmpegText(tc.input)
			if result != tc.expected {
				t.Fatalf("escapeFFmpegText(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MockVideoEncoder tests
// ---------------------------------------------------------------------------

func TestMockVideoEncoder_RecordsCalls(t *testing.T) {
	mock := &MockVideoEncoder{}
	ctx := t.Context()

	t.Run("AssembleVideoFromSequence", func(t *testing.T) {
		err := mock.AssembleVideoFromSequence(ctx, "frames/%05d.jpg", 30, "out.mp4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.AssembleCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.AssembleCalls))
		}
		call := mock.AssembleCalls[0]
		if call.Pattern != "frames/%05d.jpg" || call.FPS != 30 || call.OutputPath != "out.mp4" {
			t.Fatalf("unexpected call args: %+v", call)
		}
	})

	t.Run("AssembleVideoWithWatermark", func(t *testing.T) {
		err := mock.AssembleVideoWithWatermark(ctx, "frames/%05d.jpg", 25, "out.mp4", "Preview")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.AssembleWatermarkCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.AssembleWatermarkCalls))
		}
		call := mock.AssembleWatermarkCalls[0]
		if call.WatermarkText != "Preview" {
			t.Fatalf("expected watermark 'Preview', got %q", call.WatermarkText)
		}
	})

	t.Run("ConvertToGIF", func(t *testing.T) {
		err := mock.ConvertToGIF(ctx, "in.mp4", "out.gif", 640, 12)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.ConvertCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.ConvertCalls))
		}
	})

	t.Run("ConvertToMP4", func(t *testing.T) {
		err := mock.ConvertToMP4(ctx, "in.webm", "out.mp4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.ConvertMP4Calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(mock.ConvertMP4Calls))
		}
	})
}

func TestMockVideoEncoder_ReturnsConfiguredErrors(t *testing.T) {
	ctx := t.Context()

	t.Run("AssembleVideoErr", func(t *testing.T) {
		mock := &MockVideoEncoder{AssembleVideoErr: errTest}
		if err := mock.AssembleVideoFromSequence(ctx, "", 0, ""); err != errTest {
			t.Fatalf("expected errTest, got %v", err)
		}
		if err := mock.AssembleVideoWithWatermark(ctx, "", 0, "", ""); err != errTest {
			t.Fatalf("expected errTest for watermark variant, got %v", err)
		}
	})

	t.Run("ConvertGIFErr", func(t *testing.T) {
		mock := &MockVideoEncoder{ConvertGIFErr: errTest}
		if err := mock.ConvertToGIF(ctx, "", "", 0, 0); err != errTest {
			t.Fatalf("expected errTest, got %v", err)
		}
	})

	t.Run("ConvertMP4Err", func(t *testing.T) {
		mock := &MockVideoEncoder{ConvertMP4Err: errTest}
		if err := mock.ConvertToMP4(ctx, "", ""); err != errTest {
			t.Fatalf("expected errTest, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// probeModeDimensions tests
// ---------------------------------------------------------------------------

func TestProbeModeDimensions_UniformFrames(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}
	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_FirstFrameOutlier(t *testing.T) {
	// Simulates Chrome "controlled by automation" info bar adding ~50px
	// to the first frame. The mode should be the steady-state dimensions.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 770) // outlier
	for i := 1; i < 8; i++ {
		writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 720 {
		t.Fatalf("expected mode dimensions 1280x720 (not outlier 1280x770), got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_LastFrameOutlier(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 7; i++ {
		writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", i)), 1920, 1080)
	}
	// Last frame captured during shutdown at different dimensions
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 7)), 1920, 1030)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1920 || h != 1080 {
		t.Fatalf("expected 1920x1080, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_SingleFrame(t *testing.T) {
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 640, 480)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 640 || h != 480 {
		t.Fatalf("expected 640x480, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_MissingFirstFrame(t *testing.T) {
	dir := t.TempDir()
	// No frames at all
	_, _, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err == nil {
		t.Fatal("expected error for missing first frame")
	}
	if !strings.Contains(err.Error(), "first frame") {
		t.Fatalf("expected first-frame error, got: %v", err)
	}
}

func TestProbeModeDimensions_TwoEqualGroups_PrefersSmaller(t *testing.T) {
	// When two dimension groups tie on count, the SMALLER dimensions must
	// win. Larger frames get cropped (imperceptible), while smaller frames
	// scaled up cause visible content shift that appears as flicker.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 770)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 770)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both groups have count=2; smaller area (1280x720) must win.
	if w != 1280 || h != 720 {
		t.Fatalf("expected smaller dimensions 1280x720 on tie, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_ThreeGroups_ModeWins(t *testing.T) {
	// Three distinct dimension groups: the one with the most frames wins
	// even if it's not the smallest.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 770) // outlier (1)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 720) // mode (4)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1920, 1080) // outlier (2)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 4)), 1920, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 5)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 6)), 1280, 720)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 720 {
		t.Fatalf("expected mode 1280x720, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_TieBreakingIsDeterministic(t *testing.T) {
	// Run the tie-breaking scenario multiple times to verify determinism.
	// Before the fix, Go map iteration randomness meant this could flip-flop.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 770)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1280, 770)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 720)

	pattern := filepath.Join(dir, "frame-%05d.jpg")
	for i := 0; i < 50; i++ {
		w, h, err := probeModeDimensions(pattern)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if w != 1280 || h != 720 {
			t.Fatalf("iteration %d: expected deterministic 1280x720, got %dx%d (non-deterministic tie-breaking!)", i, w, h)
		}
	}
}

func TestProbeModeDimensions_TieBreakWidth(t *testing.T) {
	// Tie on count with different widths: smaller area wins.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1920, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1920, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 1080)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 1080 {
		t.Fatalf("expected smaller-area 1280x1080 on tie, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_ScrollbarWidthVariation(t *testing.T) {
	// Scrollbar appearing reduces effective width by ~17px for some frames.
	// Most frames should be at the wider dimension.
	dir := t.TempDir()
	for i := 0; i < 6; i++ {
		writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}
	// Two frames with scrollbar
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 6)), 1263, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 7)), 1263, 720)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 1280 || h != 720 {
		t.Fatalf("expected mode 1280x720, got %dx%d", w, h)
	}
}

// ---------------------------------------------------------------------------
// assembleVideo integration test (probing + filter chain)
// ---------------------------------------------------------------------------

func TestAssembleVideo_ProbesMultipleFrames(t *testing.T) {
	// Create a frame sequence where the first frame is an outlier.
	// Verify assembleVideo uses the mode dimensions.
	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00000.jpg"), 1280, 770) // outlier
	for i := 1; i < 6; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}

	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	pattern := filepath.Join(framesDir, "frame-%05d.jpg")

	err := encoder.assembleVideo(t.Context(), pattern, 25, filepath.Join(dir, "out.mp4"), "")

	// Should fail on ffmpeg execution, NOT on probing
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	if strings.Contains(err.Error(), "probe") || strings.Contains(err.Error(), "dimension") {
		t.Fatalf("probe should succeed but got probe error: %v", err)
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("expected ffmpeg execution error, got: %v", err)
	}
}

func TestAssembleVideo_FailsOnMissingFirstFrame(t *testing.T) {
	encoder := NewFFmpegEncoder("ffmpeg")
	err := encoder.assembleVideo(
		t.Context(),
		filepath.Join(t.TempDir(), "frame-%05d.jpg"),
		25,
		filepath.Join(t.TempDir(), "out.mp4"),
		"",
	)
	if err == nil {
		t.Fatal("expected error for missing first frame")
	}
	if !strings.Contains(err.Error(), "first frame") {
		t.Fatalf("expected first-frame probe error, got: %v", err)
	}
}

func TestAssembleVideo_DefaultFPS(t *testing.T) {
	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00000.jpg"), 640, 480)

	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	pattern := filepath.Join(framesDir, "frame-%05d.jpg")

	// fps=0 should default to 25 (not error)
	err := encoder.assembleVideo(t.Context(), pattern, 0, filepath.Join(dir, "out.mp4"), "")
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	// Should get past probe and fps check, fail only on ffmpeg exec
	if strings.Contains(err.Error(), "probe") || strings.Contains(err.Error(), "first frame") {
		t.Fatalf("should not fail on probe: %v", err)
	}
}

func TestAssembleVideo_NegativeFPS(t *testing.T) {
	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00000.jpg"), 640, 480)

	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	pattern := filepath.Join(framesDir, "frame-%05d.jpg")

	// Negative fps should be treated like 0 (defaults to 25)
	err := encoder.assembleVideo(t.Context(), pattern, -5, filepath.Join(dir, "out.mp4"), "")
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	if strings.Contains(err.Error(), "probe") || strings.Contains(err.Error(), "first frame") {
		t.Fatalf("should not fail on probe: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ConvertToGIF defaults
// ---------------------------------------------------------------------------

func TestConvertToGIF_DefaultFPS(t *testing.T) {
	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	// fps=0 should use default (12), not error before exec
	err := encoder.ConvertToGIF(t.Context(), "in.mp4", "out.gif", 640, 0)
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("expected ffmpeg execution error, got: %v", err)
	}
}

func TestConvertToGIF_DefaultWidth(t *testing.T) {
	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	// targetWidth=0 should use default, not error before exec
	err := encoder.ConvertToGIF(t.Context(), "in.mp4", "out.gif", 0, 12)
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("expected ffmpeg execution error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ConvertToMP4 filter chain
// ---------------------------------------------------------------------------

func TestConvertToMP4_NoPadFilter(t *testing.T) {
	// ConvertToMP4 must NOT use pad (which adds black bars that caused
	// flickering). It should use scale+crop like AssembleVideoFromSequence.
	// We can't inspect the actual args without running, but verify execution.
	encoder := NewFFmpegEncoder("__nonexistent_ffmpeg__")
	err := encoder.ConvertToMP4(t.Context(), "in.webm", "out.mp4")
	if err == nil {
		t.Fatal("expected ffmpeg error with nonexistent binary")
	}
	if !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("expected ffmpeg execution error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func writeTestJPEG(t *testing.T, width, height int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.jpg")
	writeTestJPEGAt(t, path, width, height)
	return path
}

func writeTestJPEGAt(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color so JPEG encoding produces valid data
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write test JPEG: %v", err)
	}
}

func writeTestPNG(t *testing.T, width, height int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.png")
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}
	return path
}

var errTest = fmt.Errorf("test error")
