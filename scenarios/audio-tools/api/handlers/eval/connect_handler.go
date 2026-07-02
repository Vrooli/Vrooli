package eval

import (
	"context"
	"errors"
	"fmt"

	inteval "audio-tools/internal/eval"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

type reportRunner struct{ deps Deps }

var (
	errEvalNotConfigured  = errors.New("eval service not configured (no corpus/database)")
	errEvalNoProvider     = errors.New("eval requires a transcription provider (Whisper) — none configured")
	errEvalEmptyCorpus    = errors.New("corpus is empty — record clips before running an eval")
	errEvalInvalidRequest = errors.New("invalid eval request")
)

// RunReport runs the existing synchronous eval harness. It is exported so the
// experiment worker can reuse the same compute path instead of cloning it.
func RunReport(ctx context.Context, deps Deps, clipIDs []string, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32) (inteval.EvalReport, error) {
	return RunReportWithOptions(ctx, deps, clipIDs, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{})
}

func RunReportWithOptions(ctx context.Context, deps Deps, clipIDs []string, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	h := &reportRunner{deps: deps}
	if h.deps.Corpus == nil {
		return inteval.EvalReport{}, errEvalNotConfigured
	}
	if h.deps.NewProvider == nil {
		return inteval.EvalReport{}, errEvalNoProvider
	}

	clips, err := h.loadClips(ctx, clipIDs)
	if err != nil {
		return inteval.EvalReport{}, err
	}
	if len(clips) == 0 {
		return inteval.EvalReport{}, errEvalEmptyCorpus
	}
	return RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, opts)
}

// RunReportForClips runs the eval harness over already-materialized clips.
// Experiment recipe builders use this to evaluate synthetic long-form inputs
// without cloning strategy/session construction.
func RunReportForClips(ctx context.Context, deps Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32) (inteval.EvalReport, error) {
	return RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{})
}

func RunReportForClipsWithOptions(ctx context.Context, deps Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts inteval.EvalOptions) (inteval.EvalReport, error) {
	h := &reportRunner{deps: deps}
	if h.deps.NewProvider == nil {
		return inteval.EvalReport{}, errEvalNoProvider
	}
	if len(clips) == 0 {
		return inteval.EvalReport{}, errEvalEmptyCorpus
	}
	specs, err := h.buildSpecs(strategies)
	if err != nil {
		return inteval.EvalReport{}, fmt.Errorf("%w: %v", errEvalInvalidRequest, err)
	}
	opts.ChunkMs = int(chunkMs)
	opts.QualityPass = true
	opts.RealtimeRepeats = int(realtimeRepeats)
	return inteval.RunReport(ctx, clips, specs, opts), nil
}

func ReportToProto(r inteval.EvalReport) *evalv1.EvalReport {
	out := &evalv1.EvalReport{
		QualityMeasured: r.QualityMeasured,
		LatencyMeasured: r.LatencyMeasured,
		PerStrategy:     make([]*evalv1.StrategyReport, 0, len(r.PerStrategy)),
		Summary: &evalv1.EvalReportSummary{
			WinnerStrategy:  r.Summary.WinnerStrategy,
			WinnerLabel:     r.Summary.WinnerLabel,
			Recommendation:  r.Summary.Recommendation,
			Confidence:      r.Summary.Confidence,
			Reasons:         append([]string(nil), r.Summary.Reasons...),
			ConfidenceNotes: append([]string(nil), r.Summary.ConfidenceNotes...),
		},
		Warnings: reportWarningsToProto(r.Warnings),
		NormalizationPolicy: &evalv1.NormalizationPolicy{
			WerPolicy:              r.NormalizationPolicy.WERPolicy,
			OverlapAgreementPolicy: r.NormalizationPolicy.OverlapAgreementPolicy,
		},
		LatencyHonesty: "Wall-clock finalization and commit timing are intra-experiment-only; compare cross-experiment runs by WER, calls, audio-seconds, RTF, safety gates, and pinned machine metadata.",
	}
	for _, s := range r.PerStrategy {
		out.PerStrategy = append(out.PerStrategy, strategyReportToProto(s))
	}
	return out
}

