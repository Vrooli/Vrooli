package sketch

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	internal "react-component-library/internal/sketch"
)

func TestMountedPlacePersistsDeclaredIntentAndRejectsUnresolvedPinnedAsset(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{
	  "page": {"purpose": "Operations overview"},
	  "regions": [{"id": "hero", "purpose": "Primary message", "elements": ["title"]}],
	  "sketch": {"regions": [{"id": "hero"}]}
	}`), 0o600))
	store := internal.NewStore(root)
	before, err := store.Read("demo", "home")
	require.NoError(t, err)

	got := moduleFromDeps(root, nil, nil, Deps{Store: store})
	router := mux.NewRouter()
	got.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := experienceconnect.NewExperienceServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	target := &sharedv1.SketchTarget{Scenario: "demo", Page: "home"}

	placed, err := client.Place(ctx, connect.NewRequest(&sketchv1.PlaceRequest{
		Target: target, Region: "hero", Asset: "feedback.alert", Note: "Keep current risk visible",
		ExpectedContentHash: before.ContentHash,
	}))
	require.NoError(t, err)
	require.True(t, placed.Msg.GetChanged())
	require.NotEqual(t, before.ContentHash, placed.Msg.GetContentHash())
	require.Len(t, placed.Msg.GetSketch().GetPlacements(), 1)
	placement := placed.Msg.GetSketch().GetPlacements()[0]
	require.Equal(t, "hero", placement.GetRegion())
	require.Equal(t, "feedback.alert", placement.GetFills().GetAsset())
	require.Empty(t, placement.GetFills().GetVersion())
	require.Equal(t, "declared", placement.GetState(), "unversioned intent must not claim an exact built implementation")
	require.Equal(t, "Keep current risk visible", placement.GetNote())

	saved, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, placed.Msg.GetContentHash(), saved.ContentHash)
	require.Equal(t, "feedback.alert", saved.Document.Placements[0].Fills.Asset)
	require.Equal(t, "declared", saved.Document.Placements[0].State)

	_, err = client.Place(ctx, connect.NewRequest(&sketchv1.PlaceRequest{
		Target: target, Region: "details", Asset: "feedback.alert", Version: "1.0.0",
		ExpectedContentHash: saved.ContentHash,
	}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "a pinned asset requires an exact implementation resolver")

	unchanged, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, saved.ContentHash, unchanged.ContentHash, "failed pinned placement must not write a partial revision")
	require.Len(t, unchanged.Document.Placements, 1)
}
