package render

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder for probeImageDimensions
	_ "image/png"  // register PNG decoder for probeImageDimensions
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// VideoEncoder abstracts video encoding operations.
// This seam enables testing replay rendering without requiring FFmpeg.
type VideoEncoder interface {
	// AssembleVideoFromSequence compiles a sequence of image frames into an MP4 video.
	// pattern is a printf-style path like "frames/frame-%05d.jpg"
	// fps is the target frames per second
	// outputPath is the destination file path
	AssembleVideoFromSequence(ctx context.Context, pattern string, fps int, outputPath string) error

	// AssembleVideoWithWatermark compiles frames into video with a watermark overlay.
	// watermarkText is displayed in the corner of the video.
	AssembleVideoWithWatermark(ctx context.Context, pattern string, fps int, outputPath string, watermarkText string) error

	// ConvertToGIF converts a video file to an animated GIF.
	// inputPath is the source video file
	// outputPath is the destination GIF file
	// targetWidth is the desired output width (height scales proportionally)
	// fps is the target frame rate for the GIF
	ConvertToGIF(ctx context.Context, inputPath, outputPath string, targetWidth int, fps int) error

	// ConvertToMP4 transcodes a video file to MP4 (H.264).
	ConvertToMP4(ctx context.Context, inputPath, outputPath string) error
}

// FFmpegEncoder implements VideoEncoder using the FFmpeg CLI.
type FFmpegEncoder struct {
	ffmpegPath string
}

// NewFFmpegEncoder creates a new FFmpeg-based video encoder.
func NewFFmpegEncoder(ffmpegPath string) *FFmpegEncoder {
	return &FFmpegEncoder{ffmpegPath: ffmpegPath}
}

// AssembleVideoFromSequence uses ffmpeg to compile a sequence of frames into an MP4 video.
func (e *FFmpegEncoder) AssembleVideoFromSequence(ctx context.Context, pattern string, fps int, outputPath string) error {
	return e.assembleVideo(ctx, pattern, fps, outputPath, "")
}

// AssembleVideoWithWatermark compiles frames into video with a watermark overlay.
func (e *FFmpegEncoder) AssembleVideoWithWatermark(ctx context.Context, pattern string, fps int, outputPath string, watermarkText string) error {
	return e.assembleVideo(ctx, pattern, fps, outputPath, watermarkText)
}

// assembleVideo is the internal implementation that optionally adds a watermark.
func (e *FFmpegEncoder) assembleVideo(ctx context.Context, pattern string, fps int, outputPath string, watermarkText string) error {
	if fps <= 0 {
		fps = 25
	}

	// Probe multiple frames to determine the most common (mode) dimensions.
	// Using only frame-00000 caused bottom-of-frame flickering when the first
	// frame had atypical dimensions (e.g., Chrome "controlled by automation"
	// info bar adds ~50px height). By sampling several frames and picking the
	// mode, we ensure outlier frames are normalized instead of dictating the
	// target.
	targetW, targetH, probeErr := probeModeDimensions(pattern)
	if probeErr != nil {
		return fmt.Errorf("failed to probe frame dimensions: %w", probeErr)
	}

	// Round up to even dimensions for H.264 compatibility.
	targetW = ceilEven(targetW)
	targetH = ceilEven(targetH)

	vf := buildAssemblyFilterChain(watermarkText, targetW, targetH)

	args := []string{
		"-y",
		"-framerate", strconv.Itoa(fps),
		"-start_number", "0",
		"-i", pattern,
		"-vf", vf,
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "21",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg sequence assembly failed: %w (%s)", err, stderr.String())
	}
	return nil
}

