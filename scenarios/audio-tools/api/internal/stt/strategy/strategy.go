package strategy

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// Strategy is the per-session pipeline that turns AudioChunks into
// StreamEvents. The Segmenter constructs one Strategy per session via
// the StrategySelector, calls Run once, and closes the events channel
// after Run returns.
//
// Strategies are stateless across sessions: the Segmenter passes all
// per-session state through Run's arguments. Constructors may capture
// long-lived dependencies (e.g. the BatchExecutor for BufferedFallback)
// but must not capture session-scoped state.
//
// seam: Strategy is the streaming-strategy seam (SEAMS.md row
// "stt.StreamingStrategy"). Production wires VAD / overlap-agree /
// passthrough implementations; tests substitute fakes.
type Strategy interface {
	// Kind identifies which selector cell produced this strategy. The
	// selector uses this to enforce the compatibility matrix and the
	// per-provider whitelist; consumers can read it for telemetry.
	Kind() sttchain.StrategyKind

	// Run drives the strategy until either chunks closes (graceful end
	// of session) or ctx is cancelled. Run is responsible for emitting
	// all events for the session — including a terminal StreamEventDone.
	// Run does NOT close events; the Segmenter owns the channel.
	Run(
		ctx context.Context,
		start sttchain.StreamStart,
		chunks <-chan sttchain.AudioChunk,
		events chan<- sttchain.StreamEvent,
		cursor *sttchain.ConsumptionCursor,
	) error
}
