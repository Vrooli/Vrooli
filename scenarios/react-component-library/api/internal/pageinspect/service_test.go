package pageinspect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
)

func TestInspectUsesCanonicalBundleAndResolvesL4IdentityWithoutGuessingUnstampedNodes(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	bundleDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(bundleDir, "outline"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(bundleDir, "identity"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(bundleDir, "pixels"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "outline", "outline.json"), []byte(`{"schema":"bas-agent-outline/v1","nodes":[{"id":"n1","kind":"action","tag":"button","name":"Save","text":"Save","selector":"#save","rect":{"x":2,"y":3,"width":40,"height":20}},{"id":"n2","kind":"text","tag":"div","text":"plain","selector":"#plain","rect":{"x":1,"y":2,"width":30,"height":10}}],"unsupported":[]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "identity", "components.json"), []byte(`{"schema":"bas-component-identity/v1","element_count":2,"stamped_count":1,"components":[{"selector":"#save","asset":"react-component-library:Button","version":"2","stamp":"vite"}]}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "pixels", "desktop.png"), []byte("png"), 0o600))
	calls := 0
	bas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			"schema_version": "ux-capture.v1", "id": "bundle-id",
			"source":    map[string]any{"kind": "scenario", "scenario": "react-component-library", "route": "/coverage?state=%5BREDACTED%5D"},
			"lenses":    map[string]any{"L0": map[string]any{"status": "present"}, "L1": map[string]any{"status": "absent", "reason": "fixture"}, "L2": map[string]any{"status": "present"}, "L3": map[string]any{"status": "partial", "reason": "fixture"}, "L4": map[string]any{"status": "present"}, "L5": map[string]any{"status": "present"}, "L6": map[string]any{"status": "absent", "reason": "fixture"}},
			"artifacts": []any{map[string]any{"kind": "pixels", "path": "pixels/desktop.png"}, map[string]any{"kind": "outline", "path": "outline/outline.json"}, map[string]any{"kind": "component-identity", "path": "identity/components.json"}},
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"bundle_dir": bundleDir, "manifest": manifest}))
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
	require.Equal(t, filepath.Join(bundleDir, "pixels", "desktop.png"), result.ScreenshotPath)
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
}

func TestInspectRejectsNonConcreteRoutesAndOversizedWaitSelectorsBeforeDiscovery(t *testing.T) {
	for _, route := range []string{"", "https://example.test/", "//example.test/", "/assets/:id", "/*", "/\\example"} {
		_, err := (Service{}).Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: route})
		require.ErrorContains(t, err, "concrete same-origin route", route)
	}
	_, err := (Service{}).Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: "/", WaitSelector: string(make([]byte, 513))})
	require.ErrorContains(t, err, "wait selector must be at most 512 characters")
}
