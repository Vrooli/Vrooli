package discovery_test

import (
	"context"
	"testing"

	"brand-manager/handlers/discovery"
	internaldiscovery "brand-manager/internal/discovery"
	mocks "brand-manager/internal/discovery/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery/discovery_v1connect"
)

// newClient wires the real internal discovery service over in-memory fakes behind
// the generated Connect handler, exercising handler + adapter + service together.
func newClient(t *testing.T, scanner *mocks.FakeScanner, store *mocks.FakeBrandStore) discoveryconnect.DiscoveryServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	svc := internaldiscovery.NewService(scanner, store, logger)
	path, handler := discoveryconnect.NewDiscoveryServiceHandler(discovery.NewConnectHandler(discovery.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return discoveryconnect.NewDiscoveryServiceClient(server.Client(), server.URL)
}

func brandedScanner(t *testing.T) *mocks.FakeScanner {
	t.Helper()
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("web-console")
	sc.SeedFile("web-console", ".vrooli/branding.json", []byte(`{"site_name":"Acme","theme":{"primary":"#112233"}}`))
	return sc
}

func TestConnect_DiscoverReturnsDraftAndSources(t *testing.T) {
	client := newClient(t, brandedScanner(t), &mocks.FakeBrandStore{})

	resp, err := client.DiscoverScenario(context.Background(), connect.NewRequest(&discoveryv1.DiscoverScenarioRequest{
		ScenarioName: "web-console",
	}))
	require.NoError(t, err)
	require.Equal(t, "web-console", resp.Msg.Scenario)
	require.NotEmpty(t, resp.Msg.Sources)
	require.NotNil(t, resp.Msg.DraftBrand)
	require.Equal(t, "Acme", resp.Msg.DraftBrand.Identity.DisplayName)
	require.Equal(t, "#112233", resp.Msg.DraftBrand.Colors.Primary)
}

func TestConnect_DiscoverEmptyScenarioHasNullDraft(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("blank")
	client := newClient(t, sc, &mocks.FakeBrandStore{})

	resp, err := client.DiscoverScenario(context.Background(), connect.NewRequest(&discoveryv1.DiscoverScenarioRequest{
		ScenarioName: "blank",
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.Sources)
	require.Nil(t, resp.Msg.DraftBrand)
	require.NotEmpty(t, resp.Msg.Suggestions)
}

func TestConnect_DiscoverMissingScenarioIsNotFound(t *testing.T) {
	client := newClient(t, &mocks.FakeScanner{}, &mocks.FakeBrandStore{})

	_, err := client.DiscoverScenario(context.Background(), connect.NewRequest(&discoveryv1.DiscoverScenarioRequest{
		ScenarioName: "ghost",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_DiscoverMissingArgIsInvalidArgument(t *testing.T) {
	client := newClient(t, &mocks.FakeScanner{}, &mocks.FakeBrandStore{})

	_, err := client.DiscoverScenario(context.Background(), connect.NewRequest(&discoveryv1.DiscoverScenarioRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_ImportCreatesBrand(t *testing.T) {
	store := &mocks.FakeBrandStore{}
	client := newClient(t, brandedScanner(t), store)

	resp, err := client.ImportBrand(context.Background(), connect.NewRequest(&discoveryv1.ImportBrandRequest{
		ScenarioName: "web-console",
	}))
	require.NoError(t, err)
	require.Equal(t, "brand-1", resp.Msg.BrandId)
	require.Equal(t, int32(1), resp.Msg.BrandVersion)
	require.Len(t, store.Recorded(), 1)
}

func TestConnect_ImportNoBrandingIsFailedPrecondition(t *testing.T) {
	sc := &mocks.FakeScanner{}
	sc.SeedScenario("blank")
	client := newClient(t, sc, &mocks.FakeBrandStore{})

	_, err := client.ImportBrand(context.Background(), connect.NewRequest(&discoveryv1.ImportBrandRequest{
		ScenarioName: "blank",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
