package eval

import (
	"testing"

	"github.com/stretchr/testify/require"
	internalrouting "search-hub/internal/routing"
)

func TestRequestedStrategiesArePreflightedAgainstExecutableCatalog(t *testing.T) {
	catalog := []internalrouting.RetrievalStrategy{
		{Name: "active", Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageLexical}}},
		{Name: "candidate", Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageCrossEncoder}}},
		{Name: "retired", Stages: []internalrouting.RetrievalStage{{Kind: internalrouting.StageLLM}}},
	}

	require.NoError(t, validateRequestedStrategies([]string{"active", "candidate"}, catalog))
	require.ErrorContains(t, validateRequestedStrategies([]string{"active", "typo"}, catalog), "unknown retrieval strategy")
	require.ErrorContains(t, validateRequestedStrategies([]string{"active", "retired"}, catalog), "retired")
}

func TestRunSuiteTierRejectsUnsupportedAndProviderDirectStrategyRequests(t *testing.T) {
	require.NoError(t, validateRunSuiteTier("", ""))
	require.NoError(t, validateRunSuiteTier("provider_direct", ""))
	require.NoError(t, validateRunSuiteTier("federated", "lexical-fallback"))
	require.ErrorContains(t, validateRunSuiteTier("provider_direct", "lexical-fallback"), "not supported")
	require.ErrorContains(t, validateRunSuiteTier("direct", ""), "unsupported eval tier")
}
