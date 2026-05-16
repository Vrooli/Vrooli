package ttschain

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/tts"
)

// LocalProvider wraps tts.Service.Synthesize (Kokoro backend).
type LocalProvider struct {
	svc           *tts.Service
	capabilityKey string
}

func NewLocalProvider(svc *tts.Service) *LocalProvider {
	return &LocalProvider{svc: svc}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.svc == nil {
		return false
	}
	// tts.Service exposes Kokoro readiness via Deps.KokoroCapability inside
	// Synthesize; ttschain treats absence of an explicit error as ready.
	// A future enhancement may add a dedicated probe.
	return true
}

func (p *LocalProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.svc == nil {
		return nil, fmt.Errorf("audio-tools/ttschain: local provider not configured")
	}
	start := time.Now()
	in := tts.SynthesizeInput{
		Input:          req.Text,
		Voice:          resolveLocalVoice(req.Voice, req.VoiceOverrides),
		ResponseFormat: req.ResponseFormat,
		Speed:          req.Speed,
		EventID:        req.EventID,
		Version:        req.Version,
	}
	out, err := p.svc.Synthesize(ctx, in)
	if err != nil {
		return nil, err
	}
	return &Result{
		Audio:       out.Audio,
		ContentType: out.ContentType,
		Tier:        TierLocal,
		ProviderID:  "kokoro-local",
		ModelID:     "kokoro",
		VoiceUsed:   in.Voice,
		Latency:     time.Since(start),
	}, nil
}

func (p *LocalProvider) Model() string { return "kokoro" }

// resolveLocalVoice maps canonical voice IDs to Kokoro voice names using the
// shared canonical voice catalog. Overrides keyed "local:kokoro-local" win.
func resolveLocalVoice(canonical string, overrides map[string]string) string {
	if v, ok := overrides["local:kokoro-local"]; ok && v != "" {
		return v
	}
	// Fallback table — kept here until internal/tts/voice_catalog.go is wired.
	switch canonical {
	case "voice.feminine.warm":
		return "af_bella"
	case "voice.feminine.neutral":
		return "af_sarah"
	case "voice.masculine.warm":
		return "am_adam"
	case "voice.masculine.neutral":
		return "am_michael"
	case "voice.neutral.default":
		return "af_nicole"
	default:
		return canonical
	}
}
