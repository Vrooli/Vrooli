// Package stt is the streaming STT orchestration layer.
//
// It hosts the single decision boundary for "which technique produces
// events for this session?" (Selector) and, in subsequent phases, the
// transport-free pipeline that drives it (Segmenter, under
// internal/stt/segmenter). Both transports — browser WS and Connect
// bidi — call into this package; neither knows about the other and
// neither contains its own audio orchestration.
//
// The architecture this package implements is documented in
// scenarios/audio-tools/docs/domains/stt/streaming-pipeline.md.
package stt

import (
	"context"
	"errors"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/stt/strategy"
	"audio-tools/internal/sttengine"
)

// ErrIncompatibleStrategyProvider is returned by Select when the
// requested strategy is refused by the negotiated provider's
// ProviderTraits.Strategies whitelist or by the global compatibility
// matrix.
var ErrIncompatibleStrategyProvider = errors.New("audio-tools/stt: strategy is not compatible with the negotiated provider")

// ErrStreamingDisabled is returned by Select when StreamConfig.Mode is
// "off". The caller is expected to use BufferedFallback instead — the
// Segmenter handles this fan-out so transports do not branch on the
// error.
var ErrStreamingDisabled = errors.New("audio-tools/stt: streaming disabled by operator config")

// ErrNoEligibleProvider is returned by Select when no enabled tier is
// available for the request (e.g. BYOK key missing, Vrooli flag off,
// Local resource down). The selector never falls through silently —
// the caller decides whether to surface this as an error or downgrade
// to BufferedFallback.
var ErrNoEligibleProvider = errors.New("audio-tools/stt: no eligible provider for the session")

// StrategyPreference is the operator-tunable strategy hint
// (stt.strategy_preference). "auto" lets the selector pick per the
// global matrix; the explicit values narrow the search.
type StrategyPreference string

const (
	PreferenceAuto        StrategyPreference = "auto"
	PreferenceVAD         StrategyPreference = "vad"
	PreferenceOverlap     StrategyPreference = "overlap"
	PreferencePassthrough StrategyPreference = "passthrough"
)

// StreamingMode is the operator-tunable streaming master switch
// (stt.streaming_mode). "off" forces BufferedFallback; "auto" lets the
// selector negotiate per the available providers and matrix.
type StreamingMode string

const (
	ModeAuto StreamingMode = "auto"
	ModeOff  StreamingMode = "off"
)

// StreamConfig is the resolved per-session operator control surface.
// The Segmenter loads it once at session start and passes it to the
// selector. Field semantics are documented in
// scenarios/audio-tools/docs/reference/configuration.md.
type StreamConfig struct {
	Mode               StreamingMode
	StrategyPreference StrategyPreference
	// EngineID is the active STT engine selection (sttengine manifest id,
	// e.g. "whisper-local"). Empty resolves to the registry's default engine.
	// Only the Local tier honors it; BYOK/Vrooli tiers stream natively.
	EngineID          string
	VADSilenceMs      int
	OverlapWindowMs   int
	OverlapCommitRuns int
	// Boundary-accuracy levers for VADSegmenter. See pipeline.Config
	// for documentation and acceptable ranges.
	VADPreRollMs          int
	VADTrailingPadMs      int
	VADInitialPromptWords int

	// Post-recognition egress-gate levers. The Segmenter builds the gate
	// from these (Phase 2 derives the stage set from the engine manifest;
	// these remain the operator tunables the manifest-derived stages read).
	//
	// HallucinationFilterEnabled toggles the text-domain phrase filter.
	// VADFilterEnabled toggles faster-whisper's built-in silence filter on
	// the /asr request (stamped onto StreamStart.VADFilter by the Segmenter).
	// NoSpeechThreshold / LogProbThreshold drive the signal-domain stage:
	// a segment drops when no_speech_prob > NoSpeechThreshold AND
	// avg_logprob < LogProbThreshold.
	HallucinationFilterEnabled bool
	VADFilterEnabled           bool
	NoSpeechThreshold          float64
	LogProbThreshold           float64
}

