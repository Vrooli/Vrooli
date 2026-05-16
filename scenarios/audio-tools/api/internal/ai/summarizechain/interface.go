// Package summarizechain implements the summarization provider chain.
//
// Routing precedence (fixed): BYOK -> Vrooli/LPBS -> Local.
// Local is backed by internal/summarize.OllamaClient; BYOK starter is
// openrouter; Vrooli routes through LPBS chat with `Operation: audio.summarize`.
package summarizechain

import (
	"context"
	"errors"
	"time"
)

type ProviderTier string

const (
	TierBYOK   ProviderTier = "byok"
	TierVrooli ProviderTier = "vrooli"
	TierLocal  ProviderTier = "local"
)

type Request struct {
	Text           string
	Level          string             // "light" | "moderate" | "heavy"
	Model          string             // optional override
	TimeoutSeconds int

	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string
}

type Result struct {
	Text         string
	PromptTokens int
	OutputTokens int
	Tier         ProviderTier
	ProviderID   string
	ModelID      string
	Latency      time.Duration
}

type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool
	Summarize(ctx context.Context, req Request) (*Result, error)
	Model() string
}

var (
	ErrInsufficientCredits = errors.New("audio-tools: insufficient credits for Vrooli tier")
	ErrAllProvidersFailed  = errors.New("audio-tools: all providers failed")
	ErrUnknownBYOKProvider = errors.New("audio-tools: unknown BYOK provider")
	ErrMissingBYOKProvider = errors.New("audio-tools: BYOK key set without BYOK provider")
)
