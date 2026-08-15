package routing

import (
	"context"
	"errors"
	"fmt"
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

type countingDescriptionEmbedder struct {
	calls  int
	failed map[string]bool
}

func (e *countingDescriptionEmbedder) Available(context.Context) bool { return true }

func (e *countingDescriptionEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	e.calls++
	if e.failed != nil {
		for token := range e.failed {
			if strings.Contains(strings.ToLower(text), token) {
				return nil, errors.New("synthetic embedding failure")
			}
		}
	}
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

// TestEmbeddingDescriptionIndexShortlistsAndNamesOmissions [REQ:REQ-P0-014]
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

// TestDescriptionIndexUnavailableUsesBoundedFallbackReason [REQ:REQ-P0-014]
func TestDescriptionIndexUnavailableUsesBoundedFallbackReason(t *testing.T) {
	index := NewEmbeddingDescriptionIndex(nil)
	profiles := []ProviderProfile{{ProviderID: "a", Description: "unrelated"}, {ProviderID: "b", Description: "database commands"}, {ProviderID: "c", Description: "unrelated"}}
	shortlist, result := index.Shortlist(context.Background(), "database commands", profiles, 2)
	require.False(t, result.Available)
	require.Equal(t, "routing_index_unavailable", result.Reason)
	require.Equal(t, []string{"b", "a"}, []string{shortlist[0].ProviderID, shortlist[1].ProviderID})
	require.Equal(t, []string{"c"}, result.Omitted)
}

func TestDescriptionIndexFallbackIsDeterministic(t *testing.T) {
	index := NewEmbeddingDescriptionIndex(nil)
	profiles := []ProviderProfile{{ProviderID: "z", Description: "code"}, {ProviderID: "a", Description: "code"}, {ProviderID: "m", Description: "other"}}
	first, _ := index.Shortlist(context.Background(), "code", profiles, 2)
	second, _ := index.Shortlist(context.Background(), "code", profiles, 2)
	require.Equal(t, []string{"a", "z"}, []string{first[0].ProviderID, first[1].ProviderID})
	require.Equal(t, []string{first[0].ProviderID, first[1].ProviderID}, []string{second[0].ProviderID, second[1].ProviderID})
}

func TestDescriptionIndexDropsOnlyFailedEntry(t *testing.T) {
	index := NewEmbeddingDescriptionIndex(&countingDescriptionEmbedder{failed: map[string]bool{"bad corpus": true}})
	profiles := []ProviderProfile{{ProviderID: "good", Description: "code commands"}, {ProviderID: "bad", Description: "bad corpus"}}
	shortlist, result := index.Shortlist(context.Background(), "code commands", profiles, 1)
	require.Equal(t, []string{"good"}, []string{shortlist[0].ProviderID})
	require.Equal(t, []string{"bad"}, result.Dropped)
}

// TestDescriptionIndexReembedsOnlyChangedDescriptorAndSurvivesRestart [REQ:REQ-P0-014]
func TestDescriptionIndexReembedsOnlyChangedDescriptorAndSurvivesRestart(t *testing.T) {
	cache := t.TempDir() + "/description-index.json"
	firstEmbedder := &countingDescriptionEmbedder{}
	profiles := []ProviderProfile{{ProviderID: "a", Description: "code commands"}, {ProviderID: "b", Description: "external web"}}
	first := NewPersistentEmbeddingDescriptionIndex(firstEmbedder, cache)
	first.Shortlist(context.Background(), "code", profiles, 1)
	firstCalls := firstEmbedder.calls
	first.Shortlist(context.Background(), "code", []ProviderProfile{{ProviderID: "a", Description: "changed code commands"}, profiles[1]}, 1)
	require.Equal(t, firstCalls+2, firstEmbedder.calls, "one query and one changed descriptor should be embedded")

	secondEmbedder := &countingDescriptionEmbedder{}
	second := NewPersistentEmbeddingDescriptionIndex(secondEmbedder, cache)
	second.Shortlist(context.Background(), "code", []ProviderProfile{{ProviderID: "a", Description: "changed code commands"}, profiles[1]}, 1)
	require.Equal(t, 1, secondEmbedder.calls, "restart should reuse persisted descriptor vectors and embed only the query")
}

func TestDescriptionIndexInvalidatesPersistedVectorsWhenModelChanges(t *testing.T) {
	t.Setenv("SEARCH_HUB_DESCRIPTION_EMBED_MODEL", "model-a")
	cache := t.TempDir() + "/description-index.json"
	profiles := []ProviderProfile{{ProviderID: "a", Description: "code commands"}, {ProviderID: "b", Description: "external web"}}
	firstEmbedder := &countingDescriptionEmbedder{}
	NewPersistentEmbeddingDescriptionIndex(firstEmbedder, cache).Shortlist(context.Background(), "code", profiles, 1)

	t.Setenv("SEARCH_HUB_DESCRIPTION_EMBED_MODEL", "model-b")
	secondEmbedder := &countingDescriptionEmbedder{}
	NewPersistentEmbeddingDescriptionIndex(secondEmbedder, cache).Shortlist(context.Background(), "code", profiles, 1)
	require.Equal(t, 3, secondEmbedder.calls, "model change must invalidate both cached descriptor vectors plus the query")
}

func BenchmarkDescriptionIndexShortlist(b *testing.B) {
	profiles := make([]ProviderProfile, 250)
	for n := range profiles {
		profiles[n] = ProviderProfile{
			ProviderID:  fmt.Sprintf("provider-%03d", n),
			Group:       "project knowledge",
			Type:        "documentation",
			Description: fmt.Sprintf("Searchable project documentation for subsystem %d, ownership, contracts, and operational behavior.", n),
		}
	}

	b.ReportAllocs()
	for _, size := range []int{25, 100, 250} {
		b.Run(fmt.Sprintf("providers-%d", size), func(b *testing.B) {
			corpus := profiles[:size]
			index := NewEmbeddingDescriptionIndex(fixtureDescriptionEmbedder{})
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				index.Shortlist(context.Background(), "where are the project contracts documented", corpus, 8)
			}
		})
	}
}