// Defaults returns the operator defaults documented in
// docs/reference/configuration.md. Phase E swaps these for values
// loaded from the settings store.
//
// IMPORTANT: keep these aligned with handlers/stt/mappers.go::defaultStreamCfg
// (the persisted-doc defaults the admin handler ships to the browser via
// GetStreamConfig). When the two diverge, server-side VAD ends up using
// one value while the mic-button ring shows another — see the 2026-05-17
// regression where VADSilenceMs=700 here vs VadSilenceMs=1200 there caused
// the ring to fill to ~58% before the server cut.
func Defaults() StreamConfig {
	return StreamConfig{
		Mode:                       ModeAuto,
		StrategyPreference:         PreferenceAuto,
		VADSilenceMs:               1200,
		OverlapWindowMs:            2000,
		OverlapCommitRuns:          2,
		VADPreRollMs:               300,
		VADTrailingPadMs:           200,
		VADInitialPromptWords:      20,
		HallucinationFilterEnabled: true,
		VADFilterEnabled:           true,
		// faster-whisper defaults: a segment is "no speech" when
		// no_speech_prob > 0.6 and avg_logprob < -1.0.
		NoSpeechThreshold: 0.6,
		LogProbThreshold:  -1.0,
	}
}

// Selection is the result of Selector.Select. It carries both the
// strategy to run and the locked provider so the Segmenter can stamp
// trace fields and so the caller does not need to redo tier
// negotiation.
type Selection struct {
	Strategy strategy.Strategy
	Provider sttchain.Provider
	Tier     sttchain.ProviderTier
	Kind     sttchain.StrategyKind
}

// ProviderEligibility describes one tier the selector may consider.
// The caller (Segmenter) prepares these from the chain's per-request
// state — the selector itself does not reach into the chain.
type ProviderEligibility struct {
	Provider  sttchain.Provider
	Tier      sttchain.ProviderTier
	Available bool
}

// Selector picks one (Strategy, Provider) pair per session. It is the
// single explicit home for "which technique?" — every conditional that
// would otherwise scatter across transports/providers lives here.
//
// Phase A ships a deliberately narrow implementation: it always returns
// BufferedFallback paired with the first available tier. Phase D
// replaces Select's body with the full compatibility matrix from
// docs/domains/stt/streaming-pipeline.md.
type Selector struct {
	// BatchExecutor is the unary chain dependency used to construct
	// BufferedFallback strategies. Required.
	BatchExecutor strategy.BatchExecutor

	// Engine is the audio-format substrate, consulted by the capability
	// gate to decide whether a live PCM decode is possible for the
	// declared input format. May be nil (tests / no-decode-gate); a nil
	// Engine is treated as "decode available" so the gate only downgrades
	// when it can prove ffmpeg is missing.
	Engine *audioformat.Engine

	// Registry is the engine-capability manifest. When set, the Local tier's
	// eligible strategy whitelist is derived from the active engine's manifest
	// entry instead of the provider's hardcoded ProviderTraits.Strategies —
	// the manifest is the single source of truth. nil falls back to provider
	// traits (preserves pre-manifest behavior for tests).
	Registry *sttengine.Registry
}

// NewSelector constructs a Selector with the given batch executor and no
// audio-format engine (the capability gate is permissive).
func NewSelector(exec strategy.BatchExecutor) *Selector {
	return &Selector{BatchExecutor: exec}
}

// NewSelectorWith constructs a Selector with a batch executor and the
// audio-format engine used by the streaming capability gate.
func NewSelectorWith(exec strategy.BatchExecutor, engine *audioformat.Engine) *Selector {
	return &Selector{BatchExecutor: exec, Engine: engine}
}

// NewSelectorWithRegistry constructs a Selector wired to the engine-capability
// manifest, so the Local tier's eligible strategies are manifest-derived.
func NewSelectorWithRegistry(exec strategy.BatchExecutor, engine *audioformat.Engine, reg *sttengine.Registry) *Selector {
	return &Selector{BatchExecutor: exec, Engine: engine, Registry: reg}
}

