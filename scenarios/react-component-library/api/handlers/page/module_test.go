package page

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	pageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page/page_v1connect"
	"react-component-library/internal/pageinspect"
)

func TestMountedInspectReturnsRedactedCaptureAndRejectsUnsafeRoute(t *testing.T) {
	outline := []byte(`{"schema":"bas-agent-outline/v1","nodes":[{"id":"save","kind":"action","tag":"button","name":"Save","selector":"#save","rect":{"x":1,"y":2,"width":80,"height":32}}],"unsupported":[]}`)
	digest := sha256.Sum256(outline)
	artifactPath := "outline/page.json"
	bundleID := "123e4567-e89b-42d3-a456-426614174000"
	bas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			require.Equal(t, "/api/v1/ux-capture/bundles/"+bundleID+"/artifacts/"+artifactPath, r.URL.Path)
			_, _ = w.Write(outline)
			return
		}
		require.Equal(t, "/api/v1/ux-capture/scenario", r.URL.Path)
		var input map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
		require.Equal(t, "/settings?token=secret", input["route"])
		_ = json.NewEncoder(w).Encode(map[string]any{"manifest": map[string]any{
			"schema_version": "ux-capture.v1", "id": bundleID,
			"source":    map[string]any{"kind": "scenario", "scenario": "sample-app", "route": "/settings?token=%5BREDACTED%5D"},
			"lenses":    map[string]any{},
			"artifacts": []any{map[string]any{"kind": "outline", "path": artifactPath, "size_bytes": len(outline), "sha256": hex.EncodeToString(digest[:])}},
		}})
	}))
	defer bas.Close()

	service := pageinspect.Service{
		ResolveURL: func(_ context.Context, scenario, key string) (string, error) {
			if scenario == "browser-automation-studio" && key == "API_PORT" {
				return bas.URL, nil
			}
			require.Equal(t, "sample-app", scenario)
			require.Equal(t, "UI_PORT", key)
			return "http://ui.example.test", nil
		},
		HTTPClient: bas.Client(),
	}
	path, connectHandler := pageconnect.NewPageServiceHandler(Handler{Service: service})
	router := mux.NewRouter()
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})
	server := httptest.NewServer(router)
	defer server.Close()
	client := pageconnect.NewPageServiceClient(server.Client(), server.URL)

	response, err := client.Inspect(context.Background(), connect.NewRequest(&pagev1.InspectRequest{Scenario: "sample-app", Route: "/settings?token=secret"}))
	require.NoError(t, err)
	require.Equal(t, "sample-app", response.Msg.GetScenario())
	require.Equal(t, "/settings?token=%5BREDACTED%5D", response.Msg.GetRoute())
	require.Equal(t, "http://ui.example.test/settings?token=%5BREDACTED%5D", response.Msg.GetUrl())
	require.Equal(t, bundleID, response.Msg.GetExecutionId())
	require.Equal(t, uint32(1), response.Msg.GetNodeCount())
	require.Len(t, response.Msg.GetTree().GetChildren(), 1)
	require.Equal(t, "unstamped", response.Msg.GetTree().GetChildren()[0].GetSource().GetStatus())

	_, err = client.Inspect(context.Background(), connect.NewRequest(&pagev1.InspectRequest{Scenario: "sample-app", Route: "https://outside.example/"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
