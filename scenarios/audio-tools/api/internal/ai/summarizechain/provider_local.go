package summarizechain

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/tts"
)

// LocalProvider wraps tts.Summarizer (Ollama backend).
//
// The Ollama summarization code currently lives inside internal/tts because
// historically it was a TTS-only summarization step. summarizechain re-exposes
// it through the chain interface; a future refactor may lift the Summarizer
// into internal/summarize and remove the cross-package import.
type LocalProvider struct {
	summarizer  *tts.Summarizer
	defaultModel string
}

func NewLocalProvider(summarizer *tts.Summarizer, defaultModel string) *LocalProvider {
	return &LocalProvider{summarizer: summarizer, defaultModel: defaultModel}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	return p != nil && p.summarizer != nil && p.summarizer.BaseURL != ""
}

func (p *LocalProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.summarizer == nil {
		return nil, fmt.Errorf("audio-tools/summarizechain: local provider not configured")
	}
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	start := time.Now()
	resp, err := p.summarizer.Summarize(ctx, req.Text, model, req.Level)
	if err != nil {
		return nil, err
	}
	return &Result{
		Text:         resp.Content,
		OutputTokens: resp.EvalCount,
		Tier:         TierLocal,
		ProviderID:   "ollama-local",
		ModelID:      model,
		Latency:      time.Since(start),
	}, nil
}

func (p *LocalProvider) Model() string {
	if p == nil {
		return ""
	}
	return p.defaultModel
}
