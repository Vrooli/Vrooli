// Package sttchain implements the speech-to-text provider chain.
//
// Routing precedence (fixed): BYOK -> Vrooli/LPBS -> Local.
// ErrInsufficientCredits from the Vrooli tier short-circuits and does NOT
// fall through to Local — credit exhaustion is a billing failure, not a
// capability failure. Per-request credentials travel through Request.
//
// Mirrors the BAS chain pattern at scenarios/browser-automation-studio/api/services/ai/.
package sttchain

import (
	"context"
	"errors"
	"time"
)

// ProviderTier identifies which tier handled a request.
type ProviderTier string

const (
	TierBYOK   ProviderTier = "byok"
	TierVrooli ProviderTier = "vrooli"
	TierLocal  ProviderTier = "local"
)

// Request is the per-call input to the chain.
type Request struct {
	Audio                   []byte
	Format                  string
	Language                string
	SkipSpeakerVerification bool
	InitialPrompt           string

	// Per-request creds carried via headers and injected by the handler interceptor.
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string

	// Optional override; chain default model applies when empty.
	Model string
}

// Result is the chain's response, including provider trace.
type Result struct {
	Text             string
	DetectedLanguage string
	DurationSeconds  float64
	Tier             ProviderTier
	ProviderID       string
	ModelID          string
	Latency          time.Duration
}

// StrategyKind identifies a streaming strategy. The StrategySelector
// uses these constants both to enumerate the global compatibility matrix
// and to interpret the optional per-provider whitelist in ProviderTraits.
type StrategyKind string

const (
	// StrategyVADSegment chunks the input by silence-bounded VAD segments
	// and calls Provider.Transcribe once per segment. Suitable for
	// batch-only providers and the Local Whisper tier.
	StrategyVADSegment StrategyKind = "vad_segment"
	// StrategyOverlapAgree runs sliding overlapping windows over the
	// input, committing prefixes that agree across consecutive runs
	// (LocalAgreement; Macháček et al. 2023). Quality upgrade for
	// batch-only local providers.
	StrategyOverlapAgree StrategyKind = "overlap_agree"
	// StrategyPassthrough forwards chunks directly to a native-streaming
	// provider and translates its event stream back. Used for providers
	// like Deepgram, Azure, Google, and future LPBS streaming.
	StrategyPassthrough StrategyKind = "passthrough"
	// StrategyBuffered drains the input, runs a single batch call at end,
	// and emits one Segment + Done. The selector picks this when
	// streaming_mode=off or when no eligible (strategy, provider) pair
	// exists for the negotiated session.
	StrategyBuffered StrategyKind = "buffered_fallback"
)

// ProviderTraits is the capability struct read once at stream-start by
// the StrategySelector. It replaces the older boolean
// StreamingCapability() seam: a provider that declares Stream=true
// implements TranscribeStreaming natively; one with Batch=true
// implements Transcribe. The Strategies whitelist, when non-empty,
// narrows which strategies the selector may pair with this provider —
// empty means "the selector decides per the global default matrix."
type ProviderTraits struct {
	Batch      bool
	Stream     bool
	Strategies []StrategyKind
}

// Supports reports whether the provider declares the given strategy in
// its whitelist (or accepts any strategy when the whitelist is empty).
func (t ProviderTraits) Supports(k StrategyKind) bool {
	if len(t.Strategies) == 0 {
		return true
	}
	for _, s := range t.Strategies {
		if s == k {
			return true
		}
	}
	return false
}

// Provider is the interface implemented by Local/BYOK/Vrooli tiers.
//
// Each concrete provider is short-lived (constructed per-request from a
// factory) so it can capture per-request creds without leaking them across
// callers. Availability is cached at the chain level with a per-tier TTL.
type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool
	Transcribe(ctx context.Context, req Request) (*Result, error)
	Model() string

	// Traits reports the provider's streaming capabilities and the
	// optional strategy whitelist. The StrategySelector reads this once
	// at session start; providers must not mutate it across calls.
	Traits() ProviderTraits

	// TranscribeStreaming runs a streaming transcription session.
	// Implementations:
	//   - Read audio bytes from `chunks` until it closes.
	//   - Emit StreamEvent values on the returned channel.
	//   - Close the returned channel after emitting the final event.
	// Returning a nil channel + non-nil error means stream-start was
	// rejected (provider not eligible); the chain falls through to the
	// next tier or to buffered mode. Returning a nil channel + nil error
	// is reserved for adapters whose Traits().Stream is false.
	TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error)
}

// StreamStart is the metadata header that opens a streaming session.
// Carried out-of-band so the provider can negotiate format/language
// before any audio arrives.
type StreamStart struct {
	Language                string
	InitialPrompt           string
	SkipSpeakerVerification bool
	SampleRate              int32 // 0 = adapter default
	// Per-request creds (same shape as Request) so streaming sessions
	// can carry BYOK/LPBS credentials without piggy-backing on Request.
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string
}

// AudioChunk is one chunk of raw audio bytes in the stream input channel.
// Format is implicit (provider-dependent; usually 16-bit PCM @ sample_rate).
type AudioChunk struct {
	Audio []byte
}

// StreamEventKind enumerates the event types emitted on the output channel.
type StreamEventKind string

const (
	StreamEventPartial          StreamEventKind = "partial"
	StreamEventSegment          StreamEventKind = "segment"
	StreamEventWakeWord         StreamEventKind = "wake_word"
	StreamEventSpeakerRejection StreamEventKind = "speaker_rejection"
	StreamEventError            StreamEventKind = "error"
	StreamEventDone             StreamEventKind = "done"
)

// StreamEvent is a tagged union of streaming events. Exactly one of the
// *Event pointer fields is populated per emission, matching Kind.
type StreamEvent struct {
	Kind StreamEventKind

	Partial          *PartialEvent
	Segment          *SegmentEvent
	WakeWord         *WakeWordEvent
	SpeakerRejection *SpeakerRejectionEvent
	Error            error
	Done             *DoneEvent
}

type PartialEvent struct {
	Text string
}

type SegmentEvent struct {
	Text             string
	StartMs          int64
	EndMs            int64
	DetectedLanguage string
	// Per-segment trace; the chain may stamp these from the locked tier
	// if the adapter doesn't fill them in.
	ProviderTier ProviderTier
	ProviderID   string
	ModelID      string
	LatencyMs    float64
}

type WakeWordEvent struct {
	Score    float64
	SampleID string
}

type SpeakerRejectionEvent struct {
	Reason       string
	FallbackUsed bool
}

type DoneEvent struct {
	FinalText       string
	LockedTier      ProviderTier
	ProviderID      string
	ModelID         string
	LatencyMs       float64
	FellBackToUnary bool
}

// ErrInsufficientCredits is returned by the Vrooli provider when LPBS reports
// the user is out of audio credits. The chain MUST NOT fall through to Local
// on this error — it is a billing failure, not an availability failure.
var ErrInsufficientCredits = errors.New("audio-tools: insufficient credits for Vrooli tier")

// ErrAllProvidersFailed is returned when every enabled tier was tried and
// none succeeded.
var ErrAllProvidersFailed = errors.New("audio-tools: all providers failed")

// ErrUnknownBYOKProvider is returned when the BYOK provider header (see
// envelope.HeaderProvider) names a provider not in the registry.
var ErrUnknownBYOKProvider = errors.New("audio-tools: unknown BYOK provider")

// ErrMissingBYOKProvider is returned when the BYOK key header is set but the
// BYOK provider header is absent — silent provider selection is forbidden.
var ErrMissingBYOKProvider = errors.New("audio-tools: BYOK key set without BYOK provider")
