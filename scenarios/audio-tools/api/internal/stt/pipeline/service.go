package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"
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
	transcode  func(context.Context, []byte) ([]byte, error)
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

// NewService constructs a Service. transcodeFn may be nil — TranscribeBytes
// then passes the raw audio through.
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
	transcode func(context.Context, []byte) ([]byte, error),
) *Service {
	return &Service{
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
		transcode:       transcode,
	}
}

// SetWhisperURL is provided for tests that point the Service at an httptest server.
func (s *Service) SetWhisperURL(u string) { s.whisperURL = u }

// SetHTTPClient is provided for tests that need to substitute outbound
// transcription transport.
func (s *Service) SetHTTPClient(c HTTPDoer) { s.httpClient = c }

// SetTranscode is provided for tests that need a passthrough transcoder.
func (s *Service) SetTranscode(fn func(context.Context, []byte) ([]byte, error)) {
	s.transcode = fn
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

func (s *Service) transcribe(ctx context.Context, audio []byte, language string, doTranscode bool, initialPrompt string) (string, error) {
	return TranscribeBytes(ctx, s.whisperURL, s.httpClient, s.transcode, audio, language, doTranscode, initialPrompt)
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

func (s *Service) Transcribe(ctx context.Context, audio []byte, language string) (string, error) {
	return s.transcribe(ctx, audio, language, true, "")
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

func (s *Service) EnrollSpeaker(ctx context.Context, audio []byte, profileID, displayName, notes string) (SpeakerEnrollment, error) {
	if s.speakerClient == nil {
		return SpeakerEnrollment{}, errors.New("speaker verification resource is not configured")
	}
	enrollment, err := s.speakerClient.Enroll(ctx, audio, profileID, displayName, notes)
	if err != nil {
		return SpeakerEnrollment{}, err
	}
	return SpeakerEnrollment(enrollment), nil
}

func (s *Service) DeleteSpeakerBackend(ctx context.Context, profileID string) error {
	if s.speakerClient == nil {
		return errors.New("speaker verification resource is not configured")
	}
	return s.speakerClient.DeleteProfile(ctx, profileID)
}
