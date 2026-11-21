package eval

import (
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/protoint"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

func promotionVerdicts(report inteval.EvalReport) []inteval.PromotionVerdict {
	return inteval.PromotionVerdicts(report)
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
		LatencyHonesty:    "Wall-clock finalization and commit timing are intra-experiment-only; compare cross-experiment runs by WER, calls, audio-seconds, RTF, safety gates, and pinned machine metadata.",
		PromotionVerdicts: promotionVerdictsToProto(r.PromotionVerdicts),
	}
	for _, s := range r.PerStrategy {
		out.PerStrategy = append(out.PerStrategy, strategyReportToProto(s))
	}
	return out
}

func promotionVerdictsToProto(in []inteval.PromotionVerdict) []*evalv1.PromotionVerdict {
	out := make([]*evalv1.PromotionVerdict, 0, len(in))
	for _, verdict := range in {
		out = append(out, &evalv1.PromotionVerdict{
			EngineId:      verdict.EngineID,
			ModelId:       verdict.ModelID,
			Strategy:      verdict.Strategy,
			PolicyProfile: verdict.PolicyProfile,
			Stable:        verdict.Stable,
			Reasons:       append([]string(nil), verdict.Reasons...),
		})
	}
	return out
}

func strategyReportToProto(s inteval.StrategyReport) *evalv1.StrategyReport {
	sr := &evalv1.StrategyReport{
		Strategy:                 string(s.Strategy),
		Label:                    s.Label,
		Wer:                      s.WER,
		Substitutions:            protoint.FromInt64(int64(s.EditCounts.Substitutions)),
		Insertions:               protoint.FromInt64(int64(s.EditCounts.Insertions)),
		Deletions:                protoint.FromInt64(int64(s.EditCounts.Deletions)),
		RefWords:                 protoint.FromInt64(int64(s.RefWords)),
		WhisperCalls:             protoint.FromInt64(int64(s.WhisperCalls)),
		WhisperAudioSeconds:      s.WhisperAudioSeconds,
		Rtf:                      s.RTF,
		FinalizationLatencyP50Ms: s.FinalizationLatencyP50Ms,
		FinalizationLatencyP95Ms: s.FinalizationLatencyP95Ms,
		PartialRevisions:         protoint.FromInt64(int64(s.PartialRevisions)),
		CommitCount:              protoint.FromInt64(int64(s.CommitCount)),
		SpeakerRejectionCount:    protoint.FromInt64(int64(s.SpeakerRejectionCount)),
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
		EngineId:                 s.EngineID,
		ModelId:                  s.ModelID,
		PolicyProfile:            s.PolicyProfile,
		ReplayLane:               s.ReplayLane,
		FaultProfile:             s.FaultProfile,
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
			WhisperCalls:             protoint.FromInt64(int64(c.WhisperCalls)),
			WhisperAudioSeconds:      c.WhisperAudioSeconds,
			Rtf:                      c.RTF,
			SegmentCount:             protoint.FromInt64(int64(c.SegmentCount)),
			PartialRevisions:         protoint.FromInt64(int64(c.PartialRevisions)),
			FinalizationLatencyP50Ms: c.FinalizationLatencyP50Ms(),
			FinalizationLatencyP95Ms: c.FinalizationLatencyP95Ms(),
			Error:                    errStr,
			Substitutions:            protoint.FromInt64(int64(c.WER.Substitutions)),
			Insertions:               protoint.FromInt64(int64(c.WER.Insertions)),
			Deletions:                protoint.FromInt64(int64(c.WER.Deletions)),
			RefWords:                 protoint.FromInt64(int64(c.WER.RefWords)),
			HypWords:                 protoint.FromInt64(int64(c.WER.HypWords)),
			NormalizedReference:      c.NormalizedReference,
			NormalizedHypothesis:     c.NormalizedHypothesis,
			EditOperations:           editOperationsToProto(c.EditOperations),
			CommitTimeline:           commitTimelineToProto(c.CommitTimeline),
			TimeToFirstCommitMs:      c.TimeToFirstCommitMs,
			CommitCount:              protoint.FromInt64(int64(c.CommitCount)),
			SpeakerRejectionCount:    protoint.FromInt64(int64(c.SpeakerRejectionCount)),
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
		in.ComputeFit == (inteval.ScalingModelFit{}) &&
		len(in.MetricFits) == 0 {
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
		MetricFits:            make([]*evalv1.ScalingModelFit, 0, len(in.MetricFits)),
	}
	for _, fit := range in.MetricFits {
		out.MetricFits = append(out.MetricFits, scalingModelFitToProto(fit))
	}
	for _, point := range in.Points {
		out.Points = append(out.Points, &evalv1.ScalingPoint{
			ClipId:                         point.ClipID,
			TargetDurationMs:               point.TargetDurationMs,
			RealizedDurationMs:             point.RealizedDurationMs,
			Wer:                            point.WER,
			FinalizationLatencyP50Ms:       point.FinalizationLatencyP50Ms,
			FinalizationLatencyP95Ms:       point.FinalizationLatencyP95Ms,
			FinalizationLatencySampleCount: protoint.FromInt64(int64(point.FinalizationLatencySampleCount)),
			TimeToFirstCommitMs:            point.TimeToFirstCommitMs,
			CommitCount:                    protoint.FromInt64(int64(point.CommitCount)),
			PartialRevisions:               protoint.FromInt64(int64(point.PartialRevisions)),
			MaxDroppedSpanWords:            protoint.FromInt64(int64(point.MaxDroppedSpanWords)),
			WhisperCalls:                   protoint.FromInt64(int64(point.WhisperCalls)),
			WhisperAudioSeconds:            point.WhisperAudioSeconds,
			ProviderLatencyMs:              point.ProviderLatencyMs,
			Rtf:                            point.RTF,
		})
	}
	return out
}

func scalingModelFitToProto(in inteval.ScalingModelFit) *evalv1.ScalingModelFit {
	return &evalv1.ScalingModelFit{
		Metric:           in.Metric,
		Model:            in.Model,
		SlopePerSecond:   in.SlopePerSecond,
		Intercept:        in.Intercept,
		RSquared:         in.RSquared,
		SampleCount:      protoint.FromInt64(int64(in.SampleCount)),
		Reason:           in.Reason,
		Exponent:         in.Exponent,
		ExponentRSquared: in.ExponentR2,
		Unit:             in.Unit,
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
			ReferenceIndex:  protoint.FromInt64(int64(op.ReferenceIndex)),
			HypothesisIndex: protoint.FromInt64(int64(op.HypothesisIndex)),
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
		MaxDroppedSpanWords:       protoint.FromInt64(int64(in.MaxDroppedSpanWords)),
		DroppedSpanThresholdWords: protoint.FromInt64(int64(in.DroppedSpanThresholdWords)),
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
		IngressLostWords:   protoint.FromInt64(int64(in.IngressLostWords)),
		StrategyLostWords:  protoint.FromInt64(int64(in.StrategyLostWords)),
		EgressLostWords:    protoint.FromInt64(int64(in.EgressLostWords)),
		EgressRejectEvents: protoint.FromInt64(int64(in.EgressRejectEvents)),
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
			ClipCount:                protoint.FromInt64(int64(curve.ClipCount)),
			Wer:                      curve.WER,
			FinalizationLatencyP95Ms: curve.FinalizationLatencyP95Ms,
			MeanTimeToFirstCommitMs:  curve.MeanTimeToFirstCommitMs,
			MaxDroppedSpanWords:      protoint.FromInt64(int64(curve.MaxDroppedSpanWords)),
		})
	}
	return out
}