// requiresPCMDecode reports whether the strategy consumes canonical PCM
// frames (VADSegment and OverlapAgree run int16 RMS + byte slicing) and
// therefore needs the inbound audio decoded to PCM before it runs.
// BufferedFallback (whole-file → Whisper) and Passthrough (native provider
// decodes) do not.
func requiresPCMDecode(kind sttchain.StrategyKind) bool {
	return kind == sttchain.StrategyVADSegment || kind == sttchain.StrategyOverlapAgree
}

// canDecodeStream reports whether a live PCM decoder can be built for the
// declared input format: declared PCM takes the ffmpeg-free fast-path;
// any other (or undeclared) format needs ffmpeg. A nil Engine is treated
// as capable so the gate only downgrades on a proven-missing ffmpeg.
func (s *Selector) canDecodeStream(inputFormat string) bool {
	if audioformat.CodecFromString(inputFormat).IsCanonicalPCM() {
		return true
	}
	if s.Engine == nil {
		return true
	}
	return s.Engine.HasFfmpeg()
}

// Select chooses the (Strategy, Provider) pair for the session.
// providers is the precedence-ordered list (BYOK -> Vrooli -> Local)
// the caller prepared from the chain.
func (s *Selector) Select(
	ctx context.Context,
	cfg StreamConfig,
	start sttchain.StreamStart,
	providers []ProviderEligibility,
) (Selection, error) {
	if s == nil || s.BatchExecutor == nil {
		return Selection{}, fmt.Errorf("audio-tools/stt: Selector requires a BatchExecutor")
	}

	// streaming_mode=off short-circuits to BufferedFallback regardless
	// of provider availability. The transports never branch on this —
	// they just forward the chosen strategy's events. Operators see
	// "off" as one Segment + one Done per session with
	// FellBackToUnary=true.
	if cfg.Mode == ModeOff {
		chosen, _ := firstAvailable(providers)
		return Selection{
			Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor},
			Provider: chosen.Provider,
			Tier:     chosen.Tier,
			Kind:     sttchain.StrategyBuffered,
		}, nil
	}

	chosen, ok := firstAvailable(providers)
	if !ok {
		// No eligible tier at all: the buffered fallback still runs and
		// reports the error from Execute via its event stream. The
		// selector surfaces the typed error so the Segmenter can decide
		// whether to fast-fail or proceed; both transports today
		// proceed-and-emit-error for backward parity with the legacy
		// path.
		return Selection{
			Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor},
			Kind:     sttchain.StrategyBuffered,
		}, ErrNoEligibleProvider
	}

	traits := chosen.Provider.Traits()
	pref := cfg.StrategyPreference
	if pref == "" {
		pref = PreferenceAuto
	}

	// Eligible-strategy authority: for the Local tier with a wired manifest,
	// the active engine's strategies[] is the whitelist (manifest is SSOT);
	// every other case falls back to the provider's declared traits. This is
	// the only place engine selection touches strategy eligibility — no engine
	// id appears in branch logic anywhere else.
	supports := s.eligibleStrategy(cfg, chosen, traits)

	// Compatibility matrix (docs/domains/stt/streaming-pipeline.md):
	//   - Stream=true, Strategies=[passthrough] → Passthrough only
	//   - Stream=false, vad_segment in whitelist (or empty) → VADSegment
	//   - Stream=false, overlap_agree in whitelist → OverlapAgree
	//     (operator must request it via preference=overlap)
	//   - Anything else → BufferedFallback
	pick := func(kind sttchain.StrategyKind) (strategy.Strategy, sttchain.StrategyKind, error) {
		if !supports(kind) {
			return nil, "", ErrIncompatibleStrategyProvider
		}
		switch kind {
		case sttchain.StrategyPassthrough:
			if !traits.Stream {
				return nil, "", ErrIncompatibleStrategyProvider
			}
			return &strategy.Passthrough{Provider: chosen.Provider}, kind, nil
		case sttchain.StrategyVADSegment:
			if !traits.Batch {
				return nil, "", ErrIncompatibleStrategyProvider
			}
			return &strategy.VADSegmenter{
				Provider:           chosen.Provider,
				SilenceMs:          cfg.VADSilenceMs,
				PreRollMs:          cfg.VADPreRollMs,
				TrailingPadMs:      cfg.VADTrailingPadMs,
				InitialPromptWords: cfg.VADInitialPromptWords,
			}, kind, nil
		case sttchain.StrategyOverlapAgree:
			if !traits.Batch {
				return nil, "", ErrIncompatibleStrategyProvider
			}
			return &strategy.OverlapAgree{
				Provider:   chosen.Provider,
				WindowMs:   cfg.OverlapWindowMs,
				CommitRuns: cfg.OverlapCommitRuns,
			}, kind, nil
		}
		return nil, "", ErrIncompatibleStrategyProvider
	}

	// Explicit preferences honor the lever directly; auto applies the
	// global default per provider trait shape.
	var chosenStrategy strategy.Strategy
	var chosenKind sttchain.StrategyKind
	switch pref {
	case PreferencePassthrough:
		st, k, err := pick(sttchain.StrategyPassthrough)
		if err != nil {
			return Selection{Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor}, Kind: sttchain.StrategyBuffered}, err
		}
		chosenStrategy, chosenKind = st, k
	case PreferenceVAD:
		st, k, err := pick(sttchain.StrategyVADSegment)
		if err != nil {
			return Selection{Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor}, Kind: sttchain.StrategyBuffered}, err
		}
		chosenStrategy, chosenKind = st, k
	case PreferenceOverlap:
		st, k, err := pick(sttchain.StrategyOverlapAgree)
		if err != nil {
			return Selection{Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor}, Kind: sttchain.StrategyBuffered}, err
		}
		chosenStrategy, chosenKind = st, k
	default: // auto
		switch {
		case traits.Stream && supports(sttchain.StrategyPassthrough):
			st, k, _ := pick(sttchain.StrategyPassthrough)
			chosenStrategy, chosenKind = st, k
		case traits.Batch && supports(sttchain.StrategyVADSegment):
			st, k, _ := pick(sttchain.StrategyVADSegment)
			chosenStrategy, chosenKind = st, k
		default:
			chosenStrategy = &strategy.BufferedFallback{Executor: s.BatchExecutor}
			chosenKind = sttchain.StrategyBuffered
		}
	}

	// Capability gate: a PCM-consuming strategy can only run if the
	// Segmenter can produce live PCM for the declared input format. When it
	// can't (non-PCM input + no ffmpeg), downgrade to BufferedFallback,
	// which hands the whole reassembled file to Whisper's own decoder.
	if requiresPCMDecode(chosenKind) && !s.canDecodeStream(start.InputFormat) {
		return Selection{
			Strategy: &strategy.BufferedFallback{Executor: s.BatchExecutor},
			Provider: chosen.Provider,
			Tier:     chosen.Tier,
			Kind:     sttchain.StrategyBuffered,
		}, nil
	}

	return Selection{
		Strategy: chosenStrategy,
		Provider: chosen.Provider,
		Tier:     chosen.Tier,
		Kind:     chosenKind,
	}, nil
}

