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
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	candidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
	internal "react-component-library/internal/sketch"
)

func TestMountedCandidateSaveAndGetPersistsImmutableResponse(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	pagePath := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(pagePath), 0o700))
	require.NoError(t, os.WriteFile(pagePath, []byte(`{"regions":[]}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)

	module := moduleFromDeps(root, nil, nil, Deps{Store: store})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := candidateconnect.NewWorkspaceCandidateServiceClient(server.Client(), server.URL)
	target := &sharedv1.SketchTarget{Scenario: "demo", Page: "home"}
	requested := &sharedv1.Sketch{Viewport: "desktop", Regions: []*sharedv1.SketchRegion{{Id: "hero", Elements: []string{"header", "intro"}}}}
	saved, err := client.SaveCandidate(ctx, connect.NewRequest(&sketchv1.SaveCandidateRequest{
		Target: target, DesignId: "candidate-test", ExpectedContentHash: base.ContentHash, Sketch: requested,
	}))
	require.NoError(t, err)
	require.Equal(t, "demo", saved.Msg.GetCandidate().GetScenario())
	require.Equal(t, "candidate-test", saved.Msg.GetCandidate().GetDesignId())
	require.NotEmpty(t, saved.Msg.GetCandidate().GetHash())
	require.Equal(t, base.ContentHash, saved.Msg.GetBaseHash())
	require.Equal(t, "home", saved.Msg.GetPage())
	require.Equal(t, "hero", saved.Msg.GetSketch().GetRegions()[0].GetId())

	pageAfterSave, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, base.ContentHash, pageAfterSave.ContentHash, "saving an alternative must not mutate the active page")

	loaded, err := client.GetCandidate(ctx, connect.NewRequest(&sharedv1.CandidateReference{
		Scenario: "demo", DesignId: "candidate-test", Hash: saved.Msg.GetCandidate().GetHash(),
	}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.GetCandidate(), loaded.Msg.GetCandidate())
	require.Equal(t, saved.Msg.GetBaseHash(), loaded.Msg.GetBaseHash())
	require.Equal(t, "hero", loaded.Msg.GetSketch().GetRegions()[0].GetId())

	_, err = client.SaveCandidate(ctx, connect.NewRequest(&sketchv1.SaveCandidateRequest{
		Target: target, DesignId: "candidate-stale", ExpectedContentHash: "stale-revision", Sketch: requested,
	}))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
}
