package workspaces_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	visualv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison/workspacevisualcomparison_v1connect"
	"react-component-library/handlers/workspaces"
	domain "react-component-library/internal/workspaces"
)

type artifactFailureComparer struct {
	compareErr, heatmapErr, screenshotErr error
	image                                 []byte
}

func (c artifactFailureComparer) Compare(context.Context, domain.CaptureReceipt, domain.CaptureReceipt) (*diffv1.DiffResult, error) {
	if c.compareErr != nil {
		return nil, c.compareErr
	}
	return &diffv1.DiffResult{Verdict: "same"}, nil
}

func (c artifactFailureComparer) FetchHeatmap(context.Context, string) ([]byte, error) {
	return c.image, c.heatmapErr
}

func (c artifactFailureComparer) FetchScreenshot(context.Context, domain.CaptureReceipt) ([]byte, error) {
	return c.image, c.screenshotErr
}

func setupArtifactFailureServer(t *testing.T, comparer workspaces.VisualCompareRunner) (*httptest.Server, *domain.SQLiteRepository, *workspacev1.Workspace) {
	t.Helper()
	ctx := context.Background()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, workspaces.Migrate(ctx, sqlDB))
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Artifact failure paths"})
	require.NoError(t, err)
	router := mux.NewRouter()
	workspaces.ModuleFromRepositoryWithCaptureAndComparer(repo, log.New(io.Discard, "", 0), nil, comparer).Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server, repo, created
}

func seedArtifactRecords(t *testing.T, repo *domain.SQLiteRepository, workspace *workspacev1.Workspace) (stateID string, receipt domain.CaptureReceipt, comparison domain.VisualComparison) {
	t.Helper()
	ctx := context.Background()
	stateID = workspace.GetPages()[0].GetDisplays()[0].GetStates()[0].GetId()
	evidence := `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://exec-one/screenshot"}}`
	updated, err := repo.AttachCaptureEvidence(ctx, workspace.GetId(), stateID, workspace.GetRevision(), "attempt-one", "token-one", "exec-one", evidence, "2026-11-20T12:00:00Z")
	require.NoError(t, err)
	workspace.Revision = updated.GetRevision()
	receipts, err := repo.ListCaptureReceipts(ctx, workspace.GetId(), stateID)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	receipt = receipts[0]
	otherEvidence := `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://exec-two/screenshot"}}`
	updated, err = repo.AttachCaptureEvidence(ctx, workspace.GetId(), stateID, workspace.GetRevision(), "attempt-two", "token-two", "exec-two", otherEvidence, "2026-11-20T12:01:00Z")
	require.NoError(t, err)
	workspace.Revision = updated.GetRevision()
	receipts, err = repo.ListCaptureReceipts(ctx, workspace.GetId(), stateID)
	require.NoError(t, err)
	comparison, err = repo.CreateVisualComparison(ctx, workspace.GetId(), stateID, receipt.ID, receipts[0].ID, `{"verdict":"same"}`, "diff/heat.png", "2026-11-20T12:02:00Z")
	require.NoError(t, err)
	return stateID, receipt, comparison
}

func TestWorkspaceArtifactRoutesMapUnavailableAndMalformedArtifacts(t *testing.T) {
	server, repo, workspace := setupArtifactFailureServer(t, nil)
	stateID, _, comparison := seedArtifactRecords(t, repo, workspace)
	base := fmt.Sprintf("%s/api/workspaces/%s/states/%s", server.URL, workspace.GetId(), stateID)
	client := server.Client()

	invalid, err := client.Get(fmt.Sprintf("%s/api/workspaces/%%20bad%%20/states/%s/capture-receipts/receipt/screenshot", server.URL, stateID))
	require.NoError(t, err)
	invalid.Body.Close()
	require.Equal(t, http.StatusBadRequest, invalid.StatusCode)

	missing, err := client.Get(base + "/capture-receipts/receipt-missing/screenshot")
	require.NoError(t, err)
	missing.Body.Close()
	require.Equal(t, http.StatusNotFound, missing.StatusCode)

	screenshot, err := client.Get(base + "/capture-receipts/" + mustReceiptID(t, repo, workspace.GetId(), stateID) + "/screenshot")
	require.NoError(t, err)
	screenshot.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, screenshot.StatusCode, "missing screenshot fetch support is surfaced as unavailable")

	heatmap, err := client.Get(base + "/visual-comparisons/" + comparison.ID + "/heatmap")
	require.NoError(t, err)
	heatmap.Body.Close()
	require.Equal(t, http.StatusNotFound, heatmap.StatusCode, "missing heatmap fetch support is surfaced as not found")
}

