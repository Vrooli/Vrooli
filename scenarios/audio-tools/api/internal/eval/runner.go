package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/protoint"
	"audio-tools/internal/qualification"
	"audio-tools/internal/stt"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/stt/segmenter"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

// RunnerDeps contains the explicit ports needed to turn an experiment recipe
// into replay sessions. Transport packages provide adapters for corpus loading
// and optional speaker stages; the runner never depends on a handler package.
type RunnerDeps struct {
	LoadClips            func(context.Context, []string) ([]Clip, error)
	NewProvider          func() sttchain.Provider
	NewProviderForEngine func(string) sttchain.Provider
	Defaults             stt.StreamConfig
	NewSpeakerIsolation  func() egress.SpeakerIsolation
	NewSpeakerExtraction func() ingress.TargetExtractor
	// Snapshot factories are used by experiment-condition runners. They keep
	// resource adapters at composition while allowing each condition to bind a
	// fresh, isolated speaker stage.
	NewSpeakerIsolationForConfig  func(sttpipeline.SpeakerConfig, *sttpipeline.SpeakerClient) egress.SpeakerIsolation
	NewSpeakerExtractionForConfig func(sttpipeline.SpeakerConfig, *sttpipeline.SpeakerClient) ingress.TargetExtractor
	// Speaker fields are experiment metadata owned by the composition layer;
	// factories above turn a selected snapshot into pipeline stages.
	SpeakerConfig              *sttpipeline.SpeakerConfig
	SpeakerExtractionEnabled   bool
	SpeakerVerificationEnabled bool
	SpeakerResource            *sttpipeline.SpeakerClient
}

type reportRunner struct{ deps RunnerDeps }

var (
	errEvalNotConfigured  = errors.New("eval service not configured (no corpus/database)")
	errEvalNoProvider     = errors.New("eval requires a transcription provider (Whisper) — none configured")
	errEvalEmptyCorpus    = errors.New("corpus is empty — record clips before running an eval")
	errEvalInvalidRequest = errors.New("invalid eval request")
)

func RunReportWithOptions(ctx context.Context, deps RunnerDeps, clipIDs []string, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts EvalOptions) (EvalReport, error) {
	if deps.LoadClips == nil {
		return EvalReport{}, errEvalNotConfigured
	}
	if deps.NewProvider == nil {
		return EvalReport{}, errEvalNoProvider
	}
	clips, err := deps.LoadClips(ctx, clipIDs)
	if err != nil {
		return EvalReport{}, err
	}
	if len(clips) == 0 {
		return EvalReport{}, errEvalEmptyCorpus
	}
	return RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, opts)
}

func RunReportForClipsWithOptions(ctx context.Context, deps RunnerDeps, clips []Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts EvalOptions) (EvalReport, error) {
	h := reportRunner{deps: deps}
	if h.deps.NewProvider == nil {
		return EvalReport{}, errEvalNoProvider
	}
	if len(clips) == 0 {
		return EvalReport{}, errEvalEmptyCorpus
	}
	specs, err := h.buildSpecs(strategies)
	if err != nil {
		return EvalReport{}, fmt.Errorf("%w: %v", errEvalInvalidRequest, err)
	}
	opts.ChunkMs, opts.QualityPass, opts.RealtimeRepeats = int(chunkMs), true, int(realtimeRepeats)
	return RunReport(ctx, clips, specs, opts), nil
}

