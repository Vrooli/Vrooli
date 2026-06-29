package eval

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/stt/strategy"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

// defaultStrategyKinds is the trio compared when a RunEval request lists no
// strategies: the batch oracle plus the two streaming strategies.
var defaultStrategyKinds = []string{"batch", "vad_segment", "overlap_agree"}

// loadClips resolves the requested clip ids (or the whole corpus) into
// replayable eval.Clips, fetching each clip's audio from the blob store.
func (h *connectHandler) loadClips(ctx context.Context, ids []string) ([]inteval.Clip, error) {
	var metas []intcorpus.Clip
	if len(ids) == 0 {
		all, err := h.deps.Corpus.ListClips(ctx, intcorpus.ListFilter{})
		if err != nil {
			return nil, err
		}
		metas = all
	} else {
		for _, id := range ids {
			c, err := h.deps.Corpus.GetClip(ctx, id)
			if err != nil {
				return nil, err
			}
			metas = append(metas, c)
		}
	}

	clips := make([]inteval.Clip, 0, len(metas))
	for _, m := range metas {
		audio, _, err := h.deps.Corpus.GetClipAudio(ctx, m.ID)
		if err != nil {
			return nil, fmt.Errorf("load audio for clip %q: %w", m.ID, err)
		}
		sr := m.SampleRateHz
		if sr <= 0 {
			sr = 16000
		}
		clips = append(clips, inteval.Clip{ID: m.ID, PCM: audio, SampleRate: sr, Reference: m.ReferenceText, Format: m.Format})
	}
	return clips, nil
}

// buildSpecs translates the requested EvalStrategies (or the default trio)
// into eval.StrategySpecs, each carrying a BuildSession closure that wires a
// fresh metered provider to the strategy.
func (h *connectHandler) buildSpecs(reqStrategies []*evalv1.EvalStrategy) ([]inteval.StrategySpec, error) {
	type cfg struct {
		kind  string
		label string
		stall int
		win   int
		runs  int
		vad   int
	}
	var wanted []cfg
	if len(reqStrategies) == 0 {
		for _, k := range defaultStrategyKinds {
			wanted = append(wanted, cfg{kind: k})
		}
	} else {
		for _, s := range reqStrategies {
			wanted = append(wanted, cfg{
				kind: s.GetKind(), label: s.GetLabel(),
				stall: int(s.GetOverlapMaxStallRejects()),
				win:   int(s.GetOverlapWindowMs()),
				runs:  int(s.GetOverlapCommitRuns()),
				vad:   int(s.GetVadSilenceMs()),
			})
		}
	}

	specs := make([]inteval.StrategySpec, 0, len(wanted))
	for _, w := range wanted {
		w := w
		label := w.label
		if label == "" {
			label = w.kind
		}
		switch w.kind {
		case "batch", "buffered", "buffered_fallback":
			specs = append(specs, inteval.StrategySpec{
				Kind: sttchain.StrategyBuffered, Label: label,
				BuildSession: func(clip inteval.Clip) (inteval.Session, *inteval.MeteredProvider) {
					meter := h.newMeter(clip)
					return batchSession(meter, clip.Format), meter
				},
			})
		case "vad_segment", "vad":
			silence := h.deps.Defaults.VADSilenceMs
			if w.vad > 0 {
				silence = w.vad
			}
			specs = append(specs, inteval.StrategySpec{
				Kind: sttchain.StrategyVADSegment, Label: label,
				BuildSession: func(clip inteval.Clip) (inteval.Session, *inteval.MeteredProvider) {
					meter := h.newMeter(clip)
					strat := &strategy.VADSegmenter{
						Provider:           meter,
						SilenceMs:          silence,
						PreRollMs:          h.deps.Defaults.VADPreRollMs,
						TrailingPadMs:      h.deps.Defaults.VADTrailingPadMs,
						InitialPromptWords: h.deps.Defaults.VADInitialPromptWords,
					}
					return strategySession(strat, clip.SampleRate), meter
				},
			})
		case "overlap_agree", "overlap":
			stall := h.deps.Defaults.OverlapMaxStallRejects
			if w.stall >= 0 {
				stall = w.stall // 0 = disabled is honored
			}
			win := h.deps.Defaults.OverlapWindowMs
			if w.win > 0 {
				win = w.win
			}
			runs := h.deps.Defaults.OverlapCommitRuns
			if w.runs > 0 {
				runs = w.runs
			}
			specs = append(specs, inteval.StrategySpec{
				Kind: sttchain.StrategyOverlapAgree, Label: label,
				BuildSession: func(clip inteval.Clip) (inteval.Session, *inteval.MeteredProvider) {
					meter := h.newMeter(clip)
					strat := &strategy.OverlapAgree{
						Provider:        meter,
						WindowMs:        win,
						CommitRuns:      runs,
						MaxStallRejects: stall,
						SampleRate:      clip.SampleRate,
					}
					return strategySession(strat, clip.SampleRate), meter
				},
			})
		default:
			return nil, fmt.Errorf("unknown strategy kind %q (want batch|vad_segment|overlap_agree)", w.kind)
		}
	}
	return specs, nil
}

func (h *connectHandler) newMeter(clip inteval.Clip) *inteval.MeteredProvider {
	return inteval.NewMeteredProvider(h.deps.NewProvider(), float64(clip.SampleRate*2))
}

// strategySession adapts a PCM-consuming strategy's Run into a harness
// Session, stamping the canonical-PCM input format so the provider knows
// the bytes' codec.
func strategySession(strat strategy.Strategy, _ int) inteval.Session {
	start := sttchain.StreamStart{InputFormat: "pcm_s16le"}
	return inteval.StrategySession(func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		return strat.Run(ctx, start, chunks, events)
	})
}

// batchSession is the oracle row: drain the whole clip, make ONE metered
// transcribe pass, and emit a single Segment + Done with that text.
func batchSession(meter *inteval.MeteredProvider, format string) inteval.Session {
	if format == "" {
		format = "pcm_s16le"
	}
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		var audio []byte
		for ch := range chunks {
			audio = append(audio, ch.Audio...)
		}
		res, err := meter.Transcribe(ctx, sttchain.Request{Audio: audio, Format: format})
		text := ""
		if res != nil {
			text = res.Text
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: text}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: text}}
		close(events)
		return err
	}
}
