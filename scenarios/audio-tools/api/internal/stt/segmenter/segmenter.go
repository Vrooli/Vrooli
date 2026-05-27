// Package segmenter is the transport-free streaming STT orchestrator.
//
// One Segmenter is constructed per session by the transport adapter
// (Connect bidi handler, browser WS handler). It owns:
//
//   - session lifecycle (start, cancel, terminal Done)
//   - candidate-provider enumeration (via sttchain.Chain.StreamCandidates)
//   - strategy negotiation (via stt.Selector)
//   - chunks-in / events-out channel ownership (closes events on return)
//
// The Segmenter knows nothing about the wire shape on either side: the
// transport translates inbound frames into AudioChunks and outbound
// StreamEvents into wire messages.
package segmenter

import (
	"context"
	"fmt"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/stt"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	voice "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/sttengine"
)

// Deps is the long-lived dependency bundle held by a Segmenter
// factory. Transports build one Deps at startup and reuse it across
// sessions.
type Deps struct {
	Chain    *sttchain.Chain
	Selector *stt.Selector
	// Engine is the audio-format substrate. The Segmenter routes inbound
	// chunks through it to guarantee PCM-consuming strategies receive
	// canonical PCM. May be nil — a default Engine is built lazily.
	Engine *audioformat.Engine

	// Registry is the engine-capability manifest. When set, the egress gate's
	// stage set is derived from the active engine's manifest entry. nil falls
	// back to a direct cfg-driven gate (preserves pre-manifest test behavior).
	Registry *sttengine.Registry

	// SpeakerIsolation is the per-session audio-domain identity check, built by
	// the transport from the live SpeakerConfig + resource client (nil when
	// speaker isolation is disabled/off). The egress gate's audio-domain stage
	// uses it; it never changes mid-session.
	SpeakerIsolation egress.SpeakerIsolation

	// SpeakerExtraction is the per-session PRE-recognition target-speaker
	// extractor, built by the transport from the live SpeakerConfig + resource
	// client (nil when extraction is disabled/off). The ingress pipeline runs it
	// to isolate the enrolled speaker's audio before the VAD/recognizer see it —
	// engine-agnostic, unlike the egress SpeakerIsolation gate. Never changes
	// mid-session.
	SpeakerExtraction ingress.TargetExtractor
}

// Segmenter is the per-session orchestrator. Construct via New, call
// Run exactly once, then discard.
type Segmenter struct {
	deps Deps
}

// New constructs a Segmenter bound to the supplied dependencies.
func New(deps Deps) *Segmenter {
	return &Segmenter{deps: deps}
}

