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
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"
	visualv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison/workspacevisualcomparison_v1connect"
	"google.golang.org/protobuf/proto"
	"react-component-library/handlers/workspaces"
	"react-component-library/internal/componenttests"
	domain "react-component-library/internal/workspaces"
)

type captureStub struct {
	request *capturev1.CaptureRequest
	result  *capturev1.CaptureResponse
	err     error
}

func (s *captureStub) CaptureSurface(_ context.Context, request *capturev1.CaptureRequest) (*capturev1.CaptureResponse, error) {
	s.request = request
	return s.result, s.err
}

type compareStub struct {
	baseline, current           string
	imageBaseline, imageCurrent []byte
	result                      *diffv1.DiffResult
	heatmap                     []byte
	err                         error
}

func (s *compareStub) CompareImages(_ context.Context, baseline, current []byte) (*diffv1.DiffResult, error) {
	s.imageBaseline, s.imageCurrent = append([]byte(nil), baseline...), append([]byte(nil), current...)
	return s.result, s.err
}

func (s *compareStub) FetchScreenshot(_ context.Context, _ domain.CaptureReceipt) ([]byte, error) {
	return []byte("current capture image"), s.err
}

func (s *compareStub) Compare(_ context.Context, b, c domain.CaptureReceipt) (*diffv1.DiffResult, error) {
	s.baseline, s.current = b.ID, c.ID
	return s.result, s.err
}

func (s *compareStub) FetchHeatmap(_ context.Context, _ string) ([]byte, error) {
	return s.heatmap, nil
}

