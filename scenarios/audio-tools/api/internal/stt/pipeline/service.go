package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"
	"audio-tools/internal/sttbackend"
)

// Service owns the STT pipeline's state and behaviour: stream config,
// wake-word templates, speaker-verification config and resource client,
// and the Whisper transcribe entry point. Consumed directly by the
// Connect STT handler and the chain-routed LocalProvider.
type Service struct {
	configMu   sync.RWMutex
	config     Config
	configPath string

	wakeWordMu   sync.RWMutex
	wakeWord     *WakeWordTemplate
	wakeWordPath string

	speakerMu   sync.RWMutex
	speaker     SpeakerConfig
	speakerPath string

	speakerClient *SpeakerClient

	capabilities *capabilities.Registry

	skipVerifyCount *atomic.Int64

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
	Logger     logx.Logger
}

// log returns the injected Logger or falls back to logx.Std with the
// default *log.Logger. Tests inject mocks.FakeLogger via SetLogger.
func (s *Service) log() logx.Logger {
	if s.Logger == nil {
		return logx.Std{}
	}
	return s.Logger
}

// SetLogger overrides the service's logger. Tests use this to swap
// in mocks.FakeLogger after construction without changing the
// NewService signature.
func (s *Service) SetLogger(l logx.Logger) {
	s.Logger = l
}

// NewService constructs a Service. engine may be nil — the Service then
// builds a default audioformat.Engine (production ffmpeg backends).
func NewService(
	cfg Config,
	cfgPath string,
	wake *WakeWordTemplate,
	wakePath string,
	speaker SpeakerConfig,
	speakerPath string,
	speakerClient *SpeakerClient,
	caps *capabilities.Registry,
	skipVerifyCount *atomic.Int64,
	whisperURL string,
	httpClient HTTPDoer,
	engine *audioformat.Engine,
) *Service {
	if engine == nil {
		engine = audioformat.New()
	}
	return &Service{
		whisperSem:      make(chan struct{}, DefaultWhisperConcurrency),
		config:          cfg,
		configPath:      cfgPath,
		wakeWord:        wake,
		wakeWordPath:    wakePath,
		speaker:         speaker,
		speakerPath:     speakerPath,
		speakerClient:   speakerClient,
		capabilities:    caps,
		skipVerifyCount: skipVerifyCount,
		whisperURL:      whisperURL,
		httpClient:      httpClient,
		engine:          engine,
	}
}

// SetWhisperURL is provided for tests that point the Service at an httptest server.
func (s *Service) SetWhisperURL(u string) { s.whisperURL = u }

// SetHTTPClient is provided for tests that need to substitute outbound
// transcription transport.
func (s *Service) SetHTTPClient(c HTTPDoer) { s.httpClient = c }

// SetBackendEnsurer wires the on-demand recovery seam (plan L1): when a
// transcribe hits a down backend, the Service asks the ensurer to start
// `resource` once, then retries the request. Production wires
// sttbackend.NewCLIEnsurer() with "whisper"; tests inject a fake. A nil ensurer
// (the default) preserves the legacy behavior — a down backend returns the typed
// error without an auto-start.
func (s *Service) SetBackendEnsurer(e sttbackend.Ensurer, resource string) {
	s.ensurer = e
	s.backendResource = resource
}

// SetAutoEnsure toggles on-demand recovery at runtime (STT_AUTO_ENSURE). When
// false, a down backend returns the typed operator-action error immediately
// without attempting a `resource start`.
func (s *Service) SetAutoEnsure(on bool) { s.autoEnsure = on }

// SetEngine overrides the audio-format engine. Tests use this to inject
// an Engine wired to a fake ffmpeg Runner / process.
func (s *Service) SetEngine(e *audioformat.Engine) { s.engine = e }

// SetWhisperConcurrency resizes the Whisper concurrency bound. n<=0
// disables the bound. Intended for operator config and tests; existing
// in-flight calls keep their slot until they finish.
func (s *Service) SetWhisperConcurrency(n int) {
	if n <= 0 {
		s.whisperSem = nil
		return
	}
	s.whisperSem = make(chan struct{}, n)
}

// SetSpeakerClient is provided for tests that need a custom resource client.
func (s *Service) SetSpeakerClient(c *SpeakerClient) { s.speakerClient = c }

// SpeakerClient returns the configured resource client (may be nil).
func (s *Service) SpeakerClient() *SpeakerClient { return s.speakerClient }

