package pipeline

import (
	"context"
	"sync"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"
	"audio-tools/internal/sttbackend"
)

// WhisperTranscriber owns the local Whisper batch transcription and recovery path.
// Stream, wake-word, and speaker configuration are owned by their repositories.
type WhisperTranscriber struct {
	capabilities *capabilities.Registry

	whisperURL string
	httpClient HTTPDoer
	engine     *audioformat.Engine
	// ensurer brings the backing STT backend resource (backendResource) up on
	// demand when a transcribe hits a down backend (plan L1). nil disables
	// on-demand recovery (the existing behavior; tests leave it nil). autoEnsure
	// gates it at runtime (STT_AUTO_ENSURE).
	ensurer         sttbackend.Ensurer
	backendResource string
	autoEnsure      bool
	// whisperSem bounds concurrent Whisper /asr calls to the resource's
	// documented ceiling (5). Over-limit callers BLOCK on acquire (queue
	// with backpressure) — they never error. The cap is the real
	// multi-session ceiling, upstream of the audio-format layer; the format
	// substrate must not mask it. nil disables the bound.
	whisperSem chan struct{}

	// depMu guards the mutable runtime dependencies below. They are wired at
	// boot today, but SetWhisperConcurrency/SetEngine/SetHTTPClient are also
	// operator/test seams; a concurrent setter must not race a live request.
	depMu sync.RWMutex

	Logger logx.Logger
}

// runtimeDeps is a consistent snapshot of the mutable dependencies a request
// reads. Holding depMu across network I/O would serialize every stream, so
// callers snapshot once and use the values locally.
type runtimeDeps struct {
	whisperURL      string
	httpClient      HTTPDoer
	engine          *audioformat.Engine
	ensurer         sttbackend.Ensurer
	backendResource string
	autoEnsure      bool
	whisperSem      chan struct{}
}

func (s *WhisperTranscriber) snapshotDeps() runtimeDeps {
	s.depMu.RLock()
	defer s.depMu.RUnlock()
	return runtimeDeps{
		whisperURL:      s.whisperURL,
		httpClient:      s.httpClient,
		engine:          s.engine,
		ensurer:         s.ensurer,
		backendResource: s.backendResource,
		autoEnsure:      s.autoEnsure,
		whisperSem:      s.whisperSem,
	}
}

// log returns the injected Logger or falls back to logx.Std with the
// default *log.Logger. Tests inject mocks.FakeLogger via SetLogger.
func (s *WhisperTranscriber) log() logx.Logger {
	if s.Logger == nil {
		return logx.Std{}
	}
	return s.Logger
}

// SetLogger overrides the service's logger. Tests use this to swap
// in mocks.FakeLogger after construction without changing the
// NewService signature.
func (s *WhisperTranscriber) SetLogger(l logx.Logger) {
	s.Logger = l
}

// NewWhisperTranscriber constructs a WhisperTranscriber. engine may be nil — it then
// builds a default audioformat.Engine (production ffmpeg backends).
func NewWhisperTranscriber(
	caps *capabilities.Registry,
	whisperURL string,
	httpClient HTTPDoer,
	engine *audioformat.Engine,
) *WhisperTranscriber {
	if engine == nil {
		engine = audioformat.New()
	}
	return &WhisperTranscriber{
		whisperSem:   make(chan struct{}, DefaultWhisperConcurrency),
		capabilities: caps,
		whisperURL:   whisperURL,
		httpClient:   httpClient,
		engine:       engine,
	}
}

// SetWhisperURL is provided for tests that point the Service at an httptest server.
func (s *WhisperTranscriber) SetWhisperURL(u string) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	s.whisperURL = u
}

// SetHTTPClient is provided for tests that need to substitute outbound
// transcription transport.
func (s *WhisperTranscriber) SetHTTPClient(c HTTPDoer) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	s.httpClient = c
}

// SetBackendEnsurer wires the on-demand recovery seam (plan L1): when a
// transcribe hits a down backend, the Service asks the ensurer to start
// `resource` once, then retries the request. Production wires
// sttbackend.NewCLIEnsurer() with "whisper"; tests inject a fake. A nil ensurer
// (the default) preserves the legacy behavior — a down backend returns the typed
// error without an auto-start.
func (s *WhisperTranscriber) SetBackendEnsurer(e sttbackend.Ensurer, resource string) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	s.ensurer = e
	s.backendResource = resource
}

// SetAutoEnsure toggles on-demand recovery at runtime (STT_AUTO_ENSURE). When
// false, a down backend returns the typed operator-action error immediately
// without attempting a `resource start`.
func (s *WhisperTranscriber) SetAutoEnsure(on bool) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	s.autoEnsure = on
}

// SetEngine overrides the audio-format engine. Tests use this to inject
// an Engine wired to a fake ffmpeg Runner / process.
func (s *WhisperTranscriber) SetEngine(e *audioformat.Engine) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	s.engine = e
}

// SetWhisperConcurrency resizes the Whisper concurrency bound. n<=0
// disables the bound. Intended for operator config and tests; existing
// in-flight calls keep their slot until they finish.
func (s *WhisperTranscriber) SetWhisperConcurrency(n int) {
	s.depMu.Lock()
	defer s.depMu.Unlock()
	if n <= 0 {
		s.whisperSem = nil
		return
	}
	s.whisperSem = make(chan struct{}, n)
}

