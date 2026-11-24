package workspaces_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacevisualcomparison/workspacevisualcomparison_v1connect"

	"react-component-library/handlers/workspaces"
	domain "react-component-library/internal/workspaces"
)

func setupServer(t *testing.T, repository domain.Repository) workspaceconnect.WorkspaceServiceClient {
	t.Helper()
	m := workspaces.ModuleFromRepository(repository, log.New(io.Discard, "", 0))
	router := mux.NewRouter()
	m.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)
}

func TestModuleReadsRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	require.NoError(t, workspaces.Migrate(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error { return workspaces.Migrate(ctx, pool) })
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "workspaces-routing", time.Minute))

	_, err = domain.NewSQLiteRepository(routed.Primary()).Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "primary only"})
	require.NoError(t, err)
	router := mux.NewRouter()
	workspaces.Module(routed).Mount(router)
	list := func(requestContext context.Context) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, workspaceconnect.WorkspaceServiceListWorkspacesProcedure, strings.NewReader(`{}`)).WithContext(requestContext)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	primaryResponse := list(ctx)
	require.Equal(t, http.StatusOK, primaryResponse.Code, primaryResponse.Body.String())
	require.Contains(t, primaryResponse.Body.String(), "primary only")
	testResponse := list(apidb.WithTestMode(ctx))
	require.Equal(t, http.StatusOK, testResponse.Code, testResponse.Body.String())
	require.NotContains(t, testResponse.Body.String(), "primary only", "request context must select isolated routed pool")
}

func setupSQLite(t *testing.T) workspaceconnect.WorkspaceServiceClient {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	if err := workspaces.Migrate(context.Background(), sqlDB); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return setupServer(t, domain.NewSQLiteRepository(sqlDB))
}

