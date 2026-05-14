package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	voiceH "web-console/handlers/voice"
	"web-console/internal/capabilities"
)

// Service owns the voice domain's state and behaviour. It implements
// web-console/handlers/voice.Backend so the Connect adapter can call into it
// without knowing about file paths, the speaker resource client, or the
// Whisper pipeline.
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
	transcode  func(context.Context, []byte) ([]byte, error)
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
		transcode:       transcode,
	}
}

// SetWhisperURL is provided for tests that point the Service at an httptest server.
func (s *Service) SetWhisperURL(u string) { s.whisperURL = u }

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

func (s *Service) ConfigPath() string             { return s.configPath }
func (s *Service) SetConfigPath(p string)          { s.configPath = p }
func (s *Service) SpeakerConfigPath() string       { return s.speakerPath }
func (s *Service) SetSpeakerConfigPath(p string)   { s.speakerPath = p }
func (s *Service) WakeWordPath() string            { return s.wakeWordPath }
func (s *Service) SetWakeWordPath(p string)        { s.wakeWordPath = p }

// SpeakerConfigSnapshot returns the in-memory speaker config. Backend's
// GetSpeakerConfig returns the transport-shaped variant; internal callers
// (stream_ws, speaker.go) want the typed snapshot.
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
	return TranscribeBytes(ctx, s.whisperURL, s.transcode, audio, language, doTranscode, initialPrompt)
}

// ----- handlers/voice.Backend implementation -----

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

func (s *Service) EvaluateSpeaker(ctx context.Context, audio []byte) voiceH.SpeakerDecision {
	d := EvaluateSpeaker(ctx, s.SpeakerConfigSnapshot(), s.speakerClient, audio)
	return voiceH.SpeakerDecision{
		Enabled:      d.Enabled,
		Applied:      d.Applied,
		Allowed:      d.Allowed,
		Matched:      d.Matched,
		ProfileID:    d.ProfileID,
		Score:        d.Score,
		Threshold:    d.Threshold,
		Mode:         d.Mode,
		ErrorMessage: d.ErrorMessage,
	}
}

func (s *Service) FormatSpeakerDecisionError(d voiceH.SpeakerDecision) string {
	return FormatSpeakerDecisionError(SpeakerDecision{
		Enabled:      d.Enabled,
		Applied:      d.Applied,
		Allowed:      d.Allowed,
		Matched:      d.Matched,
		ProfileID:    d.ProfileID,
		Score:        d.Score,
		Threshold:    d.Threshold,
		Mode:         d.Mode,
		ErrorMessage: d.ErrorMessage,
	})
}

func (s *Service) Transcribe(ctx context.Context, audio []byte, language string) (string, error) {
	return s.transcribe(ctx, audio, language, true, "")
}

func (s *Service) IsWhisperHallucination(text string) bool {
	return IsWhisperHallucination(text)
}

func (s *Service) GetStreamConfig() voiceH.StreamConfig {
	return configToTransport(s.GetConfig())
}

