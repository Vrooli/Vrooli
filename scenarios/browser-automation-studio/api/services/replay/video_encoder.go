package replay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

// VideoEncoder abstracts video encoding operations.
// This seam enables testing replay rendering without requiring FFmpeg.
type VideoEncoder interface {
	// AssembleVideoFromSequence compiles a sequence of image frames into an MP4 video.
	// pattern is a printf-style path like "frames/frame-%05d.jpg"
	// fps is the target frames per second
	// outputPath is the destination file path
	AssembleVideoFromSequence(ctx context.Context, pattern string, fps int, outputPath string) error

	// ConvertToGIF converts a video file to an animated GIF.
	// inputPath is the source video file
	// outputPath is the destination GIF file
	// targetWidth is the desired output width (height scales proportionally)
	// fps is the target frame rate for the GIF
	ConvertToGIF(ctx context.Context, inputPath, outputPath string, targetWidth int, fps int) error
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
	if fps <= 0 {
		fps = 25
	}
	args := []string{
		"-y",
		"-framerate", strconv.Itoa(fps),
		"-start_number", "0",
		"-i", pattern,
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p",
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

// Compile-time interface enforcement
var _ VideoEncoder = (*FFmpegEncoder)(nil)

// MockVideoEncoder is a test double for VideoEncoder.
type MockVideoEncoder struct {
	AssembleVideoErr error
	ConvertGIFErr    error

	AssembleCalls []AssembleCall
	ConvertCalls  []ConvertCall
}

// AssembleCall records arguments to AssembleVideoFromSequence.
type AssembleCall struct {
	Pattern    string
	FPS        int
	OutputPath string
}

// ConvertCall records arguments to ConvertToGIF.
type ConvertCall struct {
	InputPath   string
	OutputPath  string
	TargetWidth int
	FPS         int
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

// Compile-time interface enforcement
var _ VideoEncoder = (*MockVideoEncoder)(nil)
