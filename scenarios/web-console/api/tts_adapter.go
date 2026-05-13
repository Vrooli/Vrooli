package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	ttsH "web-console/handlers/tts"
)

// ttsAdapter implements ttsH.Service against the legacy Server. It centralizes
// the validation that used to live in tts_config.go / tts_summarize_config.go
// / tts_synthesize.go / tts_cache.go / tts_voices.go / tts_playback.go, then
// returns sentinel errors from handlers/tts so the Connect classifier picks
// the right Connect code.
type ttsAdapter struct {
	srv *Server
}

func newTTSAdapter(s *Server) *ttsAdapter { return &ttsAdapter{srv: s} }

// ----------------------------------------------------------------------------
// Config
// ----------------------------------------------------------------------------

func (a *ttsAdapter) GetConfig(_ context.Context) (ttsH.Config, error) {
	return toAdapterConfig(a.srv.getTTSConfig()), nil
}

func (a *ttsAdapter) UpdateConfig(_ context.Context, patch ttsH.ConfigPatch) (ttsH.Config, error) {
	current := a.srv.getTTSConfig()
	updated := TTSConfigPatch{
		AutoEnabled: patch.AutoEnabled,
		Backend:     patch.Backend,
		KokoroVoice: patch.KokoroVoice,
		KokoroSpeed: patch.KokoroSpeed,
	}.Apply(current)

	switch updated.Backend {
	case "", "auto", "kokoro", "browser":
		if updated.Backend == "" {
			updated.Backend = "auto"
		}
	default:
		return ttsH.Config{}, fmt.Errorf("%w: backend must be auto, kokoro, or browser", ttsH.ErrInvalidArgument)
	}
	if updated.KokoroVoice == "" {
		updated.KokoroVoice = "af_heart"
	}
	if updated.KokoroSpeed < 0.5 || updated.KokoroSpeed > 4.0 {
		return ttsH.Config{}, fmt.Errorf("%w: kokoroSpeed must be between 0.5 and 4.0", ttsH.ErrInvalidArgument)
	}

	a.srv.setTTSConfig(updated)
	if err := saveTTSConfig(a.srv.ttsConfigPath, updated); err != nil {
		log.Printf("tts-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("tts-config: updated: autoEnabled=%v backend=%s voice=%s speed=%.1f",
		updated.AutoEnabled, updated.Backend, updated.KokoroVoice, updated.KokoroSpeed)
	return toAdapterConfig(updated), nil
}

// ----------------------------------------------------------------------------
// Status / playback events
// ----------------------------------------------------------------------------

func (a *ttsAdapter) GetStatus(ctx context.Context) (ttsH.Status, error) {
	s := a.srv
	hookRegistered, hookCode, hookReason, hookSettingsPath := s.getClaudeHookStatus()
	lastRouting, lastRoutingAt := s.getLastTTSRouting()
	lastHookRouting, lastHookRoutingAt := s.getLastTTSRoutingBySource("claude_hook")
	lastTailerRouting, lastTailerRoutingAt := s.getLastTTSRoutingBySource("codex_tailer")
	lastAck, lastAckAt := s.getLastTTSAck()
	lastHookAck, lastHookAckAt := s.getLastTTSAckBySource("claude_hook")
	lastTailerAck, lastTailerAckAt := s.getLastTTSAckBySource("codex_tailer")
	lastPlaybackEvent, lastPlaybackAt := s.getLastTTSPlaybackEvent()

	kokoroCapability := "unknown"
	kokoroCapabilityLabel := "Kokoro status not checked yet"
	for _, cap := range s.capabilities.ResolveLiveness(ctx) {
		if cap.ID == "kokoro-tts" {
			kokoroCapability = string(cap.Status)
			kokoroCapabilityLabel = strings.TrimSpace(cap.Message)
			if kokoroCapabilityLabel == "" {
				kokoroCapabilityLabel = "Kokoro status available"
			}
			break
		}
	}

	st := ttsH.Status{
		Config:                toAdapterConfig(s.getTTSConfig()),
		HookRegistered:        hookRegistered,
		HookCode:              hookCode,
		HookReason:            hookReason,
		HookSettingsPath:      hookSettingsPath,
		LastRouting:           appendResultToAdapter(lastRouting),
		LastRoutingAt:         rfc3339(lastRoutingAt),
		LastHookRouting:       appendResultToAdapter(lastHookRouting),
		LastHookRoutingAt:     rfc3339(lastHookRoutingAt),
		LastTailerRouting:     appendResultToAdapter(lastTailerRouting),
		LastTailerRoutingAt:   rfc3339(lastTailerRoutingAt),
		LastAck:               ackToAdapter(lastAck),
		LastAckAt:             rfc3339(lastAckAt),
		LastHookAck:           ackToAdapter(lastHookAck),
		LastHookAckAt:         rfc3339(lastHookAckAt),
		LastTailerAck:         ackToAdapter(lastTailerAck),
		LastTailerAckAt:       rfc3339(lastTailerAckAt),
		LastPlaybackEvent:     playbackToAdapter(lastPlaybackEvent),
		LastPlaybackAt:        rfc3339(lastPlaybackAt),
		KokoroCapability:      kokoroCapability,
		KokoroCapabilityLabel: kokoroCapabilityLabel,
	}
	return st, nil
}

func (a *ttsAdapter) RecordPlaybackEvent(_ context.Context, ev ttsH.PlaybackEvent) error {
	a.srv.recordTTSPlaybackEvent(TTSPlaybackEvent{
		Source:    ev.Source,
		Stage:     ev.Stage,
		Backend:   ev.Backend,
		SessionID: ev.SessionID,
		Message:   ev.Message,
	})
	return nil
}

// ----------------------------------------------------------------------------
// Summarize config
// ----------------------------------------------------------------------------

func (a *ttsAdapter) GetSummarizeConfig(_ context.Context) (ttsH.SummarizeConfig, error) {
	return toAdapterSummarize(a.srv.getTTSSummarizeConfig()), nil
}

func (a *ttsAdapter) UpdateSummarizeConfig(_ context.Context, patch ttsH.SummarizeConfigPatch) (ttsH.SummarizeConfig, error) {
	current := a.srv.getTTSSummarizeConfig()
	updated := TTSSummarizeConfigPatch{
		Enabled:        patch.Enabled,
		CharThreshold:  patch.CharThreshold,
		Level:          patch.Level,
		Model:          patch.Model,
		TimeoutSeconds: patch.TimeoutSeconds,
	}.Apply(current)

	if !validSummarizeLevels[updated.Level] {
		return ttsH.SummarizeConfig{}, fmt.Errorf("%w: level must be light, moderate, or heavy", ttsH.ErrInvalidArgument)
	}
	if updated.CharThreshold < 0 {
		return ttsH.SummarizeConfig{}, fmt.Errorf("%w: charThreshold must be non-negative", ttsH.ErrInvalidArgument)
	}
	if updated.TimeoutSeconds < minSummarizeTimeoutSeconds || updated.TimeoutSeconds > maxSummarizeTimeoutSeconds {
		return ttsH.SummarizeConfig{}, fmt.Errorf("%w: timeoutSeconds must be between %d and %d", ttsH.ErrInvalidArgument, minSummarizeTimeoutSeconds, maxSummarizeTimeoutSeconds)
	}
	if updated.Model == "" {
		return ttsH.SummarizeConfig{}, fmt.Errorf("%w: model must not be empty", ttsH.ErrInvalidArgument)
	}

	a.srv.setTTSSummarizeConfig(updated)
	if err := saveTTSSummarizeConfig(a.srv.ttsSummarizePath, updated); err != nil {
		log.Printf("tts-summarize-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("tts-summarize-config: updated: enabled=%v threshold=%d level=%s model=%s timeout=%ds",
		updated.Enabled, updated.CharThreshold, updated.Level, updated.Model, updated.TimeoutSeconds)
	return toAdapterSummarize(updated), nil
}

// ----------------------------------------------------------------------------
// Synthesize / cache / voices
// ----------------------------------------------------------------------------

func (a *ttsAdapter) Synthesize(ctx context.Context, in ttsH.SynthesizeInput) (ttsH.SynthesizeResult, error) {
	s := a.srv
	if s.ttsSynthesizer == nil {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: TTS synthesis is not configured", ttsH.ErrFailedPrecondition)
	}
	if !s.capabilities.IsAvailable(ctx, "kokoro-tts") {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: Kokoro TTS is not available", ttsH.ErrUnavailable)
	}

	input := strings.TrimSpace(in.Input)
	if input == "" {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: input is required", ttsH.ErrInvalidArgument)
	}
	if len(input) > maxSynthesizeInputLength {
		log.Printf("tts-synthesize: input too long (%d chars, limit %d)", len(input), maxSynthesizeInputLength)
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: input exceeds maximum length of 5000 characters", ttsH.ErrInvalidArgument)
	}

	voice := in.Voice
	if voice == "" {
		voice = s.getTTSConfig().KokoroVoice
		if voice == "" {
			voice = "af_heart"
		}
	}
	format := in.ResponseFormat
	if format == "" {
		format = "mp3"
	}
	if _, ok := formatContentTypes[format]; !ok {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: unsupported response_format; use mp3, wav, opus, or flac", ttsH.ErrInvalidArgument)
	}
	speed := in.Speed
	const maxTTSSpeed = 4.0
	if speed <= 0 {
		speed = 1.0
	} else if speed > maxTTSSpeed {
		speed = maxTTSSpeed
	}

	body, contentType, err := s.ttsSynthesizer.Synthesize(ctx, SynthesizeRequest{
		Input:          input,
		Voice:          voice,
		ResponseFormat: format,
		Speed:          speed,
	})
	if err != nil {
		log.Printf("tts-synthesize: synthesis failed: %v", err)
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: synthesis failed", ttsH.ErrInternal)
	}
	defer body.Close()

	data, readErr := io.ReadAll(body)
	if readErr != nil {
		log.Printf("tts-synthesize: read response: %v", readErr)
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: synthesis failed", ttsH.ErrInternal)
	}

	if in.EventID != "" && s.ttsCache != nil && len(data) > 0 {
		version := in.Version
		if version == "" {
			version = "active"
		}
		s.ttsCache.Put(TTSCacheKey{
			EventID: in.EventID,
			Voice:   voice,
			Speed:   speed,
			Version: version,
		}, data, contentType)
	}

	return ttsH.SynthesizeResult{Audio: data, ContentType: contentType}, nil
}

func (a *ttsAdapter) GetCache(_ context.Context, q ttsH.CacheLookup) (ttsH.SynthesizeResult, error) {
	s := a.srv
	if s.ttsCache == nil {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: TTS cache disabled", ttsH.ErrNotFound)
	}
	voice := q.Voice
	if voice == "" {
		voice = s.getTTSConfig().KokoroVoice
		if voice == "" {
			voice = "af_heart"
		}
	}
	speed := q.Speed
	if speed <= 0 {
		speed = 1.0
	}
	version := q.Version
	if version == "" {
		version = "active"
	}

	entry, ok := s.ttsCache.Get(TTSCacheKey{
		EventID: q.EventID,
		Voice:   voice,
		Speed:   speed,
		Version: version,
	})
	if !ok {
		return ttsH.SynthesizeResult{}, fmt.Errorf("%w: no cached audio for this event", ttsH.ErrNotFound)
	}
	return ttsH.SynthesizeResult{Audio: entry.Audio, ContentType: entry.ContentType}, nil
}

func (a *ttsAdapter) ListVoices(ctx context.Context) ([]ttsH.Voice, error) {
	s := a.srv
	if s.ttsVoiceLister == nil {
		return nil, fmt.Errorf("%w: TTS voice listing is not configured", ttsH.ErrFailedPrecondition)
	}
	if !s.capabilities.IsAvailable(ctx, "kokoro-tts") {
		return nil, fmt.Errorf("%w: Kokoro TTS is not available", ttsH.ErrUnavailable)
	}
	out, err := s.ttsVoiceLister.ListVoices(ctx)
	if err != nil {
		log.Printf("tts-voices: list failed: %v", err)
		return nil, fmt.Errorf("%w: failed to list voices", ttsH.ErrInternal)
	}
	voices := make([]ttsH.Voice, 0, len(out))
	for _, v := range out {
		voices = append(voices, ttsH.Voice{ID: v.ID, Name: v.Name})
	}
	return voices, nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func toAdapterConfig(c TTSConfig) ttsH.Config {
	return ttsH.Config{
		AutoEnabled: c.AutoEnabled,
		Backend:     c.Backend,
		KokoroVoice: c.KokoroVoice,
		KokoroSpeed: c.KokoroSpeed,
	}
}

func toAdapterSummarize(c TTSSummarizeConfig) ttsH.SummarizeConfig {
	return ttsH.SummarizeConfig{
		Enabled:        c.Enabled,
		CharThreshold:  c.CharThreshold,
		Level:          c.Level,
		Model:          c.Model,
		TimeoutSeconds: c.TimeoutSeconds,
	}
}

func appendResultToAdapter(r *ConversationAppendResult) *ttsH.AppendResult {
	if r == nil {
		return nil
	}
	return &ttsH.AppendResult{
		Appended:  r.Appended,
		Code:      r.Code,
		Reason:    r.Reason,
		Source:    r.Source,
		SessionID: r.SessionID,
		EventID:   r.EventID,
		Sequence:  r.Sequence,
		Duplicate: r.Duplicate,
	}
}

func ackToAdapter(a *TTSClientAck) *ttsH.ClientAck {
	if a == nil {
		return nil
	}
	return &ttsH.ClientAck{
		EventID:   a.EventID,
		Source:    a.Source,
		SessionID: a.SessionID,
		Stage:     a.Stage,
		Backend:   a.Backend,
		Message:   a.Message,
	}
}

func playbackToAdapter(p *TTSPlaybackEvent) *ttsH.PlaybackEvent {
	if p == nil {
		return nil
	}
	return &ttsH.PlaybackEvent{
		Source:    p.Source,
		Stage:     p.Stage,
		Backend:   p.Backend,
		SessionID: p.SessionID,
		Message:   p.Message,
	}
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// errors.Is forwarding used for static-analysis trick: keep this import in
// the file even if the rest doesn't reach it.
var _ = errors.Is
