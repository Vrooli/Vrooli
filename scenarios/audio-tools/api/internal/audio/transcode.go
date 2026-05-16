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

// runFfmpeg runs ffmpeg with the given args; stdin is the input payload.
// Returns stdout (the transformed audio) or an error joining the exit
// status and stderr tail.
func runFfmpeg(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	if !hasFfmpeg() {
		return nil, ErrFFmpegMissing
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", append([]string{"-y", "-loglevel", "error"}, args...)...)
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
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