func TestWorkspaceServiceHappyPath(t *testing.T) {
	client := setupSQLite(t)
	ctx := context.Background()

	created, err := client.CreateWorkspace(ctx, connect.NewRequest(&workspacev1.CreateWorkspaceRequest{
		Name:            "Harbor onboarding",
		Summary:         "Four-page onboarding flow",
		Tags:            []string{"onboarding"},
		BindingScenario: "agent-inbox",
		BindingPage:     "inbox",
		Pages: []*workspacev1.WorkspacePage{{
			Name:     "Welcome",
			Route:    "/onboarding/welcome",
			SpecJson: `{"root":{"kind":"page"}}`,
			Displays: []*workspacev1.WorkspaceDisplay{{
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
				Width:  1180,
				Height: 780,
				States: []*workspacev1.WorkspaceState{
					{Name: "Default", ArgsJson: `{"hero":"none"}`},
					{Name: "Empty"},
				},
			}},
		}},
	}))
	require.NoError(t, err)
	workspace := created.Msg.GetWorkspace()
	require.True(t, len(workspace.GetId()) > 3)
	require.Equal(t, int32(1), workspace.GetRevision())
	require.Len(t, workspace.GetPages(), 1)

	got, err := client.GetWorkspace(ctx, connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: workspace.GetId()}))
	require.NoError(t, err)
	require.Equal(t, workspace.GetName(), got.Msg.GetWorkspace().GetName())
	require.Equal(t, "agent-inbox", got.Msg.GetWorkspace().GetBindingScenario())

	listed, err := client.ListWorkspaces(ctx, connect.NewRequest(&workspacev1.ListWorkspacesRequest{Query: "harbor"}))
	require.NoError(t, err)
	require.Len(t, listed.Msg.GetWorkspaces(), 1)
	require.Equal(t, int32(1), listed.Msg.GetWorkspaces()[0].GetPageCount())
	require.Equal(t, int32(2), listed.Msg.GetWorkspaces()[0].GetStateCount())

	workspace.Name = "Harbor onboarding v2"
	saved, err := client.SaveWorkspace(ctx, connect.NewRequest(&workspacev1.SaveWorkspaceRequest{
		Workspace:    workspace,
		BaseRevision: workspace.GetRevision(),
		Actor:        "operator@example.test",
	}))
	require.NoError(t, err)
	require.Equal(t, int32(2), saved.Msg.GetWorkspace().GetRevision())
	require.Equal(t, "Harbor onboarding v2", saved.Msg.GetWorkspace().GetName())

	revisions, err := client.ListWorkspaceRevisions(ctx, connect.NewRequest(&workspacev1.ListWorkspaceRevisionsRequest{
		Id: workspace.GetId(), Limit: 10,
	}))
	require.NoError(t, err)
	require.Len(t, revisions.Msg.GetRevisions(), 2)
	require.Equal(t, int32(2), revisions.Msg.GetRevisions()[0].GetRevision())
	require.Equal(t, "operator@example.test", revisions.Msg.GetRevisions()[0].GetActor())
	require.Contains(t, revisions.Msg.GetRevisions()[0].GetSnapshotJson(), `"name":"Harbor onboarding v2"`)

	validation, err := client.ValidateWorkspace(ctx, connect.NewRequest(&workspacev1.ValidateWorkspaceRequest{
		Workspace: &workspacev1.Workspace{
			Name: "",
			Pages: []*workspacev1.WorkspacePage{{
				Displays: []*workspacev1.WorkspaceDisplay{{
					Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_UNSPECIFIED,
				}},
			}},
		},
	}))
	require.NoError(t, err)
	require.False(t, validation.Msg.GetValid())
	require.NotEmpty(t, validation.Msg.GetIssues())

	forked, err := client.ForkWorkspace(ctx, connect.NewRequest(&workspacev1.ForkWorkspaceRequest{Id: workspace.GetId()}))
	require.NoError(t, err)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, forked.Msg.GetWorkspace().GetProvenance())
	require.Empty(t, forked.Msg.GetWorkspace().GetBindingScenario())
	require.Empty(t, forked.Msg.GetWorkspace().GetBindingPage())
	require.Equal(t, workspace.GetId(), forked.Msg.GetWorkspace().GetOriginWorkspaceId())
	require.Equal(t, int32(1), forked.Msg.GetWorkspace().GetRevision())
	require.Equal(t, "Copy of Harbor onboarding v2", forked.Msg.GetWorkspace().GetName())

	deleted, err := client.DeleteWorkspace(ctx, connect.NewRequest(&workspacev1.DeleteWorkspaceRequest{Id: workspace.GetId()}))
	require.NoError(t, err)
	require.True(t, deleted.Msg.GetDeleted())

	_, err = client.GetWorkspace(ctx, connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: workspace.GetId()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestWorkspaceServiceMapsValidationAndMissingIDs(t *testing.T) {
	client := setupSQLite(t)
	ctx := context.Background()

	_, err := client.CreateWorkspace(ctx, connect.NewRequest(&workspacev1.CreateWorkspaceRequest{Name: "  "}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.GetWorkspace(ctx, connect.NewRequest(&workspacev1.GetWorkspaceRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.GetWorkspace(ctx, connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: "ws_missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = client.DeleteWorkspace(ctx, connect.NewRequest(&workspacev1.DeleteWorkspaceRequest{Id: "ws_missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = client.ForkWorkspace(ctx, connect.NewRequest(&workspacev1.ForkWorkspaceRequest{Id: "ws_missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = client.ListWorkspaceRevisions(ctx, connect.NewRequest(&workspacev1.ListWorkspaceRevisionsRequest{Id: "ws_missing"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// conflictRepository is the smallest fake that proves the transport mapping:
// the embedded nil Repository is never called because only Save is overridden.
type conflictRepository struct {
	domain.Repository
	current *workspacev1.Workspace
}

func (r conflictRepository) Save(_ context.Context, _ *workspacev1.SaveWorkspaceRequest) (*workspacev1.Workspace, error) {
	return nil, &domain.ConflictError{Current: r.current}
}

func TestWorkspaceServiceMapsConflictToAborted(t *testing.T) {
	current := &workspacev1.Workspace{Id: "ws_current", Name: "Current", Revision: 7}
	client := setupServer(t, conflictRepository{current: current})

	_, err := client.SaveWorkspace(context.Background(), connect.NewRequest(&workspacev1.SaveWorkspaceRequest{
		Workspace:    &workspacev1.Workspace{Id: "ws_current", Name: "Stale", Revision: 6},
		BaseRevision: 6,
	}))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	require.Contains(t, err.Error(), "current revision is 7")
}

type invalidRepository struct {
	domain.Repository
}

func (r invalidRepository) Create(_ context.Context, _ *workspacev1.CreateWorkspaceRequest) (*workspacev1.Workspace, error) {
	return nil, &domain.InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "name", Message: "name is required"}}}
}

func TestWorkspaceServiceMapsInvalidToInvalidArgument(t *testing.T) {
	client := setupServer(t, invalidRepository{})

	_, err := client.CreateWorkspace(context.Background(), connect.NewRequest(&workspacev1.CreateWorkspaceRequest{Name: "x"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "name")
}

type failingRepository struct {
	domain.Repository
}

func (r failingRepository) List(_ context.Context, _ workspacev1.WorkspaceProvenance, _ string, _ int32) ([]*workspacev1.WorkspaceSummary, error) {
	return nil, errors.New("disk on fire")
}

func TestWorkspaceServiceMapsUnexpectedFailureToInternal(t *testing.T) {
	client := setupServer(t, failingRepository{})

	_, err := client.ListWorkspaces(context.Background(), connect.NewRequest(&workspacev1.ListWorkspacesRequest{}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "disk on fire", "internal errors must not leak storage detail")
}

type activeWorkspaceRPCRepository struct {
	domain.Repository
	listProvenance workspacev1.WorkspaceProvenance
	listQuery      string
	listLimit      int32
	listResult     []*workspacev1.WorkspaceSummary
	listErr        error
	validateInput  *workspacev1.Workspace
	validateResult []*workspacev1.ValidationIssue
	validateErr    error
	revisionID     string
	revisionLimit  int32
	revisionResult []*workspacev1.WorkspaceRevision
	revisionErr    error
}

func (r *activeWorkspaceRPCRepository) List(_ context.Context, provenance workspacev1.WorkspaceProvenance, query string, limit int32) ([]*workspacev1.WorkspaceSummary, error) {
	r.listProvenance, r.listQuery, r.listLimit = provenance, query, limit
	return r.listResult, r.listErr
}

func (r *activeWorkspaceRPCRepository) Validate(_ context.Context, workspace *workspacev1.Workspace) ([]*workspacev1.ValidationIssue, error) {
	r.validateInput = workspace
	return r.validateResult, r.validateErr
}

func (r *activeWorkspaceRPCRepository) Revisions(_ context.Context, id string, limit int32) ([]*workspacev1.WorkspaceRevision, error) {
	r.revisionID, r.revisionLimit = id, limit
	return r.revisionResult, r.revisionErr
}

func TestWorkspaceServiceForwardsActiveReadRPCArgumentsAndResults(t *testing.T) {
	ctx := context.Background()
	summary := &workspacev1.WorkspaceSummary{Id: "ws_active", Name: "Active"}
	revision := &workspacev1.WorkspaceRevision{Revision: 3}
	issues := []*workspacev1.ValidationIssue{{Path: "pages[0].name", Message: "required"}}
	input := &workspacev1.Workspace{Name: "Candidate"}
	repo := &activeWorkspaceRPCRepository{
		listResult: summaryList(summary), validateResult: issues,
		revisionResult: []*workspacev1.WorkspaceRevision{revision},
	}
	client := setupServer(t, repo)

	listed, err := client.ListWorkspaces(ctx, connect.NewRequest(&workspacev1.ListWorkspacesRequest{
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, Query: "  active  ", Limit: 7,
	}))
	require.NoError(t, err)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, repo.listProvenance)
	require.Equal(t, "  active  ", repo.listQuery)
	require.Equal(t, int32(7), repo.listLimit)
	require.Equal(t, summary.GetId(), listed.Msg.GetWorkspaces()[0].GetId())
	require.Equal(t, summary.GetName(), listed.Msg.GetWorkspaces()[0].GetName())

	validated, err := client.ValidateWorkspace(ctx, connect.NewRequest(&workspacev1.ValidateWorkspaceRequest{Workspace: input}))
	require.NoError(t, err)
	require.Equal(t, input.GetName(), repo.validateInput.GetName())
	require.False(t, validated.Msg.GetValid())
	require.Equal(t, issues[0].GetPath(), validated.Msg.GetIssues()[0].GetPath())
	require.Equal(t, issues[0].GetMessage(), validated.Msg.GetIssues()[0].GetMessage())

	revisions, err := client.ListWorkspaceRevisions(ctx, connect.NewRequest(&workspacev1.ListWorkspaceRevisionsRequest{Id: "ws_active", Limit: 4}))
	require.NoError(t, err)
	require.Equal(t, "ws_active", repo.revisionID)
	require.Equal(t, int32(4), repo.revisionLimit)
	require.Equal(t, revision.GetRevision(), revisions.Msg.GetRevisions()[0].GetRevision())
}

func summaryList(items ...*workspacev1.WorkspaceSummary) []*workspacev1.WorkspaceSummary {
	return items
}

func TestWorkspaceServiceMapsReadValidationAndRevisionFailures(t *testing.T) {
	ctx := context.Background()
	t.Run("invalid list filter", func(t *testing.T) {
		client := setupSQLite(t)
		_, err := client.ListWorkspaces(ctx, connect.NewRequest(&workspacev1.ListWorkspacesRequest{Provenance: workspacev1.WorkspaceProvenance(99)}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("validation repository failure", func(t *testing.T) {
		client := setupServer(t, &activeWorkspaceRPCRepository{validateErr: errors.New("validator unavailable")})
		_, err := client.ValidateWorkspace(ctx, connect.NewRequest(&workspacev1.ValidateWorkspaceRequest{Workspace: &workspacev1.Workspace{Name: "Candidate"}}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "validator unavailable")
	})
	t.Run("revision repository failure", func(t *testing.T) {
		client := setupServer(t, &activeWorkspaceRPCRepository{revisionErr: domain.ErrNotFound})
		_, err := client.ListWorkspaceRevisions(ctx, connect.NewRequest(&workspacev1.ListWorkspaceRevisionsRequest{Id: "ws_missing"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
}

func TestEndpointsCoverEveryProcedure(t *testing.T) {
	expected := map[string]string{
		"workspaces_list":                      workspaceconnect.WorkspaceServiceListWorkspacesProcedure,
		"workspaces_get":                       workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		"workspaces_create":                    workspaceconnect.WorkspaceServiceCreateWorkspaceProcedure,
		"workspaces_save":                      workspaceconnect.WorkspaceServiceSaveWorkspaceProcedure,
		"workspaces_delete":                    workspaceconnect.WorkspaceServiceDeleteWorkspaceProcedure,
		"workspaces_fork":                      workspaceconnect.WorkspaceServiceForkWorkspaceProcedure,
		"workspaces_validate":                  workspaceconnect.WorkspaceServiceValidateWorkspaceProcedure,
		"workspaces_revisions":                 workspaceconnect.WorkspaceServiceListWorkspaceRevisionsProcedure,
		"workspaces_assets_list":               workspaceconnect.WorkspaceServiceListWorkspaceAssetsProcedure,
		"workspaces_assets_list_all":           workspaceconnect.WorkspaceServiceListAllWorkspaceAssetsProcedure,
		"workspaces_assets_create":             workspaceconnect.WorkspaceServiceCreateWorkspaceAssetProcedure,
		"workspaces_assets_delete":             workspaceconnect.WorkspaceServiceDeleteWorkspaceAssetProcedure,
		"workspaces_capture_state":             workspaceconnect.WorkspaceServiceCaptureWorkspaceStateProcedure,
		"workspaces_capture_evidence_get":      workspaceconnect.WorkspaceServiceGetWorkspaceStateCaptureProcedure,
		"workspaces_capture_receipts_list":     workspaceconnect.WorkspaceServiceListWorkspaceCaptureReceiptsProcedure,
		"workspaces_visual_comparisons_create": visualconnect.WorkspaceVisualComparisonServiceCompareWorkspaceCaptureReceiptsProcedure,
		"workspaces_visual_comparisons_list":   visualconnect.WorkspaceVisualComparisonServiceListWorkspaceVisualComparisonsProcedure,
	}
	seen := make(map[string]bool, len(workspaces.Endpoints))
	for _, endpoint := range workspaces.Endpoints {
		path, ok := expected[endpoint.ID]
		if !ok {
			continue
		}
		require.Equal(t, path, endpoint.Path)
		require.Equal(t, "POST", endpoint.Method)
		require.NotEmpty(t, endpoint.Summary)
		seen[endpoint.ID] = true
	}
	require.Len(t, seen, len(expected), "every Connect RPC keeps its descriptor")
}

func TestBinaryAssetAndEvidenceRoutesAreDescribedAsREST(t *testing.T) {
	want := map[string]string{
		"workspaces_assets_get":                 "/api/workspaces/{id}/assets/{assetId}",
		"workspaces_capture_receipt_screenshot": "/api/workspaces/{id}/states/{stateId}/capture-receipts/{receiptId}/screenshot",
		"workspaces_visual_comparison_heatmap":  "/api/workspaces/{id}/states/{stateId}/visual-comparisons/{comparisonId}/heatmap",
	}
	seen := map[string]bool{}
	for _, endpoint := range workspaces.Endpoints {
		path, ok := want[endpoint.ID]
		if !ok {
			continue
		}
		require.Equal(t, path, endpoint.Path)
		require.NotNil(t, endpoint.RESTException)
		require.NotNil(t, endpoint.RESTException.ProtoPayloads)
		require.Equal(t, "binary", endpoint.RESTException.ProtoPayloads.Response.Transport)
		seen[endpoint.ID] = true
	}
	require.Len(t, seen, len(want))
}

func TestModuleShape(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	m := workspaces.Module(sqlDB)
	require.Equal(t, "workspaces", m.Name)
	require.NotNil(t, m.Mount)
	require.Len(t, m.Endpoints, 26, "nineteen WorkspaceService RPCs, four visual comparison RPCs and three binary asset routes")
	require.NotEmpty(t, workspaces.Schema())
}

func TestCuratedWorkspaceForksThroughTheAPIButRefusesWrites(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, workspaces.Schema())
	require.NoError(t, err)
	_, err = domain.Seed(ctx, sqlDB, []byte(`{"schemaVersion":1,"seedVersion":1,"workspaces":[{
		"id":"ws_template","name":"Reviewed template","provenance":"WORKSPACE_PROVENANCE_CURATED",
		"revision":1,"createdAt":"2026-11-20T00:00:00.000Z","updatedAt":"2026-11-20T00:00:00.000Z",
		"pages":[{"id":"pg_t","name":"Welcome","specJson":"{}","displays":[{"id":"dp_t",
			"device":"DISPLAY_DEVICE_DESKTOP","states":[{"id":"st_t","name":"Default","argsJson":"{}"}]}]}]
	}]}`))
	require.NoError(t, err)
	client := setupServer(t, domain.NewSQLiteRepository(sqlDB))
	got, err := client.GetWorkspace(ctx, connect.NewRequest(&workspacev1.GetWorkspaceRequest{Id: "ws_template"}))
	require.NoError(t, err)
	template := got.Msg.GetWorkspace()

	_, err = client.SaveWorkspace(ctx, connect.NewRequest(&workspacev1.SaveWorkspaceRequest{
		Workspace:    template,
		BaseRevision: template.GetRevision(),
	}))
	require.Equalf(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "save error: %v", err)
	require.Contains(t, err.Error(), "curated workspace is immutable")

	_, err = client.DeleteWorkspace(ctx, connect.NewRequest(&workspacev1.DeleteWorkspaceRequest{Id: template.GetId()}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	forked, err := client.ForkWorkspace(ctx, connect.NewRequest(&workspacev1.ForkWorkspaceRequest{Id: template.GetId()}))
	require.NoError(t, err)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, forked.Msg.GetWorkspace().GetProvenance())
}

func TestCreateWorkspaceCannotMintACuratedTemplate(t *testing.T) {
	client := setupSQLite(t)
	_, err := client.CreateWorkspace(context.Background(), connect.NewRequest(&workspacev1.CreateWorkspaceRequest{
		Name:       "Not reviewed",
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "reviewed seed")
}

func TestCreateWorkspaceBindingConflictIsRetryableByTheClient(t *testing.T) {
	client := setupSQLite(t)
	ctx := context.Background()
	request := &workspacev1.CreateWorkspaceRequest{Name: "Welcome", BindingScenario: "harbor", BindingPage: "welcome"}
	created, err := client.CreateWorkspace(ctx, connect.NewRequest(request))
	require.NoError(t, err)

	_, err = client.CreateWorkspace(ctx, connect.NewRequest(request))
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	require.Contains(t, err.Error(), created.Msg.GetWorkspace().GetId())
}
