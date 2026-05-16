// Package ttschain implements the text-to-speech provider chain.
//
// Routing precedence (fixed): BYOK -> Vrooli/LPBS -> Local.
// ErrInsufficientCredits short-circuits (no fallthrough to Local).
package ttschain

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
	Voice          string             // canonical id, e.g., "voice.feminine.warm"
	VoiceOverrides map[string]string  // keyed by "tier:provider-id"
	Speed          float64
	ResponseFormat string             // "mp3" | "wav" | "opus" | "flac"

	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string

	Model string

	// Optional cache control. EventID + Version snapshot the synthesized audio
	// into the event-index alongside the content-addressable store.
	EventID string
	Version string
}

type Result struct {
	Audio        []byte
	ContentType  string
	ContentHash  string
	Tier         ProviderTier
	ProviderID   string
	ModelID      string
	VoiceUsed    string
	Latency      time.Duration
}

type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool
	Synthesize(ctx context.Context, req Request) (*Result, error)
	Model() string
}

var (
	ErrInsufficientCredits  = errors.New("audio-tools: insufficient credits for Vrooli tier")
	ErrAllProvidersFailed   = errors.New("audio-tools: all providers failed")
	ErrUnknownBYOKProvider  = errors.New("audio-tools: unknown BYOK provider")
	ErrMissingBYOKProvider  = errors.New("audio-tools: BYOK key set without BYOK provider")
	ErrUnknownCanonicalVoice = errors.New("audio-tools: canonical voice has no mapping for active adapter")
)
