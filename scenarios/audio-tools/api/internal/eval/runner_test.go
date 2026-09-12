package eval

import (
	"context"
	"testing"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/segmenter/testaudio"

	"github.com/stretchr/testify/require"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func TestRunnerBuildCellSpecsUsesDeclaredEngineFactory(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Stream: true, Strategies: []sttchain.StrategyKind{sttchain.StrategyPassthrough}})
	runner := reportRunner{deps: RunnerDeps{NewProviderForEngine: func(engineID string) sttchain.Provider {
		if engineID == "kyutai" {
			return provider
		}
		return nil
	}}}

	specs, err := runner.buildCellSpecs([]*experimentv1.EvaluationCell{{EngineId: "kyutai", Strategy: "passthrough", Label: "kyutai native"}})
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, sttchain.StrategyPassthrough, specs[0].Kind)
	require.Equal(t, "fake", specs[0].ModelID)
	require.Equal(t, "kyutai native", specs[0].Label)
	_, err = runner.buildCellSpecs([]*experimentv1.EvaluationCell{{EngineId: "missing", Strategy: "passthrough"}})
	require.ErrorContains(t, err, "unavailable engine")
}

func TestEvaluationCellStrategySupportsProviderNeutralStrategies(t *testing.T) {
	for _, strategy := range []string{"batch", "buffered_fallback", "vad_segment", "overlap_agree", "passthrough"} {
		_, _, _, ok := evaluationCellStrategy(strategy)
		require.Truef(t, ok, "strategy %q must remain executable", strategy)
	}
	_, _, _, ok := evaluationCellStrategy("unknown")
	require.False(t, ok)
}

func TestRunnerLegacyOverlapUsesResolvedEvaluationEngine(t *testing.T) {
	provider := mocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{
		Batch:      true,
		Strategies: []sttchain.StrategyKind{sttchain.StrategyOverlapAgree},
	})
	provider.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "hello", Tier: sttchain.TierLocal, Latency: time.Millisecond}, nil
	}
	runner := reportRunner{deps: RunnerDeps{
		NewProvider: func() sttchain.Provider { return provider },
	}}

	specs, err := runner.buildSpecs([]*evalv1.EvalStrategy{{Kind: "overlap_agree"}})
	require.NoError(t, err)
	require.Len(t, specs, 1)

	clip := Clip{
		ID:         "legacy-overlap",
		PCM:        append(testaudio.VoicedSamples(1000, 1000), testaudio.SilenceSamples(1200)...),
		SampleRate: testaudio.SampleRateHz,
		Reference:  "hello",
	}
	session, meter := specs[0].BuildSession(clip)
	require.NotNil(t, session)
	Replay(context.Background(), clip, ReplayOptions{Mode: ModeDeterministic, ChunkMs: 100}, session)

	// A legacy --strategies request has no explicit engine id. The harness
	// must still bind its resolved evaluation engine to the LocalEngines map;
	// otherwise Segmenter selects BufferedFallback and makes exactly one
	// terminal batch call instead of exercising OverlapAgree.
	require.GreaterOrEqual(t, meter.Snapshot().Calls, 2)
}
