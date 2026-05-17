// Package audio provides ffmpeg-backed transformations. All functions
// require ffmpeg on $PATH; when it's missing they return
// ErrFFmpegMissing so handlers can map to connect.CodeFailedPrecondition.
// No silent passthrough — greenfield contract.
package audio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
)

// ErrFFmpegMissing is returned when the ffmpeg binary cannot be found.
var ErrFFmpegMissing = errors.New("audio: ffmpeg not installed")

var (
	ffmpegOnce      sync.Once
	ffmpegAvailable bool
	ffprobeOnce     sync.Once
	ffprobeAvailable bool
)

func hasFfmpeg() bool {
	ffmpegOnce.Do(func() {
		_, err := exec.LookPath("ffmpeg")
		ffmpegAvailable = err == nil
	})
	return ffmpegAvailable
}

func hasFfprobe() bool {
	ffprobeOnce.Do(func() {
		_, err := exec.LookPath("ffprobe")
		ffprobeAvailable = err == nil
	})
	return ffprobeAvailable
}

// Runner is the seam tests substitute for ffmpeg/ffprobe invocation.
// Production uses os/exec; tests inject a fake to capture argv and
// inject canned stdout/err pairs without depending on a binary.
type Runner interface {
	Run(ctx context.Context, name string, stdin []byte, args ...string) (stdout []byte, err error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// DefaultRunner is the production Runner used by every Transcode/Trim/
// Volume/etc. call. Tests overwrite it (via t.Cleanup) to substitute a
// fake without touching the real ffmpeg binary.
var DefaultRunner Runner = execRunner{}

// runFfmpeg runs ffmpeg with the given args; stdin is the input payload.
// Returns stdout (the transformed audio) or an error joining the exit
// status and stderr tail.
func runFfmpeg(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	if !hasFfmpeg() {
		return nil, ErrFFmpegMissing
	}
	full := append([]string{"-y", "-loglevel", "error"}, args...)
	return DefaultRunner.Run(ctx, "ffmpeg", input, full...)
}

// Transcode pipes audio through ffmpeg to produce the requested
// sample rate / channels / format. Sensible defaults: 16 kHz mono WAV.
func Transcode(ctx context.Context, audio []byte) ([]byte, error) {
	return runFfmpeg(ctx, audio,
		"-i", "pipe:0",
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		"pipe:1",
	)
}

// TranscodeOpts lets handlers parameterise sample-rate/channels/bitrate.
func TranscodeOpts(ctx context.Context, audio []byte, format string, sampleRate, channels, bitrate int) ([]byte, error) {
	if format == "" {
		format = "wav"
	}
	args := []string{"-i", "pipe:0"}
	if sampleRate > 0 {
		args = append(args, "-ar", fmt.Sprint(sampleRate))
	}
	if channels > 0 {
		args = append(args, "-ac", fmt.Sprint(channels))
	}
	if bitrate > 0 {
		args = append(args, "-b:a", fmt.Sprint(bitrate))
	}
	args = append(args, "-f", format, "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}
