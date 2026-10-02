package exports_service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/internal/testutil/databasemocks"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	exportsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports/exportsconnect"
)

// =============================================================================
// Test fakes
// =============================================================================

func newFakeRepo() *databasemocks.MockRepository {
	return databasemocks.NewMockRepository()
}

type fakeExecutor struct {
	exists bool
}

func (f *fakeExecutor) GetExecution(_ context.Context, _ uuid.UUID) (*database.ExecutionIndex, error) {
	if !f.exists {
		return nil, database.ErrNotFound
	}
	return &database.ExecutionIndex{}, nil
}

type fakeCatalog struct{}

func (fakeCatalog) GetWorkflow(_ context.Context, _ uuid.UUID) (*basapi.WorkflowSummary, error) {
	return nil, database.ErrNotFound
}

type fakeOpener struct {
	revealCalled, openCalled bool
	revealErr, openErr       error
	lastPath                 string
}

func (f *fakeOpener) Reveal(path string) error {
	f.revealCalled = true
	f.lastPath = path
	return f.revealErr
}

func (f *fakeOpener) OpenFolder(path string) error {
	f.openCalled = true
	f.lastPath = path
	return f.openErr
}

type fakeAIClient struct {
	resp string
	err  error
}

func (f fakeAIClient) ExecutePrompt(_ context.Context, _ string) (string, error) {
	return f.resp, f.err
}

func (fakeAIClient) Model() string { return "test-model" }

type fakeAIFactory struct{ client fakeAIClient }

func (f fakeAIFactory) CreateClient(_ ai.ClientOptions) ai.AIClient { return f.client }

// =============================================================================
// Harness
// =============================================================================

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type harness struct {
	repo     *databasemocks.MockRepository
	executor *fakeExecutor
	opener   *fakeOpener
	factory  *fakeAIFactory
	client   exportsconnect.ExportsServiceClient
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})

	repo := newFakeRepo()
	executor := &fakeExecutor{exists: true}
	opener := &fakeOpener{}
	factory := &fakeAIFactory{client: fakeAIClient{resp: "A cool caption."}}

	mount := Module(Deps{
		Repo:            repo,
		Executor:        executor,
		Catalog:         fakeCatalog{},
		AIClientFactory: factory,
		Opener:          opener,
		Logger:          logger,
	})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
	t.Cleanup(srv.Close)
	client := exportsconnect.NewExportsServiceClient(srv.Client(), srv.URL)
	return &harness{repo: repo, executor: executor, opener: opener, factory: factory, client: client}
}

// =============================================================================
// Module wiring
// =============================================================================

func TestModulePanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Repo: newFakeRepo(), Executor: &fakeExecutor{}, Catalog: fakeCatalog{}})
	})
}

func TestModulePanicsWithoutRepo(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Logger: logrus.New(), Executor: &fakeExecutor{}, Catalog: fakeCatalog{}})
	})
}

func TestModulePanicsWithoutExecutor(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Logger: logrus.New(), Repo: newFakeRepo(), Catalog: fakeCatalog{}})
	})
}

func TestModulePanicsWithoutCatalog(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Logger: logrus.New(), Repo: newFakeRepo(), Executor: &fakeExecutor{}})
	})
}

// =============================================================================
// CreateExport
// =============================================================================

func TestCreateExport_Happy(t *testing.T) {
	h := newHarness(t)
	execID := uuid.New()
	resp, err := h.client.CreateExport(context.Background(), connect.NewRequest(&exportsv1.CreateExportRequest{
		ExecutionId: execID.String(),
		Name:        "demo",
		Format:      "MP4",
	}))
	require.NoError(t, err)
	require.Equal(t, "created", resp.Msg.GetStatus())
	require.NotEmpty(t, resp.Msg.GetExportId())
	require.Equal(t, "mp4", resp.Msg.GetExport().GetFormat())
	require.Equal(t, execID.String(), resp.Msg.GetExport().GetExecutionId())
}

