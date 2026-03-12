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
//  1. scale: normalize all frames to fill target dimensions (handles
//     inconsistent capture sizes from viewport changes or CDP screencast)
//  2. crop: trim any overflow to exact target dimensions
//  3. drawtext (optional): semi-transparent watermark in bottom-right
//  4. format: convert to yuv420p for broad player compatibility
//
// Why increase+crop instead of decrease+pad:
// Frames captured via CDP screencast or polling can vary by 10-50px due to
// scrollbar appearance, browser chrome changes, or compositor timing. The
// previous approach (decrease+pad) left shorter frames at their original size
// and filled the gap with black, causing visible flickering at the bottom of
// the video. The increase+crop approach scales UP to fill the target so every
// frame covers the full area, then crops any slight overflow. The cropping is
// imperceptible (a few pixels at most) and eliminates flickering entirely.
func buildAssemblyFilterChain(watermarkText string, targetW, targetH int) string {
	// Scale all frames to fill the target dimensions. force_original_aspect_ratio=increase
	// ensures the smaller dimension is scaled to match the target, and the larger
	// dimension may slightly overflow. This guarantees no frame is ever smaller
	// than the target — eliminating black padding that caused flickering.
	filters := []string{
		fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase", targetW, targetH),
	}

	// Crop to exact target dimensions, centered. Any overflow from the increase
	// scaling is trimmed equally from both edges. For typical dimension
	// mismatches (10-50px from scrollbar/chrome changes), the cropped content
	// is imperceptible.
	filters = append(filters, fmt.Sprintf("crop=%d:%d:(iw-%d)/2:(ih-%d)/2", targetW, targetH, targetW, targetH))

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

// dimKey is a comparable key for width×height pairs.
type dimKey struct{ w, h int }

// probeModeDimensions samples multiple frames from a sequence and returns the
// most common (mode) dimensions. This prevents the first frame from dictating
// the target when it has atypical dimensions (e.g., browser info bar adding
// ~50px during startup). If no clear mode exists, the smallest dimensions are
// used to favour cropping (imperceptible) over scaling (causes content shift).
//
// Sampling strategy: probe frames 0–9 plus logarithmic samples beyond that.
// For short sequences this probes every frame; for long sequences it stays O(1).
func probeModeDimensions(pattern string) (width, height int, err error) {
	// Collect dimensions from available frames.
	counts := make(map[dimKey]int)
	var lastGood dimKey

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
		key := dimKey{w, h}
		counts[key]++
		lastGood = key
	}

	if len(counts) == 0 {
		return 0, 0, fmt.Errorf("no readable frames found matching pattern %s", pattern)
	}

	// If all sampled frames share the same dimensions, fast path.
	if len(counts) == 1 {
		return lastGood.w, lastGood.h, nil
	}

	// Pick the mode (most common dimensions). On ties, prefer smaller
	// dimensions so that outlier frames are cropped (imperceptible) rather
	// than scaled up (causes visible content shift / flicker).
	var best dimKey
	bestCount := 0
	for key, count := range counts {
		if count > bestCount ||
			(count == bestCount && (key.w*key.h < best.w*best.h)) {
			best = key
			bestCount = count
		}
	}

	return best.w, best.h, nil
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
// Uses scale+crop (not pad) to ensure even dimensions without adding black
// bars, consistent with AssembleVideoFromSequence.
func (e *FFmpegEncoder) ConvertToMP4(ctx context.Context, inputPath, outputPath string) error {
	// Scale to nearest even dimensions (ceil to even), then crop to exact.
	// This avoids the black-bar padding that caused flickering.
	vf := "scale=ceil(iw/2)*2:ceil(ih/2)*2:force_original_aspect_ratio=increase," +
		"crop=ceil(iw/2)*2:ceil(ih/2)*2:(iw-ceil(iw/2)*2)/2:(ih-ceil(ih/2)*2)/2," +
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
