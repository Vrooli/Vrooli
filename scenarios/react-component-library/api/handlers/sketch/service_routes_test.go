package sketch

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	internal "react-component-library/internal/sketch"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	assistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	candidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
)

func TestWorkspaceAssistAndCandidateMountWithoutLegacySketchService(t *testing.T) {
	root := t.TempDir()
	pageDir := filepath.Join(root, "scenarios", "demo", "experience", "pages")
	require.NoError(t, os.MkdirAll(pageDir, 0o700))
	store := internal.NewStore(root)
	got := moduleFromDeps(root, log.Default(), nil, Deps{Store: store})
	router := mux.NewRouter()
	got.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	assist := assistconnect.NewWorkspaceAssistServiceClient(server.Client(), server.URL)
	_, err := assist.GetSketchInference(context.Background(), connect.NewRequest(&sketchv1.GetSketchInferenceRequest{Id: "missing"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))

	candidates := candidateconnect.NewWorkspaceCandidateServiceClient(server.Client(), server.URL)
	listed, err := candidates.ListCandidates(context.Background(), connect.NewRequest(&sharedv1.SketchTarget{Scenario: "demo", Page: "home"}))
	require.NoError(t, err)
	require.Empty(t, listed.Msg.GetCandidates())

	legacyURL := server.URL + "/vrooli.react_component_library.v1.sketch.SketchService/ListCandidates"
	legacy, err := server.Client().Post(legacyURL, "application/json", http.NoBody)
	require.NoError(t, err)
	legacy.Body.Close()
	require.Equal(t, http.StatusNotFound, legacy.StatusCode)
}
