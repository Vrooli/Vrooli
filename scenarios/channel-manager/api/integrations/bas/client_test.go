package bas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
)

type staticResolver struct{ url string }

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type workflowsHandler struct {
	basconnect.UnimplementedWorkflowsServiceHandler
	t *testing.T
}

func (h workflowsHandler) ExecuteWorkflow(_ context.Context, request *connect.Request[basapi.ExecuteWorkflowRequest]) (*connect.Response[basapi.ExecuteWorkflowResponse], error) {
	require.Equal(h.t, "workflow-1", request.Msg.GetWorkflowId())
	require.Equal(h.t, "profile-1", request.Msg.GetParameters().GetSessionProfileId())
	require.Equal(h.t, "profile-1", request.Msg.GetParameters().GetSaveSessionProfileId())
	require.Equal(h.t, map[string]string{"action_id": "action-1"}, request.Msg.GetParameters().GetVariables())
	return connect.NewResponse(&basapi.ExecuteWorkflowResponse{ExecutionId: "execution-1"}), nil
}

func TestDispatchSendsOnlyActionAndProfileReferences(t *testing.T) {
	path, handler := basconnect.NewWorkflowsServiceHandler(workflowsHandler{t: t})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &Client{resolver: staticResolver{url: server.URL}, http: server.Client()}
	executionID, artifacts, err := client.Dispatch(context.Background(), "profile-1", "workflow-1", "action-1")
	require.NoError(t, err)
	require.Equal(t, "execution-1", executionID)
	require.Empty(t, artifacts)
}

func TestDispatchRejectsMissingReferences(t *testing.T) {
	client := &Client{}
	_, _, err := client.Dispatch(context.Background(), "", "workflow-1", "action-1")
	require.Error(t, err)
}
