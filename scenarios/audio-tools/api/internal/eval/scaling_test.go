package eval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildScalingAnalysis_PopulatesPerDurationPoints(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		{
			ClipID:              "long-form-30s",
			AudioDurationMs:     30_000,
			WER:                 WERResult{EditCounts: EditCounts{Substitutions: 1}, RefWords: 20},
			LatencySamplesMs:    []float64{120, 180, 240},
			TimeToFirstCommitMs: 350,
			CommitCount:         2,
			PartialRevisions:    3,
			Safety:              SafetyGateReport{MaxDroppedSpanWords: 1},
			WhisperCalls:        4,
			WhisperAudioSeconds: 45,
			ProviderLatencyMs:   900,
			RTF:                 0.03,
		},
	})

	require.Equal(t, "inconclusive", got.LatencyClassification)
	require.Equal(t, "inconclusive", got.ComputeClassification)
	require.Equal(t, "low", got.Confidence)
	require.Len(t, got.Points, 1)
	point := got.Points[0]
	require.Equal(t, "long-form-30s", point.ClipID)
	require.EqualValues(t, 30_000, point.RealizedDurationMs)
	require.InDelta(t, 0.05, point.WER, 1e-9)
	require.InDelta(t, 180, point.FinalizationLatencyP50Ms, 1e-9)
	require.InDelta(t, 234, point.FinalizationLatencyP95Ms, 1e-9)
	require.Equal(t, 3, point.FinalizationLatencySampleCount)
	require.InDelta(t, 900, point.ProviderLatencyMs, 1e-9)
	require.NotEmpty(t, got.Warnings)
	require.Equal(t, "insufficient_scaling_points", got.Warnings[0].Code)
}

func TestBuildScalingAnalysis_SkipsInvalidDurations(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{{ClipID: "bad"}})

	require.Empty(t, got.Points)
	require.Equal(t, "inconclusive", got.LatencyClassification)
	require.Equal(t, "insufficient_scaling_points", got.Warnings[0].Code)
}

func TestBuildScalingAnalysis_RequiresDistinctDurationsForClassification(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("same-a", 30_000, 300, 600),
		scalingClip("same-b", 30_000, 320, 620),
		scalingClip("same-c", 30_000, 340, 640),
	})

	require.Equal(t, "inconclusive", got.LatencyClassification)
	require.Equal(t, "inconclusive", got.ComputeClassification)
	require.Equal(t, "low", got.Confidence)
	require.Equal(t, "insufficient_scaling_points", got.Warnings[0].Code)
	require.Contains(t, got.Reasons[0], "Only 1 distinct positive duration")
}

func TestClassifyScaling_RequiresDistinctMetricDurations(t *testing.T) {
	fit, classification, reason := classifyScaling([]ScalingPoint{
		{ClipID: "same-a", RealizedDurationMs: 30_000, FinalizationLatencyP95Ms: 300, FinalizationLatencySampleCount: 1},
		{ClipID: "same-b", RealizedDurationMs: 30_000, FinalizationLatencyP95Ms: 320, FinalizationLatencySampleCount: 1},
		{ClipID: "same-c", RealizedDurationMs: 30_000, FinalizationLatencyP95Ms: 340, FinalizationLatencySampleCount: 1},
	}, testScalingMetric(metricFinalizationP95))

	require.Equal(t, "inconclusive", classification)
	require.Equal(t, "none", fit.Model)
	require.Contains(t, fit.Reason, "insufficient distinct positive durations")
	require.Contains(t, reason, "fewer than 3 distinct positive durations")
}