// eligibleStrategy returns a predicate reporting whether a strategy is allowed
// for this session. The Local tier with a wired manifest uses the active
// engine's strategies[] whitelist; everything else (no registry, unknown
// engine id, or a non-Local tier) falls back to the provider's declared
// ProviderTraits — preserving the "empty whitelist means any" semantics.
func (s *Selector) eligibleStrategy(cfg StreamConfig, chosen ProviderEligibility, traits sttchain.ProviderTraits) func(sttchain.StrategyKind) bool {
	if s.Registry != nil && chosen.Tier == sttchain.TierLocal {
		engineID := cfg.EngineID
		if engineID == "" {
			engineID = s.Registry.DefaultEngineID()
		}
		if strs := s.Registry.EligibleStrategies(engineID); len(strs) > 0 {
			set := make(map[sttchain.StrategyKind]bool, len(strs))
			for _, str := range strs {
				set[sttchain.StrategyKind(str)] = true
			}
			return func(k sttchain.StrategyKind) bool { return set[k] }
		}
	}
	return traits.Supports
}

func firstAvailable(providers []ProviderEligibility) (ProviderEligibility, bool) {
	for _, p := range providers {
		if p.Available && p.Provider != nil {
			return p, true
		}
	}
	return ProviderEligibility{}, false
}