// ----- Config accessors -----

func (s *Service) GetConfig() Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

func (s *Service) SetConfig(c Config) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = c
}

func (s *Service) ConfigPath() string            { return s.configPath }
func (s *Service) SetConfigPath(p string)        { s.configPath = p }
func (s *Service) SpeakerConfigPath() string     { return s.speakerPath }
func (s *Service) SetSpeakerConfigPath(p string) { s.speakerPath = p }
func (s *Service) WakeWordPath() string          { return s.wakeWordPath }
func (s *Service) SetWakeWordPath(p string)      { s.wakeWordPath = p }

// SpeakerConfigSnapshot returns the in-memory speaker config.
func (s *Service) SpeakerConfigSnapshot() SpeakerConfig {
	s.speakerMu.RLock()
	defer s.speakerMu.RUnlock()
	return s.speaker
}

func (s *Service) SetSpeakerConfig(c SpeakerConfig) {
	s.speakerMu.Lock()
	defer s.speakerMu.Unlock()
	s.speaker = c
}

func (s *Service) GetWakeWordTemplate() *WakeWordTemplate {
	s.wakeWordMu.RLock()
	defer s.wakeWordMu.RUnlock()
	return s.wakeWord
}

func (s *Service) SetWakeWordTemplate(tmpl *WakeWordTemplate) {
	s.wakeWordMu.Lock()
	defer s.wakeWordMu.Unlock()
	s.wakeWord = tmpl
}

func (s *Service) transcribe(ctx context.Context, audio []byte, format, language, initialPrompt string, vadFilter bool) (TranscriptionResult, error) {
	// Bound concurrent Whisper calls to the resource cap; block (queue),
	// never error, when full. ctx-aware so a cancelled session releases its
	// place in line instead of deadlocking.
	if s.whisperSem != nil {
		select {
		case s.whisperSem <- struct{}{}:
			defer func() { <-s.whisperSem }()
		case <-ctx.Done():
			return TranscriptionResult{}, ctx.Err()
		}
	}

	res, err := TranscribeBytes(ctx, s.whisperURL, s.httpClient, s.engine, audio, format, language, initialPrompt, vadFilter)
	if err == nil || !isBackendDown(err) {
		return res, err
	}

	// The backend is down (connection refused / dial failure), the exact
	// 2026-06-29 incident signature. On-demand recovery (plan L1): ensure the
	// backing resource is running, then retry ONCE. Single-flight + cooldown +
	// allowlist live in the Ensurer, so concurrent transcribes trigger at most one
	// `resource start`. When recovery is off (no ensurer / autoEnsure disabled /
	// no backend resource) we return the typed operator-action error — never the
	// raw dial string (plan L2).
	resource := s.backendResource
	if s.ensurer == nil || !s.autoEnsure || resource == "" {
		s.log().Printf("event=stt_backend_down resource=%q auto_ensure=%t recovered=false err=%v", resource, s.autoEnsure, err)
		return TranscriptionResult{}, newBackendNeedsOperator(resource)
	}

	s.log().Printf("event=stt_backend_down resource=%q action=ensure err=%v", resource, err)
	if ensureErr := s.ensurer.EnsureRunning(ctx, resource); ensureErr != nil {
		s.log().Printf("event=stt_backend_ensure_failed resource=%q err=%v", resource, ensureErr)
		return TranscriptionResult{}, newBackendNeedsOperator(resource)
	}

	res, err = TranscribeBytes(ctx, s.whisperURL, s.httpClient, s.engine, audio, format, language, initialPrompt, vadFilter)
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
	return res, err
}

// ----- Pipeline state API -----

func (s *Service) WhisperAvailable(ctx context.Context) bool {
	if s.capabilities == nil {
		return false
	}
	return s.capabilities.IsAvailable(ctx, "whisper-stt")
}

func (s *Service) IncrSkipVerification() {
	if s.skipVerifyCount != nil {
		s.skipVerifyCount.Add(1)
	}
}

func (s *Service) SpeakerCapability(ctx context.Context) (string, string) {
	if s.capabilities == nil {
		return "", ""
	}
	for _, cap := range s.capabilities.ResolveLiveness(ctx) {
		if cap.ID == "speaker-verification" {
			return string(cap.Status), cap.Message
		}
	}
	return "", ""
}

func (s *Service) EvaluateSpeaker(ctx context.Context, audio []byte) SpeakerDecision {
	return EvaluateSpeaker(ctx, s.SpeakerConfigSnapshot(), s.speakerClient, audio)
}

