package taste

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	tastev1 "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/taste"
	"music-library/internal/catalog"
	"music-library/internal/preference"
)

func TestRecommendExcludesOperatorRejectedCandidates(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(catalog.Schema),
		database.SchemaProviderFunc(preference.Schema),
	))
	catalogStore := catalog.NewStore(db)
	seedID, rejectedID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	_, err := catalogStore.Put(ctx, catalog.Track{SHA256: seedID, Title: "Liked seed", SourcePath: "/seed.wav", Approval: "liked", Tags: []string{"genre:ambient"}})
	require.NoError(t, err)
	_, err = catalogStore.Put(ctx, catalog.Track{SHA256: rejectedID, Title: "Rejected track", SourcePath: "/rejected.wav", Approval: "rejected", Tags: []string{"genre:ambient"}})
	require.NoError(t, err)
	h := &handler{store: preference.NewStore(db), catalog: catalogStore}

	response, err := h.Recommend(ctx, connect.NewRequest(&tastev1.RecommendRequest{SeedTrackSha256: seedID}))
	require.NoError(t, err)
	got := make([]string, 0, len(response.Msg.Recommendations))
	for _, item := range response.Msg.Recommendations {
		got = append(got, item.Candidate.TrackSha256)
	}
	require.NotContains(t, got, rejectedID, "operator-rejected tracks must not be recommended")
}
