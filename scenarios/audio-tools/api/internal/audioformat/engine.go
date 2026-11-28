package audioformat

import "audio-tools/internal/audio"

// Engine is the single owner of audio-format conversion. Construct one at
// startup (New) and share it across sessions — it holds no per-session
// state; every per-session decoder is created fresh via NewStreamDecoder
// so concurrent sessions never share a process.
//
// The one-shot ffmpeg runs through the existing audio.Runner seam; the
// long-lived streaming decode runs through the ProcessRunner seam. ffmpeg
// presence is probed through an injectable function so capability tests
// don't depend on a binary.
type Engine struct {
	runner    audio.Runner
	process   ProcessRunner
	hasFfmpeg func() bool
}

// Option configures an Engine.
type Option func(*Engine)

// WithRunner overrides the one-shot ffmpeg Runner (tests inject a fake to
// assert argv).
func WithRunner(r audio.Runner) Option { return func(e *Engine) { e.runner = r } }

// WithProcessRunner overrides the long-lived decode process factory
// (tests inject a fake in-memory process).
func WithProcessRunner(p ProcessRunner) Option { return func(e *Engine) { e.process = p } }

// WithFfmpegProbe overrides the ffmpeg-presence probe (tests force
// available/unavailable without touching $PATH).
func WithFfmpegProbe(fn func() bool) Option { return func(e *Engine) { e.hasFfmpeg = fn } }

// New constructs an Engine wired to production ffmpeg backends. Apply
// options to substitute seams in tests.
func New(opts ...Option) *Engine {
	e := &Engine{
		runner:    audio.DefaultRunner,
		process:   execProcessRunner{},
		hasFfmpeg: audio.HasFfmpeg,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// HasFfmpeg reports whether the engine's ffmpeg backend is available.
func (e *Engine) HasFfmpeg() bool { return e.hasFfmpeg() }
