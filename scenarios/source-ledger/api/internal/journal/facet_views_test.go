package journal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPruneRedundantFacetViewsKeepsTopicAndIsIdempotent(t *testing.T) {
	repo := journalDB(t)
	ctx := context.Background()
	vec := []float64{1, 2}
	legacy := Entry{ID: "legacy", Scope: "agent-memory", Body: "legacy", FacetID: "episode", FacetTexts: []FacetText{
		{ID: "l-topic", Kind: TopicFacetKind, Text: "legacy", Vector: vec},
		{ID: "l-rule", Kind: "rule", Text: "Implication: legacy", Vector: vec},
		{ID: "l-ent", Kind: "entities", Text: "Entities: legacy", Vector: vec},
	}}
	current := Entry{ID: "current", Scope: "agent-memory", Body: "current", FacetID: "episode", FacetTexts: []FacetText{{ID: "c-topic", Kind: TopicFacetKind, Text: "current", Vector: vec}}}
	for _, e := range []Entry{legacy, current} {
		_, err := repo.Append(ctx, e, nil)
		require.NoError(t, err)
	}

	planned, err := MeasureRedundantFacetViews(ctx, repo.db)
	require.NoError(t, err)
	require.Equal(t, 1, planned.Entries)
	require.Equal(t, 2, planned.FacetTexts)
	require.Equal(t, 2, planned.Embeddings)
	require.Positive(t, planned.EmbeddingBytes)

	removed, err := PruneRedundantFacetViews(ctx, repo.db, 1)
	require.NoError(t, err)
	require.Equal(t, 2, removed.FacetTexts)
	require.Equal(t, 2, removed.Embeddings)

	var texts, embeddings int
	require.NoError(t, repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_texts`).Scan(&texts))
	require.NoError(t, repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM embeddings`).Scan(&embeddings))
	require.Equal(t, 2, texts, "each entry keeps its topic row")
	require.Equal(t, 2, embeddings, "each entry keeps its topic vector")

	again, err := PruneRedundantFacetViews(ctx, repo.db, 0)
	require.NoError(t, err)
	require.Zero(t, again.FacetTexts)
}

func TestPruneRedundantFacetViewsNeverRemovesAnEntrysOnlyView(t *testing.T) {
	repo := journalDB(t)
	ctx := context.Background()
	_, err := repo.Append(ctx, Entry{ID: "odd", Scope: "agent-memory", Body: "odd", FacetID: "episode", FacetTexts: []FacetText{{ID: "o-rule", Kind: "rule", Text: "Implication: odd", Vector: []float64{1}}}}, nil)
	require.NoError(t, err)
	removed, err := PruneRedundantFacetViews(ctx, repo.db, 0)
	require.NoError(t, err)
	require.Zero(t, removed.FacetTexts)
}
