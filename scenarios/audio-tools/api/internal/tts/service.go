package tts

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"audio-tools/internal/logx"
)

const (
	defaultVoice             = "af_heart"
	maxSynthesizeInputLength = 5000
	minSummarizeTimeout      = 15
	maxSummarizeTimeout      = 300
	maxTTSSpeed              = 4.0
)

var formatContentTypes = map[string]struct{}{
	"mp3":  {},
	"wav":  {},
	"opus": {},
	"flac": {},
}

type CacheKey struct {
	EventID string
	Voice   string
	Speed   float64
	Version string
}

type Deps struct {
	GetConfig     func() Config
	SetConfig     func(Config)
	PersistConfig func(Config) error

	GetHookStatus        func() (bool, string, string, string)
	GetLastRouting       func() (*AppendResult, time.Time)
	GetRoutingBySource   func(string) (*AppendResult, time.Time)
	GetLastAck           func() (*ClientAck, time.Time)
	GetAckBySource       func(string) (*ClientAck, time.Time)
	GetLastPlaybackEvent func() (*PlaybackEvent, time.Time)
	RecordPlaybackEvent  func(PlaybackEvent)

	KokoroCapability func(context.Context) (string, string)
	SynthesizeAudio  func(context.Context, SynthesizeInput) (io.ReadCloser, string, error)
	GetCache         func(CacheKey) (SynthesizeResult, bool)
	PutCache         func(CacheKey, []byte, string)
	ListVoiceCatalog func(context.Context) ([]Voice, error)

	Logger logx.Logger
}

type Service struct {
	deps Deps
}

// NewService constructs a TTS Service. Deps.Logger is required (no
// fallback); a nil value panics so a forgotten wire-up surfaces at
// boot, not at request-time.
func NewService(d Deps) *Service {
	if d.Logger == nil {
		panic("tts.NewService requires Deps.Logger")
	}
	return &Service{deps: d}
}

func (s *Service) logger() logx.Logger { return s.deps.Logger }

func (s *Service) GetConfig(_ context.Context) (Config, error) {
	return s.deps.GetConfig(), nil
}

func (s *Service) UpdateConfig(_ context.Context, patch ConfigPatch) (Config, error) {
	current := s.deps.GetConfig()
	updated := current
	if patch.AutoEnabled != nil {
		updated.AutoEnabled = *patch.AutoEnabled
	}
	if patch.Backend != nil {
		updated.Backend = *patch.Backend
	}
	if patch.KokoroVoice != nil {
		updated.KokoroVoice = *patch.KokoroVoice
	}
	if patch.KokoroSpeed != nil {
		updated.KokoroSpeed = *patch.KokoroSpeed
	}

	switch updated.Backend {
	case "", "auto", "kokoro", "browser":
		if updated.Backend == "" {
			updated.Backend = "auto"
		}
	default:
		return Config{}, fmt.Errorf("%w: backend must be auto, kokoro, or browser", ErrInvalidArgument)
	}
	if updated.KokoroVoice == "" {
		updated.KokoroVoice = defaultVoice
	}
	if updated.KokoroSpeed < 0.5 || updated.KokoroSpeed > maxTTSSpeed {
		return Config{}, fmt.Errorf("%w: kokoroSpeed must be between 0.5 and 4.0", ErrInvalidArgument)
	}

	s.deps.SetConfig(updated)
	if s.deps.PersistConfig != nil {
		if err := s.deps.PersistConfig(updated); err != nil {
			s.logger().Printf("tts-config: persist failed (in-memory updated): %v", err)
		}
	}
	s.logger().Printf("tts-config: updated: autoEnabled=%v backend=%s voice=%s speed=%.1f",
		updated.AutoEnabled, updated.Backend, updated.KokoroVoice, updated.KokoroSpeed)
	return updated, nil
}

func (s *Service) GetStatus(ctx context.Context) (Status, error) {
	hookRegistered, hookCode, hookReason, hookSettingsPath := s.deps.GetHookStatus()
	lastRouting, lastRoutingAt := s.deps.GetLastRouting()
	lastHookRouting, lastHookRoutingAt := s.deps.GetRoutingBySource("claude_hook")
	lastTailerRouting, lastTailerRoutingAt := s.deps.GetRoutingBySource("codex_tailer")
	lastAck, lastAckAt := s.deps.GetLastAck()
	lastHookAck, lastHookAckAt := s.deps.GetAckBySource("claude_hook")
	lastTailerAck, lastTailerAckAt := s.deps.GetAckBySource("codex_tailer")
	lastPlaybackEvent, lastPlaybackAt := s.deps.GetLastPlaybackEvent()
	kokoroCapability, kokoroCapabilityLabel := s.deps.KokoroCapability(ctx)

	return Status{
		Config:                s.deps.GetConfig(),
		HookRegistered:        hookRegistered,
		HookCode:              hookCode,
		HookReason:            hookReason,
		HookSettingsPath:      hookSettingsPath,
		LastRouting:           lastRouting,
		LastRoutingAt:         formatTime(lastRoutingAt),
		LastHookRouting:       lastHookRouting,
		LastHookRoutingAt:     formatTime(lastHookRoutingAt),
		LastTailerRouting:     lastTailerRouting,
		LastTailerRoutingAt:   formatTime(lastTailerRoutingAt),
		LastAck:               lastAck,
		LastAckAt:             formatTime(lastAckAt),
		LastHookAck:           lastHookAck,
		LastHookAckAt:         formatTime(lastHookAckAt),
		LastTailerAck:         lastTailerAck,
		LastTailerAckAt:       formatTime(lastTailerAckAt),
		LastPlaybackEvent:     lastPlaybackEvent,
		LastPlaybackAt:        formatTime(lastPlaybackAt),
		KokoroCapability:      kokoroCapability,
		KokoroCapabilityLabel: kokoroCapabilityLabel,
	}, nil
}