func TestBuildScalingAnalysis_ClassifiesLinearScaling(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("30s", 30_000, 300, 600),
		scalingClip("60s", 60_000, 600, 1_200),
		scalingClip("120s", 120_000, 1_200, 2_400),
	})

	require.Equal(t, "linear", got.LatencyClassification)
	require.Equal(t, "linear", got.ComputeClassification)
	require.Equal(t, "medium", got.Confidence)
	require.Equal(t, "linear", got.LatencyFit.Model)
	require.Greater(t, got.LatencyFit.RSquared, 0.99)
	require.InDelta(t, 1, got.LatencyFit.Exponent, 0.05)
	require.Empty(t, got.Warnings)
}

func TestBuildScalingAnalysis_ClassifiesSuperlinearScaling(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("30s", 30_000, 90, 180),
		scalingClip("60s", 60_000, 360, 720),
		scalingClip("120s", 120_000, 1_440, 2_880),
		scalingClip("240s", 240_000, 5_760, 11_520),
	})

	require.Equal(t, "superlinear", got.LatencyClassification)
	require.Equal(t, "superlinear", got.ComputeClassification)
	require.Equal(t, "quadratic", got.LatencyFit.Model)
	require.InDelta(t, 2, got.LatencyFit.Exponent, 0.05)
	require.Equal(t, "superlinear_latency_growth", got.Warnings[0].Code)
	require.Equal(t, "superlinear_compute_growth", got.Warnings[1].Code)
}

func TestBuildScalingAnalysis_JitteredLinearDoesNotClassifySuperlinear(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("30s", 30_000, 305, 610),
		scalingClip("60s", 60_000, 590, 1_220),
		scalingClip("120s", 120_000, 1_230, 2_330),
		scalingClip("240s", 240_000, 2_350, 4_870),
	})

	require.NotEqual(t, "superlinear", got.LatencyClassification)
	require.NotEqual(t, "superlinear", got.ComputeClassification)
	require.Empty(t, got.Warnings)
}

func TestBuildScalingAnalysis_ThreeDurationsRefuseSuperlinearVerdict(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("30s", 30_000, 90, 180),
		scalingClip("60s", 60_000, 360, 720),
		scalingClip("120s", 120_000, 1_440, 2_880),
	})

	require.NotEqual(t, "superlinear", got.LatencyClassification)
	require.NotEqual(t, "superlinear", got.ComputeClassification)
	require.Empty(t, got.Warnings)
}

func TestBuildScalingAnalysis_SingleLatencySamplesCapConfidence(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		singleSampleScalingClip("30s", 30_000, 300, 600),
		singleSampleScalingClip("60s", 60_000, 600, 1_200),
		singleSampleScalingClip("120s", 120_000, 1_200, 2_400),
		singleSampleScalingClip("240s", 240_000, 2_400, 4_800),
	})

	require.Equal(t, "linear", got.LatencyClassification)
	require.Equal(t, "linear", got.ComputeClassification)
	require.Equal(t, "low", got.Confidence)
}

func TestBuildScalingAnalysis_RegistryFitsQualityAndSafetyMetrics(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClipWithQuality("30s", 30_000, 300, 600, 1, 1),
		scalingClipWithQuality("60s", 60_000, 600, 1_200, 2, 2),
		scalingClipWithQuality("120s", 120_000, 1_200, 2_400, 4, 4),
		scalingClipWithQuality("240s", 240_000, 2_400, 4_800, 8, 8),
	})

	require.NotEmpty(t, got.MetricFits)
	requireFitMetric(t, got.MetricFits, metricWER)
	requireFitMetric(t, got.MetricFits, metricTTFC)
	requireFitMetric(t, got.MetricFits, metricDroppedSpan)
}