func (s *Service) FormatSpeakerDecisionError(d SpeakerDecision) string {
	return FormatSpeakerDecisionError(d)
}

// Transcribe sends audio to Whisper. format is the audioformat codec
// vocabulary describing the bytes ("webm", "pcm_s16le", "wav", ...);
// empty means "sniff". The audioformat substrate wraps canonical PCM in a
// WAV header and passes real containers straight to Whisper's own decoder.
// vadFilter enables faster-whisper's silence filter on the request. The
// returned TranscriptionResult carries the text plus the per-segment
// confidence signals the egress gate's signal-domain stage consumes.
func (s *Service) Transcribe(ctx context.Context, audio []byte, format, language, initialPrompt string, vadFilter bool) (TranscriptionResult, error) {
	return s.transcribe(ctx, audio, format, language, initialPrompt, vadFilter)
}

func (s *Service) IsWhisperHallucination(text string) bool {
	return IsWhisperHallucination(text)
}

func (s *Service) GetStreamConfig() Config {
	return s.GetConfig()
}

func (s *Service) SaveStreamConfig(c Config) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	s.SetConfig(c)
	if err := SaveConfig(s.configPath, c); err != nil {
		s.log().Printf("voice-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

func (s *Service) GetWakeWord() WakeWordConfig {
	return WakeWordToTransport(s.GetWakeWordTemplate())
}

func (s *Service) SetWakeWord(templateJSON string) (WakeWordConfig, error) {
	var tmpl WakeWordTemplate
	if err := json.Unmarshal([]byte(templateJSON), &tmpl); err != nil {
		return WakeWordConfig{}, fmt.Errorf("%w: parse template_json: %s", ErrInvalidArgument, err.Error())
	}
	if err := ValidateWakeWordTemplate(&tmpl); err != nil {
		return WakeWordConfig{}, fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	s.SetWakeWordTemplate(&tmpl)
	if err := SaveWakeWordTemplate(s.wakeWordPath, &tmpl); err != nil {
		s.log().Printf("wakeword: persist failed (in-memory updated): %v", err)
	}
	s.log().Printf("wakeword: template saved: label=%q samples=%d threshold=%.2f",
		tmpl.Label, len(tmpl.Samples), tmpl.Threshold)
	return WakeWordToTransport(&tmpl), nil
}

func (s *Service) ClearWakeWord() error {
	s.SetWakeWordTemplate(nil)
	return DeleteWakeWordTemplate(s.wakeWordPath)
}

func (s *Service) SaveSpeakerConfig(c SpeakerConfig) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidArgument, err.Error())
	}
	s.SetSpeakerConfig(c)
	if err := SaveSpeakerConfig(s.speakerPath, c); err != nil {
		s.log().Printf("speaker-verification-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

func (s *Service) GetSpeakerConfig() SpeakerConfig {
	return s.SpeakerConfigSnapshot()
}

func (s *Service) DefaultSpeakerThreshold() float64 {
	return DefaultSpeakerConfig().Threshold
}

func (s *Service) DefaultSpeakerProfileID() string {
	return DefaultSpeakerProfileID()
}

func (s *Service) SpeakerClientConfigured() bool { return s.speakerClient != nil }

func (s *Service) SpeakerReady(ctx context.Context) bool {
	if s.speakerClient == nil {
		return false
	}
	ready, err := s.speakerClient.Ready(ctx)
	return err == nil && ready.Status == "ready"
}

func (s *Service) ListSpeakerProfiles(ctx context.Context) ([]SpeakerProfile, int, error) {
	if s.speakerClient == nil {
		return nil, 0, errors.New("speaker verification resource is not configured")
	}
	list, err := s.speakerClient.ListProfiles(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list.Profiles, list.Count, nil
}

func (s *Service) SpeakerInfo(ctx context.Context) (SpeakerResourceInfo, bool) {
	if s.speakerClient == nil {
		return SpeakerResourceInfo{}, false
	}
	info, err := s.speakerClient.Info(ctx)
	if err != nil {
		return SpeakerResourceInfo{}, false
	}
	return info, true
}

func (s *Service) DeleteSpeakerBackend(ctx context.Context, profileID string) error {
	if s.speakerClient == nil {
		return errors.New("speaker verification resource is not configured")
	}
	return s.speakerClient.DeleteProfile(ctx, profileID)
}
