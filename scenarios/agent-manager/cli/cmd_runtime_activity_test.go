package main

import (
	"connectrpc.com/connect"
	"context"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"net/http"
	"net/http/httptest"
	"testing"
)

type physicalActivityServer struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
}

func (physicalActivityServer) GetRuntimeActivity(context.Context, *connect.Request[apipb.GetRuntimeActivityRequest]) (*connect.Response[apipb.GetRuntimeActivityResponse], error) {
	return connect.NewResponse(&apipb.GetRuntimeActivityResponse{ObservedExecutingRuns: 2, LogicalQueuedRuns: 99, UnknownReasons: []string{"partial"}}), nil
}
func TestRunActivityPreservesUnknownTotalOnPublicTypedWire(t *testing.T) {
	path, handler := apiconnect.NewAgentManagerServiceHandler(physicalActivityServer{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
	body, got, err := (&RunService{api: api}).Activity()
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutingRuns != nil || got.Complete || got.ObservedExecutingRuns != 2 || got.LogicalQueuedRuns != 99 || len(body) == 0 {
		t.Fatalf("physical unknown lost on typed wire: %+v body=%s", got, body)
	}
}
