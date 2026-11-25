package hub_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

// keywordEmbedder is a deterministic stand-in for the gateway. It maps three
// synonym families into dimensions, so two paraphrases that share no words
// score close and an unrelated question does not.
type keywordEmbedder struct {
	fail bool
	dims int
}

var synonymFamilies = [][]string{
	{"delete", "drop", "remove", "erase"},
	{"database", "store", "data", "db", "records"},
	{"staging", "pre-production", "preproduction", "test", "non-production"},
	{"logo", "design", "page", "brand"},
}

func (e keywordEmbedder) Embed(_ context.Context, texts []string) (hub.Embedding, error) {
	if e.fail {
		return hub.Embedding{}, errors.New("ai-gateway unavailable")
	}
	dims := e.dims
	if dims == 0 {
		dims = len(synonymFamilies) + 1
	}
	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		lower := strings.ToLower(text)
		vector := make([]float32, dims)
		for family, words := range synonymFamilies {
			if family >= dims-1 {
				break
			}
			for _, word := range words {
				if strings.Contains(lower, word) {
					vector[family] += 1
					break
				}
			}
		}
		vector[dims-1] = 0.1 // never a zero vector
		vectors = append(vectors, vector)
	}
	return hub.Embedding{Model: "fake-embed", Dimensions: dims, Vectors: vectors}, nil
}

func TestSemanticRelatedFindsAParaphraseTheWordsMiss(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	l.service.SetEmbedder(keywordEmbedder{})

	first, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "sem-1", "Remove the pre-production data store?"))
	require.NoError(t, err)
	second, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "sem-2", "Delete the staging database?"))
	require.NoError(t, err)

	// The pair shares no content word, so the lexical leg alone would miss it.
	stored, err := l.service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, second.ID, stored.Related[0].AskID)
	require.Equal(t, "semantic", stored.Related[0].Basis)
	require.GreaterOrEqual(t, stored.Related[0].Score, 0.68)
}

func TestSemanticFailureLeavesTheLexicalRead(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	l.service.SetEmbedder(keywordEmbedder{fail: true})

	first, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "semf-1", "Should the agent delete the staging database?"))
	require.NoError(t, err)
	second, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "semf-2", "Should the agent be allowed to drop the staging DB?"))
	require.NoError(t, err)

	stored, err := l.service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, second.ID, stored.Related[0].AskID)
	require.Equal(t, "lexical", stored.Related[0].Basis)
}

func TestADimensionChangeIsIgnoredRatherThanCompared(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	l.service.SetEmbedder(keywordEmbedder{})
	first, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "semd-1", "Should the agent delete the staging database?"))
	require.NoError(t, err)
	_, _, err = l.service.AskWithSpec(ctx, readySpec(listNow, "semd-2", "Should the agent be allowed to drop the staging DB?"))
	require.NoError(t, err)

	// A different model/dimension must not be compared with the stored rows.
	l.service.SetEmbedder(keywordEmbedder{dims: 2})
	stored, err := l.service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, "lexical", stored.Related[0].Basis, "a changed embedder degrades to lexical")
}

func TestBackfillEmbeddingsMakesAnOlderAskSemantic(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	first, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "semb-1", "Remove the pre-production data store?"))
	require.NoError(t, err)
	second, _, err := l.service.AskWithSpec(ctx, readySpec(listNow, "semb-2", "Delete the staging database?"))
	require.NoError(t, err)

	l.service.SetEmbedder(keywordEmbedder{})
	embedded, err := l.service.BackfillEmbeddings(ctx, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, embedded, 2)

	stored, err := l.service.GetAsk(ctx, first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.Related)
	require.Equal(t, second.ID, stored.Related[0].AskID)
	require.Equal(t, "semantic", stored.Related[0].Basis)
}
