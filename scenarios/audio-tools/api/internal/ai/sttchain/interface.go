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
	Audio                    []byte
	Format                   string
	Language                 string
	SkipSpeakerVerification  bool
	InitialPrompt            string

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

	// StreamingCapability reports whether this provider can stream
	// transcription events natively (true) or only via the buffered
	// fall-back (false). The streaming chain filters by this flag at
	// stream-start tier negotiation; providers that return false are
	// only selected when no streaming-capable tier is available.
	StreamingCapability() bool

	// TranscribeStreaming runs a streaming transcription session.
	// Implementations:
	//   - Read audio bytes from `chunks` until it closes.
	//   - Emit StreamEvent values on the returned channel.
	//   - Close the returned channel after emitting the final event.
	// Returning a nil channel + non-nil error means stream-start was
	// rejected (provider not eligible); the chain falls through to the
	// next tier or to buffered mode. Returning a nil channel + nil error
	// is reserved for adapters that declare StreamingCapability=false.
	TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error)
}

// StreamStart is the metadata header that opens a streaming session.
// Carried out-of-band so the provider can negotiate format/language
// before any audio arrives.
type StreamStart struct {
	Language               string
	InitialPrompt          string
	SkipSpeakerVerification bool
	SampleRate             int32  // 0 = adapter default
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
	FinalText        string
	LockedTier       ProviderTier
	ProviderID       string
	ModelID          string
	LatencyMs        float64
	FellBackToUnary  bool
}

// ErrInsufficientCredits is returned by the Vrooli provider when LPBS reports
// the user is out of audio credits. The chain MUST NOT fall through to Local
// on this error — it is a billing failure, not an availability failure.
var ErrInsufficientCredits = errors.New("audio-tools: insufficient credits for Vrooli tier")

// ErrAllProvidersFailed is returned when every enabled tier was tried and
// none succeeded.
var ErrAllProvidersFailed = errors.New("audio-tools: all providers failed")

// ErrUnknownBYOKProvider is returned when X-Audio-BYOK-Provider names a
// provider not in the registry.
var ErrUnknownBYOKProvider = errors.New("audio-tools: unknown BYOK provider")

// ErrMissingBYOKProvider is returned when X-Audio-BYOK-Key is set but
// X-Audio-BYOK-Provider is absent — silent provider selection is forbidden.
var ErrMissingBYOKProvider = errors.New("audio-tools: BYOK key set without BYOK provider")
