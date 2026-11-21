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
	"strings"
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

// ErrInvalidArgument marks requests the caller can fix: negative or
// non-finite time windows, empty merge sources, unsorted split
// boundaries, and similar parameter mistakes. Handlers map it to
// connect.CodeInvalidArgument instead of the opaque Internal error.
var ErrInvalidArgument = errors.New("audio: invalid argument")

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

// StderrRunner is the optional Runner extension for invocations whose real
// output is diagnostic text on stderr (loudnorm reports its pass-1/pass-2
// measurements there). Runner's stdout-only contract would discard it, so
// runners that do not implement StderrRunner simply produce no measurement;
// normalization treats that as unknown rather than fatal.
type StderrRunner interface {
	RunWithStderr(ctx context.Context, name string, stdin []byte, args ...string) (stdout, stderr []byte, err error)
}

type execRunner struct{}

func (r execRunner) Run(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	stdout, _, err := r.RunWithStderr(ctx, name, stdin, args...)
	return stdout, err
}

func (execRunner) RunWithStderr(ctx context.Context, name string, stdin []byte, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s: %w: %s", name, err, stderr.String())
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

// DefaultRunner is the production Runner used by every Transcode/Trim/
// Volume/etc. call. Tests overwrite it (via t.Cleanup) to substitute a
// fake without touching the real ffmpeg binary.
var DefaultRunner Runner = execRunner{}

// SetFfmpegAvailableForTest seeds the package-private ffmpeg/ffprobe
// presence cache. Tests in other packages can use this together with
// swapping DefaultRunner to exercise runFfmpeg/runFfprobe paths without
// requiring the binaries on $PATH. Returns a restorer the caller must
// invoke (typically via t.Cleanup) to put the cache back.
//
// This is the documented seam referenced from docs/internal/SEAMS.md
// under "audio handler tests".
//
// The capability cache is force-detected before snapshotting so the
// restorer returns the host's real binaries. Consuming the sync.Once
// without probing would pin "missing" for every later real-binary test.
func SetFfmpegAvailableForTest(ffmpeg, ffprobe bool) func() {
	_ = hasFfmpeg()
	_ = hasFfprobe()
	prevFfmpeg, prevFfprobe := ffmpegAvailable, ffprobeAvailable
	ffmpegAvailable = ffmpeg
	ffprobeAvailable = ffprobe
	return func() {
		ffmpegAvailable = prevFfmpeg
		ffprobeAvailable = prevFfprobe
	}
}

// ffmpegBaseArgs are the global flags every invocation carries:
//
//   - -nostdin keeps ffmpeg from competing with `-i pipe:0` for the
//     process's standard input (interactive command parsing). Required
//     for pipe-driven use on headless hosts and Windows.
//   - -hide_banner suppresses the build banner so headless/CI logs (and
//     the stderr tail we attach to errors) stay readable.
//   - -y overwrites outputs without prompting (defensive: today we write
//     to pipe:1, but any future temp-file output path must not block on a
//     TTY question).
//   - -loglevel error keeps the captured stderr terse.
var ffmpegBaseArgs = []string{"-nostdin", "-hide_banner", "-y", "-loglevel", "error"}

// runFfmpeg runs ffmpeg with the given args; stdin is the input payload.
// Returns stdout (the transformed audio) or an error joining the exit
// status and stderr tail.
//
// A transform that exits 0 but emits nothing is a failure: some ffmpeg
// filter graphs (notably an acrossfade longer than the source) report
// success while writing an empty stream. Returning that as a successful
// transform would hand the caller silent data loss.
func runFfmpeg(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	out, stderr, err := runFfmpegAt(ctx, input, "error", args...)
	if err != nil {
		return out, err
	}
	if len(out) == 0 {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = "no diagnostic output"
		}
		return nil, fmt.Errorf("%w: ffmpeg produced empty audio (%s)", ErrFfmpegExec, detail)
	}
	return out, nil
}