func TestCreateExport_InvalidExecutionID(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.CreateExport(context.Background(), connect.NewRequest(&exportsv1.CreateExportRequest{
		ExecutionId: "not-a-uuid",
		Name:        "demo",
		Format:      "mp4",
	}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

func TestCreateExport_ExecutionNotFound(t *testing.T) {
	h := newHarness(t)
	h.executor.exists = false
	_, err := h.client.CreateExport(context.Background(), connect.NewRequest(&exportsv1.CreateExportRequest{
		ExecutionId: uuid.New().String(),
		Name:        "demo",
		Format:      "mp4",
	}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestCreateExport_InvalidFormat(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.CreateExport(context.Background(), connect.NewRequest(&exportsv1.CreateExportRequest{
		ExecutionId: uuid.New().String(),
		Name:        "demo",
		Format:      "avi",
	}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

// =============================================================================
// GetExport / List / Status
// =============================================================================

func seedExport(t *testing.T, h *harness, name string) *database.ExportIndex {
	t.Helper()
	row := &database.ExportIndex{
		ID:          uuid.New(),
		ExecutionID: uuid.New(),
		Name:        name,
		Format:      "mp4",
		Status:      "completed",
		StorageURL:  "/tmp/exports/" + name + ".mp4",
	}
	require.NoError(t, h.repo.CreateExport(context.Background(), row))
	return row
}

func TestGetExport_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.GetExport(context.Background(), connect.NewRequest(&exportsv1.GetExportRequest{Id: row.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, row.ID.String(), resp.Msg.GetExport().GetId())
}

func TestGetExport_NotFound(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.GetExport(context.Background(), connect.NewRequest(&exportsv1.GetExportRequest{Id: uuid.New().String()}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	require.Equal(t, connect.CodeNotFound, connectErr.Code())
}

func TestListExports_FiltersByExecution(t *testing.T) {
	h := newHarness(t)
	a := seedExport(t, h, "a")
	_ = seedExport(t, h, "b")
	resp, err := h.client.ListExports(context.Background(), connect.NewRequest(&exportsv1.ListExportsRequest{
		ExecutionId: a.ExecutionID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Msg.GetTotal())
	require.Equal(t, a.ID.String(), resp.Msg.GetExports()[0].GetId())
}

func TestGetExportStatus_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.GetExportStatus(context.Background(), connect.NewRequest(&exportsv1.GetExportStatusRequest{Id: row.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, "completed", resp.Msg.GetStatus())
	require.Equal(t, "mp4", resp.Msg.GetFormat())
}

// =============================================================================
// Update / Delete
// =============================================================================

func TestUpdateExport_AppliesPartial(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.UpdateExport(context.Background(), connect.NewRequest(&exportsv1.UpdateExportRequest{
		Id:     row.ID.String(),
		Name:   "renamed",
		Status: "failed",
	}))
	require.NoError(t, err)
	require.Equal(t, "updated", resp.Msg.GetStatus())
	require.Equal(t, "renamed", resp.Msg.GetExport().GetName())
	require.Equal(t, "failed", resp.Msg.GetExport().GetStatus())
}

func TestDeleteExport_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.DeleteExport(context.Background(), connect.NewRequest(&exportsv1.DeleteExportRequest{Id: row.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, "deleted", resp.Msg.GetStatus())
	_, getErr := h.client.GetExport(context.Background(), connect.NewRequest(&exportsv1.GetExportRequest{Id: row.ID.String()}))
	require.Error(t, getErr)
}

// =============================================================================
// GenerateExportCaption
// =============================================================================

func TestGenerateExportCaption_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.GenerateExportCaption(context.Background(), connect.NewRequest(&exportsv1.GenerateExportCaptionRequest{
		Id: row.ID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, "A cool caption.", resp.Msg.GetCaption())
	require.Equal(t, "A cool caption.", resp.Msg.GetExport().GetAiCaption())
}

func TestGenerateExportCaption_FallsBackOnAIError(t *testing.T) {
	h := newHarness(t)
	h.factory.client = fakeAIClient{err: errors.New("AI provider down")}
	row := seedExport(t, h, "demo")
	resp, err := h.client.GenerateExportCaption(context.Background(), connect.NewRequest(&exportsv1.GenerateExportCaptionRequest{
		Id: row.ID.String(),
	}))
	require.NoError(t, err)
	require.Contains(t, resp.Msg.GetCaption(), "demo")
}

// =============================================================================
// Reveal / OpenFolder
// =============================================================================

func TestRevealExport_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.RevealExport(context.Background(), connect.NewRequest(&exportsv1.RevealExportRequest{Id: row.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, "revealed", resp.Msg.GetStatus())
	require.True(t, h.opener.revealCalled)
	require.Equal(t, row.StorageURL, h.opener.lastPath)
}

func TestRevealExport_NoStoragePath(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	row.StorageURL = ""
	require.NoError(t, h.repo.UpdateExport(context.Background(), row))
	_, err := h.client.RevealExport(context.Background(), connect.NewRequest(&exportsv1.RevealExportRequest{Id: row.ID.String()}))
	require.Error(t, err)
	var connectErr *connect.Error
	require.True(t, errors.As(err, &connectErr))
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
}

func TestOpenExportFolder_Happy(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "demo")
	resp, err := h.client.OpenExportFolder(context.Background(), connect.NewRequest(&exportsv1.OpenExportFolderRequest{Id: row.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, "opened", resp.Msg.GetStatus())
	require.True(t, h.opener.openCalled)
}
