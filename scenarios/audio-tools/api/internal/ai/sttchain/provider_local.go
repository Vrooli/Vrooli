package sttchain

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/voice"
)

// LocalProvider wraps voice.Service.Transcribe (Whisper backend).
type LocalProvider struct {
	svc *voice.Service
	// availability cache is owned by the chain; LocalProvider exposes a cheap
	// readiness check via the voice service.
}

// NewLocalProvider constructs a Local STT provider.
func NewLocalProvider(svc *voice.Service) *LocalProvider {
	return &LocalProvider{svc: svc}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	if p == nil || p.svc == nil {
		return false
	}
	return p.svc.WhisperAvailable(ctx)
}

func (p *LocalProvider) Transcribe(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.svc == nil {
		return nil, fmt.Errorf("audio-tools/sttchain: local provider not configured")
	}
	start := time.Now()
	text, err := p.svc.Transcribe(ctx, req.Audio, req.Language)
	if err != nil {
		return nil, err
	}
	return &Result{
		Text:             text,
		DetectedLanguage: req.Language,
		Tier:             TierLocal,
		ProviderID:       "whisper-local",
		ModelID:          "whisper-large-v3",
		Latency:          time.Since(start),
	}, nil
}

func (p *LocalProvider) Model() string { return "whisper-large-v3" }
