package routing

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fixtureDescriptionEmbedder struct{}

func (fixtureDescriptionEmbedder) Available(context.Context) bool { return true }

func (fixtureDescriptionEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	text = strings.ToLower(text)
	vector := []float64{0, 0, 0}
	if strings.Contains(text, "code") || strings.Contains(text, "command") {
		vector[0] = 1
	}
	if strings.Contains(text, "external") || strings.Contains(text, "web") {
		vector[1] = 1
	}
	if strings.Contains(text, "memory") || strings.Contains(text, "history") {
		vector[2] = 1
	}
	return vector, nil
}

func TestEmbeddingDescriptionIndexShortlistsAndNamesOmissions(t *testing.T) {
	index := NewEmbeddingDescriptionIndex(fixtureDescriptionEmbedder{})
	profiles := []ProviderProfile{
		{ProviderID: "records", Description: "project memory and history"},
		{ProviderID: "commands", Description: "code and command evidence"},
		{ProviderID: "web", Description: "external web facts"},
	}
	shortlist, result := index.Shortlist(context.Background(), "where is the code command", profiles, 2)
	require.True(t, result.Available)
	require.Equal(t, 3, result.Total)
	require.Equal(t, 2, result.Returned)
	require.Equal(t, []string{"commands", "records"}, []string{shortlist[0].ProviderID, shortlist[1].ProviderID})
	require.Equal(t, []string{"web"}, result.Omitted)
}

func TestDescriptionIndexUnavailableUsesBoundedFallbackReason(t *testing.T) {
	index := NewEmbeddingDescriptionIndex(nil)
	profiles := []ProviderProfile{{ProviderID: "a"}, {ProviderID: "b"}, {ProviderID: "c"}}
	shortlist, result := index.Shortlist(context.Background(), "query", profiles, 2)
	require.False(t, result.Available)
	require.Equal(t, "routing_index_unavailable", result.Reason)
	require.Equal(t, []string{"a", "b"}, []string{shortlist[0].ProviderID, shortlist[1].ProviderID})
	require.Equal(t, []string{"c"}, result.Omitted)
}
