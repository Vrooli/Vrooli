package screenrecording

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// NewSystemRecorder uses the Vrooli FFmpeg resource when it is available. A
// direct ffmpeg implementation is the supported fallback for desktop smoke
// tests on hosts where ffmpeg is installed by the operating system but no
// resource-ffmpeg wrapper has been provisioned yet.
func NewSystemRecorder(executor CommandExecutor) Recorder {
	if _, err := exec.LookPath("resource-ffmpeg"); err == nil {
		return NewRecorder(executor)
	}
	return NewNativeFFmpegRecorder("ffmpeg")
}

// NativeFFmpegRecorder owns direct ffmpeg processes for X11 capture.
// It deliberately has the same start/stop lifecycle as resource-ffmpeg so
// callers do not need environment-specific recording code.
type NativeFFmpegRecorder struct {
	command  string
	mu       sync.Mutex
	nextID   uint64
	sessions map[string]*nativeSession
}

type nativeSession struct {
	cmd    *exec.Cmd
	done   chan error
	output string
}

func NewNativeFFmpegRecorder(command string) *NativeFFmpegRecorder {
	return &NativeFFmpegRecorder{command: command, sessions: make(map[string]*nativeSession)}
}

func (r *NativeFFmpegRecorder) StartCapture(ctx context.Context, cfg CaptureConfig) (string, error) {
	if cfg.Display == "" || cfg.Width <= 0 || cfg.Height <= 0 || cfg.FPS <= 0 || cfg.OutputPath == "" {
		return "", fmt.Errorf("direct ffmpeg capture requires display, positive resolution/FPS, and output path")
	}
	if _, err := exec.LookPath(r.command); err != nil {
		return "", fmt.Errorf("find ffmpeg capture command: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0o755); err != nil {
		return "", fmt.Errorf("create recording directory: %w", err)
	}
	args := []string{"-y", "-video_size", fmt.Sprintf("%dx%d", cfg.Width, cfg.Height), "-framerate", fmt.Sprint(cfg.FPS), "-f", "x11grab", "-i", cfg.Display, "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", cfg.OutputPath}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), r.command, args...)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start direct ffmpeg capture: %w", err)
	}
	session := &nativeSession{cmd: cmd, done: make(chan error, 1), output: cfg.OutputPath}
	go func() { session.done <- cmd.Wait() }()

	// Detect immediate failures (invalid display/codec) before reporting a
	// capture ID. A running capture is intentionally allowed to continue.
	select {
	case err := <-session.done:
		if err == nil {
			return "", fmt.Errorf("direct ffmpeg capture exited before recording began")
		}
		return "", fmt.Errorf("start direct ffmpeg capture: %w", err)
	case <-time.After(250 * time.Millisecond):
	}

	r.mu.Lock()
	r.nextID++
	id := fmt.Sprintf("native-ffmpeg-%d", r.nextID)
	r.sessions[id] = session
	r.mu.Unlock()
	return id, nil
}

func (r *NativeFFmpegRecorder) StopCapture(ctx context.Context, captureID string) (*CaptureResult, error) {
	r.mu.Lock()
	session, ok := r.sessions[captureID]
	if ok {
		delete(r.sessions, captureID)
	}
	r.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("recording not found: %s", captureID)
	}
	if err := session.cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return nil, fmt.Errorf("stop direct ffmpeg capture: %w", err)
	}
	select {
	case err := <-session.done:
		// ffmpeg commonly reports 255 after the interrupt used to finalize an
		// x11grab recording. The output file is the authoritative completion
		// signal and is validated below.
		_ = err
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for direct ffmpeg capture: %w", ctx.Err())
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("timed out stopping direct ffmpeg capture")
	}
	info, err := os.Stat(session.output)
	if err != nil {
		return nil, fmt.Errorf("recorded video unavailable: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("recorded video is empty")
	}
	return &CaptureResult{VideoPath: session.output, FileSizeBytes: info.Size()}, nil
}
