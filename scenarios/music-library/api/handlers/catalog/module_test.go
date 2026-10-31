package catalog

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog"
	svc "github.com/vrooli/vrooli/packages/proto/gen/go/music-library/v1/catalog/catalog_v1connect"
	"music-library/internal/catalog"
)

func TestCatalogConnectRoundTrip(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(catalog.Schema)))
	_, h := svc.NewCatalogServiceHandler(&handler{repo: catalog.NewStore(db)})
	server := httptest.NewServer(h)
	defer server.Close()
	c := svc.NewCatalogServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	id := strings.Repeat("c", 64)
	_, err := c.PutTrack(ctx, connect.NewRequest(&pb.PutTrackRequest{Track: &pb.Track{Sha256: id, Title: "Other project", SourcePath: "/music/example.wav", Tags: []string{"ambient"}, Approval: "liked"}}))
	require.NoError(t, err)
	_, err = c.RecordUse(ctx, connect.NewRequest(&pb.RecordUseRequest{Sha256: id, Use: &pb.TrackUse{Project: "podcast", Revision: "episode-2", Status: "draft"}}))
	require.NoError(t, err)
	r, err := c.ListTracks(ctx, connect.NewRequest(&pb.ListTracksRequest{Tag: "ambient", UnusedOnly: true}))
	require.NoError(t, err)
	require.Len(t, r.Msg.Tracks, 1)
	_, err = c.RecordUse(ctx, connect.NewRequest(&pb.RecordUseRequest{Sha256: id, Use: &pb.TrackUse{Project: "podcast", Revision: "episode-2", Status: "accepted"}}))
	require.NoError(t, err)
	r, err = c.ListTracks(ctx, connect.NewRequest(&pb.ListTracksRequest{UnusedOnly: true}))
	require.NoError(t, err)
	require.Empty(t, r.Msg.Tracks)
	r, err = c.ListTracks(ctx, connect.NewRequest(&pb.ListTracksRequest{Project: "podcast"}))
	require.NoError(t, err)
	require.Len(t, r.Msg.Tracks, 1)
}