func (s *Service) RecordPlaybackEvent(_ context.Context, ev PlaybackEvent) error {
	s.deps.RecordPlaybackEvent(ev)
	return nil
}

func (s *Service) Synthesize(ctx context.Context, in SynthesizeInput) (SynthesizeResult, error) {
	if s.deps.SynthesizeAudio == nil {
		return SynthesizeResult{}, fmt.Errorf("%w: TTS synthesis is not configured", ErrFailedPrecondition)
	}
	kokoroCapability, _ := s.deps.KokoroCapability(ctx)
	if kokoroCapability != "available" {
		return SynthesizeResult{}, fmt.Errorf("%w: Kokoro TTS is not available", ErrUnavailable)
	}

	input := strings.TrimSpace(in.Input)
	if input == "" {
		return SynthesizeResult{}, fmt.Errorf("%w: input is required", ErrInvalidArgument)
	}
	if len(input) > maxSynthesizeInputLength {
		s.logger().Printf("tts-synthesize: input too long (%d chars, limit %d)", len(input), maxSynthesizeInputLength)
		return SynthesizeResult{}, fmt.Errorf("%w: input exceeds maximum length of 5000 characters", ErrInvalidArgument)
	}

	normalized := in
	normalized.Input = input
	if normalized.Voice == "" {
		normalized.Voice = s.deps.GetConfig().KokoroVoice
		if normalized.Voice == "" {
			normalized.Voice = defaultVoice
		}
	}
	if normalized.ResponseFormat == "" {
		normalized.ResponseFormat = "mp3"
	}
	if _, ok := formatContentTypes[normalized.ResponseFormat]; !ok {
		return SynthesizeResult{}, fmt.Errorf("%w: unsupported response_format; use mp3, wav, opus, or flac", ErrInvalidArgument)
	}
	if normalized.Speed <= 0 {
		normalized.Speed = 1.0
	} else if normalized.Speed > maxTTSSpeed {
		normalized.Speed = maxTTSSpeed
	}

	body, contentType, err := s.deps.SynthesizeAudio(ctx, normalized)
	if err != nil {
		s.logger().Printf("tts-synthesize: synthesis failed: %v", err)
		return SynthesizeResult{}, fmt.Errorf("%w: synthesis failed", ErrInternal)
	}
	defer body.Close()

	data, readErr := io.ReadAll(body)
	if readErr != nil {
		s.logger().Printf("tts-synthesize: read response: %v", readErr)
		return SynthesizeResult{}, fmt.Errorf("%w: synthesis failed", ErrInternal)
	}

	if normalized.EventID != "" && s.deps.PutCache != nil && len(data) > 0 {
		version := normalized.Version
		if version == "" {
			version = "active"
		}
		s.deps.PutCache(CacheKey{
			EventID: normalized.EventID,
			Voice:   normalized.Voice,
			Speed:   normalized.Speed,
			Version: version,
		}, data, contentType)
	}

	return SynthesizeResult{Audio: data, ContentType: contentType}, nil
}

func (s *Service) GetCache(_ context.Context, q CacheLookup) (SynthesizeResult, error) {
	if s.deps.GetCache == nil {
		return SynthesizeResult{}, fmt.Errorf("%w: TTS cache disabled", ErrNotFound)
	}
	voice := q.Voice
	if voice == "" {
		voice = s.deps.GetConfig().KokoroVoice
		if voice == "" {
			voice = defaultVoice
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

	out, ok := s.deps.GetCache(CacheKey{
		EventID: q.EventID,
		Voice:   voice,
		Speed:   speed,
		Version: version,
	})
	if !ok {
		return SynthesizeResult{}, fmt.Errorf("%w: no cached audio for this event", ErrNotFound)
	}
	return out, nil
}

func (s *Service) ListVoices(ctx context.Context) ([]Voice, error) {
	if s.deps.ListVoiceCatalog == nil {
		return nil, fmt.Errorf("%w: TTS voice listing is not configured", ErrFailedPrecondition)
	}
	kokoroCapability, _ := s.deps.KokoroCapability(ctx)
	if kokoroCapability != "available" {
		return nil, fmt.Errorf("%w: Kokoro TTS is not available", ErrUnavailable)
	}
	out, err := s.deps.ListVoiceCatalog(ctx)
	if err != nil {
		s.logger().Printf("tts-voices: list failed: %v", err)
		return nil, fmt.Errorf("%w: failed to list voices", ErrInternal)
	}
	return out, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
