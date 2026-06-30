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
