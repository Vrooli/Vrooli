package summarizechain

import (
	"context"
	"fmt"
	"time"

	"audio-tools/internal/summarize"
)

// LocalProvider wraps summarize.Summarizer (Ollama backend).
type LocalProvider struct {
	summarizer   *summarize.Summarizer
	defaultModel string
}

func NewLocalProvider(summarizer *summarize.Summarizer, defaultModel string) *LocalProvider {
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