func TestBuildScalingAnalysis_ComputeClassificationUsesWorstComputeMetric(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		{
			ClipID:              "30s",
			AudioDurationMs:     30_000,
			LatencySamplesMs:    []float64{300},
			ProviderLatencyMs:   600,
			WhisperCalls:        1,
			WhisperAudioSeconds: 30,
			RTF:                 0.01,
			WER:                 WERResult{RefWords: 10},
		},
		{
			ClipID:              "60s",
			AudioDurationMs:     60_000,
			LatencySamplesMs:    []float64{600},
			ProviderLatencyMs:   1_200,
			WhisperCalls:        2,
			WhisperAudioSeconds: 60,
			RTF:                 0.04,
			WER:                 WERResult{RefWords: 10},
		},
		{
			ClipID:              "120s",
			AudioDurationMs:     120_000,
			LatencySamplesMs:    []float64{1_200},
			ProviderLatencyMs:   2_400,
			WhisperCalls:        4,
			WhisperAudioSeconds: 120,
			RTF:                 0.16,
			WER:                 WERResult{RefWords: 10},
		},
		{
			ClipID:              "240s",
			AudioDurationMs:     240_000,
			LatencySamplesMs:    []float64{2_400},
			ProviderLatencyMs:   4_800,
			WhisperCalls:        8,
			WhisperAudioSeconds: 240,
			RTF:                 0.64,
			WER:                 WERResult{RefWords: 10},
		},
	})

	require.Equal(t, "linear", got.LatencyClassification)
	require.Equal(t, "superlinear", got.ComputeClassification)
	require.Equal(t, "rtf", got.ComputeFit.Metric)
	require.Contains(t, got.ComputeFit.Reason, "highest-risk compute metric")
}

func TestBuildScalingAnalysis_ClassifiesFlatScaling(t *testing.T) {
	got := buildScalingAnalysis([]ClipResult{
		scalingClip("30s", 30_000, 100, 200),
		scalingClip("60s", 60_000, 103, 206),
		scalingClip("120s", 120_000, 105, 210),
	})

	require.Equal(t, "flat", got.LatencyClassification)
	require.Equal(t, "linear", got.ComputeClassification)
	require.Equal(t, "constant", got.LatencyFit.Model)
}

func scalingClip(id string, durationMs int64, latencyP95Ms float64, providerLatencyMs float64) ClipResult {
	latencySamples := []float64{latencyP95Ms * 0.98, latencyP95Ms, latencyP95Ms * 1.02}
	return ClipResult{
		ClipID:              id,
		AudioDurationMs:     durationMs,
		LatencySamplesMs:    latencySamples,
		TimeToFirstCommitMs: latencyP95Ms / 2,
		ProviderLatencyMs:   providerLatencyMs,
		WhisperCalls:        int(durationMs / 30_000),
		WhisperAudioSeconds: float64(durationMs) / 1000,
		RTF:                 providerLatencyMs / float64(durationMs),
		WER:                 WERResult{RefWords: 10},
	}
}

func singleSampleScalingClip(id string, durationMs int64, latencyP95Ms float64, providerLatencyMs float64) ClipResult {
	clip := scalingClip(id, durationMs, latencyP95Ms, providerLatencyMs)
	clip.LatencySamplesMs = []float64{latencyP95Ms}
	return clip
}

func scalingClipWithQuality(id string, durationMs int64, latencyP95Ms float64, providerLatencyMs float64, werErrors int, droppedWords int) ClipResult {
	clip := scalingClip(id, durationMs, latencyP95Ms, providerLatencyMs)
	clip.WER = WERResult{EditCounts: EditCounts{Substitutions: werErrors}, RefWords: 100}
	clip.Safety = SafetyGateReport{MaxDroppedSpanWords: droppedWords}
	return clip
}

func testScalingMetric(name scalingMetric) scalingMetricDef {
	for _, metric := range scalingMetricRegistry {
		if metric.name == name {
			return metric
		}
	}
	panic("missing scaling metric")
}

func requireFitMetric(t *testing.T, fits []ScalingModelFit, metric scalingMetric) {
	t.Helper()
	for _, fit := range fits {
		if fit.Metric == string(metric) {
			require.NotEmpty(t, fit.Model)
			require.Greater(t, fit.SampleCount, 0)
			return
		}
	}
	require.Failf(t, "missing metric fit", "metric %s was not fitted", metric)
}
