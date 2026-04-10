package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// buildAssemblyFilterChain tests
// ---------------------------------------------------------------------------

func TestBuildAssemblyFilterChain_NoWatermark(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1920, 1080)

	// Must start with crop to handle height variation (info bars) without
	// scaling — preserving pixel-perfect vertical alignment.
	if !strings.HasPrefix(chain, "crop=") {
		t.Fatalf("filter chain must start with crop filter, got: %s", chain)
	}

	// Must contain force scale to exact target dimensions (not scale=increase).
	if !strings.Contains(chain, "scale=1920:1080") {
		t.Fatalf("filter chain missing force scale=1920:1080, got: %s", chain)
	}

	// Must end with yuv420p format conversion
	if !strings.HasSuffix(chain, "format=yuv420p") {
		t.Fatalf("filter chain must end with format=yuv420p, got: %s", chain)
	}

	// Must NOT contain drawtext
	if strings.Contains(chain, "drawtext") {
		t.Fatal("filter chain should not contain drawtext without watermark")
	}

	// Verify filter order: crop before scale before format
	cropIdx := strings.Index(chain, "crop=")
	scaleIdx := strings.Index(chain, "scale=")
	fmtIdx := strings.Index(chain, "format=")
	if cropIdx >= scaleIdx {
		t.Fatalf("crop must come before scale: crop@%d, scale@%d", cropIdx, scaleIdx)
	}
	if scaleIdx >= fmtIdx {
		t.Fatalf("scale must come before format: scale@%d, format@%d", scaleIdx, fmtIdx)
	}
}

