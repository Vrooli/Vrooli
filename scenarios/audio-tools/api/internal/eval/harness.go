//go:build audio_lab

package eval

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// Clip is one corpus item the harness replays: canonical s16le mono PCM
// plus the operator-corrected reference transcript.
type Clip struct {
	ID         string
	PCM        []byte
	SampleRate int
	Reference  string
	// Format is the audioformat codec hint for the PCM bytes (e.g.
	// "pcm_s16le"). Used by the batch oracle session to tell the backend
	// the codec; empty defaults to canonical PCM downstream.
	Format           string
	TimedReference   []TimedWord
	TimingProvenance *TimingReferenceProvenance
}

// TimingReferenceProvenance is retained in reports alongside any score. A
// score is emitted only for human-reviewed references with complete review
// provenance; aligner-only annotations remain explicitly unscored.
type TimingReferenceProvenance struct {
	Status, SourceReference, AlignmentTool, AlignmentModel string
	Reviewer, ReviewMethod, ReviewedAt                     string
}

func (p *TimingReferenceProvenance) ReviewReady() bool {
	if p == nil || p.Status != "human_reviewed" || strings.TrimSpace(p.SourceReference) == "" ||
		(strings.TrimSpace(p.AlignmentTool) == "") != (strings.TrimSpace(p.AlignmentModel) == "") || strings.TrimSpace(p.Reviewer) == "" ||
		strings.TrimSpace(p.ReviewMethod) == "" || strings.TrimSpace(p.ReviewedAt) == "" {
		return false
	}
	for _, timestamp := range strings.Split(p.ReviewedAt, ";") {
		if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
			return false
		}
	}
	return true
}

// bytesPerSecond is the PCM byte rate for this clip (s16le mono).
func (c Clip) bytesPerSecond() int { return c.SampleRate * 2 }

// Duration is the clip's audio duration derived from PCM byte length.
func (c Clip) Duration() time.Duration {
	bps := c.bytesPerSecond()
	if bps <= 0 {
		return 0
	}
	return time.Duration(float64(len(c.PCM)) / float64(bps) * float64(time.Second))
}

// RunMode selects replay pacing.
type RunMode int

const (
	// ModeDeterministic feeds every chunk back-to-back with no pacing. The
	// transcript and compute meters are reproducible run-to-run (they do
	// not depend on wall-clock timing); finalization latency is NOT
	// meaningful in this mode.
	ModeDeterministic RunMode = iota
	// ModeRealtime releases chunks at 1× the audio rate (one chunk's worth
	// of audio-duration of sleep between sends), so the measured
	// last-chunk → Done latency reflects real finalization wall-time and
	// the queue effects of the 5-wide Whisper semaphore.
	ModeRealtime
)

// Session consumes the replayed chunk channel and emits StreamEvents,
// CLOSING events on return. Segmenter.Run matches this contract directly
// (bind start/cfg via closure); a bare Strategy.Run is adapted via
// StrategySession (Strategy.Run does not close events).
type Session func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error

// StreamResult is one replay's captured output.
type StreamResult struct {
	FinalText string
	// HypothesisWordTimestamps are provider-reported and remain relative to
	// the individual provider request. They are descriptive only; they are not
	// aligned or promoted to reference timings by the replay harness.
	HypothesisWordTimestamps          []sttchain.TimedWord
	HypothesisWordTimestampsTruncated bool
	Segments                          []string
	Partials                          []string
	SegmentCount                      int
	CommitTimeline                    []CommitState
	SpeakerRejectionCount             int
	// FinalizationLatency is the wall-clock gap between feeding the last
	// audio chunk and the terminal Done. Only meaningful in ModeRealtime.
	FinalizationLatency time.Duration
	Err                 error
}

