// Package ttschain implements the text-to-speech provider chain.
//
// Production routing precedence: Local -> BYOK -> Vrooli/LPBS.
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
	Voice          string            // canonical id, e.g., "voice.feminine.warm"
	VoiceOverrides map[string]string // keyed by "tier:provider-id"
	Speed          float64
	ResponseFormat string // "mp3" | "wav" | "opus" | "flac"

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
	Audio       []byte
	ContentType string
	ContentHash string
	Tier        ProviderTier
	ProviderID  string
	ModelID     string
	VoiceUsed   string
	Latency     time.Duration
}

// seam: Provider is the TTS chain-provider seam (SEAMS.md row
// "ttschain.Provider"). Production wires Local/BYOK/Vrooli tiers; tests
// wire fakes from internal/ai/ttschain/mocks.
type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool
	Synthesize(ctx context.Context, req Request) (*Result, error)
	Model() string

	// StreamingCapability reports whether this provider can stream
	// audio frames natively (true) or only via the unary-wrapped
	// fallback (false). The streaming chain selects providers based on
	// this flag plus tier precedence.
	StreamingCapability() bool

	// SynthesizeStreaming runs a streaming synthesis session. Emits
	// AudioFrame values; the final frame has IsFinal=true and carries
	// the trace fields. Returning a nil channel + nil error means the
	// provider declined streaming (caller falls through to next tier).
	SynthesizeStreaming(ctx context.Context, req Request) (<-chan AudioFrame, error)
}

// AudioFrame is one frame in a streaming synthesis response. The final
// frame has IsFinal=true and the trace fields populated; intermediate
// frames leave trace fields zero.
type AudioFrame struct {
	Audio       []byte
	ContentType string
	IsFinal     bool

	// Trace populated on the final frame:
	Tier        ProviderTier
	ProviderID  string
	ModelID     string
	VoiceUsed   string
	Latency     time.Duration
	ContentHash string
	Err         error // populated when IsFinal=true and the stream errored.
}

var (
	ErrInsufficientCredits   = errors.New("audio-tools: insufficient credits for Vrooli tier")
	ErrAllProvidersFailed    = errors.New("audio-tools: all providers failed")
	ErrUnknownBYOKProvider   = errors.New("audio-tools: unknown BYOK provider")
	ErrMissingBYOKProvider   = errors.New("audio-tools: BYOK key set without BYOK provider")
	ErrUnknownCanonicalVoice = errors.New("audio-tools: canonical voice has no mapping for active adapter")
)