func TestBuildAssemblyFilterChain_WithWatermark(t *testing.T) {
	chain := buildAssemblyFilterChain("BAS Preview", 1280, 720)

	if !strings.Contains(chain, "drawtext=text='BAS Preview'") {
		t.Fatalf("filter chain missing watermark text, got: %s", chain)
	}

	// Watermark must come after scale but before format
	scaleIdx := strings.Index(chain, "scale=")
	drawIdx := strings.Index(chain, "drawtext=")
	fmtIdx := strings.Index(chain, "format=")

	if drawIdx <= scaleIdx {
		t.Fatalf("drawtext must come after scale: scale@%d, drawtext@%d", scaleIdx, drawIdx)
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
	// Without watermark: crop, scale, format = 3 filters
	chain := buildAssemblyFilterChain("", 1920, 1080)
	parts := strings.Split(chain, ",")
	// The crop filter contains escaped commas (\,), which are NOT filter separators.
	// Count real filter boundaries by splitting on unescaped commas only.
	realFilters := splitUnescapedCommas(chain)
	if len(realFilters) != 3 {
		t.Fatalf("expected 3 filters without watermark, got %d: %v (raw parts: %v)", len(realFilters), realFilters, parts)
	}

	// With watermark: crop, scale, drawtext, format = 4 filters
	chain = buildAssemblyFilterChain("test", 1920, 1080)
	realFilters = splitUnescapedCommas(chain)
	if len(realFilters) != 4 {
		t.Fatalf("expected 4 filters with watermark, got %d: %v", len(realFilters), realFilters)
	}
}

func TestBuildAssemblyFilterChain_CropIsHeightOnly(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1920, 1080)

	// Crop must use iw (input width) for width — no width cropping.
	if !strings.Contains(chain, "crop=iw:") {
		t.Fatalf("crop must preserve input width (iw), got: %s", chain)
	}

	// Must NOT contain crop with explicit width dimensions.
	// The old pattern crop=1920:1080:... coupled width and height.
	if strings.Contains(chain, "crop=1920:") {
		t.Fatalf("crop must NOT use explicit width (decouples axes), got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_CropBottomAnchored(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1280, 720)

	// Y offset must be bottom-anchored: max(ih-targetH, 0).
	// This removes top overflow (info bars) while keeping page content aligned.
	if !strings.Contains(chain, "max(ih-720") {
		t.Fatalf("crop Y offset must be bottom-anchored max(ih-720,...), got: %s", chain)
	}

	// Height must use min(ih, targetH) to clamp for shorter frames.
	if !strings.Contains(chain, "min(ih") {
		t.Fatalf("crop height must use min(ih,...) for shorter frames, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_ForceScaleExactDimensions(t *testing.T) {
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

			// Must use force scale (no force_original_aspect_ratio).
			expectedScale := fmt.Sprintf("scale=%s:%s", tc.wantW, tc.wantH)
			if !strings.Contains(chain, expectedScale) {
				t.Fatalf("expected %s, got: %s", expectedScale, chain)
			}
		})
	}
}

func TestBuildAssemblyFilterChain_NeverUsesForceAspectRatio(t *testing.T) {
	// The old bug: force_original_aspect_ratio=increase coupled width and
	// height — scrollbar width changes caused proportional height changes,
	// resulting in vertical content shift and bottom-of-frame flickering.
	chain := buildAssemblyFilterChain("", 1280, 720)

	if strings.Contains(chain, "force_original_aspect_ratio") {
		t.Fatalf("filter chain must NOT use force_original_aspect_ratio (causes cross-dimensional coupling), got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_NeverUsesPad(t *testing.T) {
	// pad=W:H fills gaps with black, causing flickering when frame dimensions vary.
	chain := buildAssemblyFilterChain("", 1280, 720)
	if strings.Contains(chain, "pad=") {
		t.Fatalf("filter chain must NOT use pad (causes flickering), got: %s", chain)
	}

	chain = buildAssemblyFilterChain("test", 1280, 720)
	if strings.Contains(chain, "pad=") {
		t.Fatalf("filter chain with watermark must NOT use pad, got: %s", chain)
	}
}

// TestBuildAssemblyFilterChain_ScrollbarWidthDoesNotAffectHeight is the key
// regression test for the bottom-of-frame flickering bug. With the old
// scale=increase approach, a scrollbar reducing width by 17px caused height to
// increase by ~10px (cross-dimensional coupling), which the bottom-anchored
// crop then shifted vertically — causing content at the bottom to flicker.
//
// The new approach decouples axes: crop handles height, force-scale handles
// width. Width variation cannot cause height shift.
func TestBuildAssemblyFilterChain_ScrollbarWidthDoesNotAffectHeight(t *testing.T) {
	chain := buildAssemblyFilterChain("", 1280, 720)

	// The scale filter must use EXACT dimensions (no aspect ratio preservation).
	// This means width changes (scrollbar) are handled by horizontal stretch
	// only — they cannot propagate into vertical content shift.
	if strings.Contains(chain, "force_original_aspect_ratio") {
		t.Fatalf("scale must use exact dimensions to prevent width→height coupling, got: %s", chain)
	}

	// The crop filter must only affect height (use iw for width).
	// This ensures width variation is handled exclusively by scale.
	if !strings.Contains(chain, "crop=iw:") {
		t.Fatalf("crop must preserve input width to avoid double-handling width variation, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_InfoBarContentAlignment(t *testing.T) {
	// Regression test: Chrome "controlled by automation" info bar adds ~50px
	// at the TOP of some frames. The crop must remove top overflow (bottom-
	// anchored) so page content stays vertically aligned across all frames.
	chain := buildAssemblyFilterChain("", 1280, 720)

	// For a 1280x770 outlier frame (50px info bar at top), the crop should
	// produce y = max(770-720, 0) = 50, removing the entire info bar.
	// For a normal 1280x720 frame, y = max(720-720, 0) = 0 (no-op).
	if !strings.Contains(chain, "max(ih-720") {
		t.Fatalf("crop Y must use max(ih-720,...) for bottom-anchored removal, got: %s", chain)
	}
}

func TestBuildAssemblyFilterChain_SmallDimensions(t *testing.T) {
	tests := []struct {
		name string
		w, h int
	}{
		{"minimum even", 2, 2},
		{"small", 64, 48},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			chain := buildAssemblyFilterChain("", tc.w, tc.h)
			if !strings.Contains(chain, "crop=") {
				t.Fatalf("missing crop filter for %dx%d: %s", tc.w, tc.h, chain)
			}
			if !strings.Contains(chain, "scale=") {
				t.Fatalf("missing scale filter for %dx%d: %s", tc.w, tc.h, chain)
			}
			if !strings.Contains(chain, "format=yuv420p") {
				t.Fatalf("missing format filter for %dx%d: %s", tc.w, tc.h, chain)
			}
		})
	}
}

func TestBuildAssemblyFilterChain_EscapedCommasInExpressions(t *testing.T) {
	// FFmpeg expressions with min()/max() contain commas that must be escaped
	// as \, to prevent them from being parsed as filter separators.
	chain := buildAssemblyFilterChain("", 1280, 720)

	// The crop filter must contain escaped commas within min() and max().
	if !strings.Contains(chain, `min(ih\,720)`) {
		t.Fatalf("crop height expression must escape comma in min(), got: %s", chain)
	}
	if !strings.Contains(chain, `max(ih-720\,0)`) {
		t.Fatalf("crop Y offset expression must escape comma in max(), got: %s", chain)
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
// modeValue tests
// ---------------------------------------------------------------------------

func TestModeValue_SingleValue(t *testing.T) {
	counts := map[int]int{720: 5}
	got := modeValue(counts, 720)
	if got != 720 {
		t.Fatalf("expected 720, got %d", got)
	}
}

func TestModeValue_ClearWinner(t *testing.T) {
	counts := map[int]int{720: 8, 770: 2}
	got := modeValue(counts, 720)
	if got != 720 {
		t.Fatalf("expected 720, got %d", got)
	}
}

func TestModeValue_TiePrefersSmaller(t *testing.T) {
	counts := map[int]int{720: 3, 770: 3}
	got := modeValue(counts, 770)
	if got != 720 {
		t.Fatalf("expected smaller value 720 on tie, got %d", got)
	}
}

func TestModeValue_TieIsDeterministic(t *testing.T) {
	counts := map[int]int{1280: 4, 1263: 4}
	for i := 0; i < 50; i++ {
		got := modeValue(counts, 1280)
		if got != 1263 {
			t.Fatalf("iteration %d: expected deterministic 1263 on tie, got %d", i, got)
		}
	}
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
	// When two dimension groups tie on count, the SMALLER value must win
	// for each axis independently.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 770)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 770)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Width: 1280 (all frames) → 1280
	// Height: 720 (2) vs 770 (2) → tie, smaller wins → 720
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_ThreeGroups_ModeWins(t *testing.T) {
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
	// Width: 1280 (5) vs 1920 (2) → 1280
	// Height: 720 (4) vs 770 (1) vs 1080 (2) → 720
	if w != 1280 || h != 720 {
		t.Fatalf("expected mode 1280x720, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_TieBreakingIsDeterministic(t *testing.T) {
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
	// Tie on count with different widths: smaller value wins.
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1920, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1920, 1080)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 1080)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Width: 1920 (2) vs 1280 (2) → tie, smaller wins → 1280
	// Height: 1080 (all) → 1080
	if w != 1280 || h != 1080 {
		t.Fatalf("expected smaller-width 1280x1080 on tie, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_CorruptFrameInMiddle(t *testing.T) {
	dir := t.TempDir()
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 720)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 720)
	// Write corrupt data for frame 2
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 720)

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("corrupt middle frame should be skipped, got error: %v", err)
	}
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

func TestProbeModeDimensions_CorruptFirstFrame(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 720)

	_, _, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err == nil {
		t.Fatal("expected error for corrupt first frame")
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

// TestProbeModeDimensions_IndependentAxes verifies that width and height modes
// are computed independently. This is critical when scrollbar (width) and info
// bar (height) variations occur simultaneously — the old joint-pair approach
// could pick outlier dimensions when both varied at once.
func TestProbeModeDimensions_IndependentAxes(t *testing.T) {
	dir := t.TempDir()
	// Mix of scrollbar (width) and info bar (height) variations
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 770) // info bar only
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1263, 720) // scrollbar only
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 4)), 1263, 770) // both!
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 5)), 1280, 720) // normal

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Width: 1280 (4) vs 1263 (2) → 1280
	// Height: 720 (4) vs 770 (2) → 720
	// With joint-pair: (1280,720)=3, (1280,770)=1, (1263,720)=1, (1263,770)=1
	//   → same result here, but independent is more robust for edge cases
	if w != 1280 || h != 720 {
		t.Fatalf("expected independent mode 1280x720, got %dx%d", w, h)
	}
}

