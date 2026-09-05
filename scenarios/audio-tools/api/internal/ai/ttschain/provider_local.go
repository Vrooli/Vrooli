package ttschain

import (
	"context"
	"fmt"

	"audio-tools/internal/tts"

	"github.com/vrooli/api-core/schedule"
)

// LocalProvider wraps tts.Service.Synthesize (Kokoro backend).
type LocalProvider struct {
	svc *tts.Service
	clk schedule.Clock
}

func NewLocalProvider(svc *tts.Service) *LocalProvider {
	return &LocalProvider{svc: svc, clk: schedule.System()}
}

// NewLocalProviderWith constructs a LocalProvider with a custom schedule.
func NewLocalProviderWith(svc *tts.Service, clk schedule.Clock) *LocalProvider {
	if clk == nil {
		clk = schedule.System()
	}
	return &LocalProvider{svc: svc, clk: clk}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

// IsAvailable reports whether the local TTS backend can actually serve a
// request. It consults the same readiness signal Synthesize gates on, so the
// availability a caller is shown and the availability it will get are the same
// fact. Returning an unconditional true here made every operator surface --
// `settings providers`, the TTS status capability label -- report the local
// tier as up while Kokoro was not running at all.
func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.svc == nil {
		return false
	}
	return p.svc.LocalBackendReady(ctx)
}

func (p *LocalProvider) Synthesize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.svc == nil {
		return nil, fmt.Errorf("audio-tools/ttschain: local provider not configured")
	}
	clk := p.clk
	if clk == nil {
		clk = schedule.System()
	}
	start := clk.Now()
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
		Latency:     clk.Now().Sub(start),
	}, nil
}

func (p *LocalProvider) Model() string { return "kokoro" }

// StreamingCapability for Local Kokoro is false today; Phase D upgrades
// this to true once incremental synthesis is wired.
func (p *LocalProvider) StreamingCapability() bool { return false }

func (p *LocalProvider) SynthesizeStreaming(_ context.Context, _ Request) (<-chan AudioFrame, error) {
	return nil, nil
}

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
