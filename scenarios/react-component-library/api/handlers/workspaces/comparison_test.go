package workspaces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	"google.golang.org/protobuf/encoding/protojson"

	domain "react-component-library/internal/workspaces"
)

func TestBASImageToolsCompareUsesValidatedOwnerArtifacts(t *testing.T) {
	ctx := context.Background()
	png := []byte("\x89PNG\r\n\x1a\nmock-png")
	var metadataRequests, screenshotRequests, comparisonRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/browser_automation_studio.v1.ExecutionsService/GetExecutionScreenshots":
			metadataRequests++
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var request basapi.GetExecutionScreenshotsRequest
			require.NoError(t, protojson.Unmarshal(body, &request))
			require.NotEmpty(t, request.GetExecutionId())
			result, err := protojson.Marshal(&executionv1.GetScreenshotsResponse{
				ExecutionId: request.GetExecutionId(),
				Screenshots: []*executionv1.ExecutionScreenshot{{NodeId: "target", Screenshot: &domainv1.TimelineScreenshot{
					ArtifactId: "artifact-1", Url: "/api/v1/screenshots/" + request.GetExecutionId() + "/artifacts/screenshots/target.png",
					Width: 1180, Height: 780, ContentType: "image/png",
				}}},
			})
			require.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(result)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/screenshots/"):
			screenshotRequests++
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/diff/compare":
			comparisonRequests++
			require.NoError(t, r.ParseMultipartForm(1<<20))
			base, _, err := r.FormFile("base")
			require.NoError(t, err)
			defer base.Close()
			baseBytes, err := io.ReadAll(base)
			require.NoError(t, err)
			current, _, err := r.FormFile("compare")
			require.NoError(t, err)
			defer current.Close()
			currentBytes, err := io.ReadAll(current)
			require.NoError(t, err)
			require.Equal(t, png, baseBytes)
			require.Equal(t, png, currentBytes)
			_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.NotEmpty(t, params["boundary"])
			result, err := protojson.Marshal(&diffv1.DiffResult{Verdict: "different", ChangedFraction: 0.12, HeatmapRef: "diff/heat-123.png"})
			require.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(result)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	resolve := func(_ context.Context, scenario string) (string, error) {
		if scenario != "image-tools" && scenario != "browser-automation-studio" {
			return "", fmt.Errorf("unexpected scenario %s", scenario)
		}
		return server.URL, nil
	}
	client := server.Client()
	compare := &BASImageToolsCompare{
		Screenshots: screenshotBytesResolver{client: client, resolve: resolve},
		ImageTools:  imageToolsComparator{client: client, resolve: resolve},
	}
	makeReceipt := func(id string) domain.CaptureReceipt {
		return domain.CaptureReceipt{ID: "receipt-" + id, CaptureID: "execution-" + id, EvidenceJSON: fmt.Sprintf(`{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://execution-%s/screenshot"}}`, id)}
	}
	result, err := compare.Compare(ctx, makeReceipt("base"), makeReceipt("current"))
	require.NoError(t, err)
	require.Equal(t, "diff/heat-123.png", result.GetHeatmapRef())
	require.Equal(t, 2, metadataRequests)
	require.Equal(t, 2, screenshotRequests)
	require.Equal(t, 1, comparisonRequests)
}

func TestScreenshotResolverRejectsCallerControlledURLsAndWrongCaptureIdentity(t *testing.T) {
	resolver := screenshotBytesResolver{resolve: func(context.Context, string) (string, error) { return "http://127.0.0.1", nil }}
	bad := domain.CaptureReceipt{CaptureID: "execution-1", EvidenceJSON: `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"http://127.0.0.1/private.png"}}`}
	_, err := resolver.Resolve(context.Background(), bad)
	require.ErrorContains(t, err, "reference is invalid")
	wrongExecution := domain.CaptureReceipt{CaptureID: "execution-1", EvidenceJSON: `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://execution-2/screenshot"}}`}
	_, err = resolver.Resolve(context.Background(), wrongExecution)
	require.ErrorContains(t, err, "reference is invalid")
}

func TestScreenshotResolverRejectsMetadataOutsideExecutionRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var request basapi.GetExecutionScreenshotsRequest
		_ = protojson.Unmarshal(body, &request)
		response, _ := protojson.Marshal(&executionv1.GetScreenshotsResponse{ExecutionId: request.GetExecutionId(), Screenshots: []*executionv1.ExecutionScreenshot{{NodeId: "target", Screenshot: &domainv1.TimelineScreenshot{
			Url: "http://attacker.invalid/api/v1/screenshots/execution-1/image.png", ContentType: "image/png", Width: 1180, Height: 780,
		}}}})
		_, _ = w.Write(response)
	}))
	defer server.Close()
	resolver := screenshotBytesResolver{client: server.Client(), resolve: func(context.Context, string) (string, error) { return server.URL, nil }}
	receipt := domain.CaptureReceipt{CaptureID: "execution-1", EvidenceJSON: `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://execution-1/screenshot"}}`}
	_, err := resolver.Resolve(context.Background(), receipt)
	require.ErrorContains(t, err, "outside the capture owner")
}

var _ = json.Valid