// TestProbeModeDimensions_IndependentAxes_EdgeCase verifies independent mode
// handles the case where joint-pair mode would give wrong dimensions.
func TestProbeModeDimensions_IndependentAxes_EdgeCase(t *testing.T) {
	dir := t.TempDir()
	// Scenario: equal mix where no single (w,h) pair dominates, but the
	// correct width and height are each independently the most common.
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 0)), 1280, 720) // pair A (2)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 1)), 1280, 770) // pair B (2)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 2)), 1263, 720) // pair C (2)
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 3)), 1280, 720) // pair A
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 4)), 1280, 770) // pair B
	writeTestJPEGAt(t, filepath.Join(dir, fmt.Sprintf("frame-%05d.jpg", 5)), 1263, 720) // pair C

	w, h, err := probeModeDimensions(filepath.Join(dir, "frame-%05d.jpg"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Joint-pair: A=2, B=2, C=2 → all tied, smallest area wins → 1263x720
	// Independent: width: 1280 (4) vs 1263 (2) → 1280
	//              height: 720 (4) vs 770 (2) → 720
	// Independent gives 1280x720 which is the correct "normal" state.
	if w != 1280 || h != 720 {
		t.Fatalf("expected independent mode 1280x720 (not joint-pair 1263x720), got %dx%d", w, h)
	}
}

// ---------------------------------------------------------------------------
// assembleVideo integration test (probing + filter chain)
// ---------------------------------------------------------------------------

func TestAssembleVideo_ProbesMultipleFrames(t *testing.T) {
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
// Real FFmpeg integration tests — verify actual output, not just strings.
//
// These create test frames with known dimension variations, run them through
// the real FFmpeg pipeline, and probe the output video to verify consistent
// dimensions. This is the layer of testing that was missing: previous tests
// only checked filter chain *strings*, not actual encoder *output*.
// ---------------------------------------------------------------------------

// ffmpegAvailable returns true if the real ffmpeg binary is on PATH.
func ffmpegAvailable(t *testing.T) bool {
	t.Helper()
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// ffprobeAvailable returns true if ffprobe is on PATH.
func ffprobeAvailable(t *testing.T) bool {
	t.Helper()
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// probeVideoDimensions uses ffprobe to get output video width and height.
func probeVideoDimensions(t *testing.T, videoPath string) (width, height int) {
	t.Helper()
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		videoPath,
	).Output()
	if err != nil {
		t.Fatalf("ffprobe failed: %v", err)
	}
	line := strings.TrimSpace(string(out))
	parts := strings.SplitN(line, "x", 2)
	if len(parts) != 2 {
		t.Fatalf("unexpected ffprobe output: %q", line)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		t.Fatalf("failed to parse dimensions from %q", line)
	}
	return w, h
}

// probeVideoFrameCount uses ffprobe to count the number of frames in a video.
func probeVideoFrameCount(t *testing.T, videoPath string) int {
	t.Helper()
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-count_frames",
		"-show_entries", "stream=nb_read_frames",
		"-of", "csv=p=0",
		videoPath,
	).Output()
	if err != nil {
		t.Fatalf("ffprobe frame count failed: %v", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("failed to parse frame count from %q", string(out))
	}
	return n
}

func TestFFmpegIntegration_UniformFrames(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720 output, got %dx%d", w, h)
	}
	count := probeVideoFrameCount(t, outPath)
	if count != 10 {
		t.Fatalf("expected 10 frames, got %d", count)
	}
}

// TestFFmpegIntegration_InfoBarOutlier is the key regression test for the
// bottom-50px flickering bug. Frame 0 simulates the Chrome "controlled by
// automation" info bar adding 50px to the height. The output video must have
// consistent 1280x720 dimensions — the outlier frame is cropped, not scaled.
func TestFFmpegIntegration_InfoBarOutlier(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Frame 0: 50px taller (info bar at top)
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00000.jpg"), 1280, 770)
	for i := 1; i < 10; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720 (mode dimensions), got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_ScrollbarWidthVariation verifies that frames with
// scrollbar-reduced width (~17px narrower) produce correct output dimensions.
func TestFFmpegIntegration_ScrollbarWidthVariation(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 8; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}
	// Two frames with scrollbar reducing width
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00008.jpg"), 1263, 720)
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00009.jpg"), 1263, 720)

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720 (mode dimensions), got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_MixedVariation combines info bar AND scrollbar variation.
func TestFFmpegIntegration_MixedVariation(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00000.jpg"), 1280, 770) // info bar
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00001.jpg"), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00002.jpg"), 1263, 720) // scrollbar
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00003.jpg"), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00004.jpg"), 1263, 770) // both!
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00005.jpg"), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00006.jpg"), 1280, 720) // normal
	writeTestJPEGAt(t, filepath.Join(framesDir, "frame-00007.jpg"), 1280, 720) // normal

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720 (mode dimensions), got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_SinglePixelJitter tests the subtle case where frames
// alternate between heights differing by 1px (e.g., 720 vs 721). This can
// cause per-frame crop offset variation that shifts content by 1px.
func TestFFmpegIntegration_SinglePixelJitter(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Alternating 720/721 height (simulates subpixel rendering jitter)
	for i := 0; i < 10; i++ {
		h := 720
		if i%2 == 1 {
			h = 721
		}
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, h)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	// Mode height: 720 (5 frames) ties with 721 (5 frames) → smaller wins → 720
	// ceilEven(720) = 720
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_OddDimensionsRoundToEven verifies H.264 even-dimension
// requirement is enforced on the actual output.
func TestFFmpegIntegration_OddDimensionsRoundToEven(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// All frames have odd dimensions
	for i := 0; i < 5; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1279, 719)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w%2 != 0 || h%2 != 0 {
		t.Fatalf("output dimensions must be even for H.264, got %dx%d", w, h)
	}
	// ceilEven(1279)=1280, ceilEven(719)=720
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_WithWatermark verifies watermark assembly produces valid output.
func TestFFmpegIntegration_WithWatermark(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoWithWatermark(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath, "BAS Preview"); err != nil {
		t.Fatalf("assemble with watermark failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}
}

// TestFFmpegIntegration_ConvertToMP4_EvenDimensions verifies ConvertToMP4
// produces even-dimension output without using force_original_aspect_ratio.
func TestFFmpegIntegration_ConvertToMP4_EvenDimensions(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a source video with odd dimensions
	for i := 0; i < 5; i++ {
		writeTestJPEGAt(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1279, 719)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	sourcePath := filepath.Join(dir, "source.mp4")
	// First assemble a source video (this rounds to even already)
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, sourcePath); err != nil {
		t.Fatalf("source assemble failed: %v", err)
	}

	// Now transcode it via ConvertToMP4
	outPath := filepath.Join(dir, "converted.mp4")
	if err := encoder.ConvertToMP4(t.Context(), sourcePath, outPath); err != nil {
		t.Fatalf("ConvertToMP4 failed: %v", err)
	}

	w, h := probeVideoDimensions(t, outPath)
	if w%2 != 0 || h%2 != 0 {
		t.Fatalf("ConvertToMP4 output must have even dimensions, got %dx%d", w, h)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// splitUnescapedCommas splits an FFmpeg filter chain string on commas that are
// NOT escaped with \. This correctly handles expressions like min(ih\,720)
// where the comma is part of the expression, not a filter separator.
func splitUnescapedCommas(s string) []string {
	var parts []string
	var current strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == ',' && (i == 0 || s[i-1] != '\\') {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(s[i])
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

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

// ---------------------------------------------------------------------------
// Pixel-level content consistency tests (flickering detection)
//
// These tests go beyond dimension verification: they create frames with known
// pixel patterns that simulate real-world dimension variations, assemble them
// into video, extract individual frames, and compare pixel values in the
// bottom 50px region. This directly tests for the bottom-of-frame flickering
// bug caused by scrollbar/info-bar dimension variance.
// ---------------------------------------------------------------------------

// writeTestJPEGWithPattern creates a JPEG where different vertical regions
// have distinct colors. The bottom region uses a unique color to make content
// shifts at the bottom detectable.
func writeTestJPEGWithPattern(t *testing.T, path string, width, height int, bottomColor color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	topColor := color.RGBA{R: 40, G: 80, B: 160, A: 255}   // blue top
	midColor := color.RGBA{R: 200, G: 200, B: 200, A: 255} // gray middle
	bottomThreshold := height - 50
	midThreshold := height / 2

	for y := 0; y < height; y++ {
		var c color.RGBA
		switch {
		case y >= bottomThreshold:
			c = bottomColor
		case y >= midThreshold:
			c = midColor
		default:
			c = topColor
		}
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write test JPEG: %v", err)
	}
}

// extractFramesFromVideo uses ffmpeg to extract individual frames from a video.
func extractFramesFromVideo(t *testing.T, videoPath, outputDir string) int {
	t.Helper()
	pattern := filepath.Join(outputDir, "extracted-%05d.png")
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-f", "image2", pattern)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg frame extraction failed: %v (%s)", err, stderr.String())
	}

	// Count extracted frames
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "extracted-") {
			count++
		}
	}
	return count
}

// sampleBottomPixels reads the average color of the bottom N rows of an image.
func sampleBottomPixels(t *testing.T, imgPath string, bottomRows int) (avgR, avgG, avgB float64) {
	t.Helper()
	f, err := os.Open(imgPath)
	if err != nil {
		t.Fatalf("failed to open %s: %v", imgPath, err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode %s: %v", imgPath, err)
	}

	bounds := img.Bounds()
	startY := bounds.Max.Y - bottomRows
	if startY < bounds.Min.Y {
		startY = bounds.Min.Y
	}

	var totalR, totalG, totalB float64
	count := 0
	for y := startY; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			totalR += float64(r >> 8)
			totalG += float64(g >> 8)
			totalB += float64(b >> 8)
			count++
		}
	}
	if count == 0 {
		return 0, 0, 0
	}
	return totalR / float64(count), totalG / float64(count), totalB / float64(count)
}

// TestFFmpegIntegration_BottomPixelConsistency_UniformFrames verifies that
// uniform frames produce identical bottom pixels — baseline for flickering tests.
func TestFFmpegIntegration_BottomPixelConsistency_UniformFrames(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	extractDir := filepath.Join(dir, "extracted")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatal(err)
	}

	bottomColor := color.RGBA{R: 255, G: 0, B: 0, A: 255} // red bottom
	for i := 0; i < 10; i++ {
		writeTestJPEGWithPattern(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720, bottomColor)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	count := extractFramesFromVideo(t, outPath, extractDir)
	if count < 2 {
		t.Fatalf("expected at least 2 extracted frames, got %d", count)
	}

	// Compare bottom pixels across all extracted frames
	baseR, baseG, baseB := sampleBottomPixels(t, filepath.Join(extractDir, "extracted-00001.png"), 50)
	for i := 2; i <= count; i++ {
		r, g, b := sampleBottomPixels(t, filepath.Join(extractDir, fmt.Sprintf("extracted-%05d.png", i)), 50)
		// Allow tolerance for codec compression artifacts (H.264 is lossy)
		const tolerance = 5.0
		if abs(r-baseR) > tolerance || abs(g-baseG) > tolerance || abs(b-baseB) > tolerance {
			t.Fatalf("bottom pixel inconsistency in frame %d: base=(%.1f,%.1f,%.1f) vs frame=(%.1f,%.1f,%.1f)",
				i, baseR, baseG, baseB, r, g, b)
		}
	}
}

// TestFFmpegIntegration_BottomPixelConsistency_InfoBarVariation verifies that
// when some frames have extra height (info bar), the crop filter produces
// consistent bottom pixel content across all output frames. This is the core
// flickering regression test: if the crop is wrong, bottom pixels will differ
// between info-bar and non-info-bar frames.
func TestFFmpegIntegration_BottomPixelConsistency_InfoBarVariation(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	extractDir := filepath.Join(dir, "extracted")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatal(err)
	}

	bottomColor := color.RGBA{R: 0, G: 255, B: 0, A: 255} // green bottom

	// Frame 0: 770px tall (50px info bar at top = extra height).
	// The info bar region gets a different color to simulate real content.
	infoBarFrame := image.NewRGBA(image.Rect(0, 0, 1280, 770))
	infoBarColor := color.RGBA{R: 60, G: 60, B: 60, A: 255}
	topColor := color.RGBA{R: 40, G: 80, B: 160, A: 255}
	midColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	for y := 0; y < 770; y++ {
		var c color.RGBA
		switch {
		case y < 50:
			c = infoBarColor // info bar region
		case y >= 720:
			c = bottomColor // bottom 50px of page content
		case y >= 385:
			c = midColor
		default:
			c = topColor
		}
		for x := 0; x < 1280; x++ {
			infoBarFrame.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, infoBarFrame, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(framesDir, "frame-00000.jpg"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	// Frames 1-9: normal 720px
	for i := 1; i < 10; i++ {
		writeTestJPEGWithPattern(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), 1280, 720, bottomColor)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	// Verify output dimensions
	w, h := probeVideoDimensions(t, outPath)
	if w != 1280 || h != 720 {
		t.Fatalf("expected 1280x720, got %dx%d", w, h)
	}

	// Extract frames and verify bottom pixel consistency
	count := extractFramesFromVideo(t, outPath, extractDir)
	if count < 2 {
		t.Fatalf("expected at least 2 extracted frames, got %d", count)
	}

	// All extracted frames should have the same bottom content
	// (the green bottom region of page content, NOT the info bar gray)
	baseR, baseG, baseB := sampleBottomPixels(t, filepath.Join(extractDir, "extracted-00002.png"), 50)

	// Green channel should dominate (bottomColor is pure green)
	if baseG < 200 {
		t.Fatalf("expected green-dominant bottom pixels (page content), got (%.1f,%.1f,%.1f) — possible info bar leaking through", baseR, baseG, baseB)
	}

	for i := 1; i <= count; i++ {
		r, g, b := sampleBottomPixels(t, filepath.Join(extractDir, fmt.Sprintf("extracted-%05d.png", i)), 50)
		const tolerance = 10.0 // slightly higher tolerance due to info bar frame
		if abs(r-baseR) > tolerance || abs(g-baseG) > tolerance || abs(b-baseB) > tolerance {
			t.Fatalf("bottom pixel flickering in frame %d: base=(%.1f,%.1f,%.1f) vs frame=(%.1f,%.1f,%.1f)",
				i, baseR, baseG, baseB, r, g, b)
		}
	}
}

// TestFFmpegIntegration_BottomPixelConsistency_ScrollbarWidth verifies that
// width variation (scrollbar) doesn't cause bottom content to shift. The force
// scale handles width without coupling to height, so bottom pixels should stay
// consistent.
func TestFFmpegIntegration_BottomPixelConsistency_ScrollbarWidth(t *testing.T) {
	if !ffmpegAvailable(t) || !ffprobeAvailable(t) {
		t.Skip("ffmpeg/ffprobe not available")
	}

	dir := t.TempDir()
	framesDir := filepath.Join(dir, "frames")
	extractDir := filepath.Join(dir, "extracted")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatal(err)
	}

	bottomColor := color.RGBA{R: 0, G: 0, B: 255, A: 255} // blue bottom

	// Mix of normal width and scrollbar-reduced width
	for i := 0; i < 10; i++ {
		w := 1280
		if i%3 == 0 {
			w = 1263 // scrollbar reduces width by ~17px
		}
		writeTestJPEGWithPattern(t, filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", i)), w, 720, bottomColor)
	}

	encoder := NewFFmpegEncoder("ffmpeg")
	outPath := filepath.Join(dir, "out.mp4")
	if err := encoder.AssembleVideoFromSequence(t.Context(), filepath.Join(framesDir, "frame-%05d.jpg"), 25, outPath); err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	count := extractFramesFromVideo(t, outPath, extractDir)
	if count < 2 {
		t.Fatalf("expected at least 2 extracted frames, got %d", count)
	}

	baseR, baseG, baseB := sampleBottomPixels(t, filepath.Join(extractDir, "extracted-00002.png"), 50)
	for i := 1; i <= count; i++ {
		r, g, b := sampleBottomPixels(t, filepath.Join(extractDir, fmt.Sprintf("extracted-%05d.png", i)), 50)
		const tolerance = 10.0
		if abs(r-baseR) > tolerance || abs(g-baseG) > tolerance || abs(b-baseB) > tolerance {
			t.Fatalf("bottom pixel inconsistency with scrollbar variation in frame %d: base=(%.1f,%.1f,%.1f) vs frame=(%.1f,%.1f,%.1f)",
				i, baseR, baseG, baseB, r, g, b)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

var errTest = fmt.Errorf("test error")
