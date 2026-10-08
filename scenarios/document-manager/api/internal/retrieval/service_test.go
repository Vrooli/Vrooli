package retrieval

import (
	"context"
	"database/sql"
	"testing"

	"document-manager/internal/corpus"

	"github.com/stretchr/testify/require"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	_ "modernc.org/sqlite"
)

func TestPrivacyIsFilteredBeforeLexicalAndSemanticScoring(t *testing.T) { // [REQ:DOC-P0-023] [REQ:DOC-P0-024]
	db, err := sql.Open("sqlite", "file:retrieval-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, corpus.Schema()+"\n"+Schema())
	require.NoError(t, err)
	collections := corpus.NewSQLiteRepository(db)
	require.NoError(t, collections.CreateCollection(ctx, corpus.Collection{ID: "c", Name: "public corpus", DefaultPrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC, Federated: true}))
	repo := NewSQLiteRepository(db)
	require.NoError(t, repo.AddUnit(ctx, Unit{ID: "public", CollectionID: "c", DocumentHash: "public-doc", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC, Text: "shared phrase", AnchorURI: "vrooli-anchor:1/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/1/logical/1/0-6"}))
	require.NoError(t, repo.AddUnit(ctx, Unit{ID: "secret", CollectionID: "c", DocumentHash: "secret-doc", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, Text: "shared phrase", AnchorURI: "vrooli-anchor:1/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb/1/logical/1/0-6"}))
	response, err := NewService(repo, NewSQLiteVectorStore(db)).Query(ctx, Query{Text: "shared phrase", CollectionID: "c", CallerMaxPrivacy: documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL, Federated: true, Limit: 10})
	require.NoError(t, err)
	require.True(t, response.Partial)
	require.Len(t, response.Results, 1)
	require.Equal(t, "public-doc", response.Results[0].DocumentHash)
	require.NotEmpty(t, response.Results[0].AnchorURI)
}

func TestRankingIsDeterministicAndSemanticLegFusesByRRF(t *testing.T) { // [REQ:DOC-P0-023]
	db, err := sql.Open("sqlite", "file:retrieval-rank-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, corpus.Schema()+"\n"+Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	for _, unit := range []Unit{
		{ID: "a", CollectionID: "c", DocumentHash: "a", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC, Text: "alpha", AnchorURI: "a"},
		{ID: "b", CollectionID: "c", DocumentHash: "b", PrivacyClass: documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC, Text: "beta", AnchorURI: "b"},
	} {
		require.NoError(t, repo.AddUnit(ctx, unit))
	}
	require.NoError(t, repo.AddVector(ctx, "a", []float32{1, 0}))
	require.NoError(t, repo.AddVector(ctx, "b", []float32{0, 1}))
	query := Query{Text: "alpha", Vector: []float32{0, 1}, CollectionID: "c", CallerMaxPrivacy: documentpb.PrivacyClass_PRIVACY_CLASS_PUBLIC, Limit: 10}
	one, err := NewService(repo, NewSQLiteVectorStore(db)).Query(ctx, query)
	require.NoError(t, err)
	two, err := NewService(repo, NewSQLiteVectorStore(db)).Query(ctx, query)
	require.NoError(t, err)
	require.Equal(t, one.Results, two.Results)
	require.Len(t, one.Results, 2)
}
