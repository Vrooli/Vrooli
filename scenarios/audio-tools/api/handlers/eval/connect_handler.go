//go:build audio_lab

package eval

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

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

// MetricsOnlyReportToProto retains the stable EvalReport wire shape while
// applying a fail-closed field allowlist. Newly added report fields are
// omitted until explicitly reviewed for transcript-free persistence.
func MetricsOnlyReportToProto(r inteval.EvalReport) *evalv1.EvalReport {
	out := proto.Clone(ReportToProto(r)).(*evalv1.EvalReport)
	filterReportFields(out.ProtoReflect(), metricsOnlyReportFields)
	out.RetentionMode = "metrics_only"
	return out
}

type reportFieldPolicy map[protoreflect.Name]reportFieldPolicy

var metricsOnlyReportFields = reportFieldPolicy{
	"quality_measured": {}, "latency_measured": {}, "retention_mode": {},
	"per_strategy": {"strategy": {}, "wer": {}, "substitutions": {}, "insertions": {}, "deletions": {}, "ref_words": {}, "whisper_calls": {}, "whisper_audio_seconds": {}, "rtf": {}, "finalization_latency_p50_ms": {}, "finalization_latency_p95_ms": {}, "per_clip": {
		"wer": {}, "whisper_calls": {}, "whisper_audio_seconds": {}, "rtf": {}, "segment_count": {}, "partial_revisions": {}, "finalization_latency_p50_ms": {}, "finalization_latency_p95_ms": {}, "substitutions": {}, "insertions": {}, "deletions": {}, "ref_words": {}, "hyp_words": {}, "time_to_first_commit_ms": {}, "commit_count": {}, "speaker_rejection_count": {}, "audio_duration_ms": {},
		"timestamp_drift_skip_reason": {},
		"timestamp_drift":             {"reference_words": {}, "matched_words": {}, "coverage": {}, "start_abs_p50_ms": {}, "start_abs_p95_ms": {}, "end_abs_p50_ms": {}, "end_abs_p95_ms": {}, "start_bias_ms": {}, "end_bias_ms": {}, "max_start_abs_ms": {}, "max_end_abs_ms": {}, "drift_rate_observed": {}, "start_drift_ms_per_hour": {}, "end_drift_ms_per_hour": {}},
		"timestamp_reference":         {"status": {}, "source_reference": {}, "alignment_tool": {}, "alignment_model": {}, "reviewer": {}, "review_method": {}, "reviewed_at": {}},
	}, "engine_id": {}, "model_id": {}, "policy_profile": {}, "replay_lane": {}, "fault_profile": {}},
}

func filterReportFields(message protoreflect.Message, allowed reportFieldPolicy) {
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		childPolicy, keep := allowed[field.Name()]
		if !keep {
			message.Clear(field)
			return true
		}
		if len(childPolicy) == 0 || field.Message() == nil {
			return true
		}
		if field.IsList() {
			list := value.List()
			for i := 0; i < list.Len(); i++ {
				filterReportFields(list.Get(i).Message(), childPolicy)
			}
		} else if field.IsMap() {
			// No map-valued fields are admitted in the allowlist. This branch is
			// defensive in case one is added later without a dedicated policy.
			message.Clear(field)
		} else {
			filterReportFields(value.Message(), childPolicy)
		}
		return true
	})
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
			TimestampDrift:           timestampDriftToProto(c.TimestampDrift),
			TimestampDriftSkipReason: c.TimestampDriftSkipReason,
			TimestampReference:       timingReferenceToProto(c.TimestampReference),
		})
	}
	return sr
}

func timestampDriftToProto(in *inteval.TimestampDriftResult) *evalv1.TimestampDrift {
	if in == nil {
		return nil
	}
	return &evalv1.TimestampDrift{
		ReferenceWords:      protoint.FromInt64(int64(in.ReferenceWords)),
		MatchedWords:        protoint.FromInt64(int64(in.MatchedWords)),
		Coverage:            in.Coverage,
		StartAbsP50Ms:       in.StartAbsP50Ms,
		StartAbsP95Ms:       in.StartAbsP95Ms,
		EndAbsP50Ms:         in.EndAbsP50Ms,
		EndAbsP95Ms:         in.EndAbsP95Ms,
		StartBiasMs:         in.StartBiasMs,
		EndBiasMs:           in.EndBiasMs,
		MaxStartAbsMs:       in.MaxStartAbsMs,
		MaxEndAbsMs:         in.MaxEndAbsMs,
		DriftRateObserved:   in.DriftRateObserved,
		StartDriftMsPerHour: in.StartDriftMsPerHour,
		EndDriftMsPerHour:   in.EndDriftMsPerHour,
	}
}

func timingReferenceToProto(in *inteval.TimingReferenceProvenance) *evalv1.TimingReferenceProvenance {
	if in == nil {
		return nil
	}
	return &evalv1.TimingReferenceProvenance{
		Status: in.Status, SourceReference: in.SourceReference,
		AlignmentTool: in.AlignmentTool, AlignmentModel: in.AlignmentModel,
		Reviewer: in.Reviewer, ReviewMethod: in.ReviewMethod, ReviewedAt: in.ReviewedAt,
	}
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
