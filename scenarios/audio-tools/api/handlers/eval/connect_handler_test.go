package eval

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/sttchain/mocks"
	inteval "audio-tools/internal/eval"

	"github.com/stretchr/testify/require"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func TestBuildCellSpecs_UsesDeclaredEngineFactory(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}})
	h := reportRunner{deps: Deps{NewProviderForEngine: func(engineID string) sttchain.Provider {
		if engineID == "kyutai" {
			return provider
		}
		return nil
	}}}
	specs, err := h.buildCellSpecs([]*experimentv1.EvaluationCell{{EngineId: "kyutai", Strategy: "passthrough", Label: "kyutai native"}})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, sttchain.StrategyPassthrough, specs[0].Kind)
	require.Equal(t, "fake", specs[0].ModelID)
	require.Equal(t, "kyutai native", specs[0].Label)

	_, err = h.buildCellSpecs([]*experimentv1.EvaluationCell{{EngineId: "missing", Strategy: "passthrough"}})
	require.ErrorContains(t, err, "unavailable engine")
}

func TestEvaluationCellStrategySupportsEveryProviderNeutralStrategy(t *testing.T) {
	cases := []struct {
		name      string
		wantKind  sttchain.StrategyKind
		wantBatch bool
		wantOK    bool
	}{
		{name: "batch", wantKind: sttchain.StrategyBuffered, wantBatch: true, wantOK: true},
		{name: "buffered_fallback", wantKind: sttchain.StrategyBuffered, wantBatch: true, wantOK: true},
		{name: "vad_segment", wantKind: sttchain.StrategyVADSegment, wantOK: true},
		{name: "overlap_agree", wantKind: sttchain.StrategyOverlapAgree, wantOK: true},
		{name: "passthrough", wantKind: sttchain.StrategyPassthrough, wantOK: true},
		{name: "unknown", wantOK: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, gotKind, gotBatch, gotOK := evaluationCellStrategy(tc.name)
			require.Equal(t, tc.wantOK, gotOK)
			if tc.wantOK {
				require.Equal(t, tc.wantKind, gotKind)
				require.Equal(t, tc.wantBatch, gotBatch)
			}
		})
	}
}

func TestRunReportForCells_ExecutesNativeProviderThroughSegmenter(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}})
	provider.StreamFn = func(_ context.Context, _ sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		events := make(chan sttchain.StreamEvent, 2)
		go func() {
			for range chunks {
			}
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{Text: "native cell"}}
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "native cell"}}
			close(events)
		}()
		return events, nil
	}
	report, err := RunReportForCells(context.Background(), Deps{NewProviderForEngine: func(engineID string) sttchain.Provider {
		if engineID == "kyutai" {
			return provider
		}
		return nil
	}}, []inteval.Clip{{ID: "clip", PCM: []byte{0, 0, 0, 0}, SampleRate: 16_000, Format: "pcm_s16le", Reference: "native cell"}}, []*experimentv1.EvaluationCell{{EngineId: "kyutai", Strategy: "passthrough", Label: "kyutai native", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, RepeatCount: 1}}, 100, inteval.EvalOptions{})
	require.NoError(t, err)
	require.Len(t, report.PerStrategy, 1)
	require.Equal(t, "kyutai native", report.PerStrategy[0].Label)
	require.Equal(t, 0.0, report.PerStrategy[0].WER)
	require.Equal(t, "kyutai", report.PerStrategy[0].EngineID)
	require.Equal(t, "deterministic", report.PerStrategy[0].ReplayLane)
}