func RunReportForCells(ctx context.Context, deps RunnerDeps, clips []Clip, cells []*experimentv1.EvaluationCell, chunkMs int32, opts EvalOptions) (EvalReport, error) {
	h := reportRunner{deps: deps}
	if len(clips) == 0 {
		return EvalReport{}, errEvalEmptyCorpus
	}
	if len(cells) == 0 {
		return EvalReport{}, fmt.Errorf("%w: no evaluation cells", errEvalInvalidRequest)
	}
	reports := make([]EvalReport, 0, len(cells))
	for i, cell := range cells {
		if err := validateExecutableCell(cell); err != nil {
			return EvalReport{}, fmt.Errorf("%w: cells[%d]: %v", errEvalInvalidRequest, i, err)
		}
		specs, err := h.buildCellSpecs([]*experimentv1.EvaluationCell{cell})
		if err != nil {
			return EvalReport{}, fmt.Errorf("%w: %v", errEvalInvalidRequest, err)
		}
		cellOpts := opts
		cellOpts.ChunkMs, cellOpts.QualityPass, cellOpts.RealtimeRepeats = int(chunkMs), true, 0
		switch cell.GetReplayLane() {
		case experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC:
			for repeat := int32(0); repeat < cell.GetRepeatCount(); repeat++ {
				repeatSpecs := specs
				if cell.GetRepeatCount() > 1 {
					repeatSpecs = append([]StrategySpec(nil), specs...)
					repeatSpecs[0].Label = fmt.Sprintf("%s [repeat %d/%d]", specs[0].Label, repeat+1, cell.GetRepeatCount())
					repeatSpecs[0].CellID = fmt.Sprintf("%s/repeat-%d", specs[0].CellID, repeat+1)
				}
				reports = append(reports, RunReport(ctx, clips, repeatSpecs, cellOpts))
			}
		case experimentv1.ReplayLane_REPLAY_LANE_REALTIME:
			cellOpts.QualityPass, cellOpts.RealtimeRepeats = false, int(cell.GetRepeatCount())
			reports = append(reports, RunReport(ctx, clips, specs, cellOpts))
		default:
			return EvalReport{}, fmt.Errorf("%w: cells[%d] has unsupported replay lane", errEvalInvalidRequest, i)
		}
	}
	report := CombineReports(reports...)
	report.PromotionVerdicts = PromotionVerdicts(report)
	return report, nil
}

func RunReportCellsWithOptions(ctx context.Context, deps RunnerDeps, clipIDs []string, cells []*experimentv1.EvaluationCell, chunkMs int32, opts EvalOptions) (EvalReport, error) {
	if deps.LoadClips == nil {
		return EvalReport{}, errEvalNotConfigured
	}
	clips, err := deps.LoadClips(ctx, clipIDs)
	if err != nil {
		return EvalReport{}, err
	}
	return RunReportForCells(ctx, deps, clips, cells, chunkMs, opts)
}

// PromotionVerdicts derives qualification evidence from evaluated strategy rows.
func PromotionVerdicts(report EvalReport) []PromotionVerdict {
	measurements := make([]trustfloor.ReplayMeasurement, 0, len(report.PerStrategy))
	for _, row := range report.PerStrategy {
		if row.EngineID == "" {
			continue
		}
		measurement := trustfloor.ReplayMeasurement{EngineID: row.EngineID, ModelID: row.ModelID, Strategy: string(row.Strategy), PolicyProfile: row.PolicyProfile, WER: row.WER, ReplayLane: row.ReplayLane, SafetyObserved: true, SafetyPassed: row.Safety.Passed}
		for _, clip := range row.PerClip {
			measurement.ClipDurationsMS = append(measurement.ClipDurationsMS, clip.AudioDurationMs)
		}
		measurements = append(measurements, measurement)
	}
	assessed := trustfloor.EvaluateReplayMeasurements(measurements, trustfloor.DefaultThresholds)
	verdicts := make([]PromotionVerdict, 0, len(assessed))
	for _, verdict := range assessed {
		verdicts = append(verdicts, PromotionVerdict{EngineID: verdict.EngineID, ModelID: verdict.ModelID, Strategy: verdict.Strategy, PolicyProfile: verdict.PolicyProfile, Stable: verdict.Verdict.Stable, Reasons: verdict.Verdict.Reasons})
	}
	return verdicts
}

func validateExecutableCell(cell *experimentv1.EvaluationCell) error {
	if cell == nil {
		return errors.New("cell is required")
	}
	if strings.TrimSpace(cell.GetFaultProfile()) != "" {
		return fmt.Errorf("fault profile %q requires the dedicated fault harness", cell.GetFaultProfile())
	}
	if strings.TrimSpace(cell.GetPolicyProfile()) != "" {
		return fmt.Errorf("policy profile %q requires the policy evaluation harness", cell.GetPolicyProfile())
	}
	switch cell.GetReplayLane() {
	case experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, experimentv1.ReplayLane_REPLAY_LANE_REALTIME:
		return nil
	case experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH:
		return errors.New("product-path evidence must run through the browser qualification harness")
	default:
		return errors.New("replay lane is required")
	}
}

var defaultStrategyKinds = []string{"batch", "vad_segment", "overlap_agree"}