// Run drives the session: negotiates a strategy via the selector,
// invokes Strategy.Run, and closes events when the strategy returns
// (success or error). The strategy is responsible for emitting a
// terminal StreamEventDone; Run does not synthesize one.
//
// Run respects ctx cancellation: when ctx fires before the strategy
// returns, the strategy sees it via the same ctx and is expected to
// drain and exit promptly.
func (s *Segmenter) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	cfg stt.StreamConfig,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	defer close(events)

	if s == nil || s.deps.Chain == nil || s.deps.Selector == nil {
		err := fmt.Errorf("audio-tools/stt/segmenter: Segmenter requires a Chain and Selector")
		emitTerminal(events, err)
		return err
	}

	// Resolve + stamp the active engine id onto the start so the chain can
	// pick the right Local-tier provider (Whisper vs Kyutai). Both are
	// Local-tier engines distinguished only by this id; resolving it here
	// (the single per-session orchestration point) keeps engine selection out
	// of the transports.
	engineID := cfg.EngineID
	if engineID == "" && s.deps.Registry != nil {
		engineID = s.deps.Registry.DefaultEngineID()
	}
	start.EngineID = engineID

	candidates := s.deps.Chain.StreamCandidates(ctx, start)
	eligibility := make([]stt.ProviderEligibility, 0, len(candidates))
	for _, p := range candidates {
		eligibility = append(eligibility, stt.ProviderEligibility{
			Provider:  p,
			Tier:      p.Type(),
			Available: true,
		})
	}

	selection, err := s.deps.Selector.Select(ctx, cfg, start, eligibility)
	if err != nil && selection.Strategy == nil {
		// Hard selection failure with no fallback strategy. Surface as
		// an error + Done so consumers see a consistent shape.
		emitTerminal(events, err)
		return err
	}
	// The selector may return a non-nil error alongside a fallback
	// strategy (e.g. ErrNoEligibleProvider returns BufferedFallback).
	// In that case we proceed with the strategy and let its event
	// stream carry the underlying problem.

	// PCM-consuming strategies (VADSegment, OverlapAgree) require canonical
	// PCM. Route the inbound chunks through the audioformat substrate here
	// — the single per-session injection point both transports inherit, so
	// the WS and Connect paths cannot drift. BufferedFallback and
	// Passthrough see the raw bytes (the former hands the whole file to
	// Whisper, the latter's native provider decodes for itself).
	pcmChunks := chunks
	if s.requiresPCM(selection.Kind, engineID) {
		normalized, nstart, cleanup, nerr := s.normalizeChunks(ctx, start, chunks, events)
		if nerr != nil {
			return nerr // error + Done already emitted
		}
		defer cleanup()
		pcmChunks = normalized
		start = nstart
	}

	// Pre-recognition ingress enhancement (denoise). Scoped to the Whisper
	// PCM-buffering strategies (VADSegment/OverlapAgree) — the streaming
	// Passthrough engines are validated separately — and gated on config +
	// ffmpeg availability inside buildIngress. Wraps the canonical-PCM stream
	// so both the server-side VAD and Whisper see the cleaned audio.
	if requiresPCMDecode(selection.Kind) {
		if pipeline := s.buildIngress(cfg); pipeline != nil {
			enhanced, cleanup, ierr := pipeline.Process(ctx, pcmChunks)
			if ierr != nil {
				emitTerminal(events, ierr)
				return ierr
			}
			defer cleanup()
			pcmChunks = enhanced
		}
	}

	// Stamp the session VAD-filter lever onto the start so every batch call
	// the strategy makes enables faster-whisper's silence filter.
	start.VADFilter = cfg.VADFilterEnabled

	// Post-recognition egress gate: the single seam every SegmentEvent
	// passes through before the wire. The strategy writes to an internal
	// channel; the interceptor runs each segment through the gate and
	// forwards the surviving events. Strategies never see the gate.
	gate := s.buildGate(cfg)
	inner := make(chan sttchain.StreamEvent)
	forwardDone := make(chan struct{})
	go runEgress(ctx, gate, inner, events, forwardDone)

	err = selection.Strategy.Run(ctx, start, pcmChunks, inner)
	close(inner)
	<-forwardDone
	return err
}

// buildGate constructs the per-session egress gate. When a manifest registry
// is wired, the stage SET is derived from the active engine's capabilities
// (sttengine.EgressStages) — the manifest-driven path. Without a registry it
// falls back to a direct cfg-driven gate so registry-less tests keep working;
// the stages and tunables are identical either way.
func (s *Segmenter) buildGate(cfg stt.StreamConfig) *egress.Gate {
	params := sttengine.EgressParams{
		HallucinationFilterEnabled: cfg.HallucinationFilterEnabled,
		NoSpeechThreshold:          cfg.NoSpeechThreshold,
		LogProbThreshold:           cfg.LogProbThreshold,
		IsHallucination:            voice.IsWhisperHallucination,
		SpeakerIsolation:           s.deps.SpeakerIsolation,
	}
	if s.deps.Registry != nil {
		engineID := cfg.EngineID
		if engineID == "" {
			engineID = s.deps.Registry.DefaultEngineID()
		}
		return egress.NewGate(s.deps.Registry.EgressStages(engineID, params)...)
	}
	// Registry-less fallback: hallucination stage (if enabled) + the
	// signal-domain stage (no-ops without confidence signals) + the
	// audio-domain speaker stage when an isolation is wired.
	var stages []egress.Stage
	if params.HallucinationFilterEnabled {
		stages = append(stages, egress.HallucinationStage{IsHallucination: params.IsHallucination})
	}
	stages = append(stages, egress.ConfidenceStage{
		NoSpeechThreshold: params.NoSpeechThreshold,
		LogProbThreshold:  params.LogProbThreshold,
	})
	if params.SpeakerIsolation != nil {
		stages = append(stages, egress.SpeakerStage{Isolation: params.SpeakerIsolation})
	}
	return egress.NewGate(stages...)
}

