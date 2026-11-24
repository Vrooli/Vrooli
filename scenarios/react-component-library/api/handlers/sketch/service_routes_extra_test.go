package sketch

import (
	"context"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	experiencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	internal "react-component-library/internal/sketch"
)

func TestMountedExperienceListDesignPagesReportsWorkspaceInventory(t *testing.T) {
	root := t.TempDir()
	pageDir := filepath.Join(root, "scenarios", "demo", "experience", "pages")
	require.NoError(t, os.MkdirAll(pageDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(pageDir, "home.json"), []byte(`{"regions":[]}`), 0o600))

	got := moduleFromDeps(root, log.Default(), nil, Deps{Store: internal.NewStore(root)})
	router := mux.NewRouter()
	got.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := experienceconnect.NewExperienceServiceClient(server.Client(), server.URL)
	response, err := client.ListDesignPages(context.Background(), connect.NewRequest(&sketchv1.ListDesignPagesRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.Len(t, response.Msg.GetPages(), 1)
	page := response.Msg.GetPages()[0]
	require.Equal(t, "home", page.GetPage())
	require.Equal(t, "contract-only", page.GetStatus())
	require.False(t, page.GetRegistered())
}

func TestMountedAdoptExperiencePageReturnsRecoveryEvidenceAndPreservesSource(t *testing.T) {
	root := t.TempDir()
	pageDir := filepath.Join(root, "scenarios", "demo", "experience", "pages")
	require.NoError(t, os.MkdirAll(pageDir, 0o700))
	source := []byte(`{"page":{"id":"home","title":"Home"},"claims":[{"id":"keep"}],"sketch":{"template":{"asset":"templates.collection-page"},"render":{"regions":[{"id":"main","slot":["body"]}]}}}`)
	path := filepath.Join(pageDir, "home.json")
	require.NoError(t, os.WriteFile(path, source, 0o600))
	calls := 0
	deps := Deps{
		Store: internal.NewStore(root),
		AdoptExperience: func(_ context.Context, scenario, page string, got []byte) (*workspacev1.Workspace, string, string, string, string, bool, error) {
			calls++
			require.Equal(t, "demo", scenario)
			require.Equal(t, "home", page)
			require.Equal(t, source, got)
			return &workspacev1.Workspace{Id: "ws_imported", BindingScenario: scenario, BindingPage: page}, "ir_receipt", "source-sha", "target-sha", `{"sourcePageJson":"preserved"}`, calls > 1, nil
		},
	}
	got := moduleFromDeps(root, log.Default(), nil, deps)
	router := mux.NewRouter()
	got.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := experienceconnect.NewExperienceServiceClient(server.Client(), server.URL)
	for i := 0; i < 2; i++ {
		response, err := client.AdoptExperiencePage(context.Background(), connect.NewRequest(&experiencev1.AdoptExperiencePageRequest{Scenario: "demo", Page: "home"}))
		require.NoError(t, err)
		require.Equal(t, "ws_imported", response.Msg.GetWorkspace().GetId())
		require.Equal(t, "ir_receipt", response.Msg.GetReceiptId())
		require.Equal(t, "source-sha", response.Msg.GetSourceHash())
		require.Equal(t, "target-sha", response.Msg.GetTargetHash())
		require.NotEmpty(t, response.Msg.GetReceiptJson())
		require.Equal(t, i == 1, response.Msg.GetAlreadyAdopted())
	}
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, source, after, "adoption must leave the original source page untouched")
}