func strategyReportToProto(s inteval.StrategyReport) *evalv1.StrategyReport {
	sr := &evalv1.StrategyReport{
		Strategy:                 string(s.Strategy),
		Label:                    s.Label,
		Wer:                      s.WER,
		Substitutions:            int32(s.EditCounts.Substitutions),
		Insertions:               int32(s.EditCounts.Insertions),
		Deletions:                int32(s.EditCounts.Deletions),
		RefWords:                 int32(s.RefWords),
		WhisperCalls:             int32(s.WhisperCalls),
		WhisperAudioSeconds:      s.WhisperAudioSeconds,
		Rtf:                      s.RTF,
		FinalizationLatencyP50Ms: s.FinalizationLatencyP50Ms,
		FinalizationLatencyP95Ms: s.FinalizationLatencyP95Ms,
		PartialRevisions:         int32(s.PartialRevisions),
		CommitCount:              int32(s.CommitCount),
		SpeakerRejectionCount:    int32(s.SpeakerRejectionCount),
		WerDeltaVsWinner:         s.WERDeltaVsWinner,
		P95DeltaMsVsWinner:       s.P95DeltaMsVsWinner,
		CallMultiplierVsWinner:   s.CallMultiplierVsWinner,
		Verdict:                  s.Verdict,
		Reasons:                  append([]string(nil), s.Reasons...),
		Warnings:                 reportWarningsToProto(s.Warnings),
		Safety:                   safetyGateToProto(s.Safety),
		StageAttribution:         stageAttributionToProto(s.StageAttribution),
		LengthCurves:             lengthCurvesToProto(s.LengthCurves),
		Scaling:                  scalingAnalysisToProto(s.Scaling),
		PerClip:                  make([]*evalv1.ClipReport, 0, len(s.PerClip)),
	}
	for _, c := range s.PerClip {
		errStr := ""
		if c.Err != nil {
			errStr = c.Err.Error()
		}
		sr.PerClip = append(sr.PerClip, &evalv1.ClipReport{
			ClipId:                   c.ClipID,
			Reference:                c.Reference,
			Hypothesis:               c.Hypothesis,
			Wer:                      c.WER.Rate(),
			WhisperCalls:             int32(c.WhisperCalls),
			WhisperAudioSeconds:      c.WhisperAudioSeconds,
			Rtf:                      c.RTF,
			SegmentCount:             int32(c.SegmentCount),
			PartialRevisions:         int32(c.PartialRevisions),
			FinalizationLatencyP50Ms: c.FinalizationLatencyP50Ms(),
			FinalizationLatencyP95Ms: c.FinalizationLatencyP95Ms(),
			Error:                    errStr,
			Substitutions:            int32(c.WER.Substitutions),
			Insertions:               int32(c.WER.Insertions),
			Deletions:                int32(c.WER.Deletions),
			RefWords:                 int32(c.WER.RefWords),
			HypWords:                 int32(c.WER.HypWords),
			NormalizedReference:      c.NormalizedReference,
			NormalizedHypothesis:     c.NormalizedHypothesis,
			EditOperations:           editOperationsToProto(c.EditOperations),
			CommitTimeline:           commitTimelineToProto(c.CommitTimeline),
			TimeToFirstCommitMs:      c.TimeToFirstCommitMs,
			CommitCount:              int32(c.CommitCount),
			SpeakerRejectionCount:    int32(c.SpeakerRejectionCount),
			AudioDurationMs:          c.AudioDurationMs,
			Safety:                   safetyGateToProto(c.Safety),
		})
	}
	return sr
}

func scalingAnalysisToProto(in inteval.ScalingAnalysis) *evalv1.ScalingAnalysis {
	if len(in.Points) == 0 &&
		in.LatencyClassification == "" &&
		in.ComputeClassification == "" &&
		in.Confidence == "" &&
		len(in.Reasons) == 0 &&
		len(in.Warnings) == 0 &&
		in.LatencyFit == (inteval.ScalingModelFit{}) &&
		in.ComputeFit == (inteval.ScalingModelFit{}) {
		return nil
	}
	out := &evalv1.ScalingAnalysis{
		Points:                make([]*evalv1.ScalingPoint, 0, len(in.Points)),
		LatencyClassification: in.LatencyClassification,
		ComputeClassification: in.ComputeClassification,
		Confidence:            in.Confidence,
		Reasons:               append([]string(nil), in.Reasons...),
		Warnings:              reportWarningsToProto(in.Warnings),
		LatencyFit:            scalingModelFitToProto(in.LatencyFit),
		ComputeFit:            scalingModelFitToProto(in.ComputeFit),
	}
	for _, point := range in.Points {
		out.Points = append(out.Points, &evalv1.ScalingPoint{
			ClipId:                         point.ClipID,
			TargetDurationMs:               point.TargetDurationMs,
			RealizedDurationMs:             point.RealizedDurationMs,
			Wer:                            point.WER,
			FinalizationLatencyP50Ms:       point.FinalizationLatencyP50Ms,
			FinalizationLatencyP95Ms:       point.FinalizationLatencyP95Ms,
			FinalizationLatencySampleCount: int32(point.FinalizationLatencySampleCount),
			TimeToFirstCommitMs:            point.TimeToFirstCommitMs,
			CommitCount:                    int32(point.CommitCount),
			PartialRevisions:               int32(point.PartialRevisions),
			MaxDroppedSpanWords:            int32(point.MaxDroppedSpanWords),
			WhisperCalls:                   int32(point.WhisperCalls),
			WhisperAudioSeconds:            point.WhisperAudioSeconds,
			ProviderLatencyMs:              point.ProviderLatencyMs,
			Rtf:                            point.RTF,
		})
	}
	return out
}

