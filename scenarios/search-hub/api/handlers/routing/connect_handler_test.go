package routing_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"

	handler "search-hub/handlers/routing"
	internalrouting "search-hub/internal/routing"
)

// fakeRouter is a hand-written Querier fake: arrange via fields, assert against
// the recorded request. Keeps the handler test off real provider fan-out (that
// path is covered by internal/routing/router_test.go).
type fakeRouter struct {
	out     *routingv1.QueryResponse
	err     error
	lastReq *routingv1.QueryRequest
}

func (f *fakeRouter) Query(_ context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
	f.lastReq = req
	return f.out, f.err
}

func newClient(t *testing.T, router handler.Querier) routingconnect.RoutingServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := routingconnect.NewRoutingServiceHandler(handler.NewConnectHandler(handler.Deps{Router: router, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return routingconnect.NewRoutingServiceClient(server.Client(), server.URL)
}

func TestQueryReturnsGroups(t *testing.T) {
	router := &fakeRouter{out: &routingv1.QueryResponse{
		Groups: []*routingv1.ProviderResultGroup{
			{ProviderId: "cli-health.commands", Count: 1, Hits: []*routingv1.SearchHit{{ProviderId: "cli-health.commands", Title: "scenario restart"}}},
		},
		CorporaSearched: []string{"cli-health.commands"},
	}}
	client := newClient(t, router)

	resp, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{
		Query: "restart a scenario", Types: []string{"command"}, Limit: 5,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetGroups(), 1)
	require.Equal(t, "cli-health.commands", resp.Msg.GetGroups()[0].GetProviderId())
	// The handler forwards the request verbatim to the router seam.
	require.Equal(t, "restart a scenario", router.lastReq.GetQuery())
	require.Equal(t, []string{"command"}, router.lastReq.GetTypes())
	require.Equal(t, int32(5), router.lastReq.GetLimit())
}

func TestQueryInvalidArgument(t *testing.T) {
	router := &fakeRouter{err: internalrouting.ErrInvalidQuery{Reason: "query text is required"}}
	client := newClient(t, router)

	_, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{All: true}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestQueryInternalErrorIsOpaque(t *testing.T) {
	router := &fakeRouter{err: errors.New("registry db on fire: /var/lib/secret")}
	client := newClient(t, router)

	_, err := client.Query(context.Background(), connect.NewRequest(&routingv1.QueryRequest{Query: "x", All: true}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "secret", "internal details must not leak to the client")
}

func TestStatusUnimplemented(t *testing.T) {
	client := newClient(t, &fakeRouter{})
	_, err := client.Status(context.Background(), connect.NewRequest(&routingv1.StatusRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}
