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

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt"
)

// Deps is the long-lived dependency bundle held by a Segmenter
// factory. Transports build one Deps at startup and reuse it across
// sessions.
type Deps struct {
	Chain    *sttchain.Chain
	Selector *stt.Selector
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

	return selection.Strategy.Run(ctx, start, chunks, events)
}

func emitTerminal(events chan<- sttchain.StreamEvent, err error) {
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FellBackToUnary: true}}
}
