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

// Confidence carries the per-segment confidence signals a batch backend
// reports (faster-whisper's no_speech_prob / avg_logprob). It is nil on
// Result/SegmentEvent when the backend provides none (a non-Whisper tier,
// or a manifest engine whose provides.confidenceSignals is empty), in which
// case the signal-domain egress stage is skipped.
type Confidence struct {
	NoSpeechProb float64
	AvgLogProb   float64
}

// Request is the per-call input to the chain.
type Request struct {
	Audio                   []byte
	Format                  string
	Language                string
	SkipSpeakerVerification bool
	InitialPrompt           string
	// VADFilter requests the backend's built-in voice-activity filter
	// (faster-whisper vad_filter). The selector/segmenter derives it from
	// the session StreamConfig; batch strategies copy it from StreamStart.
	VADFilter bool

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
	// Confidence carries the backend's per-segment confidence signals when
	// available (Local/Whisper). nil for tiers that report none.
	Confidence *Confidence
	// Words carries the backend's per-word timing (start/end seconds
	// relative to the request audio) when available. Empty for tiers that
	// don't report words. OverlapAgree uses this to advance its committed
	// audio cursor to a real word boundary so committed text is never
	// re-emitted on the next transcription.
	Words []TimedWord
}

// TimedWord is a per-word timestamp pair (seconds, relative to the request
// audio's start). Mirrored from pipeline.TimedWord; lives here so chain
// consumers don't import the pipeline package.
type TimedWord struct {
	Word  string
	Start float64
	End   float64
	Prob  float64
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
	// BackendAcknowledgements means the adapter emits
	// StreamEventAcknowledgement only after its backend confirms processed
	// coverage. Passthrough must not also acknowledge those chunks locally.
	BackendAcknowledgements bool
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
//
// seam: Provider is the STT chain-provider seam (SEAMS.md row
// "sttchain.Provider"). Production wires Local/BYOK/Vrooli tiers; tests
// wire fakes from internal/ai/sttchain/mocks.
type Provider interface {
	Type() ProviderTier
	IsAvailable(ctx context.Context) bool

	// Transcribe converts one audio buffer to text.
	//
	// Implementations must be safe for concurrent use on a single instance.
	// VADSegment issues bounded preview transcriptions on its own goroutine
	// while a segment-boundary transcription may be in flight, so a provider
	// that keeps mutable per-call state on itself will corrupt one of them.
	// The shipped providers satisfy this by holding only immutable
	// collaborators and building request state on the stack.
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
	// rejected (provider not eligible). Callers that set RequireStreaming
	// receive that failure instead of an implicit whole-turn fallback.
	// Returning a nil channel + nil error is reserved for adapters whose
	// Traits().Stream is false.
	TranscribeStreaming(ctx context.Context, start StreamStart, chunks <-chan AudioChunk) (<-chan StreamEvent, error)
}

// StreamStart is the metadata header that opens a streaming session.
// Carried out-of-band so the provider can negotiate format/language
// before any audio arrives.
type StreamStart struct {
	// ProtocolVersion is the negotiated transport contract. Version 2 adds
	// replay-safe chunk and acknowledgement identities.
	ProtocolVersion         int32
	SessionID               string
	Generation              uint64
	ResumeToken             string
	Language                string
	InitialPrompt           string
	SkipSpeakerVerification bool
	SampleRate              int32 // 0 = adapter default
	// InputFormat declares the codec of the inbound AudioChunk bytes
	// (audioformat vocabulary: "webm", "opus", "pcm_s16le", ...). Empty
	// means undeclared — the Segmenter sniffs the first chunk. The
	// Segmenter rewrites this to "pcm_s16le" before handing the start to a
	// PCM-consuming strategy, so a strategy's Request.Format always matches
	// the bytes it actually holds.
	InputFormat string
	// InputSampleRate is a hint about the inbound bytes' sample rate; it
	// never changes the fixed internal target (16 kHz). 0 = unknown.
	InputSampleRate int32
	// VADFilter requests the backend's built-in voice-activity filter for
	// every batch call in the session. The Segmenter stamps it from the
	// resolved StreamConfig before handing the start to a batch strategy.
	VADFilter bool
	// EngineID is the active STT engine selection (sttengine manifest id,
	// e.g. "whisper-local", "kyutai", or "sherpa-streaming"). The Segmenter stamps it from
	// resolved StreamConfig before enumerating candidates so the chain can
	// resolve which Local-tier provider serves the session — local engines are
	// distinguished only by this id.
	// Empty resolves to the chain's default local provider (Whisper).
	EngineID string
	// Per-request creds (same shape as Request) so streaming sessions
	// can carry BYOK/LPBS credentials without piggy-backing on Request.
	BYOKProvider string
	BYOKKey      string
	LPBSToken    string
	UserIdentity string
	// RequireStreaming makes stream negotiation fail closed. It is set by
	// production streaming entry points so a missing or failed native
	// streaming provider cannot silently accumulate an arbitrarily long
	// turn for unary transcription. The intentional streaming_mode=off
	// path leaves this false and selects BufferedFallback explicitly.
	RequireStreaming bool
}

// AudioChunk is one chunk of audio bytes in the stream input channel.
//
// The codec is whatever StreamStart.InputFormat declared on ingress. By
// the time a strategy consumes chunks, the Segmenter has routed them
// through the audioformat substrate, so PCM-consuming strategies
// (VADSegment, OverlapAgree) are guaranteed canonical 16-bit LE PCM,
// mono, 16 kHz. The raw, pre-normalization bytes only ever reach
// BufferedFallback (which hands the whole reassembled file to Whisper)
// and Passthrough (whose native provider decodes for itself).
type AudioChunk struct {
	Audio       []byte
	Sequence    uint64
	StartSample int64
	EndSample   int64
	Digest      []byte
}

// ConsumptionCursor records the highest contiguous audio range a strategy has
// handed to its recognizer. It deliberately tracks coverage rather than
// committed text: a streaming recognizer may consume many seconds of fluent
// speech before it commits a durable segment.
//
// The cursor emits at most one acknowledgement per interval of audio and is
// flushed by the segmenter when the strategy returns. The transport owns the
// callback, so strategies remain independent of WebSocket/Connect details.
type ConsumptionCursor struct {
	emit                     func(StreamEvent)
	minAdvanceSamples        int64
	receivedSequence         int64
	processedSequence        int64
	receivedEndSample        int64
	processedEndSample       int64
	lastAcknowledgedSequence int64
	lastAckEndSample         int64
	hasCoverage              bool
}

func NewConsumptionCursor(emit func(StreamEvent), minAdvanceSamples int64) *ConsumptionCursor {
	if minAdvanceSamples <= 0 {
		minAdvanceSamples = 1600 // approximately 100 ms at the canonical 16 kHz rate
	}
	return &ConsumptionCursor{
		emit: emit, minAdvanceSamples: minAdvanceSamples,
		receivedSequence: -1, processedSequence: -1, lastAcknowledgedSequence: -1,
	}
}

// Observe advances coverage for one contiguous input chunk. Repeated or
// out-of-order observations are ignored so a reconnect/replay cannot create
// duplicate acknowledgements or move the processed cursor backwards.
func (c *ConsumptionCursor) Observe(chunk AudioChunk) {
	if c == nil || c.emit == nil || chunk.EndSample < chunk.StartSample {
		return
	}
	if c.hasCoverage && chunk.Sequence <= uint64(c.processedSequence) {
		return
	}
	c.receivedSequence = int64(chunk.Sequence)
	c.receivedEndSample = chunk.EndSample
	if !c.hasCoverage || chunk.Sequence == uint64(c.processedSequence+1) {
		c.processedSequence = int64(chunk.Sequence)
		c.processedEndSample = chunk.EndSample
		c.hasCoverage = true
		if c.processedSequence == 0 || c.processedEndSample-c.lastAckEndSample >= c.minAdvanceSamples {
			c.emitCurrent()
		}
	}
}

// Flush emits the latest coverage even when the remaining tail is shorter
// than the normal acknowledgement interval.
func (c *ConsumptionCursor) Flush() {
	if c == nil || !c.hasCoverage || c.processedSequence <= c.lastAcknowledgedSequence {
		return
	}
	c.emitCurrent()
}

func (c *ConsumptionCursor) emitCurrent() {
	if c == nil || !c.hasCoverage || c.processedSequence <= c.lastAcknowledgedSequence {
		return
	}
	c.emit(StreamEvent{
		Kind: StreamEventAcknowledgement,
		Acknowledgement: &AcknowledgementEvent{
			ReceivedSequence: c.receivedSequence, ProcessedSequence: c.processedSequence,
			ReceivedEndSample: c.receivedEndSample, ProcessedEndSample: c.processedEndSample,
		},
	})
	c.lastAcknowledgedSequence = c.processedSequence
	c.lastAckEndSample = c.processedEndSample
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
	StreamEventAcknowledgement  StreamEventKind = "acknowledgement"
	StreamEventSessionStatus    StreamEventKind = "session_status"
	// StreamEventVadState carries periodic snapshots of the server-side
	// VAD silence clock. Consumers (UI mic-button ring) render the
	// silence-elapsed value with light client-side interpolation so the
	// visible auto-stop progress lines up exactly with the moment the
	// strategy actually cuts a segment. Throttled by the emitting
	// strategy (~20 Hz in silence, ~2 Hz in voiced).
	StreamEventVadState StreamEventKind = "vad_state"
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
	VadState         *VadStateEvent
	Acknowledgement  *AcknowledgementEvent
	SessionStatus    *SessionStatusEvent
}

type DeliveryClass string

const (
	DeliveryProgress DeliveryClass = "progress"
	DeliveryDurable  DeliveryClass = "durable"
)

// DeliveryClass is the explicit bounded-queue registry. High-rate partial,
// VAD, and transient status updates are coalescible; commit, acknowledgement,
// policy, error, and terminal events are ordered and durable.
func (e StreamEvent) DeliveryClass() DeliveryClass {
	switch e.Kind {
	case StreamEventPartial, StreamEventVadState:
		return DeliveryProgress
	default:
		return DeliveryDurable
	}
}

func (e StreamEvent) Durable() bool { return e.DeliveryClass() == DeliveryDurable }

// IsDroppable is the inverse of Durable: true only for Partial events, the one
// class the pipeline may coalesce or drop to stay backpressure-safe.
func (e StreamEvent) IsDroppable() bool { return !e.Durable() }

type PartialEvent struct {
	Text string
}

type SegmentEvent struct {
	Text             string
	SegmentID        string
	Generation       uint64
	StartMs          int64
	EndMs            int64
	StartSample      int64
	EndSample        int64
	AlignmentQuality string
	DetectedLanguage string
	// Per-segment trace; the chain may stamp these from the locked tier
	// if the adapter doesn't fill them in.
	ProviderTier ProviderTier
	ProviderID   string
	ModelID      string
	LatencyMs    float64

	// Confidence and Audio are the post-recognition egress-gate inputs. The
	// strategy stamps them when producing the segment; the Segmenter's gate
	// reads them (signal-domain stage reads Confidence, audio-domain speaker
	// stage reads Audio) and never forwards them on the wire — the transport
	// mappers ignore both. Confidence is nil when the backend reports none;
	// Audio is the canonical-PCM segment bytes, nil for non-PCM strategies.
	Confidence *Confidence
	Audio      []byte
}

type WakeWordEvent struct {
	Score    float64
	SampleID string
}

type SpeakerRejectionEvent struct {
	Reason       string
	FallbackUsed bool
	// Score is the best cosine-similarity the active profiles produced for the
	// rejected segment (0 when verification could not run); Threshold is the
	// configured match cutoff. The browser transport forwards both so the
	// rejection banner shows the real numbers rather than 0.00/0.00.
	Score     float64
	Threshold float64
}

// VadStateEvent is a periodic snapshot of the server-side VAD silence
// clock. Emitted by VADSegmenter on state transitions and at a throttled
// cadence during sustained silence (~20 Hz) / sustained voiced (~2 Hz)
// so the UI mic-button ring can render server-derived progress and avoid
// the client/server silence-threshold drift documented in
// /home/matthalloran8/.vrooli/plans/server-driven-mic-ring-streamvadstate-event.md.
//
// SilenceElapsedMs is server-relative (no clock-sync concern). The
// client stamps receivedAt on arrival and interpolates from there.
// SilenceTimeoutMs echoes the active StreamConfig.VADSilenceMs so the
// client doesn't have to combine this with its own config store.
// TickSeq is a per-stream monotonic counter the client uses to drop
// out-of-order frames on transports that don't guarantee ordering.
type VadStateEvent struct {
	Voiced           bool
	SilenceElapsedMs int64
	SilenceTimeoutMs int64
	TickSeq          uint64
	// SilenceTimedOut is true only on the threshold-crossing tick where
	// SilenceElapsedMs first reaches SilenceTimeoutMs (the same frame the
	// segment is cut). It is the self-describing one-shot auto-stop signal;
	// after the cut no further ticks are emitted until voice resumes, so
	// clients must latch on it rather than re-deriving it from a float
	// comparison inside a staleness window. See StreamVadState in stt.proto.
	SilenceTimedOut bool
}

type DoneEvent struct {
	FinalText          string
	LockedTier         ProviderTier
	ProviderID         string
	ModelID            string
	LatencyMs          float64
	FellBackToUnary    bool
	ProcessedSequence  int64
	ProcessedEndSample int64
	TerminalReason     string
}

type AcknowledgementEvent struct {
	ReceivedSequence   int64
	ProcessedSequence  int64
	ReceivedEndSample  int64
	ProcessedEndSample int64
}

type SessionStatusEvent struct {
	SessionID         string
	Generation        uint64
	State             string
	QueuePosition     int32
	CapabilityOutcome string
	RecoveryGuidance  string
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

// ErrStreamingUnavailable is returned when a caller explicitly requires a
// native streaming provider but none can accept the session.
var ErrStreamingUnavailable = errors.New("audio-tools: native streaming unavailable")
