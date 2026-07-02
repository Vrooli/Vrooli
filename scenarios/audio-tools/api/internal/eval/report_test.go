package eval

import (
	"testing"

	"audio-tools/internal/ai/sttchain"

	"github.com/stretchr/testify/require"
)

func TestExplainReport_RecommendsVADForSampleShape(t *testing.T) {
	report := EvalReport{
		QualityMeasured: true,
		LatencyMeasured: true,
		PerStrategy: []StrategyReport{
			{
				Strategy: "batch", Label: "batch", WER: 0.022,
				EditCounts: EditCounts{Substitutions: 1}, RefWords: 45,
				WhisperCalls: 7, WhisperAudioSeconds: 59.7, RTF: 0.05,
				FinalizationLatencyP50Ms: 842, FinalizationLatencyP95Ms: 2169,
				PerClip: []ClipResult{{ClipID: "c1"}},
			},
			{
				Strategy: sttchain.StrategyVADSegment, Label: "vad_segment", WER: 0.022,
				EditCounts: EditCounts{Substitutions: 1}, RefWords: 45,
				WhisperCalls: 7, WhisperAudioSeconds: 58.5, RTF: 0.05,
				FinalizationLatencyP50Ms: 570, FinalizationLatencyP95Ms: 1239,
				PerClip: []ClipResult{{ClipID: "c1"}},
			},
			{
				Strategy: sttchain.StrategyOverlapAgree, Label: "overlap_agree", WER: 0.033,
				EditCounts: EditCounts{Substitutions: 2}, RefWords: 60,
				WhisperCalls: 15, WhisperAudioSeconds: 116.3, RTF: 0.10,
				FinalizationLatencyP50Ms: 1106, FinalizationLatencyP95Ms: 1525, PartialRevisions: 7,
				PerClip: []ClipResult{{ClipID: "c1"}},
			},
		},
	}

	got := explainReport(report)
	require.Equal(t, "vad_segment", got.Summary.WinnerStrategy)
	require.Contains(t, got.Summary.Recommendation, "vad_segment")
	require.Equal(t, "winner", got.PerStrategy[1].Verdict)
	require.Equal(t, "loser", got.PerStrategy[2].Verdict)
	require.NotEmpty(t, got.PerStrategy[2].Warnings)
	require.NotEmpty(t, got.NormalizationPolicy.WERPolicy)
	require.NotEmpty(t, got.Warnings, "single-clip corpus should carry adequacy warning")
}

func TestExplainReport_PreservesAggregateRankingWithoutScalingData(t *testing.T) {
	report := EvalReport{
		QualityMeasured: true,
		LatencyMeasured: true,
		PerStrategy: []StrategyReport{
			{
				Strategy: "batch", Label: "batch", WER: 0.02,
				WhisperCalls: 1, WhisperAudioSeconds: 30, RTF: 0.1,
				FinalizationLatencyP95Ms: 500,
				PerClip:                  []ClipResult{{ClipID: "c1"}},
			},
			{
				Strategy: "vad_segment", Label: "vad_segment", WER: 0.02,
				WhisperCalls: 1, WhisperAudioSeconds: 30, RTF: 0.1,
				FinalizationLatencyP95Ms: 200,
				PerClip:                  []ClipResult{{ClipID: "c1"}},
			},
		},
	}

	got := explainReport(report)

	require.Equal(t, "vad_segment", got.Summary.WinnerStrategy)
	require.Equal(t, "winner", got.PerStrategy[1].Verdict)
}

func TestExplainReport_PrefersLongFormScalingForNearTieWER(t *testing.T) {
	report := EvalReport{
		QualityMeasured: true,
		LatencyMeasured: true,
		PerStrategy: []StrategyReport{
			{
				Strategy: "batch", Label: "batch", WER: 0.0200,
				WhisperCalls: 1, WhisperAudioSeconds: 300, RTF: 0.1,
				FinalizationLatencyP95Ms: 900,
				Scaling:                  reportScaling("linear", "linear"),
				PerClip:                  []ClipResult{{ClipID: "c1"}},
			},
			{
				Strategy: "overlap_agree", Label: "overlap_agree", WER: 0.0203,
				WhisperCalls: 1, WhisperAudioSeconds: 300, RTF: 0.1,
				FinalizationLatencyP95Ms: 100,
				Scaling:                  reportScaling("superlinear", "linear"),
				PerClip:                  []ClipResult{{ClipID: "c1"}},
			},
		},
	}

	got := explainReport(report)

	require.Equal(t, "batch", got.Summary.WinnerStrategy)
	require.Equal(t, "winner", got.PerStrategy[0].Verdict)
	require.Equal(t, "tradeoff", got.PerStrategy[1].Verdict)
	require.Contains(t, got.PerStrategy[1].Reasons, "Shows superlinear finalization-latency growth over the measured duration sweep.")
	require.Contains(t, got.Summary.Reasons[2], "Short-form aggregate tie-breaks favored overlap_agree")
}

func TestExplainReport_DoesNotTrustDuplicateDurationScalingEvidence(t *testing.T) {
	report := EvalReport{
		QualityMeasured: true,
		LatencyMeasured: true,
		PerStrategy: []StrategyReport{
			{
				Strategy: "batch", Label: "batch", WER: 0.0200,
				WhisperCalls: 1, WhisperAudioSeconds: 300, RTF: 0.1,
				FinalizationLatencyP95Ms: 900,
				Scaling:                  reportScaling("linear", "linear"),
				PerClip:                  []ClipResult{{ClipID: "c1"}},
			},
			{
				Strategy: "overlap_agree", Label: "overlap_agree", WER: 0.0203,
				WhisperCalls: 1, WhisperAudioSeconds: 300, RTF: 0.1,
				FinalizationLatencyP95Ms: 100,
				Scaling: ScalingAnalysis{
					Points: []ScalingPoint{
						{ClipID: "same-a", RealizedDurationMs: 30_000},
						{ClipID: "same-b", RealizedDurationMs: 30_000},
						{ClipID: "same-c", RealizedDurationMs: 30_000},
					},
					LatencyClassification: "linear",
					ComputeClassification: "linear",
				},
				PerClip: []ClipResult{{ClipID: "c1"}},
			},
		},
	}

	got := explainReport(report)

	require.Equal(t, "batch", got.Summary.WinnerStrategy)
	require.Equal(t, "tradeoff", got.PerStrategy[1].Verdict)
	require.Contains(t, got.PerStrategy[1].Reasons, "Scaling evidence is inconclusive for long-form dictation.")
}

func reportScaling(latency, compute string) ScalingAnalysis {
	return ScalingAnalysis{
		Points: []ScalingPoint{
			{ClipID: "30s", RealizedDurationMs: 30_000},
			{ClipID: "60s", RealizedDurationMs: 60_000},
			{ClipID: "120s", RealizedDurationMs: 120_000},
		},
		LatencyClassification: latency,
		ComputeClassification: compute,
		Confidence:            "medium",
	}
}