// buildIngress constructs the per-session pre-recognition enhancement pipeline,
// composing (in order) denoise then target-speaker extraction. Both stages are
// engine-agnostic (they operate on canonical PCM upstream of recognition), so
// they are gated here on capability + config rather than via a per-engine
// manifest flag. Returns nil when no stage applies (the Pipeline is then never
// wired — zero overhead).
//
//   - Denoise: enabled in config AND ffmpeg available. "config on but no ffmpeg"
//     degrades to a no-op (mirrors the selector's BufferedFallback downgrade).
//   - Extraction: a SpeakerExtraction extractor was built for the session (the
//     transport builds it only when extraction is enabled + a profile is bound).
//
// Ordering: denoise first (clean steady background noise), then extraction
// (isolate the target speaker from any remaining co-occurring voices).
func (s *Segmenter) buildIngress(cfg stt.StreamConfig) *ingress.Pipeline {
	var enhancers []ingress.Enhancer

	if cfg.DenoiseEnabled {
		eng := s.deps.Engine
		if eng == nil {
			eng = audioformat.New()
		}
		if eng.HasFfmpeg() {
			enhancers = append(enhancers, ingress.DenoiseEnhancer{Runner: eng})
		}
	}

	if s.deps.SpeakerExtraction != nil {
		enhancers = append(enhancers, ingress.ExtractionEnhancer{Extractor: s.deps.SpeakerExtraction})
	}

	if len(enhancers) == 0 {
		return nil
	}
	return ingress.NewPipeline(enhancers...)
}

// runEgress reads events from the strategy's inner channel, runs each
// SegmentEvent through the gate, and forwards survivors to out. Dropped
// segments are suppressed; rejected segments become a speaker-rejection
// event. When any segment was suppressed, the terminal Done's FinalText is
// rebuilt from the surviving segment texts so the final transcript matches
// what the consumer actually saw. It closes done when in is drained so the
// Segmenter can sequence the events-channel close after the strategy returns.
func runEgress(ctx context.Context, gate *egress.Gate, in <-chan sttchain.StreamEvent, out chan<- sttchain.StreamEvent, done chan<- struct{}) {
	defer close(done)
	var finalParts []string
	gatedAny := false
	for ev := range in {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			seg := ev.Segment
			if seg == nil {
				out <- ev
				continue
			}
			dec := gate.Apply(ctx, egress.SegmentDecision{
				Text:       seg.Text,
				Language:   seg.DetectedLanguage,
				Confidence: seg.Confidence,
				Audio:      seg.Audio,
			})
			switch dec.Outcome {
			case egress.Drop:
				gatedAny = true
			case egress.Reject:
				gatedAny = true
				out <- sttchain.StreamEvent{
					Kind: sttchain.StreamEventSpeakerRejection,
					SpeakerRejection: &sttchain.SpeakerRejectionEvent{
						Reason:       dec.Reason,
						FallbackUsed: dec.FallbackUsed,
					},
				}
			default: // Emit
				// Strip gate-only fields so they never reach the wire.
				seg.Text = dec.Text
				seg.Confidence = nil
				seg.Audio = nil
				finalParts = append(finalParts, seg.Text)
				out <- ev
			}
		case sttchain.StreamEventDone:
			if gatedAny && ev.Done != nil {
				ev.Done.FinalText = strings.Join(finalParts, " ")
			}
			out <- ev
		default:
			out <- ev
		}
	}
}

// requiresPCMDecode mirrors stt.Selector's gate: only VADSegment and
// OverlapAgree consume canonical PCM frames client-side.
func requiresPCMDecode(kind sttchain.StrategyKind) bool {
	return kind == sttchain.StrategyVADSegment || kind == sttchain.StrategyOverlapAgree
}

// requiresPCM decides whether inbound chunks must be normalized to canonical
// PCM before the strategy runs. VADSegment/OverlapAgree always need it (they
// slice PCM client-side). Passthrough needs it ONLY when the active engine
// declares requires.pcm16kMono in the manifest (Kyutai does; a Passthrough
// BYOK vendor that decodes for itself does not). BufferedFallback never needs
// it (Whisper decodes the whole reassembled file). Manifest-driven — no engine
// id branching here.
func (s *Segmenter) requiresPCM(kind sttchain.StrategyKind, engineID string) bool {
	if requiresPCMDecode(kind) {
		return true
	}
	if kind == sttchain.StrategyPassthrough && s.deps.Registry != nil {
		return s.deps.Registry.RequiresPCM(engineID)
	}
	return false
}

func emitTerminal(events chan<- sttchain.StreamEvent, err error) {
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FellBackToUnary: true}}
}
