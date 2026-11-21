package workspaces_test

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"

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

func setupSQLite(t *testing.T) workspaceconnect.WorkspaceServiceClient {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	if _, err := sqlDB.ExecContext(context.Background(), workspaces.Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
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

func TestEndpointsCoverEveryProcedure(t *testing.T) {
	expected := map[string]string{
		"workspaces_list":      workspaceconnect.WorkspaceServiceListWorkspacesProcedure,
		"workspaces_get":       workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		"workspaces_create":    workspaceconnect.WorkspaceServiceCreateWorkspaceProcedure,
		"workspaces_save":      workspaceconnect.WorkspaceServiceSaveWorkspaceProcedure,
		"workspaces_delete":    workspaceconnect.WorkspaceServiceDeleteWorkspaceProcedure,
		"workspaces_fork":      workspaceconnect.WorkspaceServiceForkWorkspaceProcedure,
		"workspaces_validate":  workspaceconnect.WorkspaceServiceValidateWorkspaceProcedure,
		"workspaces_revisions": workspaceconnect.WorkspaceServiceListWorkspaceRevisionsProcedure,
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

// The REST asset routes are the one non-Connect transport in this module. They
// are declared in the same slice so the endpoint registry, the CLI parity test
// and `make endpoints` see one surface.
func TestAssetEndpointsAreDescribedAsREST(t *testing.T) {
	expected := map[string]string{
		"workspaces_assets_list":     "/api/workspaces/{id}/assets",
		"workspaces_assets_list_all": "/api/workspace-assets",
		"workspaces_assets_create":   "/api/workspaces/{id}/assets",
		"workspaces_assets_get":      "/api/workspaces/{id}/assets/{assetId}",
		"workspaces_assets_delete":   "/api/workspaces/{id}/assets/{assetId}",
	}
	seen := make(map[string]bool, len(expected))
	for _, endpoint := range workspaces.Endpoints {
		path, ok := expected[endpoint.ID]
		if !ok {
			continue
		}
		require.Equal(t, path, endpoint.Path)
		require.NotEmpty(t, endpoint.Summary)
		require.NotNil(t, endpoint.RESTException, "a non-Connect route must declare why it is REST")
		seen[endpoint.ID] = true
	}
	require.Len(t, seen, len(expected))
}

func TestModuleShape(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	m := workspaces.Module(sqlDB)
	require.Equal(t, "workspaces", m.Name)
	require.NotNil(t, m.Mount)
	require.Len(t, m.Endpoints, 14, "nine Connect RPCs plus five asset routes")
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