func (s *WhisperTranscriber) transcribe(ctx context.Context, audio []byte, format, language, initialPrompt string, vadFilter bool) (TranscriptionResult, error) {
	d := s.snapshotDeps()

	// Bound concurrent Whisper calls to the resource cap; block (queue),
	// never error, when full. ctx-aware so a cancelled session releases its
	// place in line instead of deadlocking.
	if d.whisperSem != nil {
		select {
		case d.whisperSem <- struct{}{}:
			defer func() { <-d.whisperSem }()
		case <-ctx.Done():
			return TranscriptionResult{}, ctx.Err()
		}
	}

	call := func() (TranscriptionResult, error) {
		return TranscribeBytes(ctx, d.whisperURL, d.httpClient, d.engine, audio, format, language, initialPrompt, vadFilter)
	}
	res, err := call()
	if err == nil {
		return res, err
	}
	if isBackendTimeout(err) {
		resource := d.backendResource
		if resource == "" {
			resource = "whisper"
		}
		s.log().Printf("event=stt_backend_degraded resource=%q err=%v", resource, err)
		return TranscriptionResult{}, newBackendDegraded(resource)
	}
	if !isBackendDown(err) {
		return res, err
	}

	// The backend is down (connection refused / dial failure), the exact
	// 2026-06-29 incident signature. On-demand recovery (plan L1): ensure the
	// backing resource is running, then retry ONCE. Single-flight + cooldown +
	// allowlist live in the Ensurer, so concurrent transcribes trigger at most one
	// `resource start`. When recovery is off (no ensurer / autoEnsure disabled /
	// no backend resource) we return the typed operator-action error — never the
	// raw dial string (plan L2).
	resource := d.backendResource
	if d.ensurer == nil || !d.autoEnsure || resource == "" {
		s.log().Printf("event=stt_backend_down resource=%q auto_ensure=%t recovered=false err=%v", resource, d.autoEnsure, err)
		return TranscriptionResult{}, newBackendNeedsOperator(resource)
	}

	s.log().Printf("event=stt_backend_down resource=%q action=ensure err=%v", resource, err)
	if ensureErr := d.ensurer.EnsureRunning(ctx, resource); ensureErr != nil {
		s.log().Printf("event=stt_backend_ensure_failed resource=%q err=%v", resource, ensureErr)
		return TranscriptionResult{}, newBackendNeedsOperator(resource)
	}

	res, err = call()
	if err == nil {
		s.log().Printf("event=stt_backend_recovered resource=%q", resource)
		return res, nil
	}
	if isBackendDown(err) {
		// Ensured, but the backend is not serving yet — report transient so the UI
		// shows a "starting…" state and the user retries shortly.
		s.log().Printf("event=stt_backend_starting resource=%q err=%v", resource, err)
		return TranscriptionResult{}, newBackendStarting(resource)
	}
	if isBackendTimeout(err) {
		s.log().Printf("event=stt_backend_degraded_after_ensure resource=%q err=%v", resource, err)
		return TranscriptionResult{}, newBackendDegraded(resource)
	}
	return res, err
}

// ----- Pipeline state API -----

func (s *WhisperTranscriber) WhisperAvailable(ctx context.Context) bool {
	if s.capabilities == nil {
		return false
	}
	return s.capabilities.IsAvailable(ctx, "whisper-stt")
}

// EnsureWhisperAvailable probes Whisper and, when it is unavailable, asks the
// configured backend ensurer to reconcile the resource before forcing a fresh
// capability check. This covers stream-start/provider-picker paths that consult
// availability before any batch transcription request can hit the retry path.
func (s *WhisperTranscriber) EnsureWhisperAvailable(ctx context.Context) bool {
	if s.WhisperAvailable(ctx) {
		return true
	}
	d := s.snapshotDeps()
	resource := d.backendResource
	if s.capabilities == nil || d.ensurer == nil || !d.autoEnsure || resource == "" {
		return false
	}
	s.log().Printf("event=stt_backend_unavailable resource=%q action=ensure_probe", resource)
	if err := d.ensurer.EnsureRunning(ctx, resource); err != nil {
		s.log().Printf("event=stt_backend_ensure_probe_failed resource=%q err=%v", resource, err)
		return false
	}
	for _, state := range s.capabilities.ResolveForce(ctx) {
		if state.ID == "whisper-stt" {
			recovered := state.Status == capabilities.StatusAvailable
			s.log().Printf("event=stt_backend_ensure_probe_result resource=%q recovered=%t status=%q", resource, recovered, state.Status)
			return recovered
		}
	}
	return false
}

// Transcribe sends audio to Whisper. format is the audioformat codec
// vocabulary describing the bytes ("webm", "pcm_s16le", "wav", ...);
// empty means "sniff". The audioformat substrate wraps canonical PCM in a
// WAV header and passes real containers straight to Whisper's own decoder.
// vadFilter enables faster-whisper's silence filter on the request. The
// returned TranscriptionResult carries the text plus the per-segment
// confidence signals the egress gate's signal-domain stage consumes.
func (s *WhisperTranscriber) Transcribe(ctx context.Context, audio []byte, format, language, initialPrompt string, vadFilter bool) (TranscriptionResult, error) {
	return s.transcribe(ctx, audio, format, language, initialPrompt, vadFilter)
}
