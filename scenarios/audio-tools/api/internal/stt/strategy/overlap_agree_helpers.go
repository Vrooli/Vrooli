package strategy

import (
	"context"
	"fmt"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	voice "audio-tools/internal/stt/pipeline"
)

// processAudioTriggers runs settle attempts for one appended chunk. Stopwatch
// mode uses the configured audio threshold; VAD mode drains clean boundaries
// first, then applies the MaxWindowMs safety settle in the same order as Run.
func processAudioTriggers(
	trigger string,
	pcmBytes, committedAudioBytes, nextTriggerAt int,
	minWindowBytes, maxWindowBytes int,
	scanVADBoundary func() int,
	processIteration func(int),
) {
	switch trigger {
	case TriggerStopwatch:
		if pcmBytes >= nextTriggerAt && pcmBytes-committedAudioBytes >= minWindowBytes {
			processIteration(pcmBytes)
		}
	default: // TriggerVAD
		for {
			boundary := scanVADBoundary()
			if boundary < 0 {
				break
			}
			processIteration(boundary)
		}
		if maxWindowBytes > 0 && pcmBytes-committedAudioBytes > maxWindowBytes {
			processIteration(pcmBytes)
		}
	}
}

// settleAttemptState performs one provider call and validates its result. The
// next trigger is scheduled immediately after the call, before error handling,
// matching the stream's existing retry cadence.
type settleAttemptState struct {
	pcm           []byte
	advanceBytes  int
	nextTriggerAt *int
	transcribe    func([]byte) (*sttchain.Result, error)
	logger        logx.Logger
	events        chan<- sttchain.StreamEvent
}

func (s settleAttemptState) run(audio []byte, audioBytes int) (*sttchain.Result, bool) {
	result, err := s.transcribe(audio)
	*s.nextTriggerAt = len(s.pcm) + s.advanceBytes
	if err != nil {
		// A transient provider failure is surfaced while the growing buffer
		// keeps the audio available for the next settle attempt.
		s.logger.Printf("[stt-overlap] settle error: %v", err)
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		return nil, false
	}
	if result == nil {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: provider returned no result for %d bytes; retaining audio for retry", audioBytes)}
		return nil, false
	}
	if strings.TrimSpace(result.Text) == "" {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: provider returned an empty transcript for %d bytes; retaining audio for retry", audioBytes)}
		return nil, false
	}
	return result, true
}

// flushOverlapTail performs the last bounded transcription for audio that has
// not yet been committed when a stream ends or is cancelled. It returns the
// updated committed text so Run retains ownership of session state.
func flushOverlapTail(
	callCtx context.Context,
	pcm []byte,
	committedAudioBytes int,
	absoluteStartSample int64,
	committed string,
	hasVoiced bool,
	transcribe func(context.Context, []byte) (*sttchain.Result, error),
	events chan<- sttchain.StreamEvent,
) string {
	if len(pcm)-committedAudioBytes <= 0 {
		return committed
	}
	tailBytes := append([]byte(nil), pcm[committedAudioBytes:]...)
	result, err := transcribe(callCtx, tailBytes)
	if err != nil {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: final transcription failed for %d bytes; audio was not committed: %w", len(tailBytes), err)}
		return committed
	}
	if result == nil {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: final transcription returned no result for %d bytes; audio was not committed", len(tailBytes))}
		return committed
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		// Empty output is expected when the remaining tail contains only
		// silence. Preserve an error when voiced audio remains uncommitted.
		if hasVoiced {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
				"overlap-agree: final transcription returned an empty transcript for %d bytes; audio was not committed", len(tailBytes))}
		}
		return committed
	}
	newCommit, tail, _ := appendAfterAdvance(committed, text)
	if len(newCommit) <= len(committed) || tail == "" {
		return committed
	}
	events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
		Text:         tail,
		StartSample:  absoluteStartSample,
		Words:        timedWordsForUniqueSpan(tail, result.Words),
		ProviderTier: result.Tier,
		ProviderID:   result.ProviderID,
		ModelID:      result.ModelID,
		Confidence:   result.Confidence,
	}}
	return newCommit
}

// agreementCommitState carries session state for resolving a local agreement,
// divergence rejection, stall fallback, and word-aligned cursor advance.
type agreementCommitState struct {
	strategy             *OverlapAgree
	pcm                  []byte
	rightEdge            int
	committedAudioBytes  *int
	committed            *string
	recent               *[]hypothesis
	lastAdvanced         *bool
	stallRejects         *int
	forceRetryAfterBytes *int
	lastTier             *sttchain.ProviderTier
	lastProviderID       *string
	lastModelID          *string
	logger               logx.Logger
	events               chan<- sttchain.StreamEvent
	result               *sttchain.Result
	agreed               string
	emitSegment          func(string, *sttchain.Result)
	wordEndBytes         func([]sttchain.TimedWord, int) int
	compactConsumed      func()
}

