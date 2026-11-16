package pageinspect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
)

func TestInspectUsesCanonicalBundleAndResolvesL4IdentityWithoutGuessingUnstampedNodes(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	outline := []byte(`{"schema":"bas-agent-outline/v1","nodes":[{"id":"n1","kind":"action","tag":"button","name":"Save","text":"Save","selector":"#save","rect":{"x":2,"y":3,"width":40,"height":20}},{"id":"n2","kind":"text","tag":"div","text":"plain","selector":"#plain","rect":{"x":1,"y":2,"width":30,"height":10}}],"unsupported":[]}`)
	identity := []byte(`{"schema":"bas-component-identity/v1","element_count":2,"stamped_count":1,"components":[{"selector":"#save","asset":"react-component-library:Button","version":"2","stamp":"vite"}]}`)
	pixels := []byte("png")
	data := map[string][]byte{"outline/outline.json": outline, "identity/components.json": identity, "pixels/desktop.png": pixels}
	artifact := func(kind, path string, content []byte) map[string]any {
		digest := sha256.Sum256(content)
		return map[string]any{"kind": kind, "path": path, "size_bytes": len(content), "sha256": hex.EncodeToString(digest[:])}
	}
	calls := 0
	artifactStarted := make(chan struct{}, 1)
	slowArtifacts := false
	bas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if slowArtifacts {
				artifactStarted <- struct{}{}
				<-r.Context().Done()
				return
			}
			content, ok := data[strings.TrimPrefix(r.URL.Path, "/api/v1/ux-capture/bundles/123e4567-e89b-42d3-a456-426614174000/artifacts/")]
			require.True(t, ok, r.URL.Path)
			_, _ = w.Write(content)
			return
		}
		calls++
		require.Equal(t, "/api/v1/ux-capture/scenario", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input map[string]any
		require.NoError(t, json.Unmarshal(body, &input))
		require.Equal(t, "react-component-library", input["scenario"])
		require.Equal(t, "/coverage?state=error", input["route"])
		require.Equal(t, "[data-ready]", input["wait_selector"])
		manifest := map[string]any{
			"schema_version": "ux-capture.v1", "id": "123e4567-e89b-42d3-a456-426614174000",
			"source":    map[string]any{"kind": "scenario", "scenario": "react-component-library", "route": "/coverage?state=%5BREDACTED%5D"},
			"lenses":    map[string]any{"L0": map[string]any{"status": "present"}, "L1": map[string]any{"status": "absent", "reason": "fixture"}, "L2": map[string]any{"status": "present"}, "L3": map[string]any{"status": "partial", "reason": "fixture"}, "L4": map[string]any{"status": "present"}, "L5": map[string]any{"status": "present"}, "L6": map[string]any{"status": "absent", "reason": "fixture"}},
			"artifacts": []any{artifact("pixels", "pixels/desktop.png", pixels), artifact("outline", "outline/outline.json", outline), artifact("component-identity", "identity/components.json", identity)},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"bundle_dir": "/private/server/path", "manifest": manifest}))
	}))
	defer bas.Close()
	service := Service{RepoRoot: root, ResolveURL: func(_ context.Context, scenario, key string) (string, error) {
		if scenario == "browser-automation-studio" {
			return bas.URL, nil
		}
		require.Equal(t, "UI_PORT", key)
		return "http://scenario.test", nil
	}}
	result, err := service.Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: "/coverage?state=error", WaitSelector: "[data-ready]"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Empty(t, result.ScreenshotPath)
	require.Equal(t, bas.URL+"/api/v1/ux-capture/bundles/123e4567-e89b-42d3-a456-426614174000/artifacts/pixels/desktop.png", result.ScreenshotUrl)
	require.Equal(t, "/coverage?state=%5BREDACTED%5D", result.Route)
	require.Equal(t, uint32(2), result.NodeCount)
	require.Equal(t, uint32(1), result.StampedCount)
	require.Equal(t, uint32(1), result.ResolvedCount)
	require.Equal(t, "unstamped", result.Tree.Source.Status)
	require.Equal(t, "react-component-library:Button", result.Tree.Children[0].Source.LibraryId)
	require.Equal(t, "consumer-major-alias", result.Tree.Children[0].Source.ResolutionRule)
	require.Equal(t, "unstamped", result.Tree.Children[1].Source.Status)
	require.Contains(t, result.Tree.Children[0].Observation.AsMap()["data"].(map[string]any), "rclSourceSlot")
	require.NotEmpty(t, result.Warnings)

	data["outline/outline.json"] = append([]byte(nil), outline...)
	data["outline/outline.json"][0] = '['
	_, err = service.Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: "/coverage?state=error", WaitSelector: "[data-ready]"})
	require.ErrorContains(t, err, "digest does not match")
	data["outline/outline.json"] = outline

	// Artifact reads remain part of the caller's cancellation boundary. They
	// must not detach after capture has returned its durable bundle identity.
	slowArtifacts = true
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, inspectErr := service.Inspect(ctx, &pagev1.InspectRequest{Scenario: "react-component-library", Route: "/coverage?state=error", WaitSelector: "[data-ready]"})
		finished <- inspectErr
	}()
	<-artifactStarted
	cancel()
	require.ErrorIs(t, <-finished, context.Canceled)
}

func TestInspectRejectsNonConcreteRoutesAndOversizedWaitSelectorsBeforeDiscovery(t *testing.T) {
	for _, route := range []string{"", "https://example.test/", "//example.test/", "/assets/:id", "/*", "/\\example"} {
		_, err := (Service{}).Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: route})
		require.ErrorContains(t, err, "concrete same-origin route", route)
	}
	_, err := (Service{}).Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: "/", WaitSelector: string(make([]byte, 513))})
	require.ErrorContains(t, err, "wait selector must be at most 512 characters")
}
