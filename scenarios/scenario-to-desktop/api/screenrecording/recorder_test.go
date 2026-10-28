package screenrecording

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestNativeCaptureCancellationStopsAndReapsInterruptIgnoringRecorder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("X11 recorder cancellation uses Unix signals")
	}
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			dir := t.TempDir()
			command := filepath.Join(dir, "ffmpeg")
			if err := os.WriteFile(command, []byte("#!/bin/sh\ntrap '' INT\nfor output; do :; done\nprintf partial > \"$output\"\nexec sleep 300\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			recorder := NewNativeFFmpegRecorder(command)
			id, err := recorder.StartCapture(context.Background(), CaptureConfig{Display: ":99", Width: 640, Height: 480, FPS: 10, OutputPath: filepath.Join(dir, "capture.mp4")})
			if err != nil {
				t.Fatal(err)
			}
			session := recorder.sessions[id]
			t.Cleanup(func() {
				if session.cmd.Process.Kill() == nil {
					select {
					case <-session.done:
					case <-time.After(2 * time.Second):
						t.Error("fixture recorder did not exit")
					}
				}
			})
			ctx, cancel := context.WithCancel(context.Background())
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			defer cancel()
			if _, err := recorder.StopCapture(ctx, id); !errors.Is(err, want) {
				t.Errorf("stop error = %v; want %v", err, want)
			}
			if err := session.cmd.Process.Signal(syscall.Signal(0)); !errors.Is(err, os.ErrProcessDone) {
				t.Errorf("recorder still alive or unreaped after cancellation: %v", err)
			}
		})
	}
}

// mockExecutor implements CommandExecutor for tests.
type mockExecutor struct {
	executeResult *ExecutionResult
	executeErr    error
	command       string
	args          []string
	timeout       time.Duration
}

func (m *mockExecutor) ExecuteWithResult(_ context.Context, _, command string, args, _ []string, timeout time.Duration) (*ExecutionResult, error) {
	m.command, m.args, m.timeout = command, append([]string(nil), args...), timeout
	return m.executeResult, m.executeErr
}

func TestStartCapture_Success(t *testing.T) {
	executor := &mockExecutor{
		executeResult: &ExecutionResult{
			Stdout:   "rec-20260318120000-12345\n",
			ExitCode: 0,
		},
	}

	r := NewRecorder(executor)
	id, err := r.StartCapture(context.Background(), CaptureConfig{
		Display: ":99",
		Width:   1920,
		Height:  1080,
		FPS:     30,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "rec-20260318120000-12345" {
		t.Fatalf("unexpected capture ID: %s", id)
	}
}

func TestStartCapture_CLIError(t *testing.T) {
	executor := &mockExecutor{
		executeResult: &ExecutionResult{
			Stderr:   "x11grab not available",
			ExitCode: 1,
		},
	}

	r := NewRecorder(executor)
	_, err := r.StartCapture(context.Background(), CaptureConfig{
		Display: ":99",
		Width:   640,
		Height:  480,
		FPS:     10,
	})

	if err == nil {
		t.Fatal("expected error from failed CLI")
	}
}

func TestStartCapture_ExecutionFails(t *testing.T) {
	executor := &mockExecutor{
		executeErr: fmt.Errorf("command not found: resource-ffmpeg"),
	}

	r := NewRecorder(executor)
	_, err := r.StartCapture(context.Background(), CaptureConfig{
		Display: ":99",
		Width:   640,
		Height:  480,
		FPS:     10,
	})

	if err == nil {
		t.Fatal("expected error when executor fails")
	}
}

func TestStopCapture_Success(t *testing.T) {
	executor := &mockExecutor{
		executeResult: &ExecutionResult{
			Stdout:   "/data/screen-captures/rec-123.mp4\n",
			ExitCode: 0,
		},
	}

	r := NewRecorder(executor)
	result, err := r.StopCapture(context.Background(), "rec-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.VideoPath != "/data/screen-captures/rec-123.mp4" {
		t.Fatalf("unexpected video path: %s", result.VideoPath)
	}
}

func TestStopCapture_CLIError(t *testing.T) {
	executor := &mockExecutor{
		executeResult: &ExecutionResult{
			Stderr:   "recording not found",
			ExitCode: 1,
		},
	}

	r := NewRecorder(executor)
	_, err := r.StopCapture(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("expected error for nonexistent recording")
	}
}

func TestStartCapture_WithOutputPath(t *testing.T) {
	executor := &mockExecutor{
		executeResult: &ExecutionResult{
			Stdout:   "rec-custom\n",
			ExitCode: 0,
		},
	}

	r := NewRecorder(executor)
	id, err := r.StartCapture(context.Background(), CaptureConfig{
		Display:    ":99",
		Width:      640,
		Height:     480,
		FPS:        10,
		OutputPath: "/tmp/test.mp4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "rec-custom" {
		t.Fatalf("unexpected capture ID: %s", id)
	}
	if executor.command != "resource-ffmpeg" || executor.timeout != 30*time.Second || fmt.Sprint(executor.args) != "[screen-capture start --display :99 --resolution 640x480 --framerate 10 --output /tmp/test.mp4]" {
		t.Fatalf("capture command = %q %v (timeout %v)", executor.command, executor.args, executor.timeout)
	}
}

func TestRecorderReturnsDiagnosticsAndRejectsMissingCaptureID(t *testing.T) {
	t.Run("start error preserves stderr", func(t *testing.T) {
		r := NewRecorder(&mockExecutor{executeResult: &ExecutionResult{Stderr: "permission denied", ExitCode: 17}, executeErr: fmt.Errorf("exit status 17")})
		if _, err := r.StartCapture(context.Background(), CaptureConfig{}); err == nil || err.Error() != "screen capture start failed (exit code 17): permission denied" {
			t.Fatalf("start error = %v", err)
		}
	})
	t.Run("missing id", func(t *testing.T) {
		r := NewRecorder(&mockExecutor{executeResult: &ExecutionResult{ExitCode: 0}})
		if _, err := r.StartCapture(context.Background(), CaptureConfig{}); err == nil {
			t.Fatal("expected missing capture ID error")
		}
	})
	t.Run("stop error preserves stderr", func(t *testing.T) {
		r := NewRecorder(&mockExecutor{executeResult: &ExecutionResult{Stderr: "not found", ExitCode: 3}, executeErr: fmt.Errorf("exit status 3")})
		if _, err := r.StopCapture(context.Background(), "missing"); err == nil || err.Error() != "screen capture stop failed (exit code 3): not found" {
			t.Fatalf("stop error = %v", err)
		}
	})
}
