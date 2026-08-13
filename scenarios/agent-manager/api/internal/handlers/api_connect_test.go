package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

func TestAgentManagerConnectListRunsUsesBoundedDefault(t *testing.T) {
	handler, _ := setupTestHandler(t)
	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	response, err := client.ListRuns(context.Background(), connect.NewRequest(&apipb.ListRunsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Msg)
}