// ReplayOptions tunes one Replay call.
type ReplayOptions struct {
	Mode RunMode
	// ChunkMs is the audio-duration of each replayed chunk (default 100ms).
	ChunkMs int
	// LatencyTailSeconds switches ModeRealtime into a bounded approximation:
	// the prefix is fed without wall-clock pacing and only the final tail
	// window is paced at 1x. This measures final-tail behavior for long clips
	// without sleeping through the full recording; prefix backlog effects are
	// intentionally not modeled.
	LatencyTailSeconds int
	// Sleep paces ModeRealtime; defaults to time.Sleep. Injectable so unit
	// tests can drive paced replay without real sleeping.
	Sleep func(time.Duration)
	// Now stamps event/feed times; defaults to time.Now (clock.System).
	Now func() time.Time
	// DroppedSpanThresholdWords controls the safety gate for contiguous
	// deleted reference words. 0 uses DefaultDroppedSpanThresholdWords.
	DroppedSpanThresholdWords int
}

func (o ReplayOptions) chunkMs() int {
	if o.ChunkMs <= 0 {
		return 100
	}
	return o.ChunkMs
}

func (o ReplayOptions) sleep(d time.Duration) {
	if o.Sleep != nil {
		o.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (o ReplayOptions) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// Replay feeds clip.PCM through session as a chunk stream and collects the
// emitted events, returning the assembled transcript, the partial/segment
// timeline, and (in ModeRealtime) the finalization latency. The harness
// owns chunk pacing and event collection; session owns transcription.
func Replay(ctx context.Context, clip Clip, opts ReplayOptions, session Session) StreamResult {
	chunkBytes := clip.bytesPerSecond() * opts.chunkMs() / 1000
	if chunkBytes <= 0 {
		chunkBytes = len(clip.PCM)
	}
	tailStartByte := latencyTailStartByte(clip, opts.LatencyTailSeconds)

	chunks := make(chan sttchain.AudioChunk, 4)
	events := make(chan sttchain.StreamEvent, 64)

	// The feeder and event reader run concurrently. Keep the end-of-feed
	// timestamp on a channel rather than sharing a mutable time.Time; otherwise
	// a provider that emits Done immediately after chunks close can race the
	// assignment and produce a false zero-latency sample.
	feedDone := make(chan time.Time, 1)
	go func() {
		defer func() { feedDone <- opts.now() }()
		defer close(chunks)
		for off := 0; off < len(clip.PCM); off += chunkBytes {
			end := off + chunkBytes
			if end > len(clip.PCM) {
				end = len(clip.PCM)
			}
			cp := make([]byte, end-off)
			copy(cp, clip.PCM[off:end])
			select {
			case <-ctx.Done():
				return
			case chunks <- sttchain.AudioChunk{Audio: cp}:
			}
			if opts.Mode == ModeRealtime && shouldPaceChunk(end, tailStartByte) {
				opts.sleep(time.Duration(float64(end-off) / float64(clip.bytesPerSecond()) * float64(time.Second)))
			}
		}
	}()

	sessionDone := make(chan error, 1)
	go func() { sessionDone <- session(ctx, chunks, events) }()

	var res StreamResult
	startedAt := opts.now()
	var doneAt time.Time
	for {
		var ev sttchain.StreamEvent
		var ok bool
		select {
		case ev, ok = <-events:
			if !ok {
				goto eventsDrained
			}
		case <-ctx.Done():
			// A server-owned experiment deadline must release the worker even if
			// a provider violates the stream contract and never closes events.
			// The session still receives the same context and is responsible for
			// its own asynchronous teardown.
			res.Err = ctx.Err()
			return res
		}
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			if ev.Segment != nil {
				res.Segments = append(res.Segments, ev.Segment.Text)
				appendHypothesisWordTimestamps(&res, ev.Segment.Words, ev.Segment.StartMs, ev.Segment.StartSample, clip.SampleRate)
				res.SegmentCount++
				res.CommitTimeline = append(res.CommitTimeline, CommitState{
					Text:       strings.TrimSpace(strings.Join(res.Segments, " ")),
					AtMs:       millisSince(startedAt, opts.now()),
					AudioEndMs: ev.Segment.EndMs,
				})
			}
		case sttchain.StreamEventPartial:
			if ev.Partial != nil {
				res.Partials = append(res.Partials, ev.Partial.Text)
			}
		case sttchain.StreamEventSpeakerRejection:
			res.SpeakerRejectionCount++
		case sttchain.StreamEventError:
			if ev.Error != nil {
				res.Err = ev.Error
			}
		case sttchain.StreamEventDone:
			doneAt = opts.now()
			if ev.Done != nil {
				res.FinalText = ev.Done.FinalText
				if shouldAppendFinalCommit(res.CommitTimeline, res.FinalText) {
					res.CommitTimeline = append(res.CommitTimeline, CommitState{
						Text: strings.TrimSpace(res.FinalText),
						AtMs: millisSince(startedAt, doneAt),
					})
				}
			}
		}
	}

eventsDrained:
	if err := <-sessionDone; err != nil && res.Err == nil {
		res.Err = err
	}
	lastFedAt := <-feedDone
	if !doneAt.IsZero() && !lastFedAt.IsZero() && doneAt.After(lastFedAt) {
		res.FinalizationLatency = doneAt.Sub(lastFedAt)
	}
	return res
}

// Keep report payloads bounded even if a provider returns malformed or
// unexpectedly large timing arrays. 20,000 words is well above a one-hour
// transcript while limiting each per-clip JSON addition to a predictable size.
const maxHypothesisWordTimestamps = 20_000

func appendHypothesisWordTimestamps(result *StreamResult, words []sttchain.TimedWord, segmentStartMs, segmentStartSample int64, sampleRate int) {
	remaining := maxHypothesisWordTimestamps - len(result.HypothesisWordTimestamps)
	if remaining <= 0 {
		if len(words) > 0 {
			result.HypothesisWordTimestampsTruncated = true
		}
		return
	}
	if len(words) > remaining {
		words = words[:remaining]
		result.HypothesisWordTimestampsTruncated = true
	}
	offsetSeconds := float64(segmentStartMs) / 1000
	if segmentStartMs == 0 && segmentStartSample > 0 && sampleRate > 0 {
		offsetSeconds = float64(segmentStartSample) / float64(sampleRate)
	}
	for _, word := range words {
		word.Start += offsetSeconds
		word.End += offsetSeconds
		result.HypothesisWordTimestamps = append(result.HypothesisWordTimestamps, word)
	}
}

// EvalClip replays one clip through one strategy session and computes the
// per-clip quality metrics. meter must be the MeteredProvider wrapping the
// session's provider so the Whisper-call / audio-second / RTF columns can
// be read after the run. Reference WER uses the v1 normalization policy.
func EvalClip(ctx context.Context, clip Clip, meter *MeteredProvider, opts ReplayOptions, session Session) ClipResult {
	if meter != nil {
		meter.Reset()
	}
	res := Replay(ctx, clip, opts, session)

	norm := DefaultNormalizeOptions()
	normalizedReference := Normalize(clip.Reference, norm)
	normalizedHypothesis := Normalize(res.FinalText, norm)
	refTokens := Tokenize(clip.Reference, norm)
	hypTokens := Tokenize(res.FinalText, norm)
	_, editOperations := AlignOperations(refTokens, hypTokens)
	wer := WER(refTokens, hypTokens)

	cr := ClipResult{
		ClipID:                            clip.ID,
		Reference:                         clip.Reference,
		Hypothesis:                        res.FinalText,
		HypothesisWordTimestamps:          append([]sttchain.TimedWord(nil), res.HypothesisWordTimestamps...),
		HypothesisWordTimestampsTruncated: res.HypothesisWordTimestampsTruncated,
		WER:                               wer,
		NormalizedReference:               normalizedReference,
		NormalizedHypothesis:              normalizedHypothesis,
		EditOperations:                    editOperations,
		SegmentCount:                      res.SegmentCount,
		PartialRevisions:                  PartialRevisions(res.Partials),
		CommitTimeline:                    res.CommitTimeline,
		CommitCount:                       len(res.CommitTimeline),
		TimeToFirstCommitMs:               firstCommitMs(res.CommitTimeline),
		SpeakerRejectionCount:             res.SpeakerRejectionCount,
		AudioDurationMs:                   int64(clip.Duration() / time.Millisecond),
		Err:                               res.Err,
	}
	cr.Safety = EvaluateSafety(editOperations, res.CommitTimeline, SafetyOptions{
		DroppedSpanThresholdWords: opts.DroppedSpanThresholdWords,
	})
	cr.TimestampReference = clip.TimingProvenance
	switch {
	case len(clip.TimedReference) == 0:
		cr.TimestampDriftSkipReason = "timed_reference_absent"
	case !clip.TimingProvenance.ReviewReady():
		cr.TimestampDriftSkipReason = "timed_reference_not_human_reviewed"
	case !timedReferenceMatchesTranscript(clip.TimedReference, clip.Reference):
		cr.TimestampDriftSkipReason = "timed_reference_transcript_mismatch"
	case !referenceWordsWithinDuration(clip.TimedReference, clip.Duration()):
		cr.TimestampDriftSkipReason = "timed_reference_out_of_bounds"
	case len(res.HypothesisWordTimestamps) == 0:
		cr.TimestampDriftSkipReason = "hypothesis_word_timestamps_absent"
	case res.HypothesisWordTimestampsTruncated:
		cr.TimestampDriftSkipReason = "hypothesis_word_timestamps_truncated"
	case !timedWordsWithinDuration(res.HypothesisWordTimestamps, clip.Duration()):
		cr.TimestampDriftSkipReason = "hypothesis_word_timestamps_out_of_bounds"
	default:
		reference := make([]TimedWord, len(clip.TimedReference))
		for i, word := range clip.TimedReference {
			reference[i] = TimedWord{Word: word.Word, Start: word.Start, End: word.End}
		}
		hypothesis := make([]TimedWord, len(res.HypothesisWordTimestamps))
		for i, word := range res.HypothesisWordTimestamps {
			hypothesis[i] = TimedWord{Word: word.Word, Start: word.Start, End: word.End}
		}
		drift, err := ScoreTimestampDrift(reference, hypothesis)
		if err != nil {
			if drift.ReferenceWords > 0 && drift.MatchedWords == 0 {
				cr.TimestampDriftSkipReason = "timestamp_no_lexical_matches"
			} else {
				cr.TimestampDriftSkipReason = "timestamp_evidence_invalid"
			}
		} else {
			cr.TimestampDrift = &drift
		}
	}
	if meter != nil {
		snap := meter.Snapshot()
		cr.WhisperCalls = snap.Calls
		cr.WhisperAudioSeconds = snap.AudioSeconds
		cr.ProviderLatencyMs = float64(snap.ProviderLatency) / float64(time.Millisecond)
		cr.RTF = RTF(snap.ProviderLatency, clip.Duration())
	}
	if opts.Mode == ModeRealtime {
		cr.LatencySamplesMs = []float64{float64(res.FinalizationLatency) / float64(time.Millisecond)}
	}
	return cr
}

func timedWordsWithinDuration(words []sttchain.TimedWord, duration time.Duration) bool {
	if duration <= 0 {
		return false
	}
	limit := duration.Seconds()
	for _, word := range words {
		if word.Start < 0 || word.End < word.Start || word.End > limit {
			return false
		}
	}
	return true
}

// timedReferenceMatchesTranscript protects the evaluator boundary as well as
// corpus ingestion: reports can also be built from in-memory clips and older
// persisted rows, so a human_reviewed provenance claim alone is not enough.
func timedReferenceMatchesTranscript(words []TimedWord, transcript string) bool {
	if len(words) == 0 {
		return false
	}
	options := DefaultNormalizeOptions()
	transcriptWords := Tokenize(transcript, options)
	if len(words) != len(transcriptWords) {
		return false
	}
	for i, word := range words {
		if Normalize(word.Word, options) != transcriptWords[i] {
			return false
		}
	}
	return true
}

func referenceWordsWithinDuration(words []TimedWord, duration time.Duration) bool {
	if duration <= 0 {
		return false
	}
	// Match corpus.TimedReference's import tolerance for codec/rounding tails.
	limit := duration.Seconds() + 0.05
	for _, word := range words {
		if word.Start < 0 || word.End < word.Start || word.End > limit {
			return false
		}
	}
	return true
}

// StrategySpec describes one strategy row in the report. BuildSession is
// invoked once per replay (strategies are single-use): it returns a fresh
// Session bound to a fresh MeteredProvider so per-clip compute is isolated.
type StrategySpec struct {
	Kind  sttchain.StrategyKind
	Label string
	// Cell metadata identifies a provider-neutral experiment row. It stays in
	// the internal report so persistence can retain the exact evidence lane
	// rather than inferring it from a display label.
	EngineID      string
	ModelID       string
	PolicyProfile string
	ReplayLane    string
	FaultProfile  string
	CellID        string
	BuildSession  func(clip Clip) (Session, *MeteredProvider)
}

// EvalOptions tunes a full RunReport over a corpus.
type EvalOptions struct {
	ChunkMs int
	// QualityPass runs the deterministic WER+compute measurement (default
	// true via DefaultEvalOptions).
	QualityPass bool
	// RealtimeRepeats is how many real-time-paced runs per clip feed the
	// latency distribution. 0 skips latency measurement (the fast default
	// suite path).
	RealtimeRepeats int
	// RealtimeConcurrency bounds concurrent real-time replay sessions.
	// Real-time eval mostly waits on wall-clock audio pacing, so serializing
	// every clip/repeat/strategy makes UI-triggered latency runs impractical.
	// Default is 8, low enough to avoid stampeding the local STT backend.
	RealtimeConcurrency       int
	LatencyTailSeconds        int
	Sleep                     func(time.Duration)
	Now                       func() time.Time
	DroppedSpanThresholdWords int
	Progress                  func(EvalProgress)
	ProgressScope             string
}

// EvalProgress reports one completed eval work unit. It is intentionally
// callback-only so synchronous eval callers keep the same report contract.
type EvalProgress struct {
	Scope         string
	Phase         string
	Strategy      string
	StrategyIndex int
	StrategyTotal int
	ClipID        string
	ClipIndex     int
	ClipTotal     int
	RepeatIndex   int
	RepeatTotal   int
}

// DefaultEvalOptions is the deterministic-only configuration used by the
// default (no-Whisper-needed) test suite and CLI quality runs.
func DefaultEvalOptions() EvalOptions {
	return EvalOptions{ChunkMs: 100, QualityPass: true, RealtimeRepeats: 0}
}

func (o EvalOptions) realtimeConcurrency() int {
	if o.RealtimeConcurrency > 0 {
		return o.RealtimeConcurrency
	}
	return 8
}

// RunReport replays every clip through every strategy spec and assembles the
// comparison report. Batch-oriented strategies use the deterministic pass for
// reproducible WER/compute columns. Native passthrough providers are real-time
// decoders, so their quality pass is paced at 1x; burst-feeding them is not a
// valid operating condition and can produce model-state hallucinations.
func RunReport(ctx context.Context, clips []Clip, specs []StrategySpec, opts EvalOptions) EvalReport {
	report := EvalReport{
		QualityMeasured: opts.QualityPass,
		LatencyMeasured: opts.RealtimeRepeats > 0,
	}
	tailLatencyApproximation := opts.RealtimeRepeats > 0 && opts.LatencyTailSeconds > 0
	if tailLatencyApproximation {
		report.Warnings = append(report.Warnings, ReportWarning{
			Code:     "tail_latency_approximation",
			Severity: "info",
			Message:  fmt.Sprintf("Latency repeats paced only the final %d second(s) of each clip; long-form latency reflects final-tail behavior, not full real-time playback.", opts.LatencyTailSeconds),
		})
	}
	for strategyIndex, spec := range specs {
		clipResults := make([]ClipResult, len(clips))
		for i, clip := range clips {
			var cr ClipResult
			if opts.QualityPass {
				session, meter := spec.BuildSession(clip)
				qualityMode := ModeDeterministic
				if spec.Kind == sttchain.StrategyPassthrough {
					qualityMode = ModeRealtime
				}
				cr = EvalClip(ctx, clip, meter, ReplayOptions{
					Mode: qualityMode, ChunkMs: opts.ChunkMs, Sleep: opts.Sleep, Now: opts.Now, DroppedSpanThresholdWords: opts.DroppedSpanThresholdWords,
				}, session)
				// Native passthrough quality must be paced, but that quality pass
				// is not one of the requested latency repeats. Keep its transcript,
				// compute and safety evidence while reporting latency only from the
				// explicitly requested repeat set.
				cr.LatencySamplesMs = nil
				opts.emitProgress(EvalProgress{
					Scope:         opts.ProgressScope,
					Phase:         "quality",
					Strategy:      string(spec.Kind),
					StrategyIndex: strategyIndex + 1,
					StrategyTotal: len(specs),
					ClipID:        clip.ID,
					ClipIndex:     i + 1,
					ClipTotal:     len(clips),
				})
			} else {
				cr = ClipResult{ClipID: clip.ID, Reference: clip.Reference, AudioDurationMs: int64(clip.Duration() / time.Millisecond), TimestampReference: clip.TimingProvenance}
				cr.TimestampDriftSkipReason = timingReferenceSkipReason(clip)
				if cr.TimestampDriftSkipReason == "" {
					cr.TimestampDriftSkipReason = "quality_pass_not_run"
				}
			}
			clipResults[i] = cr
		}
		if opts.RealtimeRepeats > 0 {
			runRealtimeRepeats(ctx, clips, spec, strategyIndex, len(specs), opts, clipResults)
		}
		strategyReport := aggregateStrategy(spec.Kind, spec.Label, clipResults)
		strategyReport.EngineID = spec.EngineID
		strategyReport.ModelID = spec.ModelID
		strategyReport.PolicyProfile = spec.PolicyProfile
		strategyReport.ReplayLane = spec.ReplayLane
		strategyReport.FaultProfile = spec.FaultProfile
		if spec.CellID != "" {
			strategyReport.BaseStrategy = spec.Kind
			strategyReport.Strategy = sttchain.StrategyKind(spec.CellID)
		}
		if tailLatencyApproximation {
			strategyReport.Scaling.Warnings = append(strategyReport.Scaling.Warnings, ReportWarning{
				Code:     "tail_latency_approximation",
				Severity: "info",
				Message:  "Scaling latency points use final-tail pacing; validate full long-form latency before making promotion decisions.",
			})
		}
		report.PerStrategy = append(report.PerStrategy, strategyReport)
	}
	return explainReport(report)
}

func timingReferenceSkipReason(clip Clip) string {
	if len(clip.TimedReference) == 0 {
		return "timed_reference_absent"
	}
	if !clip.TimingProvenance.ReviewReady() {
		return "timed_reference_not_human_reviewed"
	}
	return ""
}

// CombineReports merges independently executed cells into one comparison.
// Cells must be run independently when their replay lanes have different
// timing semantics; re-explaining only after the merge keeps winner deltas and
// warnings consistent with a single report.
func CombineReports(reports ...EvalReport) EvalReport {
	var combined EvalReport
	for _, report := range reports {
		combined.QualityMeasured = combined.QualityMeasured || report.QualityMeasured
		combined.LatencyMeasured = combined.LatencyMeasured || report.LatencyMeasured
		combined.PerStrategy = append(combined.PerStrategy, report.PerStrategy...)
		combined.Warnings = append(combined.Warnings, report.Warnings...)
	}
	return explainReport(combined)
}

func (o EvalOptions) emitProgress(progress EvalProgress) {
	if o.Progress == nil {
		return
	}
	o.Progress(progress)
}

type realtimeResult struct {
	clipIndex int
	repeat    int
	result    ClipResult
}

func runRealtimeRepeats(ctx context.Context, clips []Clip, spec StrategySpec, strategyIndex, strategyTotal int, opts EvalOptions, clipResults []ClipResult) {
	total := len(clips) * opts.RealtimeRepeats
	if total == 0 {
		return
	}

	sem := make(chan struct{}, opts.realtimeConcurrency())
	results := make(chan realtimeResult, total)
	var wg sync.WaitGroup
	for clipIndex, clip := range clips {
		for repeat := 0; repeat < opts.RealtimeRepeats; repeat++ {
			wg.Add(1)
			go func(clipIndex int, repeat int, clip Clip) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					results <- realtimeResult{clipIndex: clipIndex, repeat: repeat, result: ClipResult{
						ClipID: clip.ID, Reference: clip.Reference, Err: ctx.Err(),
					}}
					return
				}
				session, meter := spec.BuildSession(clip)
				rt := EvalClip(ctx, clip, meter, ReplayOptions{
					Mode: ModeRealtime, ChunkMs: opts.ChunkMs, LatencyTailSeconds: opts.LatencyTailSeconds, Sleep: opts.Sleep, Now: opts.Now, DroppedSpanThresholdWords: opts.DroppedSpanThresholdWords,
				}, session)
				opts.emitProgress(EvalProgress{
					Scope:         opts.ProgressScope,
					Phase:         "latency",
					Strategy:      string(spec.Kind),
					StrategyIndex: strategyIndex + 1,
					StrategyTotal: strategyTotal,
					ClipID:        clip.ID,
					ClipIndex:     clipIndex + 1,
					ClipTotal:     len(clips),
					RepeatIndex:   repeat + 1,
					RepeatTotal:   opts.RealtimeRepeats,
				})
				results <- realtimeResult{clipIndex: clipIndex, repeat: repeat, result: rt}
			}(clipIndex, repeat, clip)
		}
	}
	wg.Wait()
	close(results)

	for rr := range results {
		cr := clipResults[rr.clipIndex]
		cr.LatencySamplesMs = append(cr.LatencySamplesMs, rr.result.LatencySamplesMs...)
		if !opts.QualityPass && rr.repeat == 0 {
			rr.result.LatencySamplesMs = cr.LatencySamplesMs
			cr = rr.result
		}
		if cr.Err == nil {
			cr.Err = rr.result.Err
		}
		clipResults[rr.clipIndex] = cr
	}
}

