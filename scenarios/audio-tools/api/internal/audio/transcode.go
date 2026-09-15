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

// ErrFfmpegExec is returned when ffmpeg is present but exits non-zero —
// almost always because the caller-supplied audio is unsupported/corrupt
// or the requested output format is invalid. It lets handlers map the
// failure to an actionable client-facing code (InvalidArgument) rather
// than flattening every ffmpeg rejection to a server-side Internal error.
// The underlying exec error (with ffmpeg's stderr tail) is wrapped so the
// caller still sees the concrete cause.
var ErrFfmpegExec = errors.New("audio: ffmpeg execution failed")

var (
	ffmpegOnce       sync.Once
	ffmpegAvailable  bool
	ffprobeOnce      sync.Once
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

// HasFfmpeg reports whether the ffmpeg binary is available on $PATH.
// The result is cached. The audioformat substrate uses this to gate its
// capability matrix and to choose the PCM fast-path over an ffmpeg decode.
func HasFfmpeg() bool { return hasFfmpeg() }

// HasFfprobe reports whether the ffprobe binary is available on $PATH.
// The result is cached.
func HasFfprobe() bool { return hasFfprobe() }

// seam: Runner is the ffmpeg-process seam (SEAMS.md row "audio.Runner").
// Production wires audio.DefaultRunner; tests wire mocks.FakeRunner.
//
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

// SetFfmpegAvailableForTest seeds the package-private ffmpeg/ffprobe
// presence cache. Tests in other packages can use this together with
// swapping DefaultRunner to exercise runFfmpeg/runFfprobe paths without
// requiring the binaries on $PATH. Returns a restorer the caller must
// invoke (typically via t.Cleanup) to put the cache back; the package
// is greenfield and never auto-rolls-back the cell.
//
// This is the documented seam referenced from docs/internal/SEAMS.md
// under "audio handler tests".
func SetFfmpegAvailableForTest(ffmpeg, ffprobe bool) func() {
	ffmpegOnce.Do(func() {})
	ffprobeOnce.Do(func() {})
	prevFfmpeg, prevFfprobe := ffmpegAvailable, ffprobeAvailable
	ffmpegAvailable = ffmpeg
	ffprobeAvailable = ffprobe
	return func() {
		ffmpegAvailable = prevFfmpeg
		ffprobeAvailable = prevFfprobe
	}
}

// runFfmpeg runs ffmpeg with the given args; stdin is the input payload.
// Returns stdout (the transformed audio) or an error joining the exit
// status and stderr tail.
func runFfmpeg(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	if !hasFfmpeg() {
		return nil, ErrFFmpegMissing
	}
	full := append([]string{"-y", "-loglevel", "error"}, args...)
	out, err := DefaultRunner.Run(ctx, "ffmpeg", input, full...)
	if err != nil {
		// ffmpeg is installed but rejected the job. Wrap with the
		// ErrFfmpegExec sentinel (keeping the underlying cause in the
		// chain) so handlers can surface an actionable InvalidArgument
		// instead of a bare Internal error.
		return nil, fmt.Errorf("%w: %w", ErrFfmpegExec, err)
	}
	return out, nil
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
