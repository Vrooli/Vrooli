package summarizechain

import (
	"context"
	"fmt"
	"strings"

	"audio-tools/internal/summarize"

	"github.com/vrooli/api-core/schedule"
)

// LocalProvider wraps summarize.Summarizer (Ollama backend).
type LocalProvider struct {
	summarizer   *summarize.Summarizer
	defaultModel string
	clk          schedule.Clock
}

func NewLocalProvider(summarizer *summarize.Summarizer, defaultModel string) *LocalProvider {
	return &LocalProvider{summarizer: summarizer, defaultModel: defaultModel, clk: schedule.System()}
}

// NewLocalProviderWith constructs a LocalProvider with a custom schedule.
func NewLocalProviderWith(summarizer *summarize.Summarizer, defaultModel string, clk schedule.Clock) *LocalProvider {
	if clk == nil {
		clk = schedule.System()
	}
	return &LocalProvider{summarizer: summarizer, defaultModel: defaultModel, clk: clk}
}

func (p *LocalProvider) Type() ProviderTier { return TierLocal }

func (p *LocalProvider) IsAvailable(ctx context.Context) bool {
	return p != nil && p.summarizer != nil && (p.summarizer.Runner != nil || p.summarizer.Bin != "")
}

func (p *LocalProvider) Summarize(ctx context.Context, req Request) (*Result, error) {
	if p == nil || p.summarizer == nil {
		return nil, fmt.Errorf("audio-tools/summarizechain: local provider not configured")
	}
	model := req.LocalModelRole
	if model == "" {
		model = req.Model
	}
	if model == "" {
		model = p.defaultModel
	}
	clk := p.clk
	if clk == nil {
		clk = schedule.System()
	}
	start := clk.Now()
	resp, err := p.summarizer.Summarize(ctx, req.Text, model, req.Level)
	if err != nil {
		return nil, err
	}
	content := summarize.PreserveExplicitAssignments(req.Text, resp.Content)
	resolvedModel := strings.TrimSpace(resp.Model)
	if resolvedModel == "" {
		// Older gateway adapters may omit the resolved tag. Preserve the
		// selector as a compatibility fallback, but prefer the runtime fact.
		resolvedModel = model
	}
	return &Result{
		Text:         content,
		OutputTokens: resp.EvalCount,
		Tier:         TierLocal,
		ProviderID:   "ollama-local",
		ModelID:      resolvedModel,
		Latency:      clk.Now().Sub(start),
	}, nil
}

func (p *LocalProvider) Model() string {
	if p == nil {
		return ""
	}
	return p.defaultModel
}
