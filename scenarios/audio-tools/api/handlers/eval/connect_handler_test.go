package eval

import (
	"testing"

	inteval "audio-tools/internal/eval"

	"github.com/stretchr/testify/require"
)

func TestReportToProto_MapsScalingAnalysis(t *testing.T) {
	report := inteval.EvalReport{
		QualityMeasured: true,
		PerStrategy: []inteval.StrategyReport{{
			Label: "batch",
			Scaling: inteval.ScalingAnalysis{
				LatencyClassification: "linear",
				ComputeClassification: "superlinear",
				Confidence:            "medium",
				Reasons:               []string{"duration sweep has enough points"},
				Warnings: []inteval.ReportWarning{{
					Code:     "superlinear_compute_growth",
					Severity: "warning",
					Message:  "Compute grows faster than duration.",
				}},
				LatencyFit: inteval.ScalingModelFit{
					Metric:         "finalization_latency_p95_ms",
					Model:          "linear",
					SlopePerSecond: 12,
					Intercept:      40,
					RSquared:       0.91,
					SampleCount:    3,
					Reason:         "linear fit was strongest",
				},
				Points: []inteval.ScalingPoint{{
					ClipID:                         "long-form-60s",
					RealizedDurationMs:             60_000,
					WER:                            0.02,
					FinalizationLatencyP95Ms:       500,
					FinalizationLatencySampleCount: 2,
					WhisperCalls:                   6,
					WhisperAudioSeconds:            90,
					ProviderLatencyMs:              1_200,
					RTF:                            0.02,
				}},
			},
		}},
	}

	got := ReportToProto(report)
	require.Len(t, got.GetPerStrategy(), 1)
	scaling := got.GetPerStrategy()[0].GetScaling()
	require.NotNil(t, scaling)
	require.Equal(t, "linear", scaling.GetLatencyClassification())
	require.Equal(t, "superlinear", scaling.GetComputeClassification())
	require.Equal(t, "medium", scaling.GetConfidence())
	require.Equal(t, "duration sweep has enough points", scaling.GetReasons()[0])
	require.Equal(t, "superlinear_compute_growth", scaling.GetWarnings()[0].GetCode())
	require.Equal(t, "finalization_latency_p95_ms", scaling.GetLatencyFit().GetMetric())
	require.InDelta(t, 12, scaling.GetLatencyFit().GetSlopePerSecond(), 1e-9)
	require.Len(t, scaling.GetPoints(), 1)
	point := scaling.GetPoints()[0]
	require.Equal(t, "long-form-60s", point.GetClipId())
	require.EqualValues(t, 60_000, point.GetRealizedDurationMs())
	require.InDelta(t, 1_200, point.GetProviderLatencyMs(), 1e-9)
}