// buildAssemblyFilterChain constructs the FFmpeg video filter chain for frame
// sequence assembly. The chain normalizes inconsistent frame dimensions (which
// cause bottom-of-frame flickering), ensures H.264-compatible even dimensions,
// and optionally overlays a watermark.
//
// targetW and targetH must be positive even integers (use ceilEven to prepare).
//
// Filter order:
//  1. crop (height only): remove top overflow from browser chrome (info bars)
//     without any scaling — preserving pixel-perfect vertical alignment
//  2. scale: force exact target dimensions — handles width variation (scrollbar
//     ±17px) via imperceptible horizontal stretch without coupling to height
//  3. drawtext (optional): semi-transparent watermark in bottom-right
//  4. format: convert to yuv420p for broad player compatibility
//
// Why crop-then-force-scale instead of scale=increase+crop:
// The previous approach used scale with force_original_aspect_ratio=increase,
// which coupled width and height: when a scrollbar appeared (reducing width by
// ~17px), the frame was scaled UP proportionally, increasing height by ~10px.
// The bottom-anchored crop then removed the top 10px, but the scaled content
// was shifted DOWN relative to non-scrollbar frames — causing ~10px of content
// at the bottom to alternate between visible and invisible (flickering).
//
// The new approach decouples the axes:
//   - Height variation (info bars, ~50px at top) is handled by CROPPING from
//     the top (pixel-perfect, no content shift)
//   - Width variation (scrollbar, ~17px) is handled by force-scaling to exact
//     target width (~1.3% stretch, imperceptible, no height coupling)
//   - Frames shorter than target are scaled up vertically (rare, <2% stretch)
func buildAssemblyFilterChain(watermarkText string, targetW, targetH int) string {
	// Crop height only: remove top overflow (info bars, browser chrome) without
	// any scaling. Bottom-anchored so page content stays aligned at the bottom.
	// For frames at target height: ih-targetH=0, min(ih,targetH)=targetH → no-op.
	// For taller frames (info bar): removes top overflow, keeps page content.
	// For shorter frames: min(ih,targetH)=ih, offset=0 → no-op (scale handles it).
	// Width is NOT cropped — force scaling handles width variation instead.
	//
	// Note: commas within FFmpeg expressions must be escaped as \, to avoid
	// being parsed as filter separators.
	filters := []string{
		fmt.Sprintf("crop=iw:min(ih\\,%d):0:max(ih-%d\\,0)", targetH, targetH),
	}

	// Force scale to exact target dimensions. This handles width variation
	// (scrollbar ±17px) via slight horizontal stretch (~1.3%, imperceptible)
	// WITHOUT coupling width changes to height — eliminating the vertical
	// content shift that caused bottom-of-frame flickering. Frames shorter
	// than target are stretched vertically (rare edge case, <2% distortion).
	filters = append(filters, fmt.Sprintf("scale=%d:%d", targetW, targetH))

	if watermarkText != "" {
		escaped := escapeFFmpegText(watermarkText)
		filters = append(filters, fmt.Sprintf(
			"drawtext=text='%s':fontsize=16:fontcolor=white@0.5:shadowcolor=black@0.5:shadowx=1:shadowy=1:x=w-tw-10:y=h-th-10",
			escaped,
		))
	}

	filters = append(filters, "format=yuv420p")
	return strings.Join(filters, ",")
}

// probeImageDimensions reads the width and height from an image file without
// decoding the full pixel data. Supports JPEG and PNG.
func probeImageDimensions(path string) (width, height int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// probeModeDimensions samples multiple frames from a sequence and returns the
// most common (mode) width and height, computed independently. This prevents
// the first frame from dictating the target when it has atypical dimensions
// (e.g., browser info bar adding ~50px height during startup, or scrollbar
// changing width by ~17px).
//
// Width and height modes are computed separately because they vary from
// independent sources: width changes from scrollbar appearance/disappearance,
// while height changes from browser chrome (info bars). Computing them as a
// joint (w,h) pair can pick outlier dimensions when both vary simultaneously.
//
// On ties, the smaller dimension wins to favour cropping (imperceptible) over
// scaling up (causes visible content shift / flicker).
//
// Sampling strategy: probe frames 0–9 plus logarithmic samples beyond that.
// For short sequences this probes every frame; for long sequences it stays O(1).
func probeModeDimensions(pattern string) (width, height int, err error) {
	// Collect dimensions from available frames — independently for each axis.
	widthCounts := make(map[int]int)
	heightCounts := make(map[int]int)
	var lastW, lastH int
	total := 0

	// Always try the first 10 frames, then sample at exponential intervals.
	indices := make([]int, 0, 20)
	for i := 0; i < 10; i++ {
		indices = append(indices, i)
	}
	for step := 10; step < 100000; step *= 2 {
		indices = append(indices, step)
	}

	for _, idx := range indices {
		path := fmt.Sprintf(pattern, idx)
		w, h, probeErr := probeImageDimensions(path)
		if probeErr != nil {
			if idx == 0 {
				// First frame must exist — fail fast.
				return 0, 0, fmt.Errorf("first frame (%s): %w", path, probeErr)
			}
			// Frame doesn't exist (past end of sequence) or is corrupt — skip.
			continue
		}
		widthCounts[w]++
		heightCounts[h]++
		lastW = w
		lastH = h
		total++
	}

	if total == 0 {
		return 0, 0, fmt.Errorf("no readable frames found matching pattern %s", pattern)
	}

	return modeValue(widthCounts, lastW), modeValue(heightCounts, lastH), nil
}

// modeValue returns the most common value from counts. On ties, the smaller
// value wins (cropping a larger frame is imperceptible; scaling up a smaller
// frame causes visible content shift). fallback is returned when counts is
// empty (shouldn't happen in practice).
func modeValue(counts map[int]int, fallback int) int {
	if len(counts) == 1 {
		return fallback
	}
	best := 0
	bestCount := 0
	for val, count := range counts {
		if count > bestCount ||
			(count == bestCount && val < best) {
			best = val
			bestCount = count
		}
	}
	if bestCount == 0 {
		return fallback
	}
	return best
}

// ceilEven rounds n up to the nearest even integer.
// H.264 requires even dimensions for both width and height.
func ceilEven(n int) int {
	return (n + 1) &^ 1
}

// escapeFFmpegText escapes special characters for FFmpeg drawtext filter.
func escapeFFmpegText(text string) string {
	// FFmpeg drawtext requires escaping: ' : \
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		`:`, `\:`,
	)
	return replacer.Replace(text)
}

