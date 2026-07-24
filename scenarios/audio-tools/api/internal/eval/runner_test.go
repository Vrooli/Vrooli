package eval

import (
	"testing"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/sttchain/mocks"

	"github.com/stretchr/testify/require"
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