func TestWorkspaceCaptureConnectPreservesRevisionReceiptAndAttemptSemantics(t *testing.T) {
	ctx := context.Background()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, workspaces.Migrate(ctx, sqlDB))
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Capture contract"})
	require.NoError(t, err)
	page := created.Pages[0]
	display := page.Displays[0]
	state := display.States[0]
	producer := &captureStub{err: componenttests.CaptureTerminalError{Err: fmt.Errorf("browser failed")}}
	router := mux.NewRouter()
	workspaces.ModuleFromRepositoryWithCapture(repo, log.New(io.Discard, "", 0), producer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)
	request := func(attempt string, revision int32) *connect.Response[workspacev1.CaptureWorkspaceStateResponse] {
		r, e := client.CaptureWorkspaceState(ctx, connect.NewRequest(&workspacev1.CaptureWorkspaceStateRequest{WorkspaceId: created.Id, ExpectedRevision: revision, PageId: page.Id, DisplayId: display.Id, StateId: state.Id, AttemptId: attempt}))
		if e != nil {
			return nil
		}
		return r
	}
	_, err = client.CaptureWorkspaceState(ctx, connect.NewRequest(&workspacev1.CaptureWorkspaceStateRequest{WorkspaceId: created.Id, ExpectedRevision: created.Revision + 1, PageId: page.Id, DisplayId: display.Id, StateId: state.Id, AttemptId: "3db6ed20-17e4-4f82-8c24-5f39b1ac624e"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	require.Nil(t, producer.request)
	_, err = client.CaptureWorkspaceState(ctx, connect.NewRequest(&workspacev1.CaptureWorkspaceStateRequest{WorkspaceId: created.Id, ExpectedRevision: created.Revision, PageId: page.Id, DisplayId: display.Id, StateId: state.Id, AttemptId: "3db6ed20-17e4-4f82-8c24-5f39b1ac624e"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, "failed", ce.Meta().Get("attempt-state"))
	producer.err = nil
	producer.result = &capturev1.CaptureResponse{ExecutionId: "exec-1", Artifacts: []*capturev1.CaptureArtifact{{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Reference: "bas-capture://exec-1/screenshot"}, {Type: capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY, Reference: "bas-capture://exec-1/accessibility"}}}
	response := request("4db6ed20-17e4-4f82-8c24-5f39b1ac624e", created.Revision)
	require.NotNil(t, response)
	require.Equal(t, "exec-1", response.Msg.CaptureId)
	require.Equal(t, "bas-capture://exec-1/screenshot", response.Msg.Evidence.Screenshot.Reference)
	reloaded, err := repo.Get(ctx, created.Id)
	require.NoError(t, err)
	require.Equal(t, workspacev1.StateStatus_STATE_STATUS_CAPTURED, reloaded.Pages[0].Displays[0].States[0].Status)
	evidence, err := client.GetWorkspaceStateCapture(ctx, connect.NewRequest(&workspacev1.GetWorkspaceStateCaptureRequest{WorkspaceId: created.Id, StateId: state.Id}))
	require.NoError(t, err)
	require.Equal(t, "exec-1", evidence.Msg.CaptureId)
	receipts, err := client.ListWorkspaceCaptureReceipts(ctx, connect.NewRequest(&workspacev1.ListWorkspaceCaptureReceiptsRequest{WorkspaceId: created.Id, StateId: state.Id}))
	require.NoError(t, err)
	require.Len(t, receipts.Msg.Receipts, 1)
	require.Equal(t, "exec-1", receipts.Msg.Receipts[0].CaptureId)

	// A composition edit invalidates the current capture projection, while the
	// immutable receipt remains available as history.
	edited := proto.Clone(reloaded).(*workspacev1.Workspace)
	edited.Pages[0].Displays[0].States[0].ArgsJson = `{"variant":"edited"}`
	_, err = repo.Save(ctx, &workspacev1.SaveWorkspaceRequest{Workspace: edited, BaseRevision: reloaded.GetRevision()})
	require.NoError(t, err)
	_, err = client.GetWorkspaceStateCapture(ctx, connect.NewRequest(&workspacev1.GetWorkspaceStateCaptureRequest{WorkspaceId: created.Id, StateId: state.Id}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	receipts, err = client.ListWorkspaceCaptureReceipts(ctx, connect.NewRequest(&workspacev1.ListWorkspaceCaptureReceiptsRequest{WorkspaceId: created.Id, StateId: state.Id}))
	require.NoError(t, err)
	require.Len(t, receipts.Msg.GetReceipts(), 1)
	require.Equal(t, "exec-1", receipts.Msg.GetReceipts()[0].GetCaptureId())
}

func TestSourceBaselineRequiresHumanAcceptanceAndPersistsSourceAnchoredComparison(t *testing.T) {
	ctx := context.Background()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, workspaces.Migrate(ctx, sqlDB))
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Source drift"})
	require.NoError(t, err)
	page := created.GetPages()[0]
	display := page.GetDisplays()[0]
	state := display.GetStates()[0]
	sourcePNG := append([]byte("\x89PNG\r\n\x1a\n"), []byte("source-image")...)
	asset, err := repo.CreateWorkspaceAsset(ctx, domain.AssetInput{WorkspaceID: created.GetId(), Name: "source.png", Mime: "image/png", Data: sourcePNG})
	require.NoError(t, err)
	_, err = repo.AttachCaptureEvidence(ctx, created.GetId(), state.GetId(), created.GetRevision(), "attempt-1", "token-1", "capture-1", `{"schemaVersion":"rcl-workspace-state-capture/v1","screenshot":{"reference":"bas-capture://capture-1/screenshot"}}`, "2026-11-22T12:00:00Z")
	require.NoError(t, err)
	receipts, err := repo.ListCaptureReceipts(ctx, created.GetId(), state.GetId())
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	comparer := &compareStub{result: &diffv1.DiffResult{Verdict: "changed", ChangedFraction: .25, HeatmapRef: "diff/heat.png"}, heatmap: []byte("heatmap")}
	router := mux.NewRouter()
	workspaces.ModuleFromRepositoryWithCaptureAndComparer(repo, log.New(io.Discard, "", 0), nil, comparer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	workspaceClient := workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)
	visualClient := visualconnect.NewWorkspaceVisualComparisonServiceClient(server.Client(), server.URL)
	baseline, err := workspaceClient.CreateWorkspaceSourceBaseline(ctx, connect.NewRequest(&workspacev1.CreateWorkspaceSourceBaselineRequest{WorkspaceId: created.GetId(), StateId: state.GetId(), AssetId: asset.ID, SourceLocator: "https://design.example/screen"}))
	require.NoError(t, err)
	require.Empty(t, baseline.Msg.GetAcceptedAt())
	listed, err := workspaceClient.ListWorkspaceSourceBaselines(ctx, connect.NewRequest(&workspacev1.ListWorkspaceSourceBaselinesRequest{WorkspaceId: created.GetId(), StateId: state.GetId()}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetCandidates(), 1)
	require.Nil(t, listed.Msg.GetAccepted())
	_, err = visualClient.CompareWorkspaceSourceBaseline(ctx, connect.NewRequest(&visualv1.CompareWorkspaceSourceBaselineRequest{WorkspaceId: created.GetId(), StateId: state.GetId(), CurrentReceiptId: receipts[0].ID}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Empty(t, comparer.imageBaseline)
	accepted, err := workspaceClient.AcceptWorkspaceSourceBaseline(ctx, connect.NewRequest(&workspacev1.AcceptWorkspaceSourceBaselineRequest{WorkspaceId: created.GetId(), StateId: state.GetId(), BaselineId: baseline.Msg.GetId(), AcceptedBy: "reviewer"}))
	require.NoError(t, err)
	require.Equal(t, "reviewer", accepted.Msg.GetAcceptedBy())
	result, err := visualClient.CompareWorkspaceSourceBaseline(ctx, connect.NewRequest(&visualv1.CompareWorkspaceSourceBaselineRequest{WorkspaceId: created.GetId(), StateId: state.GetId(), CurrentReceiptId: receipts[0].ID}))
	require.NoError(t, err)
	require.Equal(t, baseline.Msg.GetId(), result.Msg.GetSourceBaselineId())
	require.Equal(t, asset.ID, result.Msg.GetSourceAssetId())
	require.Equal(t, receipts[0].ID, result.Msg.GetCurrentReceiptId())
	require.Equal(t, sourcePNG, comparer.imageBaseline)
	require.Equal(t, []byte("current capture image"), comparer.imageCurrent)
	history, err := visualClient.ListWorkspaceSourceBaselineComparisons(ctx, connect.NewRequest(&visualv1.ListWorkspaceSourceBaselineComparisonsRequest{WorkspaceId: created.GetId(), StateId: state.GetId()}))
	require.NoError(t, err)
	require.Len(t, history.Msg.GetComparisons(), 1)
	require.Equal(t, result.Msg.GetComparisonId(), history.Msg.GetComparisons()[0].GetComparisonId())
	heatmapURL := fmt.Sprintf("%s/api/workspaces/%s/states/%s/visual-comparisons/%s/heatmap", server.URL, created.GetId(), state.GetId(), result.Msg.GetComparisonId())
	response, err := server.Client().Get(heatmapURL)
	require.NoError(t, err)
	defer response.Body.Close()
	heatmap, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, comparer.heatmap, heatmap)
}

func TestWorkspaceCaptureRetryFromNewTabJoinsActiveBASAttempt(t *testing.T) {
	ctx := context.Background()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, workspaces.Migrate(ctx, sqlDB))
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Recover capture"})
	require.NoError(t, err)
	page, display, state := created.Pages[0], created.Pages[0].Displays[0], created.Pages[0].Displays[0].States[0]
	producer := &captureStub{err: componenttests.CaptureUnresolvedError{Err: fmt.Errorf("BAS response was interrupted")}}
	router := mux.NewRouter()
	workspaces.ModuleFromRepositoryWithCapture(repo, log.New(io.Discard, "", 0), producer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)
	request := func(attempt string) *connect.Request[workspacev1.CaptureWorkspaceStateRequest] {
		return connect.NewRequest(&workspacev1.CaptureWorkspaceStateRequest{WorkspaceId: created.Id, ExpectedRevision: created.Revision, PageId: page.Id, DisplayId: display.Id, StateId: state.Id, AttemptId: attempt})
	}
	const originalAttempt = "3db6ed20-17e4-4f82-8c24-5f39b1ac624e"
	const newTabAttempt = "4db6ed20-17e4-4f82-8c24-5f39b1ac624e"
	_, err = client.CaptureWorkspaceState(ctx, request(originalAttempt))
	require.Error(t, err)
	require.Equal(t, connect.CodeDeadlineExceeded, connect.CodeOf(err))
	originalToken := producer.request.GetRequestToken()

	producer.err = nil
	producer.result = &capturev1.CaptureResponse{ExecutionId: "exec-recovered", Artifacts: []*capturev1.CaptureArtifact{
		{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Reference: "bas-capture://exec-recovered/screenshot"},
		{Type: capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY, Reference: "bas-capture://exec-recovered/accessibility"},
	}}
	response, err := client.CaptureWorkspaceState(ctx, request(newTabAttempt))
	require.NoError(t, err)
	require.Equal(t, originalToken, producer.request.GetRequestToken(), "a new tab must resume the server's original BAS request")
	require.Equal(t, originalAttempt, response.Msg.GetEvidence().GetAttemptId())

	receipts, err := client.ListWorkspaceCaptureReceipts(ctx, connect.NewRequest(&workspacev1.ListWorkspaceCaptureReceiptsRequest{WorkspaceId: created.Id, StateId: state.Id}))
	require.NoError(t, err)
	require.Len(t, receipts.Msg.GetReceipts(), 1)
	require.Equal(t, originalAttempt, receipts.Msg.GetReceipts()[0].GetAttemptId())
}

func TestWorkspaceVisualComparisonConnectPersistsMetricsAndKeepsHeatmapBinaryREST(t *testing.T) {
	ctx := context.Background()
	sqlDB := db.NewSQLite(t)
	require.NoError(t, workspaces.Migrate(ctx, sqlDB))
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Compare"})
	require.NoError(t, err)
	stateID := created.Pages[0].Displays[0].States[0].Id
	add := func(attempt, capture string) domain.CaptureReceipt {
		e := fmt.Sprintf(`{"schemaVersion":"rcl-workspace-state-capture/v1","attemptId":%q,"sourceRevision":%d,"screenshot":{"reference":"bas-capture://%s/screenshot"}}`, attempt, created.Revision, capture)
		updated, e2 := repo.AttachCaptureEvidence(ctx, created.Id, stateID, created.Revision, attempt, "token-"+attempt, capture, e, "2026-11-20T12:00:00Z")
		require.NoError(t, e2)
		created = updated
		items, e2 := repo.ListCaptureReceipts(ctx, created.Id, stateID)
		require.NoError(t, e2)
		for _, item := range items {
			if item.AttemptID == attempt {
				return item
			}
		}
		t.Fatalf("receipt for attempt %q was not returned", attempt)
		return domain.CaptureReceipt{}
	}
	baseline := add("attempt-1", "exec-1")
	current := add("attempt-2", "exec-2")
	comparer := &compareStub{result: &diffv1.DiffResult{Verdict: "changed", ChangedFraction: .25, HeatmapRef: "diff/heat.png"}, heatmap: []byte("heatmap-png")}
	router := mux.NewRouter()
	workspaces.ModuleFromRepositoryWithCaptureAndComparer(repo, log.New(io.Discard, "", 0), nil, comparer).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := visualconnect.NewWorkspaceVisualComparisonServiceClient(server.Client(), server.URL)
	result, err := client.CompareWorkspaceCaptureReceipts(ctx, connect.NewRequest(&visualv1.CompareWorkspaceCaptureReceiptsRequest{WorkspaceId: created.Id, StateId: stateID, BaselineReceiptId: baseline.ID, CurrentReceiptId: current.ID}))
	require.NoError(t, err)
	require.Equal(t, baseline.ID, comparer.baseline)
	list, err := client.ListWorkspaceVisualComparisons(ctx, connect.NewRequest(&visualv1.ListWorkspaceVisualComparisonsRequest{WorkspaceId: created.Id, StateId: stateID}))
	require.NoError(t, err)
	require.Len(t, list.Msg.Comparisons, 1)
	require.Equal(t, result.Msg.ComparisonId, list.Msg.Comparisons[0].ComparisonId)
	heatmap, err := server.Client().Get(fmt.Sprintf("%s/api/workspaces/%s/states/%s/visual-comparisons/%s/heatmap", server.URL, created.Id, stateID, result.Msg.ComparisonId))
	require.NoError(t, err)
	defer heatmap.Body.Close()
	require.Equal(t, http.StatusOK, heatmap.StatusCode)
	bytes, err := io.ReadAll(heatmap.Body)
	require.NoError(t, err)
	require.Equal(t, comparer.heatmap, bytes)
}
