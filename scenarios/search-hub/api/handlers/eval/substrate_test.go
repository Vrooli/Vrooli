package eval

import (
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestResolveRunRerankProviderDeclarationWins(t *testing.T) {
	// A provider that explicitly declares rerank-off must keep "none" even when
	// the shared substrate advertises an active reranker (the drift this fixes).
	leg, enabled := resolveRunRerank("none", nil, "cross-encoder:BAAI/bge-reranker-v2-m3", true)
	require.Equal(t, "none", leg)
	require.False(t, enabled)

	leg, enabled = resolveRunRerank("cross-encoder:BAAI/bge-reranker-v2-m3", nil, "llm:other", true)
	require.Equal(t, "cross-encoder:BAAI/bge-reranker-v2-m3", leg)
	require.True(t, enabled)
}

func TestResolveRunRerankDeclaredTuningBeatsSubstrate(t *testing.T) {
	// Provider status is silent, so its registered tuning is authoritative: a
	// registered rerank-off must not be overwritten by the substrate reranker.
	tuning := &registryv1.Tuning{Engine: "dense", RerankEnabled: false}
	leg, enabled := resolveRunRerank("", tuning, "cross-encoder:BAAI/bge-reranker-v2-m3", true)
	require.Equal(t, "none", leg)
	require.False(t, enabled)

	leg, enabled = resolveRunRerank("unknown", tuning, "cross-encoder:BAAI/bge-reranker-v2-m3", true)
	require.Equal(t, "none", leg)
	require.False(t, enabled)
}

func TestResolveRunRerankDeclaredTuningOnUsesSubstrateLeg(t *testing.T) {
	tuning := &registryv1.Tuning{Engine: "dense", RerankEnabled: true}
	leg, enabled := resolveRunRerank("", tuning, "cross-encoder:BAAI/bge-reranker-v2-m3", true)
	require.Equal(t, "cross-encoder:BAAI/bge-reranker-v2-m3", leg)
	require.True(t, enabled)

	leg, enabled = resolveRunRerank("", tuning, "", false)
	require.Equal(t, "unknown", leg)
	require.True(t, enabled)
}

func TestResolveRunRerankSubstrateFallbackOnlyWhenUndeclared(t *testing.T) {
	leg, enabled := resolveRunRerank("", nil, "cross-encoder:BAAI/bge-reranker-v2-m3", true)
	require.Equal(t, "cross-encoder:BAAI/bge-reranker-v2-m3", leg)
	require.True(t, enabled)

	leg, enabled = resolveRunRerank("", nil, "none", false)
	require.Equal(t, "none", leg)
	require.False(t, enabled)
}
