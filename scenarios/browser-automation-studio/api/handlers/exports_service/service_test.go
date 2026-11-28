package exports_service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/internal/testutil/databasemocks"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/database"
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

type fakeCaptionClient struct {
	resp        string
	err         error
	role        string
	prompt      string
	operationID string
}

func (f *fakeCaptionClient) ExecutePromptWithRole(ctx context.Context, role, prompt string) (string, error) {
	f.role, f.prompt = role, prompt
	f.operationID = aihandlers.GatewayRootOperationIDFromContext(ctx)
	return f.resp, f.err
}

// =============================================================================
// Harness
// =============================================================================

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type harness struct {
	repo     *databasemocks.MockRepository
	executor *fakeExecutor
	opener   *fakeOpener
	caption  *fakeCaptionClient
	client   exportsconnect.ExportsServiceClient
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})

	repo := newFakeRepo()
	executor := &fakeExecutor{exists: true}
	opener := &fakeOpener{}
	caption := &fakeCaptionClient{resp: "A cool caption."}

	mount := Module(Deps{
		Repo:          repo,
		Executor:      executor,
		Catalog:       fakeCatalog{},
		CaptionClient: caption,
		Opener:        opener,
		Logger:        logger,
	})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
	t.Cleanup(srv.Close)
	client := exportsconnect.NewExportsServiceClient(srv.Client(), srv.URL)
	return &harness{repo: repo, executor: executor, opener: opener, caption: caption, client: client}
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

func TestExportIDCallersShareInvalidIDContract(t *testing.T) {
	h := newHarness(t)
	cases := []struct {
		name string
		call func() error
	}{
		{name: "get", call: func() error {
			_, err := h.client.GetExport(context.Background(), connect.NewRequest(&exportsv1.GetExportRequest{Id: "bad"}))
			return err
		}},
		{name: "update", call: func() error {
			_, err := h.client.UpdateExport(context.Background(), connect.NewRequest(&exportsv1.UpdateExportRequest{Id: "bad"}))
			return err
		}},
		{name: "delete", call: func() error {
			_, err := h.client.DeleteExport(context.Background(), connect.NewRequest(&exportsv1.DeleteExportRequest{Id: "bad"}))
			return err
		}},
		{name: "status", call: func() error {
			_, err := h.client.GetExportStatus(context.Background(), connect.NewRequest(&exportsv1.GetExportStatusRequest{Id: "bad"}))
			return err
		}},
		{name: "caption", call: func() error {
			_, err := h.client.GenerateExportCaption(context.Background(), connect.NewRequest(&exportsv1.GenerateExportCaptionRequest{Id: "bad"}))
			return err
		}},
		{name: "reveal", call: func() error {
			_, err := h.client.RevealExport(context.Background(), connect.NewRequest(&exportsv1.RevealExportRequest{Id: "bad"}))
			return err
		}},
		{name: "open folder", call: func() error {
			_, err := h.client.OpenExportFolder(context.Background(), connect.NewRequest(&exportsv1.OpenExportFolderRequest{Id: "bad"}))
			return err
		}},
		{name: "cancel", call: func() error {
			_, err := h.client.CancelExport(context.Background(), connect.NewRequest(&exportsv1.CancelExportRequest{Id: "bad"}))
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.ErrorContains(t, err, errInvalidExportID.Error())
		})
	}
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
	row := &database.ExportIndex{ID: uuid.New(), ExecutionID: uuid.New(), Name: "demo", Format: "mp4", Status: database.ExportPending}
	require.NoError(t, h.repo.CreateExport(context.Background(), row))
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

func TestUpdateExportRejectsTransitionFromTerminalState(t *testing.T) {
	h := newHarness(t)
	row := seedExport(t, h, "terminal")
	_, err := h.client.UpdateExport(context.Background(), connect.NewRequest(&exportsv1.UpdateExportRequest{
		Id:     row.ID.String(),
		Status: "failed",
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	persisted, err := h.repo.GetExport(context.Background(), row.ID)
	require.NoError(t, err)
	require.Equal(t, database.ExportCompleted, persisted.Status)
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
	request := connect.NewRequest(&exportsv1.GenerateExportCaptionRequest{
		Id: row.ID.String(),
	})
	request.Header().Set("Idempotency-Key", "aabbccddeeff00112233445566778899")
	resp, err := h.client.GenerateExportCaption(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "A cool caption.", resp.Msg.GetCaption())
	require.Equal(t, "A cool caption.", resp.Msg.GetExport().GetAiCaption())
	require.Equal(t, "write.default", h.caption.role)
	require.Contains(t, h.caption.prompt, "Generate a short, engaging caption")
	require.Equal(t, "aabbccddeeff00112233445566778899", h.caption.operationID)
}

func TestGenerateExportCaption_FailsClosedOnAIGatewayError(t *testing.T) {
	h := newHarness(t)
	h.caption.err = connect.NewError(connect.CodeUnavailable, errors.New("AI Gateway unavailable"))
	row := seedExport(t, h, "demo")
	_, err := h.client.GenerateExportCaption(context.Background(), connect.NewRequest(&exportsv1.GenerateExportCaptionRequest{
		Id: row.ID.String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	persisted, getErr := h.repo.GetExport(context.Background(), row.ID)
	require.NoError(t, getErr)
	require.Empty(t, persisted.AICaption)
}

func TestExecutePromptFailsClosedWithoutGatewayClient(t *testing.T) {
	_, err := (&service{}).executePrompt(context.Background(), "caption source")
	require.ErrorContains(t, err, "AI Gateway caption client is unavailable")
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

func TestCancelExportAcknowledgesOnlyOwnedJob(t *testing.T) {
	repo := newFakeRepo()
	id := uuid.New()
	row := &database.ExportIndex{ID: id, ExecutionID: uuid.New(), Status: "processing", Name: "film", Format: "mp4"}
	require.NoError(t, repo.CreateExport(t.Context(), row))
	called := false
	svc := &service{deps: Deps{Repo: repo, CancelExport: func(got uuid.UUID) bool { called = true; return got == id }}}
	response, err := svc.CancelExport(t.Context(), connect.NewRequest(&exportsv1.CancelExportRequest{Id: id.String()}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetCancellationRequested())
	require.True(t, called)
	persisted, err := repo.GetExport(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, database.ExportProcessing, persisted.Status, "acknowledgement cannot invent a terminal result")
	_, err = svc.CancelExport(t.Context(), connect.NewRequest(&exportsv1.CancelExportRequest{Id: "invalid"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	svc.deps.CancelExport = nil
	_, err = svc.CancelExport(t.Context(), connect.NewRequest(&exportsv1.CancelExportRequest{Id: id.String()}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
