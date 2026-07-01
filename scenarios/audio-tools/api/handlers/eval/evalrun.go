package eval

import (
	"context"
	"fmt"

	sttH "audio-tools/handlers/stt"
	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/stt"
	"audio-tools/internal/stt/segmenter"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

const evalEngineID = "eval-whisper-local"

// defaultStrategyKinds is the trio compared when a RunReport request lists no
// strategies: the batch oracle plus the two streaming strategies.
var defaultStrategyKinds = []string{"batch", "vad_segment", "overlap_agree"}

// loadClips resolves the requested clip ids (or the whole corpus) into
// replayable eval.Clips, fetching each clip's audio from the blob store.
func (h *reportRunner) loadClips(ctx context.Context, ids []string) ([]inteval.Clip, error) {
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
func (h *reportRunner) buildSpecs(reqStrategies []*evalv1.EvalStrategy) ([]inteval.StrategySpec, error) {
	type cfg struct {
		kind  string
		label string
		stall int
		win   int
		runs  int
		max   int
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
				max:   int(s.GetOverlapMaxWindowMs()),
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
			cfg := h.streamConfigFor(stt.PreferenceVAD)
			if w.vad > 0 {
				cfg.VADSilenceMs = w.vad
			}
			specs = append(specs, inteval.StrategySpec{
				Kind: sttchain.StrategyVADSegment, Label: label,
				BuildSession: func(clip inteval.Clip) (inteval.Session, *inteval.MeteredProvider) {
					meter := h.newMeter(clip)
					return h.segmenterSession(meter, cfg, clip), meter
				},
			})
		case "overlap_agree", "overlap":
			cfg := h.streamConfigFor(stt.PreferenceOverlap)
			if w.stall >= 0 {
				cfg.OverlapMaxStallRejects = w.stall // 0 = disabled is honored
			}
			if w.win > 0 {
				cfg.OverlapWindowMs = w.win
			}
			if w.runs > 0 {
				cfg.OverlapCommitRuns = w.runs
			}
			if w.max > 0 {
				cfg.OverlapMaxWindowMs = w.max
			}
			specs = append(specs, inteval.StrategySpec{
				Kind: sttchain.StrategyOverlapAgree, Label: label,
				BuildSession: func(clip inteval.Clip) (inteval.Session, *inteval.MeteredProvider) {
					meter := h.newMeter(clip)
					return h.segmenterSession(meter, cfg, clip), meter
				},
			})
		default:
			return nil, fmt.Errorf("unknown strategy kind %q (want batch|vad_segment|overlap_agree)", w.kind)
		}
	}
	return specs, nil
}

func (h *reportRunner) newMeter(clip inteval.Clip) *inteval.MeteredProvider {
	return inteval.NewMeteredProvider(h.deps.NewProvider(), float64(clip.SampleRate*2))
}

func (h *reportRunner) streamConfigFor(pref stt.StrategyPreference) stt.StreamConfig {
	cfg := h.deps.Defaults
	if cfg.Mode == "" {
		cfg = stt.Defaults()
	}
	cfg.Mode = stt.ModeAuto
	cfg.StrategyPreference = pref
	cfg.EngineID = evalEngineID
	return cfg
}

// segmenterSession routes eval streaming strategies through the production
// Segmenter path. Speaker stages stay unbound unless a caller supplies a
// per-run SpeakerConfig, so the public eval surface remains speaker-off while
// experiments can exercise extraction/verification hermetically.
func (h *reportRunner) segmenterSession(meter *inteval.MeteredProvider, cfg stt.StreamConfig, clip inteval.Clip) inteval.Session {
	chain := sttchain.NewChain(sttchain.Options{
		EnableLocal:  true,
		LocalEngines: map[string]sttchain.Provider{evalEngineID: meter},
	})
	selector := stt.NewSelector(providerBatchExecutor{provider: meter})
	deps := segmenter.Deps{Chain: chain, Selector: selector}
	if h.deps.SpeakerConfig != nil {
		if h.deps.SpeakerVerificationEnabled {
			deps.SpeakerIsolation = sttH.NewSpeakerIsolationFromConfig(*h.deps.SpeakerConfig, h.deps.SpeakerResource, h.deps.Logger)
		}
		if h.deps.SpeakerExtractionEnabled {
			deps.SpeakerExtraction = sttH.NewSpeakerExtractionFromConfig(*h.deps.SpeakerConfig, h.deps.SpeakerResource)
		}
	}
	seg := segmenter.New(deps)
	start := sttchain.StreamStart{
		InputFormat:     "pcm_s16le",
		InputSampleRate: int32(clip.SampleRate),
		EngineID:        evalEngineID,
	}
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		return seg.Run(ctx, start, cfg, chunks, events)
	}
}

type providerBatchExecutor struct {
	provider sttchain.Provider
}

func (e providerBatchExecutor) Execute(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	return e.provider.Transcribe(ctx, req)
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
