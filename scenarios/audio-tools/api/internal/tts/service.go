package tts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/logx"
)

const (
	defaultVoice             = "af_heart"
	maxSynthesizeInputLength = 5000
	maxSynthesizeAudioBytes  = 64 << 20
	minSummarizeTimeout      = 15
	maxSummarizeTimeout      = 300
	maxTTSSpeed              = 4.0
)

var errResponseBodyTooLarge = errors.New("response exceeds size limit")

func readBoundedResponse(body io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 {
		return nil, fmt.Errorf("invalid synthesis response limit %d", maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errResponseBodyTooLarge
	}
	return data, nil
}

func normalizeSynthesisSpeed(speed float64) (float64, error) {
	if math.IsNaN(speed) || math.IsInf(speed, 0) {
		return 0, fmt.Errorf("%w: speed must be a finite number", ErrInvalidArgument)
	}
	if speed <= 0 {
		return 1, nil
	}
	if speed > maxTTSSpeed {
		return maxTTSSpeed, nil
	}
	return speed, nil
}

type CacheKey struct {
	EventID        string
	Backend        string
	Voice          string
	Speed          float64
	ResponseFormat string
	Version        string
	// ChunkIndex disambiguates the N ordered paragraphs of a single event so
	// per-paragraph audio does not collide on one whole-message key. 0 for
	// single-chunk / whole-message synthesis.
	ChunkIndex int32
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

	KokoroCapability          func(context.Context) (string, string)
	InvalidateBackendLiveness func()
	SynthesizeAudio           func(context.Context, SynthesizeInput) (io.ReadCloser, string, error)
	GetCache                  func(CacheKey) (SynthesizeResult, bool)
	PutCache                  func(CacheKey, []byte, string)
	CacheBackend              string
	ListVoiceCatalog          func(context.Context) ([]Voice, error)

	Logger logx.Logger
}

type Service struct {
	deps     Deps
	configMu sync.RWMutex
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

// InvalidateBackendLiveness expires the cached provider status after a
// transport failure so the next request can observe a recovery promptly.
func (s *Service) InvalidateBackendLiveness() {
	if s != nil && s.deps.InvalidateBackendLiveness != nil {
		s.deps.InvalidateBackendLiveness()
	}
}

func (s *Service) GetConfig(_ context.Context) (Config, error) {
	return s.getConfig(), nil
}

func (s *Service) UpdateConfig(_ context.Context, patch ConfigPatch) (Config, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	current := s.deps.GetConfig()
	updated := current
	if patch.AutoEnabled != nil {
		updated.AutoEnabled = *patch.AutoEnabled
	}
	if patch.DefaultResponseFormat != nil {
		updated.DefaultResponseFormat = *patch.DefaultResponseFormat
	}
	if patch.KokoroVoice != nil {
		updated.KokoroVoice = *patch.KokoroVoice
	}
	if patch.KokoroSpeed != nil {
		updated.KokoroSpeed = *patch.KokoroSpeed
	}

	if updated.DefaultResponseFormat == "" {
		updated.DefaultResponseFormat = DefaultResponseFormat
	}
	if _, ok := audioformat.OutputFormatFromString(updated.DefaultResponseFormat); !ok {
		return Config{}, fmt.Errorf("%w: defaultResponseFormat must be mp3, wav, opus, or flac", ErrInvalidArgument)
	}
	if updated.KokoroVoice == "" {
		updated.KokoroVoice = defaultVoice
	}
	if math.IsNaN(updated.KokoroSpeed) || math.IsInf(updated.KokoroSpeed, 0) || updated.KokoroSpeed < 0.5 || updated.KokoroSpeed > maxTTSSpeed {
		return Config{}, fmt.Errorf("%w: kokoroSpeed must be between 0.5 and 4.0", ErrInvalidArgument)
	}

	s.deps.SetConfig(updated)
	if s.deps.PersistConfig != nil {
		if err := s.deps.PersistConfig(updated); err != nil {
			s.logger().Printf("tts-config: persist failed (in-memory updated): %v", err)
		}
	}
	s.logger().Printf("tts-config: updated: autoEnabled=%v responseFormat=%s voice=%s speed=%.1f",
		updated.AutoEnabled, updated.DefaultResponseFormat, updated.KokoroVoice, updated.KokoroSpeed)
	return updated, nil
}

func (s *Service) getConfig() Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.deps.GetConfig()
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
		Config:                s.getConfig(),
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

// LocalBackendReady reports whether the local synthesis backend is wired and
// its cheap liveness probe currently says it can serve. The same predicate
// gates Synthesize and answers provider-chain availability; full synthesis
// probes are reserved for explicit readiness sweeps.
func (s *Service) LocalBackendReady(ctx context.Context) bool {
	if s == nil || s.deps.SynthesizeAudio == nil || s.deps.KokoroCapability == nil {
		return false
	}
	capability, _ := s.deps.KokoroCapability(ctx)
	return capability == "available"
}

// InvalidateBackendReadiness clears a cached negative liveness result after a
// managed resource start, so the ensuing request verifies the newly started
// endpoint instead of waiting for the ordinary capability TTL.
func (s *Service) InvalidateBackendReadiness() {
	if s != nil && s.deps.InvalidateBackendLiveness != nil {
		s.deps.InvalidateBackendLiveness()
	}
}

func (s *Service) Synthesize(ctx context.Context, in SynthesizeInput) (SynthesizeResult, error) {
	if s.deps.SynthesizeAudio == nil {
		return SynthesizeResult{}, fmt.Errorf("%w: TTS synthesis is not configured", ErrFailedPrecondition)
	}
	if !s.LocalBackendReady(ctx) {
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
		normalized.Voice = s.getConfig().KokoroVoice
		if normalized.Voice == "" {
			normalized.Voice = defaultVoice
		}
	}
	if normalized.ResponseFormat == "" {
		normalized.ResponseFormat = s.getConfig().DefaultResponseFormat
		if normalized.ResponseFormat == "" {
			normalized.ResponseFormat = DefaultResponseFormat
		}
	}
	// The audioformat substrate is the single source of truth for the TTS
	// output-format vocabulary (mp3/wav/opus/flac). The kokoro engine
	// encodes to the requested format itself (as Whisper decodes ingress
	// itself), so this is a vocabulary check, not a re-encode.
	if _, ok := audioformat.OutputFormatFromString(normalized.ResponseFormat); !ok {
		return SynthesizeResult{}, fmt.Errorf("%w: unsupported response_format; use mp3, wav, opus, or flac", ErrInvalidArgument)
	}
	speed, err := normalizeSynthesisSpeed(normalized.Speed)
	if err != nil {
		return SynthesizeResult{}, err
	}
	normalized.Speed = speed

	body, contentType, err := s.deps.SynthesizeAudio(ctx, normalized)
	if err != nil {
		// Preserve caller cancellation and deadlines through the service layer.
		// In particular, do not turn a canceled request into an internal error
		// or invalidate a healthy backend for a client-side disconnect.
		if ctx.Err() != nil {
			return SynthesizeResult{}, ctx.Err()
		}
		s.logger().Printf("tts-synthesize: synthesis failed: %v", err)
		if errors.Is(err, ErrUnavailable) {
			if s.deps.InvalidateBackendLiveness != nil {
				s.deps.InvalidateBackendLiveness()
			}
			return SynthesizeResult{}, fmt.Errorf("%w: synthesis backend unavailable", ErrUnavailable)
		}
		if errors.Is(err, ErrInvalidArgument) {
			return SynthesizeResult{}, err
		}
		return SynthesizeResult{}, fmt.Errorf("%w: synthesis failed", ErrInternal)
	}
	defer body.Close()

	data, readErr := readBoundedResponse(body, maxSynthesizeAudioBytes)
	if readErr != nil {
		s.logger().Printf("tts-synthesize: read response: %v", readErr)
		if ctx.Err() != nil {
			return SynthesizeResult{}, ctx.Err()
		}
		if s.deps.InvalidateBackendLiveness != nil {
			s.deps.InvalidateBackendLiveness()
		}
		if errors.Is(readErr, errResponseBodyTooLarge) {
			return SynthesizeResult{}, fmt.Errorf("%w: synthesis response exceeds %d-byte limit", ErrUnavailable, maxSynthesizeAudioBytes)
		}
		return SynthesizeResult{}, fmt.Errorf("%w: synthesis response could not be read", ErrUnavailable)
	}
	// The response reader can complete successfully even when the caller
	// cancels during its final read. Do not return or persist an event snapshot
	// after cancellation merely because the transport delivered all bytes.
	if ctx.Err() != nil {
		return SynthesizeResult{}, ctx.Err()
	}

	if normalized.EventID != "" && s.deps.PutCache != nil && len(data) > 0 {
		version := normalized.Version
		if version == "" {
			version = "active"
		}
		s.deps.PutCache(CacheKey{
			EventID:        normalized.EventID,
			Backend:        s.deps.CacheBackend,
			Voice:          normalized.Voice,
			Speed:          normalized.Speed,
			ResponseFormat: normalized.ResponseFormat,
			Version:        version,
			ChunkIndex:     normalized.ChunkIndex,
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
		voice = s.getConfig().KokoroVoice
		if voice == "" {
			voice = defaultVoice
		}
	}
	speed, err := normalizeSynthesisSpeed(q.Speed)
	if err != nil {
		return SynthesizeResult{}, err
	}
	version := q.Version
	if version == "" {
		version = "active"
	}
	responseFormat := q.ResponseFormat
	if responseFormat == "" {
		responseFormat = s.getConfig().DefaultResponseFormat
		if responseFormat == "" {
			responseFormat = DefaultResponseFormat
		}
	}
	if _, ok := audioformat.OutputFormatFromString(responseFormat); !ok {
		return SynthesizeResult{}, fmt.Errorf("%w: unsupported response_format; use mp3, wav, opus, or flac", ErrInvalidArgument)
	}
	out, ok := s.deps.GetCache(CacheKey{
		EventID:        q.EventID,
		Backend:        s.deps.CacheBackend,
		Voice:          voice,
		Speed:          speed,
		ResponseFormat: responseFormat,
		Version:        version,
		ChunkIndex:     q.ChunkIndex,
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
