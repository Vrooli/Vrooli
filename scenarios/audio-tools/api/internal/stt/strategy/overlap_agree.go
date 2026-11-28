package strategy

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"

	"github.com/vrooli/api-core/schedule"
)

// OverlapAgree is the LocalAgreement-N streaming strategy (Macháček et
// al. 2023) for batch-only providers.
//
// It runs over a single GROWING audio buffer: on each iteration it
// re-transcribes pcm[committedAudioBytes:] (the same audio prefix as
// before, plus newly arrived chunks). Because consecutive hypotheses
// share a real audio prefix, their text shares a real text prefix —
// LocalAgreement can extract it. When CommitRuns consecutive hypotheses
// agree on a prefix, that prefix commits as a Segment and the audio
// cursor (committedAudioBytes) advances to the END timestamp of the last
// committed word, so the next iteration's audio starts immediately after
// the committed material and committed words can never be re-emitted.
//
// This replaces a prior sliding-window implementation whose windows
// covered DIFFERENT audio offsets and therefore could never share a
// prefix to agree on; that scheme only ever committed at end-of-stream.
// See docs/internal/PROBLEMS.md "OverlapAgree commit gap" for the
// original failure mode.
//
// Tuning levers:
//
//   - WindowMs: also doubles as the lower bound on the first call's
//     audio (capped at 1.5s so the first hypothesis arrives in real
//     time). Matches StreamConfig.OverlapWindowMs (default 2000).
//   - CommitRuns: how many consecutive iterations must agree on a
//     prefix before it commits. Matches StreamConfig.OverlapCommitRuns
//     (default 2).
//   - AdvanceMs: minimum new audio between iterations. Default
//     WindowMs/2.
//   - MaxWindowMs: cap on the uncommitted audio buffer. If no commit
//     happens in this much audio, force-advance the cursor so per-call
//     latency stays bounded (default 10 000ms, tuned for long-form safety).
//
// TriggerVAD makes the strategy run a settle attempt on silence
// boundaries detected from frame RMS analysis (production default).
// Whisper sees clean audio edges, transcripts are more stable, and
// LocalAgreement agreement happens at a much higher rate than the
// stopwatch alternative.
const TriggerVAD = "vad"

// TriggerStopwatch makes the strategy run a settle attempt every
// AdvanceMs of accumulated audio, regardless of voice state. Used as
// a legacy fallback and by scripted-text behaviour tests where the
// fixture audio is silent zeros that the VAD trigger would never
// classify as voiced.
const TriggerStopwatch = "stopwatch"

// Post-cursor hypotheses begin after the committed audio. Only a small
// prompt-regurgitation overlap is therefore legitimate; a long unbounded
// search can delete a genuinely repeated phrase from a long dictation.
const maxPromptOverlapWords = 8

// forceCommitRetainMs is the physical tail kept after a forced batch
// transcription. Whisper can omit words at a hard right edge, so claiming
// coverage through the exact end would make that omission unrecoverable. This
// mirrors the VAD segmenter's pre-roll and stays below the bounded prompt
// overlap used when merging the next result.
const forceCommitRetainMs = 300

type OverlapAgree struct {
	Provider sttchain.Provider

	WindowMs    int
	CommitRuns  int
	SampleRate  int
	AdvanceMs   int
	MaxWindowMs int
	// MaxAgreedTokens caps the per-iteration agreement walk so the
	// comparison cost and variance accumulation stay bounded even when
	// the uncommitted buffer holds many seconds of speech. 0 means
	// unbounded; default applied by applyDefaults is 30 tokens.
	MaxAgreedTokens int

	// MaxStallRejects is the stall-fallback commit policy. When
	// LocalAgreement keeps producing hypotheses that DIVERGE from the
	// committed prefix (the model wandered on hard audio / jittery word
	// timestamps), the divergence-reject path commits nothing and the
	// uncommitted tail grows toward the MaxWindowMs net — every settle
	// attempt then re-transcribes an ever-larger window, saturating the
	// Whisper semaphore and making finalization SLOWER than a single
	// batch call. After this many CONSECUTIVE divergence-rejects, the
	// strategy force-commits the freshest hypothesis tail and advances
	// the cursor, bounding tail growth / re-transcription cost well
	// before the 10s net. The counter resets on any forward commit.
	//
	// 0 disables the fallback (only the MaxWindowMs net applies — the
	// pre-fallback behavior). This field carries no applyDefaults
	// default precisely so that a directly-constructed OverlapAgree
	// preserves legacy behavior and so 0 ("disabled") survives end to
	// end; the quality-first operator default (0) is applied at the config layer
	// (selector.Defaults / streamCfgDoc). Acceptable operator range
	// [1,10].
	MaxStallRejects int

	// UseWordTimestampAdvance enables the latency optimization that moves the
	// audio cursor to Whisper's last committed word timestamp. It is disabled
	// in production because batch word timestamps are not durable coverage
	// acknowledgements: an over-optimistic timestamp can skip speech. The
	// growing-window merge plus MaxWindowMs fallback is the safe default.
	UseWordTimestampAdvance bool

	// Trigger selects the settle-attempt source: TriggerVAD (default)
	// or TriggerStopwatch. See the constants above.
	Trigger string

	// VAD-trigger fields. Defaults applied by applyDefaults match
	// VADSegmenter's so a silence that closes a VADSegment also
	// closes an OverlapAgree settle attempt on the same audio.
	SilenceMs  int     // sustained silence window that triggers a settle (default 1200)
	SilenceRMS float64 // amplitude threshold below which a frame is silent (default 250)
	FrameMs    int     // frame size for RMS evaluation (default 20)

	Clock schedule.Clock
	// Logger records strategy diagnostics through the scenario logging seam.
	// Nil uses logx.Std, preserving direct construction in focused tests.
	Logger logx.Logger
}

// Kind reports the strategy kind for selector enforcement.

// hypothesis is one iteration's transcription, kept in `recent` so the
// next iteration can agree against it. words is empty when the backend
// doesn't supply word timestamps; the cursor advance then degrades to
// "leave committedAudioBytes alone" and mergeAgreed prevents re-emission.
type hypothesis struct {
	text  string
	words []sttchain.TimedWord
}

// Run consumes chunks, transcribes the growing buffer, emits Partial +
// Segment events under LocalAgreement-N, and a terminal Done.
func (o *OverlapAgree) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
	cursor *sttchain.ConsumptionCursor,
) error {
	if o.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: OverlapAgree requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	// Resolve defaults on a private copy: a strategy can be shared safely by
	// concurrent sessions without mutating its caller-owned configuration.
	resolved := *o
	resolved.applyDefaults()
	logger := resolved.Logger
	if logger == nil {
		logger = logx.Std{}
	}
	session := newOverlapAgreeSession(ctx, &resolved, start, chunks, events, cursor, logger)
	return session.run()
}