// ConvertToGIF uses ffmpeg to convert a video file to an animated GIF.
func (e *FFmpegEncoder) ConvertToGIF(ctx context.Context, inputPath, outputPath string, targetWidth int, fps int) error {
	if fps <= 0 {
		fps = 12
	}
	if targetWidth <= 0 {
		targetWidth = defaultPresentationWidth
	}
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos", fps, targetWidth),
		outputPath,
	}
	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg gif conversion failed: %w (%s)", err, stderr.String())
	}
	return nil
}

// ConvertToMP4 transcodes a video file to MP4 (H.264).
// Rounds dimensions up to even (H.264 requirement) without using
// force_original_aspect_ratio or pad — both of which cause flickering
// artifacts (see buildAssemblyFilterChain for the full explanation).
func (e *FFmpegEncoder) ConvertToMP4(ctx context.Context, inputPath, outputPath string) error {
	// Round each dimension up to even independently. For already-even input
	// (the common case) this is a no-op. For odd input, the ≤1px stretch is
	// imperceptible and avoids the cross-dimensional coupling that
	// force_original_aspect_ratio=increase introduces.
	vf := "scale=ceil(iw/2)*2:ceil(ih/2)*2," +
		"format=yuv420p"
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", vf,
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "21",
		outputPath,
	}
	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg mp4 transcode failed: %w (%s)", err, stderr.String())
	}
	return nil
}

// Compile-time interface enforcement
var _ VideoEncoder = (*FFmpegEncoder)(nil)

// MockVideoEncoder is a test double for VideoEncoder.
type MockVideoEncoder struct {
	AssembleVideoErr error
	ConvertGIFErr    error
	ConvertMP4Err    error

	AssembleCalls          []AssembleCall
	AssembleWatermarkCalls []AssembleWatermarkCall
	ConvertCalls           []ConvertCall
	ConvertMP4Calls        []ConvertMP4Call
}

// AssembleCall records arguments to AssembleVideoFromSequence.
type AssembleCall struct {
	Pattern    string
	FPS        int
	OutputPath string
}

// AssembleWatermarkCall records arguments to AssembleVideoWithWatermark.
type AssembleWatermarkCall struct {
	Pattern       string
	FPS           int
	OutputPath    string
	WatermarkText string
}

// ConvertCall records arguments to ConvertToGIF.
type ConvertCall struct {
	InputPath   string
	OutputPath  string
	TargetWidth int
	FPS         int
}

// ConvertMP4Call records arguments to ConvertToMP4.
type ConvertMP4Call struct {
	InputPath  string
	OutputPath string
}

// AssembleVideoFromSequence records the call and returns the configured error.
func (m *MockVideoEncoder) AssembleVideoFromSequence(_ context.Context, pattern string, fps int, outputPath string) error {
	m.AssembleCalls = append(m.AssembleCalls, AssembleCall{
		Pattern:    pattern,
		FPS:        fps,
		OutputPath: outputPath,
	})
	return m.AssembleVideoErr
}

// AssembleVideoWithWatermark records the call and returns the configured error.
func (m *MockVideoEncoder) AssembleVideoWithWatermark(_ context.Context, pattern string, fps int, outputPath string, watermarkText string) error {
	m.AssembleWatermarkCalls = append(m.AssembleWatermarkCalls, AssembleWatermarkCall{
		Pattern:       pattern,
		FPS:           fps,
		OutputPath:    outputPath,
		WatermarkText: watermarkText,
	})
	return m.AssembleVideoErr
}

// ConvertToGIF records the call and returns the configured error.
func (m *MockVideoEncoder) ConvertToGIF(_ context.Context, inputPath, outputPath string, targetWidth int, fps int) error {
	m.ConvertCalls = append(m.ConvertCalls, ConvertCall{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		TargetWidth: targetWidth,
		FPS:         fps,
	})
	return m.ConvertGIFErr
}

// ConvertToMP4 records the call and returns the configured error.
func (m *MockVideoEncoder) ConvertToMP4(_ context.Context, inputPath, outputPath string) error {
	m.ConvertMP4Calls = append(m.ConvertMP4Calls, ConvertMP4Call{
		InputPath:  inputPath,
		OutputPath: outputPath,
	})
	return m.ConvertMP4Err
}

// Compile-time interface enforcement
var _ VideoEncoder = (*MockVideoEncoder)(nil)
