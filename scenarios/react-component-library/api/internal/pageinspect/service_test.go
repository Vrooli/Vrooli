package pageinspect

import (
	"context"
	"github.com/stretchr/testify/require"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectSharesCaptureAndResolvesRealSourceWithoutGuessingUnstampedNodes(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	calls := 0
	bas := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Contains(t, r.URL.Path, "CaptureService/Capture")
		input := &capturev1.CaptureRequest{}
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(body, input))
		require.Equal(t, "http://scenario.test/coverage?state=error", input.Url)
		require.True(t, input.InlineDomTree)
		require.Equal(t, "[data-ready]", input.WaitFor.GetSelector())
		result := &capturev1.CaptureResponse{ExecutionId: "single-session", DomTreeJson: `{"tagName":"BODY","children":[{"tagName":"BUTTON","data":{"rclAsset":"react-component-library:Button","rclVersion":"2"},"computed":{"color":"rgb(1, 2, 3)"},"rect":{"x":10}},{"tagName":"DIV","data":{"rclAsset":"unknown","rclVersion":"1"}}]}`, Artifacts: []*capturev1.CaptureArtifact{
			{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "old.png", Metadata: map[string]string{"view_url": "/old.png"}},
			{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "final.png", Primary: true, Metadata: map[string]string{"view_url": "/final.png"}},
		}}
		raw, err := proto.Marshal(result)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(raw)
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
	require.Equal(t, "final.png", result.ScreenshotPath)
	require.Equal(t, bas.URL+"/final.png", result.ScreenshotUrl)
	require.Equal(t, uint32(3), result.NodeCount)
	require.Equal(t, uint32(2), result.StampedCount)
	require.Equal(t, uint32(1), result.ResolvedCount)
	require.Equal(t, "unstamped", result.Tree.Source.Status)
	button := result.Tree.Children[0]
	require.Equal(t, "react-component-library:Button", button.Source.LibraryId)
	require.Equal(t, "consumer-major-alias", button.Source.ResolutionRule)
	_, err = os.Stat(button.Source.SourcePath)
	require.NoError(t, err)
	require.Equal(t, "rgb(1, 2, 3)", button.Observation.AsMap()["computed"].(map[string]any)["color"])
	require.Equal(t, "unresolved", result.Tree.Children[1].Source.Status)
	require.NotEmpty(t, result.Tree.Children[1].Source.Reason)
	require.NotEmpty(t, result.Warnings)
}

func TestInspectRejectsNonConcreteRoutesBeforeDiscovery(t *testing.T) {
	for _, route := range []string{"", "https://example.test/", "//example.test/", "/assets/:id", "/*", "/\\example"} {
		_, err := (Service{}).Inspect(context.Background(), &pagev1.InspectRequest{Scenario: "react-component-library", Route: route})
		require.ErrorContains(t, err, "concrete same-origin route", route)
	}
}