// handle applies the agreed transcript using the current overlap mode,
// preserving event order and cursor advancement behavior.
func (s agreementCommitState) handle() {
	var newCommit, tail string
	var ok bool
	if *s.lastAdvanced {
		newCommit, tail, ok = appendAfterAdvance(*s.committed, s.agreed)
	} else {
		newCommit, tail, ok = mergeAgreed(*s.committed, s.agreed)
	}
	if s.agreed != "" && !ok {
		*s.stallRejects++
		// Stall fallback stops waiting for an agreement that is not coming
		// after the configured number of consecutive divergence rejects.
		if s.strategy.MaxStallRejects > 0 && *s.stallRejects >= s.strategy.MaxStallRejects {
			s.logger.Printf("[stt-overlap] stall-fallback: %d consecutive divergence-rejects >= max_stall_rejects=%d — force-committing freshest hypothesis tail=%q",
				*s.stallRejects, s.strategy.MaxStallRejects, voice.TruncateForLog(s.result.Text, 80))
			newCommit, tail, _ = appendAfterAdvance(*s.committed, strings.TrimSpace(s.result.Text))
			if len(newCommit) <= len(*s.committed) || tail == "" {
				s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
					"overlap-agree: stall fallback produced no new transcript; retaining audio for retry")}
				return
			}
			s.emitSegment(tail, s.result)
			*s.committed = newCommit
			*s.forceRetryAfterBytes = 0
			*s.lastTier = s.result.Tier
			*s.lastProviderID = s.result.ProviderID
			*s.lastModelID = s.result.ModelID
			// Advance past the force-committed window so the next iteration
			// starts on genuinely new audio.
			if s.rightEdge > *s.committedAudioBytes && s.rightEdge <= len(s.pcm) {
				*s.committedAudioBytes = s.rightEdge
				s.compactConsumed()
			}
			*s.recent = nil
			*s.lastAdvanced = true
			*s.stallRejects = 0
			return
		}
		s.logger.Printf("[stt-overlap] divergence-reject: committed=%q agreed=%q (in-stream wander — no commit, stall=%d/%d)",
			voice.TruncateForLog(*s.committed, 60), voice.TruncateForLog(s.agreed, 60), *s.stallRejects, s.strategy.MaxStallRejects)
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: s.result.Text}}
		return
	}
	if len(newCommit) > len(*s.committed) {
		s.logger.Printf("[stt-overlap] commit: tail=%q committed_now=%q",
			voice.TruncateForLog(tail, 80), voice.TruncateForLog(newCommit, 100))
		s.emitSegment(tail, s.result)
		*s.committed = newCommit
		*s.stallRejects = 0
		*s.forceRetryAfterBytes = 0
		if s.strategy.UseWordTimestampAdvance {
			if advance := s.wordEndBytes(s.result.Words, len(strings.Fields(s.agreed))); advance > 0 {
				newOffset := *s.committedAudioBytes + advance
				if newOffset > len(s.pcm) {
					newOffset = len(s.pcm)
				}
				*s.committedAudioBytes = newOffset
				s.compactConsumed()
				*s.recent = nil
				*s.lastAdvanced = true
			}
		} else {
			*s.lastAdvanced = false
		}
	} else if s.result.Text != "" {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: s.result.Text}}
	}
}

// cleanPostForceBoundary carries the stream state for the optimized first
// clean VAD boundary after a forced commit.
type cleanPostForceBoundary struct {
	strategy             *OverlapAgree
	pcm                  []byte
	rightEdge            int
	committedAudioBytes  *int
	committed            *string
	result               *sttchain.Result
	forceRetryAfterBytes *int
	events               chan<- sttchain.StreamEvent
	emitSegment          func(string, *sttchain.Result)
	compactConsumed      func()
	recent               *[]hypothesis
	lastAdvanced         *bool
	stallRejects         *int
}

// handle commits a clean post-force boundary without waiting for another
// LocalAgreement run. It returns true when this optimization applied, even
// when it emitted an error and retained the audio for retry.
func (s cleanPostForceBoundary) handle() bool {
	if s.strategy.Trigger != TriggerVAD || s.strategy.UseWordTimestampAdvance || !*s.lastAdvanced || len(*s.recent) != 0 {
		return false
	}
	newCommit, tail, _ := appendAfterAdvance(*s.committed, strings.TrimSpace(s.result.Text))
	if len(newCommit) <= len(*s.committed) || tail == "" {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: clean post-force boundary produced no new transcript; retaining audio for retry")}
		return true
	}
	s.emitSegment(tail, s.result)
	*s.committed = newCommit
	*s.forceRetryAfterBytes = 0
	if s.rightEdge > *s.committedAudioBytes && s.rightEdge <= len(s.pcm) {
		*s.committedAudioBytes = s.rightEdge
		s.compactConsumed()
	}
	*s.recent = nil
	*s.lastAdvanced = true
	*s.stallRejects = 0
	return true
}