func TestRunReportForCells_HonorsRealtimeLaneAndRepeatCount(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}})
	var calls atomic.Int32
	provider.StreamFn = func(_ context.Context, _ sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		calls.Add(1)
		events := make(chan sttchain.StreamEvent, 1)
		go func() {
			for range chunks {
			}
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "realtime cell"}}
			close(events)
		}()
		return events, nil
	}
	report, err := RunReportForCells(context.Background(), Deps{NewProviderForEngine: func(engineID string) sttchain.Provider {
		if engineID == "kyutai" {
			return provider
		}
		return nil
	}}, []inteval.Clip{{ID: "clip", PCM: make([]byte, 320), SampleRate: 16_000, Format: "pcm_s16le", Reference: "realtime cell"}}, []*experimentv1.EvaluationCell{{
		EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_REALTIME, RepeatCount: 2,
	}}, 10, inteval.EvalOptions{Sleep: func(time.Duration) {}})
	require.NoError(t, err)
	require.True(t, report.LatencyMeasured)
	require.Len(t, report.PerStrategy, 1)
	require.Len(t, report.PerStrategy[0].PerClip[0].LatencySamplesMs, 2)
	require.Equal(t, "realtime", report.PerStrategy[0].ReplayLane)
	require.EqualValues(t, 2, calls.Load(), "a realtime cell must not prepend an unpaced deterministic pass")
}

func TestRunReportForCells_RejectsUnearnedEvidenceLanes(t *testing.T) {
	cases := []*experimentv1.EvaluationCell{
		{EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH, RepeatCount: 1},
		{EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, FaultProfile: "provider_busy", RepeatCount: 1},
		{EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, PolicyProfile: "speaker-filter", RepeatCount: 1},
	}
	for _, cell := range cases {
		_, err := RunReportForCells(context.Background(), Deps{}, []inteval.Clip{{ID: "clip", PCM: []byte{0, 0}, SampleRate: 16_000}}, []*experimentv1.EvaluationCell{cell}, 100, inteval.EvalOptions{})
		require.Error(t, err)
		require.ErrorContains(t, err, "harness")
	}
}

func TestRunReportForCells_GivesRepeatedDeterministicCellsDistinctStableIDs(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}})
	provider.StreamFn = func(_ context.Context, _ sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		events := make(chan sttchain.StreamEvent, 1)
		go func() {
			for range chunks {
			}
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "repeat"}}
			close(events)
		}()
		return events, nil
	}
	report, err := RunReportForCells(context.Background(), Deps{NewProviderForEngine: func(string) sttchain.Provider { return provider }}, []inteval.Clip{{ID: "clip", PCM: []byte{0, 0}, SampleRate: 16_000, Reference: "repeat"}}, []*experimentv1.EvaluationCell{{
		EngineId: "kyutai", Strategy: "passthrough", ReplayLane: experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, RepeatCount: 2,
	}}, 100, inteval.EvalOptions{})
	require.NoError(t, err)
	require.Len(t, report.PerStrategy, 2)
	require.NotEqual(t, report.PerStrategy[0].Strategy, report.PerStrategy[1].Strategy)
	require.Equal(t, sttchain.StrategyPassthrough, report.PerStrategy[0].BaseStrategy)
	require.Equal(t, sttchain.StrategyPassthrough, report.PerStrategy[1].BaseStrategy)
}

func TestPromotionVerdicts_CreditsDurationButBlocksMissingQualificationCategories(t *testing.T) {
	verdicts := promotionVerdicts(inteval.EvalReport{PerStrategy: []inteval.StrategyReport{{
		EngineID:   "kyutai",
		ReplayLane: "realtime",
		WER:        0.01,
		PerClip: []inteval.ClipResult{{
			AudioDurationMs: 60 * 60 * 1000,
		}},
	}}})
	require.Len(t, verdicts, 1)
	require.Equal(t, "kyutai", verdicts[0].EngineID)
	require.False(t, verdicts[0].Stable)
	require.NotContains(t, verdicts[0].Reasons, "missing duration profile: 60_minutes")
	require.Contains(t, verdicts[0].Reasons, "missing fault profile: provider_busy")
	require.Contains(t, verdicts[0].Reasons, "browser product-path evidence is missing")
	require.Contains(t, verdicts[0].Reasons, "manual device evidence is missing")
}