func (h reportRunner) buildSpecs(requested []*evalv1.EvalStrategy) ([]StrategySpec, error) {
	type cfg struct {
		kind, label                string
		stall, win, runs, max, vad int
	}
	wanted := make([]cfg, 0, len(requested))
	if len(requested) == 0 {
		for _, kind := range defaultStrategyKinds {
			wanted = append(wanted, cfg{kind: kind})
		}
	} else {
		for _, strategy := range requested {
			wanted = append(wanted, cfg{kind: strategy.GetKind(), label: strategy.GetLabel(), stall: int(strategy.GetOverlapMaxStallRejects()), win: int(strategy.GetOverlapWindowMs()), runs: int(strategy.GetOverlapCommitRuns()), max: int(strategy.GetOverlapMaxWindowMs()), vad: int(strategy.GetVadSilenceMs())})
		}
	}
	specs := make([]StrategySpec, 0, len(wanted))
	for _, wanted := range wanted {
		label := wanted.label
		if label == "" {
			label = wanted.kind
		}
		switch wanted.kind {
		case "batch", "buffered", "buffered_fallback":
			specs = append(specs, h.spec(sttchain.StrategyBuffered, label, stt.PreferenceAuto, true, "", StreamConfigAdjust{}))
		case "vad_segment", "vad":
			specs = append(specs, h.spec(sttchain.StrategyVADSegment, label, stt.PreferenceVAD, false, "", StreamConfigAdjust{VADSilenceMs: wanted.vad}))
		case "overlap_agree", "overlap":
			specs = append(specs, h.spec(sttchain.StrategyOverlapAgree, label, stt.PreferenceOverlap, false, "", StreamConfigAdjust{OverlapMaxStallRejects: wanted.stall, OverlapWindowMs: wanted.win, OverlapCommitRuns: wanted.runs, OverlapMaxWindowMs: wanted.max}))
		default:
			return nil, fmt.Errorf("unknown strategy kind %q (want batch|vad_segment|overlap_agree)", wanted.kind)
		}
	}
	return specs, nil
}

type StreamConfigAdjust struct{ VADSilenceMs, OverlapMaxStallRejects, OverlapWindowMs, OverlapCommitRuns, OverlapMaxWindowMs int }

func (h reportRunner) spec(kind sttchain.StrategyKind, label string, preference stt.StrategyPreference, batch bool, engineID string, adjust StreamConfigAdjust) StrategySpec {
	cfg := h.streamConfigForEngine(preference, engineID)
	if adjust.VADSilenceMs > 0 {
		cfg.VADSilenceMs = adjust.VADSilenceMs
	}
	if adjust.OverlapMaxStallRejects >= 0 && (preference == stt.PreferenceOverlap) {
		cfg.OverlapMaxStallRejects = adjust.OverlapMaxStallRejects
	}
	if adjust.OverlapWindowMs > 0 {
		cfg.OverlapWindowMs = adjust.OverlapWindowMs
	}
	if adjust.OverlapCommitRuns > 0 {
		cfg.OverlapCommitRuns = adjust.OverlapCommitRuns
	}
	if adjust.OverlapMaxWindowMs > 0 {
		cfg.OverlapMaxWindowMs = adjust.OverlapMaxWindowMs
	}
	return StrategySpec{Kind: kind, Label: label, EngineID: engineID, BuildSession: func(clip Clip) (Session, *MeteredProvider) {
		meter := h.newMeterForEngine(clip, engineID)
		if meter == nil {
			return nil, nil
		}
		if batch {
			return batchSession(meter, clip.Format), meter
		}
		return h.segmenterSession(meter, cfg, clip, engineID), meter
	}}
}

const evalEngineID = "eval-whisper-local"

func (h reportRunner) newMeterForEngine(clip Clip, engineID string) *MeteredProvider {
	if engineID == "" {
		engineID = evalEngineID
	}
	if h.deps.NewProviderForEngine != nil {
		if provider := h.deps.NewProviderForEngine(engineID); provider != nil {
			return NewMeteredProvider(provider, float64(clip.SampleRate*2))
		}
	}
	if h.deps.NewProvider == nil {
		return nil
	}
	return NewMeteredProvider(h.deps.NewProvider(), float64(clip.SampleRate*2))
}

func (h reportRunner) streamConfigForEngine(preference stt.StrategyPreference, engineID string) stt.StreamConfig {
	if engineID == "" {
		engineID = evalEngineID
	}
	cfg := h.deps.Defaults
	if cfg.Mode == "" {
		cfg = stt.Defaults()
	}
	cfg.Mode, cfg.StrategyPreference, cfg.EngineID = stt.ModeAuto, preference, engineID
	return cfg
}