// forceCommitState carries the mutable session values needed by the
// max-window safety commit. Pointer fields retain the closure's original
// in-place update semantics; callbacks keep transcription, event emission,
// and audio compaction on the same Run-local path.
type forceCommitState struct {
	strategy             *OverlapAgree
	pcm                  *[]byte
	committedAudioBytes  *int
	maxWindowBytes       int
	forceRetryAfterBytes *int
	forceRetryStepBytes  int
	committed            *string
	recent               *[]hypothesis
	lastAdvanced         *bool
	stallRejects         *int
	lastTier             *sttchain.ProviderTier
	lastProviderID       *string
	lastModelID          *string
	logger               logx.Logger
	events               chan<- sttchain.StreamEvent
	sampleBytes          int
	transcribe           func([]byte) (*sttchain.Result, error)
	emitSegment          func(string, *sttchain.Result)
	compactConsumed      func()
}

// run transcribes all uncommitted audio when it exceeds MaxWindowMs. It
// returns true when the safety path handled the iteration, including failed
// attempts that retain the audio for a later retry.
func (s forceCommitState) run() bool {
	uncommittedBytes := len(*s.pcm) - *s.committedAudioBytes
	if s.maxWindowBytes <= 0 || uncommittedBytes <= s.maxWindowBytes {
		return false
	}
	// A failed/empty force attempt retains the audio, but do not immediately
	// retry it for every incoming frame. The retry point advances by one normal
	// settle step, keeping provider work bounded while continuing speech.
	if *s.forceRetryAfterBytes > 0 && uncommittedBytes < *s.forceRetryAfterBytes {
		return false
	}
	uncommittedMs := uncommittedBytes * 1000 / (s.strategy.SampleRate * s.sampleBytes)
	s.logger.Printf("[overlap-agree] force commit: max_window_ms=%d exceeded, transcribing %dms of uncommitted audio (no silence boundary detected)",
		s.strategy.MaxWindowMs, uncommittedMs)
	n := len(*s.pcm) - *s.committedAudioBytes
	audio := make([]byte, n)
	copy(audio, (*s.pcm)[*s.committedAudioBytes:])
	res, err := s.transcribe(audio)
	if err != nil {
		*s.forceRetryAfterBytes = uncommittedBytes + s.forceRetryStepBytes
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: force-commit transcribe failed after max_window_ms=%d; retaining %dms of audio for retry: %w",
			s.strategy.MaxWindowMs, uncommittedMs, err)}
		return true
	}
	if res == nil {
		*s.forceRetryAfterBytes = uncommittedBytes + s.forceRetryStepBytes
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: force-commit returned no result; retaining %dms of audio for retry", uncommittedMs)}
		return true
	}
	newCommit, tail, _ := appendAfterAdvance(*s.committed, strings.TrimSpace(res.Text))
	if len(newCommit) <= len(*s.committed) || tail == "" {
		*s.forceRetryAfterBytes = uncommittedBytes + s.forceRetryStepBytes
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
			"overlap-agree: force-commit produced no new transcript for %dms; retaining audio for retry", uncommittedMs)}
		return true
	}
	s.emitSegment(tail, res)
	*s.committed = newCommit
	*s.lastTier = res.Tier
	*s.lastProviderID = res.ProviderID
	*s.lastModelID = res.ModelID
	// Keep a physical tail so a word omitted at the hard batch edge can be
	// recovered on the next request.
	retainedBytes := s.strategy.SampleRate * forceCommitRetainMs / 1000 * s.sampleBytes
	newCursor := len(*s.pcm) - retainedBytes
	if newCursor < *s.committedAudioBytes {
		newCursor = *s.committedAudioBytes
	}
	*s.committedAudioBytes = newCursor
	s.compactConsumed()
	*s.forceRetryAfterBytes = 0
	*s.recent = nil
	*s.lastAdvanced = true
	*s.stallRejects = 0
	return true
}

type windowThresholds struct {
	advanceBytes        int
	minWindowBytes      int
	maxWindowBytes      int
	forceRetryStepBytes int
}

// calculateWindowThresholds converts the strategy's millisecond settings
// into PCM byte thresholds. Keep the fallback and integer arithmetic here
// aligned with Run's trigger and retry semantics.
func calculateWindowThresholds(sampleRate, advanceMs, windowMs, maxWindowMs int) windowThresholds {
	const sampleBytes = 2
	advanceBytes := sampleRate * advanceMs / 1000 * sampleBytes
	if advanceBytes <= 0 {
		advanceBytes = sampleRate * windowMs / 2000 * sampleBytes
	}
	const minWindowCapMs = 1500
	minWindowBytes := sampleRate * windowMs / 1000 * sampleBytes
	if capBytes := sampleRate * minWindowCapMs / 1000 * sampleBytes; minWindowBytes > capBytes {
		minWindowBytes = capBytes
	}
	maxWindowBytes := sampleRate * maxWindowMs / 1000 * sampleBytes
	forceRetryStepBytes := advanceBytes
	if forceRetryStepBytes <= 0 {
		forceRetryStepBytes = sampleRate * 100 / 1000 * sampleBytes
	}
	if forceRetryStepBytes <= 0 {
		forceRetryStepBytes = 1
	}
	return windowThresholds{
		advanceBytes: advanceBytes, minWindowBytes: minWindowBytes,
		maxWindowBytes: maxWindowBytes, forceRetryStepBytes: forceRetryStepBytes,
	}
}