// runFfmpegCapture is runFfmpeg plus ffmpeg's stderr, which loudnorm uses
// for its measurement report. Runners that do not implement StderrRunner
// yield a nil stderr instead of an error; callers treat measurement as
// best-effort.
func runFfmpegCapture(ctx context.Context, input []byte, args ...string) ([]byte, []byte, error) {
	// loudnorm/ebur128 emit their reports at info level, which the default
	// error level would suppress.
	return runFfmpegAt(ctx, input, "info", args...)
}

func runFfmpegAt(ctx context.Context, input []byte, logLevel string, args ...string) ([]byte, []byte, error) {
	if !hasFfmpeg() {
		return nil, nil, ErrFFmpegMissing
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	full := append(ffmpegBaseArgsAt(logLevel), args...)
	var stdout, stderr []byte
	var err error
	if r, ok := DefaultRunner.(StderrRunner); ok {
		stdout, stderr, err = r.RunWithStderr(ctx, "ffmpeg", input, full...)
	} else {
		stdout, err = DefaultRunner.Run(ctx, "ffmpeg", input, full...)
	}
	if err != nil {
		// Cancellation/deadline is the caller's own signal, not an ffmpeg
		// rejection. Surface it as such so handlers map it to
		// Canceled/DeadlineExceeded rather than InvalidArgument.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, stderr, fmt.Errorf("%w: %w", ctxErr, err)
		}
		// ffmpeg is installed but rejected the job. Wrap with the
		// ErrFfmpegExec sentinel (keeping the underlying cause in the
		// chain) so handlers can surface an actionable InvalidArgument
		// instead of a bare Internal error.
		return nil, stderr, fmt.Errorf("%w: %w", ErrFfmpegExec, err)
	}
	return stdout, stderr, nil
}

// ffmpegBaseArgsAt copies the global flags with the log level replaced.
// ffmpeg honours the last -loglevel it parses, so callers that need logs
// (measurement passes) override rather than duplicate the flag.
func ffmpegBaseArgsAt(level string) []string {
	out := append([]string{}, ffmpegBaseArgs...)
	for i := 0; i+1 < len(out); i++ {
		if out[i] == "-loglevel" {
			out[i+1] = level
			break
		}
	}
	return out
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
// The input format is auto-detected by ffmpeg; use TranscodeWithInput when
// the caller declares it explicitly.
func TranscodeOpts(ctx context.Context, audio []byte, format string, sampleRate, channels, bitrate int) ([]byte, error) {
	return TranscodeWithInput(ctx, audio, "", format, sampleRate, channels, bitrate)
}

// TranscodeWithInput is TranscodeOpts plus a declared input format.
// inFormat is an AudioFormat short-code ("" = let ffmpeg probe the
// bytes). When set, the matching demuxer is forced ahead of `-i pipe:0`
// so formats ffmpeg cannot reliably probe from a pipe (e.g. raw ADTS
// AAC) still decode.
func TranscodeWithInput(ctx context.Context, audio []byte, inFormat, outFormat string, sampleRate, channels, bitrate int) ([]byte, error) {
	switch {
	case sampleRate < 0 || sampleRate > 768000:
		return nil, fmt.Errorf("%w: sample_rate must be between 0 (passthrough) and 768000", ErrInvalidArgument)
	case channels < 0 || channels > 64:
		return nil, fmt.Errorf("%w: channels must be between 0 (passthrough) and 64", ErrInvalidArgument)
	case bitrate < 0 || bitrate > 1_000_000_000:
		return nil, fmt.Errorf("%w: bitrate must be between 0 (format default) and 1000000000", ErrInvalidArgument)
	}
	if outFormat == "" {
		outFormat = "wav"
	}
	args := make([]string, 0, 12)
	if inFormat != "" {
		args = append(args, "-f", demuxerFor(inFormat))
	}
	args = append(args, "-i", "pipe:0")
	if sampleRate > 0 {
		args = append(args, "-ar", fmt.Sprint(sampleRate))
	}
	if channels > 0 {
		args = append(args, "-ac", fmt.Sprint(channels))
	}
	if bitrate > 0 {
		args = append(args, "-b:a", fmt.Sprint(bitrate))
	}
	args = append(args, "-f", muxerFor(outFormat), "pipe:1")
	return runFfmpeg(ctx, audio, args...)
}
