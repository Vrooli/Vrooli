//go:build audio_lab

package eval

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"
	"audio-tools/internal/stt/strategy"
)

func TestEvalClip_PreservesBoundedProviderWordTimingsInReport(t *testing.T) {
	words := []sttchain.TimedWord{{Word: "hello", Start: 0.1, End: 0.25, Prob: 0.97}}
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "hello", Words: words}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
		close(events)
		return nil
	}
	clip := Clip{ID: "timed", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 250), SampleRate: testaudio.SampleRateHz, Reference: "hello"}
	got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Equal(t, words, got.HypothesisWordTimestamps)
	require.Nil(t, got.TimestampDrift, "provider timings alone are not a drift score")

	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"hypothesis_word_timestamps":[{"word":"hello","start_seconds":0.1,"end_seconds":0.25,"probability":0.97}]`)
}

func TestEvalClip_ScoresOnlyHumanReviewedTimedReferences(t *testing.T) {
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
			Text: "hello world", Words: []sttchain.TimedWord{{Word: "hello", Start: .1, End: .3}, {Word: "world", Start: .4, End: .6}},
		}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello world"}}
		close(events)
		return nil
	}
	clip := Clip{
		ID: "reviewed", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 1000), SampleRate: testaudio.SampleRateHz,
		Reference: "hello world", TimedReference: []TimedWord{{Word: "hello", Start: .08, End: .29}, {Word: "world", Start: .38, End: .62}},
		TimingProvenance: &TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "source-1", Reviewer: "reviewer-1", ReviewMethod: "manual-listen-and-correct", ReviewedAt: "2026-11-20T12:00:00Z"},
	}
	got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.NotNil(t, got.TimestampDrift)
	require.Empty(t, got.TimestampDriftSkipReason)
	require.Equal(t, "reviewer-1", got.TimestampReference.Reviewer)
	require.InDelta(t, 20, got.TimestampDrift.StartAbsP50Ms, 1e-9)

	clip.TimingProvenance.Status = "aligner_only"
	got = EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Nil(t, got.TimestampDrift)
	require.Equal(t, "timed_reference_not_human_reviewed", got.TimestampDriftSkipReason)

	clip.TimingProvenance.Status = "human_reviewed"
	clip.TimingProvenance.AlignmentTool = "aligner"
	got = EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Nil(t, got.TimestampDrift, "partial alignment provenance must not be treated as reviewed")
	require.Equal(t, "timed_reference_not_human_reviewed", got.TimestampDriftSkipReason)
}

func TestEvalClip_SkipsDriftWhenHypothesisTimingExceedsAudio(t *testing.T) {
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
			Text: "hello", Words: []sttchain.TimedWord{{Word: "hello", Start: .1, End: 1.1}},
		}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
		close(events)
		return nil
	}
	clip := Clip{
		ID: "out-of-bounds", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 1000), SampleRate: testaudio.SampleRateHz,
		Reference: "hello", TimedReference: []TimedWord{{Word: "hello", Start: .1, End: .5}},
		TimingProvenance: &TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "source-1", AlignmentTool: "aligner", AlignmentModel: "rev-1", Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct", ReviewedAt: "2026-11-20T12:00:00Z"},
	}

	got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Nil(t, got.TimestampDrift)
	require.Equal(t, "hypothesis_word_timestamps_out_of_bounds", got.TimestampDriftSkipReason)
}

func TestEvalClip_ExplainsWhenHypothesisHasNoLexicalMatches(t *testing.T) {
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		words := []sttchain.TimedWord{{Word: "goodbye", Start: .1, End: .3}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "goodbye", Words: words}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "goodbye"}}
		close(events)
		return nil
	}
	clip := Clip{
		ID: "no-lexical-match", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 1000), SampleRate: testaudio.SampleRateHz,
		Reference: "hello", TimedReference: []TimedWord{{Word: "hello", Start: .1, End: .3}},
		TimingProvenance: &TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "source-1", Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct", ReviewedAt: "2026-11-20T12:00:00Z"},
	}

	got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Nil(t, got.TimestampDrift)
	require.Equal(t, "timestamp_no_lexical_matches", got.TimestampDriftSkipReason)
}

func TestEvalClip_SkipsDriftWhenReviewedReferenceDoesNotDescribeClip(t *testing.T) {
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
			Text: "hello", Words: []sttchain.TimedWord{{Word: "hello", Start: .1, End: .3}},
		}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
		close(events)
		return nil
	}
	provenance := &TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "source-1", Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct", ReviewedAt: "2026-11-20T12:00:00Z"}
	tests := []struct {
		name       string
		reference  string
		word       TimedWord
		wantReason string
	}{
		{name: "transcript mismatch", reference: "goodbye", word: TimedWord{Word: "hello", Start: .1, End: .3}, wantReason: "timed_reference_transcript_mismatch"},
		{name: "reference exceeds audio", reference: "hello", word: TimedWord{Word: "hello", Start: .1, End: 1.1}, wantReason: "timed_reference_out_of_bounds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clip := Clip{
				ID: "invalid-reference", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 1000), SampleRate: testaudio.SampleRateHz,
				Reference: tt.reference, TimedReference: []TimedWord{tt.word}, TimingProvenance: provenance,
			}
			got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
			require.Nil(t, got.TimestampDrift)
			require.Equal(t, tt.wantReason, got.TimestampDriftSkipReason)
		})
	}
}

func TestEvalClip_ExplainsMissingTimedReference(t *testing.T) {
	session := func(_ context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello"}}
		close(events)
		return nil
	}
	clip := Clip{ID: "ordinary", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 250), SampleRate: testaudio.SampleRateHz, Reference: "hello"}
	got := EvalClip(context.Background(), clip, nil, ReplayOptions{}, session)
	require.Nil(t, got.TimestampDrift)
	require.Equal(t, "timed_reference_absent", got.TimestampDriftSkipReason)
}

func TestReplay_BoundsProviderWordTimingPayload(t *testing.T) {
	words := make([]sttchain.TimedWord, maxHypothesisWordTimestamps+1)
	for i := range words {
		words[i] = sttchain.TimedWord{Word: "word", Start: float64(i), End: float64(i) + 0.1}
	}
	var got StreamResult
	appendHypothesisWordTimestamps(&got, words, 0, 0, testaudio.SampleRateHz)
	require.Len(t, got.HypothesisWordTimestamps, maxHypothesisWordTimestamps)
	require.True(t, got.HypothesisWordTimestampsTruncated)
}

func fakeProv(text string, latency time.Duration) *sttmocks.FakeProvider {
	p := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true})
	p.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, ProviderID: "whisper", ModelID: "base", Latency: latency}, nil
	}
	return p
}

// controlledSession drains the replayed chunks, makes ONE metered backend
// call over the concatenated audio (so the meter records calls +
// audio-seconds), then emits a single Segment + terminal Done carrying a
// caller-controlled final text. It isolates the harness's metric/report
// wiring from any strategy's commit quirks.
func controlledSession(meter *MeteredProvider, finalText string) Session {
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		var audio []byte
		for ch := range chunks {
			audio = append(audio, ch.Audio...)
		}
		_, _ = meter.Transcribe(ctx, sttchain.Request{Audio: audio})
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: finalText}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: finalText}}
		close(events)
		return nil
	}
}

func TestRunReport_AggregatesQualityAndCompute(t *testing.T) {
	clips := []Clip{
		{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 1000), SampleRate: testaudio.SampleRateHz, Reference: "the quick brown fox"},
		{ID: "c2", PCM: testaudio.SilenceSamplesAtRate(testaudio.SampleRateHz, 2000), SampleRate: testaudio.SampleRateHz, Reference: "hello world"},
	}

	perfect := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "perfect",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", 20*time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}
	// lossy drops the last reference word from each clip -> one deletion/clip.
	lossy := StrategySpec{
		Kind: sttchain.StrategyVADSegment, Label: "lossy",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", 10*time.Millisecond), float64(clip.bytesPerSecond()))
			words := strings.Fields(clip.Reference)
			dropped := strings.Join(words[:len(words)-1], " ")
			return controlledSession(meter, dropped), meter
		},
	}

	rep := RunReport(context.Background(), clips, []StrategySpec{perfect, lossy}, DefaultEvalOptions())
	require.True(t, rep.QualityMeasured)
	require.False(t, rep.LatencyMeasured)
	require.Len(t, rep.PerStrategy, 2)

	p := rep.PerStrategy[0]
	require.Equal(t, "perfect", p.Label)
	require.InDelta(t, 0.0, p.WER, 1e-9, "perfect strategy has zero WER")
	require.Equal(t, 6, p.RefWords, "4 + 2 reference words across both clips")
	require.Equal(t, 2, p.WhisperCalls, "one metered call per clip")
	require.InDelta(t, 3.0, p.WhisperAudioSeconds, 1e-9, "1s + 2s of audio")
	require.Greater(t, p.RTF, 0.0)
	require.Len(t, p.PerClip, 2)

	l := rep.PerStrategy[1]
	require.Equal(t, "lossy", l.Label)
	require.Equal(t, 2, l.EditCounts.Deletions, "one dropped word per clip")
	require.Equal(t, 0, l.EditCounts.Substitutions)
	require.Equal(t, 0, l.EditCounts.Insertions)
	require.InDelta(t, 2.0/6.0, l.WER, 1e-9, "micro-average WER = total edits / total ref words")
}

func TestRunReport_EmitsPerClipProgress(t *testing.T) {
	clips := []Clip{
		{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(16000, 1000), SampleRate: 16000, Reference: "one"},
		{ID: "c2", PCM: testaudio.SilenceSamplesAtRate(16000, 1000), SampleRate: 16000, Reference: "two"},
	}
	spec := StrategySpec{
		Kind: sttchain.StrategyBuffered, Label: "batch",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("x", 0), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}
	var progress []EvalProgress
	RunReport(context.Background(), clips, []StrategySpec{spec}, EvalOptions{
		ChunkMs:       100,
		QualityPass:   true,
		ProgressScope: "condition clean",
		Progress: func(update EvalProgress) {
			progress = append(progress, update)
		},
	})

	require.Len(t, progress, 2)
	require.Equal(t, "condition clean", progress[0].Scope)
	require.Equal(t, "quality", progress[0].Phase)
	require.Equal(t, string(sttchain.StrategyBuffered), progress[0].Strategy)
	require.Equal(t, "c1", progress[0].ClipID)
	require.Equal(t, 1, progress[0].ClipIndex)
	require.Equal(t, 2, progress[1].ClipIndex)
}

// TestRunReport_RealOverlapAgreePath proves the harness drives a REAL
// OverlapAgree strategy end-to-end (replay -> strategy -> events -> WER),
// metering the backend calls, without needing a live Whisper.
func TestRunReport_RealOverlapAgreePath(t *testing.T) {
	clip := Clip{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(16000, 500), SampleRate: 16000, Reference: "hello world"}

	spec := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "overlap-agree",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			strat := &strategy.OverlapAgree{
				Provider: meter, Trigger: strategy.TriggerStopwatch,
				WindowMs: 100, AdvanceMs: 100, CommitRuns: 2, SampleRate: 16000,
			}
			return StrategySession(func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
				return strat.Run(ctx, sttchain.StreamStart{}, chunks, events, nil)
			}), meter
		},
	}

	rep := RunReport(context.Background(), []Clip{clip}, []StrategySpec{spec}, DefaultEvalOptions())
	require.Len(t, rep.PerStrategy, 1)
	row := rep.PerStrategy[0]
	require.InDelta(t, 0.0, row.WER, 1e-9, "agreement commits the full reference -> zero WER")
	require.GreaterOrEqual(t, row.WhisperCalls, 2, "overlap-agree calls the backend per settle attempt")
	require.Equal(t, "hello world", strings.TrimSpace(row.PerClip[0].Hypothesis))
}

// TestRunReport_RealtimeProducesLatencySamples proves the real-time pass
// records one finalization-latency sample per repeat. Sleep is stubbed and
// Now is a deterministic monotone counter, so the test never actually
// sleeps and stays reproducible.
func TestRunReport_RealtimeProducesLatencySamples(t *testing.T) {
	clip := Clip{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(16000, 300), SampleRate: 16000, Reference: "hello world"}
	spec := StrategySpec{
		Kind: sttchain.StrategyOverlapAgree, Label: "overlap-agree",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, "hello world"), meter
		},
	}

	opts := EvalOptions{
		ChunkMs: 100, QualityPass: true, RealtimeRepeats: 3,
		Sleep: func(time.Duration) {}, // never actually sleep
	}
	rep := RunReport(context.Background(), []Clip{clip}, []StrategySpec{spec}, opts)
	require.True(t, rep.LatencyMeasured)
	require.Len(t, rep.PerStrategy[0].PerClip, 1)
	require.Len(t, rep.PerStrategy[0].PerClip[0].LatencySamplesMs, 3, "one latency sample per real-time repeat")
}

func TestRunReport_QualityOnlyNativeStreamDoesNotClaimLatency(t *testing.T) {
	clip := Clip{ID: "native", PCM: testaudio.SilenceSamplesAtRate(16000, 300), SampleRate: 16000, Reference: "hello world"}
	spec := StrategySpec{
		Kind: sttchain.StrategyPassthrough, Label: "native",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}

	report := RunReport(context.Background(), []Clip{clip}, []StrategySpec{spec}, DefaultEvalOptions())
	require.True(t, report.QualityMeasured)
	require.False(t, report.LatencyMeasured)
	require.Empty(t, report.PerStrategy[0].PerClip[0].LatencySamplesMs)
	require.Zero(t, report.PerStrategy[0].FinalizationLatencyP95Ms)
}

func TestRunReport_NativePassthroughSamplesOnlyRequestedLatencyRepeats(t *testing.T) {
	clip := Clip{ID: "native", PCM: testaudio.SilenceSamplesAtRate(16000, 300), SampleRate: 16000, Reference: "hello world"}
	spec := StrategySpec{
		Kind: sttchain.StrategyPassthrough, Label: "native",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv("hello world", time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}
	var ticks atomic.Int64
	report := RunReport(context.Background(), []Clip{clip}, []StrategySpec{spec}, EvalOptions{
		ChunkMs: 100, QualityPass: true, RealtimeRepeats: 2, RealtimeConcurrency: 1,
		Sleep: func(time.Duration) {},
		Now:   func() time.Time { return time.Unix(0, ticks.Add(1)*int64(time.Millisecond)) },
	})
	require.True(t, report.QualityMeasured)
	require.True(t, report.LatencyMeasured)
	require.Len(t, report.PerStrategy[0].PerClip[0].LatencySamplesMs, 2, "paced quality pass must not add a third sample")
	require.Greater(t, report.PerStrategy[0].FinalizationLatencyP95Ms, 0.0)
}

func TestReplay_RealtimeCapturesEndOfFeedBeforeImmediateDone(t *testing.T) {
	clip := Clip{ID: "immediate-done", PCM: testaudio.SilenceSamplesAtRate(16000, 300), SampleRate: 16000}
	var ticks atomic.Int64
	result := Replay(context.Background(), clip, ReplayOptions{
		Mode:    ModeRealtime,
		ChunkMs: 100,
		Sleep:   func(time.Duration) {},
		Now: func() time.Time {
			return time.Unix(0, ticks.Add(1)*int64(time.Millisecond))
		},
	}, func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		close(events)
		return nil
	})

	require.NoError(t, result.Err)
	require.Greater(t, result.FinalizationLatency, time.Duration(0), "immediate Done must retain a measured end-of-feed timestamp")
}

func TestReplay_ContextDeadlineReturnsWhenSessionDoesNotCloseEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	clip := Clip{ID: "deadline", PCM: testaudio.SilenceSamplesAtRate(16000, 100), SampleRate: 16000}
	result := make(chan StreamResult, 1)
	go func() {
		result <- Replay(ctx, clip, ReplayOptions{ChunkMs: 100}, func(ctx context.Context, _ <-chan sttchain.AudioChunk, _ chan<- sttchain.StreamEvent) error {
			<-ctx.Done()
			// A misbehaving stream may return without closing events. The harness
			// must still honor the experiment's runtime deadline.
			return ctx.Err()
		})
	}()

	select {
	case got := <-result:
		require.ErrorIs(t, got.Err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("Replay ignored context cancellation while waiting for events")
	}
}

func TestReplay_RealtimeTailPacesOnlyFinalWindow(t *testing.T) {
	clip := Clip{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(16000, 2000), SampleRate: 16000, Reference: "hello world"}
	var sleeps []time.Duration
	session := func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		for range chunks {
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "hello world"}}
		close(events)
		return nil
	}

	res := Replay(context.Background(), clip, ReplayOptions{
		Mode:               ModeRealtime,
		ChunkMs:            100,
		LatencyTailSeconds: 1,
		Sleep: func(d time.Duration) {
			sleeps = append(sleeps, d)
		},
	}, session)

	require.NoError(t, res.Err)
	require.Len(t, sleeps, 10, "one-second tail on a two-second clip should skip prefix pacing")
	for _, sleep := range sleeps {
		require.Equal(t, 100*time.Millisecond, sleep)
	}

	sleeps = nil
	res = Replay(context.Background(), clip, ReplayOptions{
		Mode:               ModeRealtime,
		ChunkMs:            100,
		LatencyTailSeconds: 0,
		Sleep: func(d time.Duration) {
			sleeps = append(sleeps, d)
		},
	}, session)
	require.NoError(t, res.Err)
	require.Len(t, sleeps, 20, "zero tail preserves full real-time pacing")
}

func TestRunReport_RealtimeRepeatsRunConcurrentlyWithinBound(t *testing.T) {
	clips := []Clip{
		{ID: "c1", PCM: testaudio.SilenceSamplesAtRate(16000, 100), SampleRate: 16000, Reference: "one"},
		{ID: "c2", PCM: testaudio.SilenceSamplesAtRate(16000, 100), SampleRate: 16000, Reference: "two"},
		{ID: "c3", PCM: testaudio.SilenceSamplesAtRate(16000, 100), SampleRate: 16000, Reference: "three"},
		{ID: "c4", PCM: testaudio.SilenceSamplesAtRate(16000, 100), SampleRate: 16000, Reference: "four"},
	}
	spec := StrategySpec{
		Kind: sttchain.StrategyBuffered, Label: "batch",
		BuildSession: func(clip Clip) (Session, *MeteredProvider) {
			meter := NewMeteredProvider(fakeProv(clip.Reference, time.Millisecond), float64(clip.bytesPerSecond()))
			return controlledSession(meter, clip.Reference), meter
		},
	}

	var mu sync.Mutex
	active := 0
	maxActive := 0
	twoActive := make(chan struct{})
	var closeOnce sync.Once
	sleep := func(time.Duration) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		if active == 2 {
			closeOnce.Do(func() { close(twoActive) })
		}
		mu.Unlock()

		<-twoActive

		mu.Lock()
		active--
		mu.Unlock()
	}

	rep := RunReport(context.Background(), clips, []StrategySpec{spec}, EvalOptions{
		ChunkMs: 100, QualityPass: false, RealtimeRepeats: 1, RealtimeConcurrency: 2, Sleep: sleep,
	})
	require.True(t, rep.LatencyMeasured)
	require.Len(t, rep.PerStrategy[0].PerClip, 4)
	require.Equal(t, 2, maxActive, "real-time replay should use bounded concurrency instead of serializing every clip")
}
