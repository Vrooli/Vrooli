package search

import (
	"context"
	"testing"
	"time"

	"offer-desk/internal/aisearch"
	"offer-desk/internal/catalog"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/search"
)

func newHandler(t *testing.T) (handler, *catalog.Store, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string { return (&catalog.Store{}).Schema() })))
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	store := catalog.NewStore(db, func() time.Time { return now })
	source := aisearch.NewStoreSource(store, func() time.Time { return now })
	return handler{service: aisearch.NewService(source)}, store, ctx
}

func TestSearchProjectsCatalogWithKindAndFollowUp(t *testing.T) {
	h, store, ctx := newHandler(t)
	node, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	response, err := h.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: "Aquila launch", Limit: 10}))
	require.NoError(t, err)
	require.NotEmpty(t, response.Msg.Results)
	var found *searchv1.SearchResult
	for _, result := range response.Msg.Results {
		if result.Id == "node:"+node.Id {
			found = result
		}
	}
	require.NotNil(t, found, "expected the authoritative node record in results")
	require.Equal(t, "node-offer", found.Kind)
	require.Equal(t, "catalog/node/"+node.Id, found.FollowUp)
	require.NotEmpty(t, response.Msg.Generation)
	require.NotEmpty(t, response.Msg.MaterializedAt)
}

func TestStatusDoesNotAdvanceOnUnchangedRead(t *testing.T) {
	h, store, ctx := newHandler(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	first, err := h.Status(ctx, connect.NewRequest(&searchv1.StatusRequest{}))
	require.NoError(t, err)
	require.True(t, first.Msg.Available)
	require.Positive(t, first.Msg.IndexedCount)

	second, err := h.Status(ctx, connect.NewRequest(&searchv1.StatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, first.Msg.Generation, second.Msg.Generation)
	require.Equal(t, first.Msg.LastIndexedAt, second.Msg.LastIndexedAt, "an unchanged read must not fabricate a new index time")
}

func TestSearchGibberishReturnsNoHits(t *testing.T) {
	h, store, ctx := newHandler(t)
	_, err := store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Aquila launch offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)

	response, err := h.Search(ctx, connect.NewRequest(&searchv1.SearchRequest{Query: "florbnax zzqq nonsense", Limit: 10}))
	require.NoError(t, err)
	require.Empty(t, response.Msg.Results)
}