func scalingModelFitToProto(in inteval.ScalingModelFit) *evalv1.ScalingModelFit {
	return &evalv1.ScalingModelFit{
		Metric:         in.Metric,
		Model:          in.Model,
		SlopePerSecond: in.SlopePerSecond,
		Intercept:      in.Intercept,
		RSquared:       in.RSquared,
		SampleCount:    int32(in.SampleCount),
		Reason:         in.Reason,
	}
}

func reportWarningsToProto(in []inteval.ReportWarning) []*evalv1.ReportWarning {
	out := make([]*evalv1.ReportWarning, 0, len(in))
	for _, w := range in {
		out = append(out, &evalv1.ReportWarning{
			Code:     w.Code,
			Message:  w.Message,
			Severity: w.Severity,
		})
	}
	return out
}

func editOperationsToProto(in []inteval.EditOperation) []*evalv1.EditOperation {
	out := make([]*evalv1.EditOperation, 0, len(in))
	for _, op := range in {
		out = append(out, &evalv1.EditOperation{
			Kind:            op.Kind,
			ReferenceToken:  op.ReferenceToken,
			HypothesisToken: op.HypothesisToken,
			ReferenceIndex:  int32(op.ReferenceIndex),
			HypothesisIndex: int32(op.HypothesisIndex),
		})
	}
	return out
}

func commitTimelineToProto(in []inteval.CommitState) []*evalv1.CommitState {
	out := make([]*evalv1.CommitState, 0, len(in))
	for _, state := range in {
		out = append(out, &evalv1.CommitState{
			Text:       state.Text,
			AtMs:       state.AtMs,
			AudioEndMs: state.AudioEndMs,
		})
	}
	return out
}

func safetyGateToProto(in inteval.SafetyGateReport) *evalv1.SafetyGateReport {
	out := &evalv1.SafetyGateReport{
		Passed:                    in.Passed,
		RetractionFree:            in.RetractionFree,
		DroppedSpanFree:           in.DroppedSpanFree,
		MaxDroppedSpanWords:       int32(in.MaxDroppedSpanWords),
		DroppedSpanThresholdWords: int32(in.DroppedSpanThresholdWords),
		Reasons:                   append([]string(nil), in.Reasons...),
	}
	for _, ev := range in.RetractionEvents {
		out.RetractionEvents = append(out.RetractionEvents, &evalv1.RetractionEvent{
			PreviousText: ev.PreviousText,
			CurrentText:  ev.CurrentText,
			AtMs:         ev.AtMs,
		})
	}
	return out
}

func stageAttributionToProto(in inteval.StageAttribution) *evalv1.StageAttribution {
	return &evalv1.StageAttribution{
		IngressLostWords:   int32(in.IngressLostWords),
		StrategyLostWords:  int32(in.StrategyLostWords),
		EgressLostWords:    int32(in.EgressLostWords),
		EgressRejectEvents: int32(in.EgressRejectEvents),
		Notes:              append([]string(nil), in.Notes...),
	}
}

func lengthCurvesToProto(in []inteval.LengthBucketCurve) []*evalv1.LengthBucketCurve {
	out := make([]*evalv1.LengthBucketCurve, 0, len(in))
	for _, curve := range in {
		out = append(out, &evalv1.LengthBucketCurve{
			Bucket:                   curve.Bucket,
			MinDurationMs:            curve.MinDurationMs,
			MaxDurationMs:            curve.MaxDurationMs,
			ClipCount:                int32(curve.ClipCount),
			Wer:                      curve.WER,
			FinalizationLatencyP95Ms: curve.FinalizationLatencyP95Ms,
			MeanTimeToFirstCommitMs:  curve.MeanTimeToFirstCommitMs,
			MaxDroppedSpanWords:      int32(curve.MaxDroppedSpanWords),
		})
	}
	return out
}
