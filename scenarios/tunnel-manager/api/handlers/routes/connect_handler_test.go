package routes_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/handlers/routes"
	"tunnel-manager/internal/authz"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
	routesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes/routes_v1connect"

	internalroutes "tunnel-manager/internal/routes"
)

// fakeService implements internalroutes.Service for handler tests.
type fakeService struct {
	listOut     []internalroutes.Route
	getOut      internalroutes.Route
	getErr      error
	createOut   internalroutes.Route
	createErr   error
	createIn    internalroutes.CreateInput
	createCalls int
	deletedOut  bool
	deleteCalls int
}

func (f *fakeService) Create(_ context.Context, in internalroutes.CreateInput) (internalroutes.Route, error) {
	f.createCalls++
	f.createIn = in
	return f.createOut, f.createErr
}

func (f *fakeService) Get(_ context.Context, _ string) (internalroutes.Route, error) {
	return f.getOut, f.getErr
}

func (f *fakeService) GetBySubdomain(_ context.Context, _ string) (internalroutes.Route, error) {
	return f.getOut, f.getErr
}

func (f *fakeService) List(_ context.Context, _ internalroutes.Tier) ([]internalroutes.Route, error) {
	return f.listOut, nil
}

func (f *fakeService) Update(_ context.Context, _ string, _ internalroutes.UpdateInput) (internalroutes.Route, error) {
	return f.getOut, f.getErr
}

func (f *fakeService) Delete(_ context.Context, _ string) (bool, error) {
	f.deleteCalls++
	return f.deletedOut, nil
}

func newClient(t *testing.T, svc internalroutes.Service) routesconnect.RoutesServiceClient {
	t.Helper()
	return newClientWithAuthorizer(t, svc, nil)
}

func newClientWithAuthorizer(t *testing.T, svc internalroutes.Service, authorizer authz.Authorizer) routesconnect.RoutesServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := routesconnect.NewRoutesServiceHandler(routes.NewConnectHandler(routes.Deps{
		Service:    svc,
		Logger:     logger,
		Authorizer: authorizer,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return routesconnect.NewRoutesServiceClient(server.Client(), server.URL)
}

func TestHandlerListDerivesPublicURL(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{listOut: []internalroutes.Route{
		{ID: "a", Subdomain: "agent-manager", Scenario: "agent-manager", Domain: "itsagitime.com", LocalPort: 21100, Tier: internalroutes.TierCore, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}})

	resp, err := client.ListRoutes(context.Background(), connect.NewRequest(&routesv1.ListRoutesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Routes, 1)
	require.Equal(t, "https://agent-manager.itsagitime.com", resp.Msg.Routes[0].PublicUrl)
	require.Equal(t, routesv1.Tier_TIER_CORE, resp.Msg.Routes[0].Tier)
}

func TestHandlerCreateMapsInput(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	fake := &fakeService{createOut: internalroutes.Route{ID: "new", Subdomain: "web-console", Scenario: "web-console", Domain: "itsagitime.com", LocalPort: 3000, Tier: internalroutes.TierLeased, Enabled: true, CreatedAt: now, UpdatedAt: now}}
	client := newClient(t, fake)

	resp, err := client.CreateRoute(context.Background(), connect.NewRequest(&routesv1.CreateRouteRequest{
		Subdomain: "web-console", Scenario: "web-console", LocalPort: 3000, Tier: routesv1.Tier_TIER_LEASED,
	}))
	require.NoError(t, err)
	require.Equal(t, "new", resp.Msg.Route.Id)
	require.Equal(t, "web-console", fake.createIn.Subdomain)
	require.Equal(t, internalroutes.TierLeased, fake.createIn.Tier)
	require.Equal(t, 3000, fake.createIn.LocalPort)
}

func TestHandlerCreateInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{createErr: internalroutes.ErrInvalidRoute{Field: "subdomain", Reason: "required"}})
	_, err := client.CreateRoute(context.Background(), connect.NewRequest(&routesv1.CreateRouteRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestHandlerCreateConflict(t *testing.T) {
	client := newClient(t, &fakeService{createErr: internalroutes.ErrRouteConflict{Subdomain: "dup"}})
	_, err := client.CreateRoute(context.Background(), connect.NewRequest(&routesv1.CreateRouteRequest{Subdomain: "dup", Scenario: "s", LocalPort: 1}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestHandlerCreateRequiresOperatorTokenWhenEnforced(t *testing.T) {
	fake := &fakeService{}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})

	_, err := client.CreateRoute(context.Background(), connect.NewRequest(&routesv1.CreateRouteRequest{
		Subdomain: "web-console", Scenario: "web-console", LocalPort: 3000,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, fake.createCalls, "denied create must not reach the service")
}

func TestHandlerDeleteAcceptsOperatorBearer(t *testing.T) {
	fake := &fakeService{deletedOut: true}
	client := newClientWithAuthorizer(t, fake, authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"})
	req := connect.NewRequest(&routesv1.DeleteRouteRequest{Id: "x"})
	req.Header().Set("Authorization", "Bearer secret")

	resp, err := client.DeleteRoute(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.Deleted)
	require.Equal(t, 1, fake.deleteCalls)
}

func TestHandlerGetNotFound(t *testing.T) {
	client := newClient(t, &fakeService{getErr: internalroutes.ErrRouteNotFound{ID: "ghost"}})
	_, err := client.GetRoute(context.Background(), connect.NewRequest(&routesv1.GetRouteRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestHandlerDelete(t *testing.T) {
	client := newClient(t, &fakeService{deletedOut: true})
	resp, err := client.DeleteRoute(context.Background(), connect.NewRequest(&routesv1.DeleteRouteRequest{Id: "x"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Deleted)
}