func TestPromotionVerdicts_RejectsObservedSafetyFailure(t *testing.T) {
	verdicts := promotionVerdicts(inteval.EvalReport{PerStrategy: []inteval.StrategyReport{{
		EngineID:   "whisper-local",
		ReplayLane: "realtime",
		WER:        0.01,
		Safety:     inteval.SafetyGateReport{Passed: false},
		PerClip: []inteval.ClipResult{{
			AudioDurationMs: 60 * 60 * 1000,
		}},
	}}})
	require.Len(t, verdicts, 1)
	require.Contains(t, verdicts[0].Reasons, "dropped-span rate exceeds trust threshold")
}

func TestPromotionVerdicts_DoesNotCreditDeterministicDurationAsRealtimeEvidence(t *testing.T) {
	verdicts := promotionVerdicts(inteval.EvalReport{PerStrategy: []inteval.StrategyReport{{
		EngineID:   "kyutai",
		ReplayLane: "deterministic",
		WER:        0.01,
		PerClip: []inteval.ClipResult{{
			AudioDurationMs: 60 * 60 * 1000,
		}},
	}}})
	require.Len(t, verdicts, 1)
	require.Contains(t, verdicts[0].Reasons, "missing duration profile: 60_minutes")
}

func TestReportToProto_MapsScalingAnalysis(t *testing.T) {
	report := inteval.EvalReport{
		QualityMeasured:   true,
		PromotionVerdicts: []inteval.PromotionVerdict{{EngineID: "kyutai", Stable: false, Reasons: []string{"manual device evidence is missing"}}},
		PerStrategy: []inteval.StrategyReport{{
			Label:      "batch",
			EngineID:   "kyutai",
			ReplayLane: "realtime",
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
					Unit:           "ms",
					Model:          "linear",
					SlopePerSecond: 12,
					Intercept:      40,
					RSquared:       0.91,
					SampleCount:    3,
					Exponent:       1.05,
					ExponentR2:     0.89,
					Reason:         "linear fit was strongest",
				},
				MetricFits: []inteval.ScalingModelFit{{
					Metric:      "wer",
					Unit:        "rate",
					Model:       "linear",
					RSquared:    0.82,
					SampleCount: 3,
					Exponent:    1.1,
					ExponentR2:  0.8,
				}},
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
	require.Len(t, got.GetPromotionVerdicts(), 1)
	require.Equal(t, "kyutai", got.GetPromotionVerdicts()[0].GetEngineId())
	require.False(t, got.GetPromotionVerdicts()[0].GetStable())
	require.Equal(t, "kyutai", got.GetPerStrategy()[0].GetEngineId())
	require.Equal(t, "realtime", got.GetPerStrategy()[0].GetReplayLane())
	scaling := got.GetPerStrategy()[0].GetScaling()
	require.NotNil(t, scaling)
	require.Equal(t, "linear", scaling.GetLatencyClassification())
	require.Equal(t, "superlinear", scaling.GetComputeClassification())
	require.Equal(t, "medium", scaling.GetConfidence())
	require.Equal(t, "duration sweep has enough points", scaling.GetReasons()[0])
	require.Equal(t, "superlinear_compute_growth", scaling.GetWarnings()[0].GetCode())
	require.Equal(t, "finalization_latency_p95_ms", scaling.GetLatencyFit().GetMetric())
	require.InDelta(t, 12, scaling.GetLatencyFit().GetSlopePerSecond(), 1e-9)
	require.Equal(t, "ms", scaling.GetLatencyFit().GetUnit())
	require.InDelta(t, 1.05, scaling.GetLatencyFit().GetExponent(), 1e-9)
	require.InDelta(t, 0.89, scaling.GetLatencyFit().GetExponentRSquared(), 1e-9)
	require.Len(t, scaling.GetMetricFits(), 1)
	require.Equal(t, "wer", scaling.GetMetricFits()[0].GetMetric())
	require.Len(t, scaling.GetPoints(), 1)
	point := scaling.GetPoints()[0]
	require.Equal(t, "long-form-60s", point.GetClipId())
	require.EqualValues(t, 60_000, point.GetRealizedDurationMs())
	require.InDelta(t, 1_200, point.GetProviderLatencyMs(), 1e-9)
}