func mustReceiptID(t *testing.T, repo *domain.SQLiteRepository, workspaceID, stateID string) string {
	t.Helper()
	receipts, err := repo.ListCaptureReceipts(context.Background(), workspaceID, stateID)
	require.NoError(t, err)
	require.NotEmpty(t, receipts)
	return receipts[0].ID
}

func TestWorkspaceArtifactRoutesMapFetcherFailuresAndRejectUnsupportedScreenshot(t *testing.T) {
	comparer := artifactFailureComparer{heatmapErr: fmt.Errorf("upstream private detail"), screenshotErr: fmt.Errorf("upstream private detail")}
	server, repo, workspace := setupArtifactFailureServer(t, comparer)
	stateID, receipt, comparison := seedArtifactRecords(t, repo, workspace)
	base := fmt.Sprintf("%s/api/workspaces/%s/states/%s", server.URL, workspace.GetId(), stateID)
	client := server.Client()

	screenshot, err := client.Get(base + "/capture-receipts/" + receipt.ID + "/screenshot")
	require.NoError(t, err)
	body, err := io.ReadAll(screenshot.Body)
	screenshot.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusBadGateway, screenshot.StatusCode)
	require.NotContains(t, string(body), "upstream private detail")

	heatmap, err := client.Get(base + "/visual-comparisons/" + comparison.ID + "/heatmap")
	require.NoError(t, err)
	body, err = io.ReadAll(heatmap.Body)
	heatmap.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusBadGateway, heatmap.StatusCode)
	require.NotContains(t, string(body), "upstream private detail")

	server, repo, workspace = setupArtifactFailureServer(t, artifactFailureComparer{image: []byte("not an image")})
	stateID, receipt, _ = seedArtifactRecords(t, repo, workspace)
	response, err := server.Client().Get(fmt.Sprintf("%s/api/workspaces/%s/states/%s/capture-receipts/%s/screenshot", server.URL, workspace.GetId(), stateID, receipt.ID))
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, http.StatusBadGateway, response.StatusCode)
}

func TestVisualComparisonConnectReportsComparerAndReceiptFailures(t *testing.T) {
	server, repo, workspace := setupArtifactFailureServer(t, artifactFailureComparer{compareErr: fmt.Errorf("private comparer detail")})
	stateID, receipt, _ := seedArtifactRecords(t, repo, workspace)
	client := visualconnect.NewWorkspaceVisualComparisonServiceClient(server.Client(), server.URL)

	_, err := client.CompareWorkspaceCaptureReceipts(context.Background(), connect.NewRequest(&visualv1.CompareWorkspaceCaptureReceiptsRequest{WorkspaceId: workspace.GetId(), StateId: stateID, BaselineReceiptId: receipt.ID, CurrentReceiptId: receipt.ID}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.CompareWorkspaceCaptureReceipts(context.Background(), connect.NewRequest(&visualv1.CompareWorkspaceCaptureReceiptsRequest{WorkspaceId: workspace.GetId(), StateId: stateID, BaselineReceiptId: receipt.ID, CurrentReceiptId: "receipt-missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	// Use a second valid receipt so the operation reaches the injected comparer.
	receipts, err := repo.ListCaptureReceipts(context.Background(), workspace.GetId(), stateID)
	require.NoError(t, err)
	require.Len(t, receipts, 2)
	_, err = client.CompareWorkspaceCaptureReceipts(context.Background(), connect.NewRequest(&visualv1.CompareWorkspaceCaptureReceiptsRequest{WorkspaceId: workspace.GetId(), StateId: stateID, BaselineReceiptId: receipts[0].ID, CurrentReceiptId: receipts[1].ID}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "private comparer detail")
}