func (s *Service) SaveStreamConfig(c voiceH.StreamConfig) error {
	internal := Config{
		FlushIntervalMs:   c.FlushIntervalMs,
		MinDeltaBytes:     c.MinDeltaBytes,
		OverlapBytes:      c.OverlapBytes,
		PersistentMode:    c.PersistentMode,
		WakeWordEnabled:   c.WakeWordEnabled,
		WakeWordThreshold: c.WakeWordThreshold,
		SegmentSilenceMs:  c.SegmentSilenceMs,
	}
	if err := internal.Validate(); err != nil {
		return fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	s.SetConfig(internal)
	if err := SaveConfig(s.configPath, internal); err != nil {
		log.Printf("voice-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

func (s *Service) GetWakeWord() voiceH.WakeWordConfig {
	return wakeWordToTransport(s.GetWakeWordTemplate())
}

func (s *Service) SetWakeWord(templateJSON string) (voiceH.WakeWordConfig, error) {
	var tmpl WakeWordTemplate
	if err := json.Unmarshal([]byte(templateJSON), &tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: parse template_json: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	if err := ValidateWakeWordTemplate(&tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	s.SetWakeWordTemplate(&tmpl)
	if err := SaveWakeWordTemplate(s.wakeWordPath, &tmpl); err != nil {
		log.Printf("wakeword: persist failed (in-memory updated): %v", err)
	}
	log.Printf("wakeword: template saved: label=%q samples=%d threshold=%.2f",
		tmpl.Label, len(tmpl.Samples), tmpl.Threshold)
	return wakeWordToTransport(&tmpl), nil
}

func (s *Service) ClearWakeWord() error {
	s.SetWakeWordTemplate(nil)
	return DeleteWakeWordTemplate(s.wakeWordPath)
}

func (s *Service) SaveSpeakerConfig(c voiceH.SpeakerConfig) error {
	internal := SpeakerConfig{
		Enabled:                     c.Enabled,
		ProfileIDs:                  append([]string(nil), c.ProfileIDs...),
		Threshold:                   c.Threshold,
		Mode:                        c.Mode,
		RejectBehavior:              c.RejectBehavior,
		FallbackWithoutVerification: c.FallbackWithoutVerification,
		ExtractionEnabled:           c.ExtractionEnabled,
	}
	if err := internal.Validate(); err != nil {
		return fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	s.SetSpeakerConfig(internal)
	if err := SaveSpeakerConfig(s.speakerPath, internal); err != nil {
		log.Printf("speaker-verification-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

func (s *Service) GetSpeakerConfig() voiceH.SpeakerConfig {
	return speakerConfigToTransport(s.SpeakerConfigSnapshot())
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

func (s *Service) ListSpeakerProfiles(ctx context.Context) ([]voiceH.SpeakerProfile, int, error) {
	if s.speakerClient == nil {
		return nil, 0, errors.New("speaker verification resource is not configured")
	}
	list, err := s.speakerClient.ListProfiles(ctx)
	if err != nil {
		return nil, 0, err
	}
	return profilesToTransport(list.Profiles), list.Count, nil
}

func (s *Service) SpeakerInfo(ctx context.Context) (voiceH.SpeakerResourceInfo, bool) {
	if s.speakerClient == nil {
		return voiceH.SpeakerResourceInfo{}, false
	}
	info, err := s.speakerClient.Info(ctx)
	if err != nil {
		return voiceH.SpeakerResourceInfo{}, false
	}
	return voiceH.SpeakerResourceInfo{
		Backend:      info.Backend,
		Model:        info.Model,
		Device:       info.Device,
		SampleRate:   info.SampleRate,
		Version:      info.Version,
		EmbeddingDim: info.EmbeddingDim,
	}, true
}

func (s *Service) EnrollSpeaker(ctx context.Context, audio []byte, profileID, displayName, notes string) (voiceH.SpeakerEnrollment, error) {
	if s.speakerClient == nil {
		return voiceH.SpeakerEnrollment{}, errors.New("speaker verification resource is not configured")
	}
	enrollment, err := s.speakerClient.Enroll(ctx, audio, profileID, displayName, notes)
	if err != nil {
		return voiceH.SpeakerEnrollment{}, err
	}
	return voiceH.SpeakerEnrollment{
		ProfileID:              enrollment.ProfileID,
		DisplayName:            enrollment.DisplayName,
		EmbeddingDim:           enrollment.EmbeddingDim,
		SampleRate:             enrollment.SampleRate,
		EnrollmentAudioSeconds: enrollment.EnrollmentAudioSeconds,
		ModelName:              enrollment.ModelName,
		CreatedAt:              enrollment.CreatedAt,
	}, nil
}

func (s *Service) DeleteSpeakerBackend(ctx context.Context, profileID string) error {
	if s.speakerClient == nil {
		return errors.New("speaker verification resource is not configured")
	}
	return s.speakerClient.DeleteProfile(ctx, profileID)
}

// ----- mappers -----

func configToTransport(c Config) voiceH.StreamConfig {
	return voiceH.StreamConfig{
		FlushIntervalMs:   c.FlushIntervalMs,
		MinDeltaBytes:     c.MinDeltaBytes,
		OverlapBytes:      c.OverlapBytes,
		PersistentMode:    c.PersistentMode,
		WakeWordEnabled:   c.WakeWordEnabled,
		WakeWordThreshold: c.WakeWordThreshold,
		SegmentSilenceMs:  c.SegmentSilenceMs,
	}
}

func wakeWordToTransport(tmpl *WakeWordTemplate) voiceH.WakeWordConfig {
	if tmpl == nil {
		return voiceH.WakeWordConfig{Configured: false}
	}
	data, err := json.Marshal(tmpl)
	if err != nil {
		log.Printf("wakeword: marshal failed: %v", err)
		return voiceH.WakeWordConfig{Configured: true}
	}
	return voiceH.WakeWordConfig{Configured: true, TemplateJSON: string(data)}
}

func speakerConfigToTransport(c SpeakerConfig) voiceH.SpeakerConfig {
	return voiceH.SpeakerConfig{
		Enabled:                     c.Enabled,
		ProfileIDs:                  append([]string(nil), c.ProfileIDs...),
		Threshold:                   c.Threshold,
		Mode:                        c.Mode,
		RejectBehavior:              c.RejectBehavior,
		FallbackWithoutVerification: c.FallbackWithoutVerification,
		ExtractionEnabled:           c.ExtractionEnabled,
	}
}

func profilesToTransport(in []SpeakerProfile) []voiceH.SpeakerProfile {
	out := make([]voiceH.SpeakerProfile, 0, len(in))
	for _, p := range in {
		out = append(out, voiceH.SpeakerProfile{
			ID:                     p.ID,
			DisplayName:            p.DisplayName,
			CreatedAt:              p.CreatedAt,
			UpdatedAt:              p.UpdatedAt,
			ModelName:              p.ModelName,
			EmbeddingDim:           p.EmbeddingDim,
			SampleRate:             p.SampleRate,
			EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
			Notes:                  p.Notes,
		})
	}
	return out
}
