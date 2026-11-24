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

func TestMountedBuildBriefReturnsOrderedPinnedChecklistWithoutMutatingPage(t *testing.T) {
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	page := []byte(`{
	  "page": {"purpose": "Operations overview"},
	  "regions": [
	    {"id": "header", "purpose": "Page heading", "elements": ["title"]},
	    {"id": "content", "purpose": "Main information", "elements": ["alert"]}
	  ],
	  "sketch": {
	    "template": {"asset": "templates.operations", "version": "1.2.3"},
	    "regions": [
	      {"id": "header", "origin": "template"},
	      {"id": "content", "note": "Keep the active incident visible"}
	    ],
	    "placements": [
	      {"region": "content", "fills": {"asset": "feedback.alert", "version": "2.0.1"}, "intent": "Show current risk"}
	    ],
	    "unplaced": [
	      {"was": "legacy history row", "region": "history", "reason": "No declared destination"}
	    ]
	  }
	}`)
	require.NoError(t, os.WriteFile(pagePath, page, 0o600))
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

	response, err := client.BuildBrief(ctx, connect.NewRequest(&sketchv1.BuildBriefRequest{
		Target: &sharedv1.SketchTarget{Scenario: "demo", Page: "home"},
	}))
	require.NoError(t, err)
	require.Len(t, response.Msg.GetItems(), 4)
	items := response.Msg.GetItems()
	require.Equal(t, int32(1), items[0].GetOrder())
	require.Equal(t, int32(1), items[0].GetTier())
	require.Equal(t, "templates.operations@1.2.3", items[0].GetAsset())
	require.Equal(t, int32(2), items[1].GetTier())
	require.Equal(t, "content", items[1].GetRegion())
	require.Equal(t, "Keep the active incident visible", items[1].GetConstraint())
	require.Equal(t, "feedback.alert@2.0.1", items[2].GetAsset())
	require.Equal(t, int32(3), items[3].GetTier())
	require.Equal(t, "history", items[3].GetRegion())
	require.Contains(t, response.Msg.GetMarkdown(), "Show current risk")
	require.Contains(t, response.Msg.GetClosingGate(), "phone and desktop sizes")

	after, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, before.ContentHash, after.ContentHash, "building a brief is a read-only operation")
	require.Equal(t, before.Document, after.Document)

	_, err = client.BuildBrief(ctx, connect.NewRequest(&sketchv1.BuildBriefRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a missing target must be a client error")
	still, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, before.ContentHash, still.ContentHash)
}
