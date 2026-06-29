package eval

import (
	"context"
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
	Format string
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
	FinalText    string
	Segments     []string
	Partials     []string
	SegmentCount int
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
	// Sleep paces ModeRealtime; defaults to time.Sleep. Injectable so unit
	// tests can drive paced replay without real sleeping.
	Sleep func(time.Duration)
	// Now stamps event/feed times; defaults to time.Now (clock.System).
	Now func() time.Time
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

	chunks := make(chan sttchain.AudioChunk, 4)
	events := make(chan sttchain.StreamEvent, 64)

	var lastFedAt time.Time
	go func() {
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
				lastFedAt = opts.now()
				return
			case chunks <- sttchain.AudioChunk{Audio: cp}:
			}
			if opts.Mode == ModeRealtime {
				opts.sleep(time.Duration(float64(end-off) / float64(clip.bytesPerSecond()) * float64(time.Second)))
			}
		}
		lastFedAt = opts.now()
	}()

	sessionDone := make(chan error, 1)
	go func() { sessionDone <- session(ctx, chunks, events) }()

	var res StreamResult
	var doneAt time.Time
	for ev := range events {
		switch ev.Kind {
		case sttchain.StreamEventSegment:
			if ev.Segment != nil {
				res.Segments = append(res.Segments, ev.Segment.Text)
				res.SegmentCount++
			}
		case sttchain.StreamEventPartial:
			if ev.Partial != nil {
				res.Partials = append(res.Partials, ev.Partial.Text)
			}
		case sttchain.StreamEventError:
			if ev.Error != nil {
				res.Err = ev.Error
			}
		case sttchain.StreamEventDone:
			doneAt = opts.now()
			if ev.Done != nil {
				res.FinalText = ev.Done.FinalText
			}
		}
	}
	if err := <-sessionDone; err != nil && res.Err == nil {
		res.Err = err
	}
	if !doneAt.IsZero() && !lastFedAt.IsZero() && doneAt.After(lastFedAt) {
		res.FinalizationLatency = doneAt.Sub(lastFedAt)
	}
	return res
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
	wer := WER(Tokenize(clip.Reference, norm), Tokenize(res.FinalText, norm))

	cr := ClipResult{
		ClipID:           clip.ID,
		Reference:        clip.Reference,
		Hypothesis:       res.FinalText,
		WER:              wer,
		SegmentCount:     res.SegmentCount,
		PartialRevisions: PartialRevisions(res.Partials),
		Err:              res.Err,
	}
	if meter != nil {
		snap := meter.Snapshot()
		cr.WhisperCalls = snap.Calls
		cr.WhisperAudioSeconds = snap.AudioSeconds
		cr.RTF = RTF(snap.ProviderLatency, clip.Duration())
	}
	if opts.Mode == ModeRealtime {
		cr.LatencySamplesMs = []float64{float64(res.FinalizationLatency) / float64(time.Millisecond)}
	}
	return cr
}

// StrategySpec describes one strategy row in the report. BuildSession is
// invoked once per replay (strategies are single-use): it returns a fresh
// Session bound to a fresh MeteredProvider so per-clip compute is isolated.
type StrategySpec struct {
	Kind         sttchain.StrategyKind
	Label        string
	BuildSession func(clip Clip) (Session, *MeteredProvider)
}

// EvalOptions tunes a full RunEval over a corpus.
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
	RealtimeConcurrency int
	Sleep               func(time.Duration)
	Now                 func() time.Time
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

// RunEval replays every clip through every strategy spec and assembles the
// comparison report. The deterministic pass (when QualityPass) yields the
// reproducible WER/compute columns; RealtimeRepeats real-time passes feed
// the per-clip latency samples that aggregate into p50/p95.
func RunEval(ctx context.Context, clips []Clip, specs []StrategySpec, opts EvalOptions) EvalReport {
	report := EvalReport{
		QualityMeasured: opts.QualityPass,
		LatencyMeasured: opts.RealtimeRepeats > 0,
	}
	for _, spec := range specs {
		clipResults := make([]ClipResult, len(clips))
		for i, clip := range clips {
			var cr ClipResult
			if opts.QualityPass {
				session, meter := spec.BuildSession(clip)
				cr = EvalClip(ctx, clip, meter, ReplayOptions{
					Mode: ModeDeterministic, ChunkMs: opts.ChunkMs, Sleep: opts.Sleep, Now: opts.Now,
				}, session)
			} else {
				cr = ClipResult{ClipID: clip.ID, Reference: clip.Reference}
			}
			clipResults[i] = cr
		}
		if opts.RealtimeRepeats > 0 {
			runRealtimeRepeats(ctx, clips, spec, opts, clipResults)
		}
		report.PerStrategy = append(report.PerStrategy, aggregateStrategy(spec.Kind, spec.Label, clipResults))
	}
	return report
}

type realtimeResult struct {
	clipIndex int
	repeat    int
	result    ClipResult
}

func runRealtimeRepeats(ctx context.Context, clips []Clip, spec StrategySpec, opts EvalOptions, clipResults []ClipResult) {
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
					Mode: ModeRealtime, ChunkMs: opts.ChunkMs, Sleep: opts.Sleep, Now: opts.Now,
				}, session)
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
