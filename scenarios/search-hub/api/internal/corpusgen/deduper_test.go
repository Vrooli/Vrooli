package corpusgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJaccardDeduper_ExactAndReordered(t *testing.T) {
	d := JaccardDeduper{}
	seen := []string{"restart the api service"}
	require.True(t, d.IsDuplicate("restart the api service", seen), "exact match is a duplicate")
	require.True(t, d.IsDuplicate("Restart   THE   service api", seen), "reordering + case + spacing still duplicates")
}

func TestJaccardDeduper_DistinctQueriesPass(t *testing.T) {
	d := JaccardDeduper{}
	seen := []string{"restart the api service"}
	require.False(t, d.IsDuplicate("list running scenarios", seen), "a genuinely different query is not a duplicate")
}

func TestJaccardDeduper_SharedVocabBelowThresholdPasses(t *testing.T) {
	d := JaccardDeduper{}
	// Shares "the service" but asks something else — below 0.8.
	require.False(t, d.IsDuplicate("how do i configure the service ports", []string{"restart the service"}))
}

func TestJaccardDeduper_EmptyCandidateIsDuplicate(t *testing.T) {
	require.True(t, JaccardDeduper{}.IsDuplicate("   ", []string{"anything"}),
		"an empty candidate carries no signal and is never proposed")
}

func TestJaccardDeduper_ThresholdHonored(t *testing.T) {
	// A lenient threshold flags near-overlaps a strict one would pass.
	lenient := JaccardDeduper{Threshold: 0.4}
	require.True(t, lenient.IsDuplicate("configure the service ports", []string{"restart the service"}))
	strict := JaccardDeduper{Threshold: 0.95}
	require.False(t, strict.IsDuplicate("restart the service now", []string{"restart the service"}))
}
