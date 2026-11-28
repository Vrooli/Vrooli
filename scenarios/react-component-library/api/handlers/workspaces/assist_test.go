package workspaces_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	aiinference "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"
	"google.golang.org/protobuf/encoding/protojson"

	db "github.com/vrooli/api-core/databasetest"

	"react-component-library/handlers/workspaces"
	"react-component-library/internal/designinference"
	domain "react-component-library/internal/workspaces"
)

type fakeAssistant struct {
	request designinference.Request
	value   string
	state   string
	err     error
	calls   int
}

func (f *fakeAssistant) Run(_ context.Context, _ string, request designinference.Request) (designinference.Operation, error) {
	f.calls++
	f.request = request
	if f.err != nil {
		return designinference.Operation{}, f.err
	}
	raw, err := protojson.Marshal(&aiinference.RunResponse{
		ValueJson: f.value,
		Validated: true,
		Provider:  "fake",
		Model:     "test-model",
	})
	if err != nil {
		return designinference.Operation{}, err
	}
	state := f.state
	if state == "" {
		state = "completed"
	}
	return designinference.Operation{ID: "op-1", State: state, ResponseJSON: string(raw)}, nil
}

const assistSpec = `{"id":"root","type":"page","props":{},"children":[{"id":"hero.title","type":"text","props":{"text":"Old headline","variant":"title"}}]}`

func setupAssist(t *testing.T, assistant workspaces.Assistant) (workspaceconnect.WorkspaceServiceClient, *workspacev1.Workspace) {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	ctx := context.Background()
	_, err := sqlDB.ExecContext(ctx, workspaces.Schema())
	require.NoError(t, err)
	repo := domain.NewSQLiteRepository(sqlDB)
	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{
		Name: "Assist",
		Pages: []*workspacev1.WorkspacePage{{
			Name:     "Welcome",
			SpecJson: assistSpec,
			Displays: []*workspacev1.WorkspaceDisplay{{
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
				Width:  1180,
				Height: 780,
				States: []*workspacev1.WorkspaceState{{
					Name:     "Default",
					ArgsJson: `{"spec":` + assistSpec + `}`,
				}},
			}},
		}},
	})
	require.NoError(t, err)

	m := workspaces.ModuleWithRecorderAndAssistant(sqlDB, nil, assistant)
	router := mux.NewRouter()
	m.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL), created
}

func assistRequest(workspace *workspacev1.Workspace, mutate func(*workspacev1.AssistWorkspaceRequest)) *workspacev1.AssistWorkspaceRequest {
	page := workspace.GetPages()[0]
	display := page.GetDisplays()[0]
	request := &workspacev1.AssistWorkspaceRequest{
		Id:             workspace.GetId(),
		BaseRevision:   workspace.GetRevision(),
		PageId:         page.GetId(),
		ActiveStates:   []*workspacev1.WorkspaceAssistState{{DisplayId: display.GetId(), StateId: display.GetStates()[0].GetId()}},
		Prompt:         "Make the headline specific",
		Scope:          &workspacev1.WorkspaceAssistScope{Kind: "layer", DisplayId: display.GetId(), NodeId: "hero.title"},
		IdempotencyKey: "assist-one-1234",
	}
	if mutate != nil {
		mutate(request)
	}
	return request
}

func TestAssistWorkspaceReturnsAValidatedProposal(t *testing.T) {
	assistant := &fakeAssistant{value: `{"summary":"Rewrite the headline","operations":[{"kind":"setText","nodeId":"hero.title","text":"Welcome aboard"}]}`}
	client, workspace := setupAssist(t, assistant)

	response, err := client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, nil)))
	require.NoError(t, err)
	require.Equal(t, workspace.GetRevision(), response.Msg.GetRevision())
	require.Equal(t, "op-1", response.Msg.GetOperation().GetId())
	require.Equal(t, "fake", response.Msg.GetOperation().GetProvider())

	var proposal struct {
		Summary    string `json:"summary"`
		Operations []struct {
			Kind   string `json:"kind"`
			NodeID string `json:"nodeId"`
		} `json:"operations"`
	}
	require.NoError(t, json.Unmarshal([]byte(response.Msg.GetProposalJson()), &proposal))
	require.Equal(t, "Rewrite the headline", proposal.Summary)
	require.Equal(t, "hero.title", proposal.Operations[0].NodeID)

	// The model saw the selected layer, not the whole page.
	require.Contains(t, assistant.request.Source, "hero.title")
	require.Equal(t, 1, assistant.calls)
}

func TestAssistWorkspaceRefusesAStaleRevision(t *testing.T) {
	client, workspace := setupAssist(t, &fakeAssistant{value: `{"summary":"x","operations":[{"kind":"setText","nodeId":"hero.title","text":"y"}]}`})
	_, err := client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, func(r *workspacev1.AssistWorkspaceRequest) {
		r.BaseRevision = 99
	})))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
}

func TestAssistWorkspaceRejectsUnknownPagesAndBadKeys(t *testing.T) {
	client, workspace := setupAssist(t, &fakeAssistant{})

	_, err := client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, func(r *workspacev1.AssistWorkspaceRequest) {
		r.PageId = "pg_missing"
	})))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, func(r *workspacev1.AssistWorkspaceRequest) {
		r.IdempotencyKey = "short"
	})))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAssistWorkspaceReportsUnavailableWithoutAnAssistant(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), workspaces.Schema())
	require.NoError(t, err)
	m := workspaces.ModuleFromRepository(domain.NewSQLiteRepository(sqlDB), log.New(io.Discard, "", 0))
	router := mux.NewRouter()
	m.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := workspaceconnect.NewWorkspaceServiceClient(server.Client(), server.URL)

	_, err = client.AssistWorkspace(context.Background(), connect.NewRequest(&workspacev1.AssistWorkspaceRequest{Id: "ws_x", IdempotencyKey: "assist-one-1234"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestAssistWorkspaceRefusesAProposalItCannotApply(t *testing.T) {
	assistant := &fakeAssistant{value: `{"summary":"Remove everything","operations":[{"kind":"setProp","nodeId":"hero.title","key":"src","value":"https://example.com/x.png"}]}`}
	client, workspace := setupAssist(t, assistant)
	_, err := client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, nil)))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAssistWorkspaceMapsGatewayFailureToUnavailable(t *testing.T) {
	client, workspace := setupAssist(t, &fakeAssistant{err: errors.New("gateway down")})
	_, err := client.AssistWorkspace(context.Background(), connect.NewRequest(assistRequest(workspace, nil)))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