func latencyTailStartByte(clip Clip, tailSeconds int) int {
	if tailSeconds <= 0 {
		return 0
	}
	bps := clip.bytesPerSecond()
	if bps <= 0 {
		return 0
	}
	tailBytes := bps * tailSeconds
	if tailBytes >= len(clip.PCM) {
		return 0
	}
	return len(clip.PCM) - tailBytes
}

func shouldPaceChunk(endByte, tailStartByte int) bool {
	return tailStartByte <= 0 || endByte > tailStartByte
}

func millisSince(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return int64(end.Sub(start) / time.Millisecond)
}

func shouldAppendFinalCommit(timeline []CommitState, finalText string) bool {
	finalText = strings.TrimSpace(finalText)
	if finalText == "" {
		return len(timeline) > 0 && strings.TrimSpace(timeline[len(timeline)-1].Text) != ""
	}
	return len(timeline) == 0 || strings.TrimSpace(timeline[len(timeline)-1].Text) != finalText
}

func firstCommitMs(timeline []CommitState) float64 {
	if len(timeline) == 0 {
		return 0
	}
	return float64(timeline[0].AtMs)
}

// StrategySession adapts a bare strategy.Strategy-style Run (which does NOT
// close events) into a Session (which must). The start header is bound by
// the caller. The runFn is `strat.Run` with start pre-applied.
func StrategySession(runFn func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error) Session {
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		err := runFn(ctx, chunks, events)
		close(events)
		return err
	}
}