func (h reportRunner) segmenterSession(meter *MeteredProvider, cfg stt.StreamConfig, clip Clip, engineID string) Session {
	chain := sttchain.NewChain(sttchain.Options{EnableLocal: true, LocalEngines: map[string]sttchain.Provider{engineID: meter}})
	deps := segmenter.Deps{Chain: chain, Selector: stt.NewSelector(providerBatchExecutor{provider: meter})}
	if h.deps.NewSpeakerIsolation != nil {
		deps.SpeakerIsolation = h.deps.NewSpeakerIsolation()
	}
	if h.deps.NewSpeakerExtraction != nil {
		deps.SpeakerExtraction = h.deps.NewSpeakerExtraction()
	}
	seg := segmenter.New(deps)
	start := sttchain.StreamStart{InputFormat: "pcm_s16le", InputSampleRate: protoint.FromInt(clip.SampleRate), EngineID: engineID}
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		return seg.Run(ctx, start, cfg, chunks, events)
	}
}

func (h reportRunner) buildCellSpecs(cells []*experimentv1.EvaluationCell) ([]StrategySpec, error) {
	if len(cells) == 0 {
		return nil, fmt.Errorf("no evaluation cells")
	}
	if h.deps.NewProviderForEngine == nil {
		return nil, fmt.Errorf("provider-neutral cell factory is not configured")
	}
	specs := make([]StrategySpec, 0, len(cells))
	for i, cell := range cells {
		if cell == nil || cell.GetEngineId() == "" {
			return nil, fmt.Errorf("cells[%d].engine_id is required", i)
		}
		engineID := cell.GetEngineId()
		provider := h.deps.NewProviderForEngine(engineID)
		if provider == nil {
			return nil, fmt.Errorf("cells[%d] names unavailable engine %q", i, engineID)
		}
		preference, kind, batch, ok := evaluationCellStrategy(cell.GetStrategy())
		if !ok {
			return nil, fmt.Errorf("cells[%d] has unsupported strategy %q", i, cell.GetStrategy())
		}
		label := cell.GetLabel()
		if label == "" {
			label = engineID + ":" + cell.GetStrategy()
		}
		cfg := h.streamConfigForEngine(preference, engineID)
		spec := h.spec(kind, label, preference, batch, engineID, StreamConfigAdjust{})
		spec.ModelID, spec.PolicyProfile, spec.ReplayLane, spec.FaultProfile, spec.CellID = provider.Model(), cell.GetPolicyProfile(), evaluationReplayLaneName(cell.GetReplayLane()), cell.GetFaultProfile(), engineID+"/"+cell.GetStrategy()+"/"+evaluationReplayLaneName(cell.GetReplayLane())
		spec.BuildSession = func(clip Clip) (Session, *MeteredProvider) {
			meter := h.newMeterForEngine(clip, engineID)
			if meter == nil {
				return nil, nil
			}
			if batch {
				return batchSession(meter, clip.Format), meter
			}
			return h.segmenterSession(meter, cfg, clip, engineID), meter
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func evaluationReplayLaneName(lane experimentv1.ReplayLane) string {
	switch lane {
	case experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC:
		return "deterministic"
	case experimentv1.ReplayLane_REPLAY_LANE_REALTIME:
		return "realtime"
	case experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH:
		return "product_path"
	default:
		return "unspecified"
	}
}

func evaluationCellStrategy(kind string) (stt.StrategyPreference, sttchain.StrategyKind, bool, bool) {
	switch kind {
	case "batch", "buffered_fallback":
		return stt.PreferenceAuto, sttchain.StrategyBuffered, true, true
	case "vad_segment":
		return stt.PreferenceVAD, sttchain.StrategyVADSegment, false, true
	case "overlap_agree", "overlap":
		return stt.PreferenceOverlap, sttchain.StrategyOverlapAgree, false, true
	case "passthrough":
		return stt.PreferencePassthrough, sttchain.StrategyPassthrough, false, true
	default:
		return "", "", false, false
	}
}

type providerBatchExecutor struct{ provider sttchain.Provider }

func (e providerBatchExecutor) Execute(ctx context.Context, request sttchain.Request) (*sttchain.Result, error) {
	return e.provider.Transcribe(ctx, request)
}

func batchSession(meter *MeteredProvider, format string) Session {
	if format == "" {
		format = "pcm_s16le"
	}
	return func(ctx context.Context, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent) error {
		var audio []byte
		for chunk := range chunks {
			audio = append(audio, chunk.Audio...)
		}
		result, err := meter.Transcribe(ctx, sttchain.Request{Audio: audio, Format: format})
		transcript := ""
		if result != nil {
			transcript = result.Text
		}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: transcript}}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: transcript}}
		close(events)
		return err
	}
}
