// Package screenrecording provides desktop screen capture for smoke test validation.
// It shells out to the FFmpeg resource's screen-capture CLI for actual recording,
// making the FFmpeg resource the single source of truth for capture implementation.
package screenrecording

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Recorder manages screen capture sessions via the FFmpeg resource CLI.
type Recorder interface {
	// StartCapture begins recording the given display.
	StartCapture(ctx context.Context, cfg CaptureConfig) (captureID string, err error)
	// StopCapture ends a recording and returns the result.
	StopCapture(ctx context.Context, captureID string) (*CaptureResult, error)
}

// CaptureConfig holds parameters for a screen capture session.
type CaptureConfig struct {
	Display    string // X display, e.g. ":99"
	Width      int
	Height     int
	FPS        int
	OutputPath string
}

// CaptureResult holds the outcome of a completed capture.
type CaptureResult struct {
	VideoPath     string
	DurationMs    int64
	FileSizeBytes int64
}

// ExecutionResult mirrors the relevant fields from smoketest.ExecutionResult
// to avoid an import cycle between screenrecording and smoketest.
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// CommandExecutor abstracts command execution. This is a local interface
// to avoid importing smoketest (which imports screenrecording).
// smoketest.ProcessExecutor satisfies this interface.
type CommandExecutor interface {
	ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*ExecutionResult, error)
}

// FFmpegRecorder implements Recorder by shelling out to `resource-ffmpeg screen-capture`.
type FFmpegRecorder struct {
	executor CommandExecutor
}

// NewRecorder creates a new FFmpegRecorder.
func NewRecorder(executor CommandExecutor) *FFmpegRecorder {
	return &FFmpegRecorder{executor: executor}
}

// StartCapture starts a screen recording via the FFmpeg resource CLI.
func (r *FFmpegRecorder) StartCapture(ctx context.Context, cfg CaptureConfig) (string, error) {
	args := []string{
		"screen-capture", "start",
		"--display", cfg.Display,
		"--resolution", fmt.Sprintf("%dx%d", cfg.Width, cfg.Height),
		"--framerate", strconv.Itoa(cfg.FPS),
	}
	if cfg.OutputPath != "" {
		args = append(args, "--output", cfg.OutputPath)
	}

	result, err := r.executor.ExecuteWithResult(ctx, "", "resource-ffmpeg", args, nil, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("screen capture start failed: %w", err)
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("screen capture start returned exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// The CLI prints the recording ID to stdout.
	captureID := strings.TrimSpace(result.Stdout)
	if captureID == "" {
		return "", fmt.Errorf("screen capture start returned no recording ID")
	}
	return captureID, nil
}

// StopCapture stops a recording and parses the output path and duration.
func (r *FFmpegRecorder) StopCapture(ctx context.Context, captureID string) (*CaptureResult, error) {
	args := []string{"screen-capture", "stop", captureID}

	result, err := r.executor.ExecuteWithResult(ctx, "", "resource-ffmpeg", args, nil, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("screen capture stop failed: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("screen capture stop returned exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// The CLI prints the output path to stdout.
	videoPath := strings.TrimSpace(result.Stdout)

	return &CaptureResult{
		VideoPath: videoPath,
	}, nil
}
