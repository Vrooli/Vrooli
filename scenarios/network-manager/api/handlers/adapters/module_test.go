package adapters

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters/adapters_v1connect"
	"network-manager/internal/module"
	domainresolver "network-manager/internal/resolver"
)

func TestModuleExposesEndpoints(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	m := Module(d)
	if m.Name != "adapters" {
		t.Fatalf("module name = %q, want adapters", m.Name)
	}
	if len(m.Endpoints) == 0 {
		t.Fatal("expected adapters endpoints")
	}
}

func TestEndpointDescriptorsMatchAdapterRPCPayloads(t *testing.T) {
	// [REQ:NM-P0-006] Machine-readable endpoint metadata must match the API contract.
	descriptors := make(map[string]module.EndpointDescriptor, len(Endpoints))
	for _, endpoint := range Endpoints {
		descriptors[endpoint.ID] = endpoint
	}

	cases := []struct {
		id, requestType, responseType string
		request, response             map[string]string
	}{
		{
			id: "adapters_capabilities", requestType: "object", responseType: "object",
			request: map[string]string{}, response: map[string]string{"capabilities": "array<Capability>"},
		},
		{
			id: "adapters_explain_unsupported", requestType: "object", responseType: "object",
			request:  map[string]string{"action": "string"},
			response: map[string]string{"capability": "Capability", "manual_steps": "array<string>"},
		},
		{
			id: "adapters_platform", requestType: "object", responseType: "object",
			request: map[string]string{}, response: map[string]string{"summary": "PlatformSummary"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			endpoint, ok := descriptors[tc.id]
			require.True(t, ok, "RPC endpoint must be described")
			require.Equal(t, tc.requestType, endpoint.Request.Type)
			require.Equal(t, tc.request, endpoint.Request.Properties)
			require.Equal(t, tc.responseType, endpoint.Response.Type)
			require.Equal(t, tc.response, endpoint.Response.Properties)
		})
	}
}

func TestConnectAdapterEndpointsReturnTypedResponses(t *testing.T) {
	// [REQ:NM-P0-006] Capability discovery and unsupported-action handling must survive transport.
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(Schema),
		apidb.SchemaProviderFunc(domainresolver.Schema),
	))

	router := mux.NewRouter()
	Module(d).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := adaptersconnect.NewAdapterServiceClient(server.Client(), server.URL)

	capabilities, err := client.ListCapabilities(ctx, connect.NewRequest(&adaptersv1.ListCapabilitiesRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, capabilities.Msg.Capabilities)
	var manualRouter *adaptersv1.Capability
	for _, capability := range capabilities.Msg.Capabilities {
		if capability.GetAdapter() == "manual-router" && capability.GetAction() == "router_dns_enforcement" {
			manualRouter = capability
		}
	}
	require.NotNil(t, manualRouter)
	require.False(t, manualRouter.GetSupported())
	require.Contains(t, manualRouter.GetReason(), "manual router instructions")
	var resolverStatus *adaptersv1.Capability
	for _, capability := range capabilities.Msg.Capabilities {
		if capability.GetAdapter() == "adguard-home" && capability.GetAction() == "resolver_status" {
			resolverStatus = capability
		}
	}
	require.NotNil(t, resolverStatus)
	require.False(t, resolverStatus.GetSupported(), "unconfigured resolvers must fail closed")

	resolverRepo := domainresolver.NewSQLiteRepository(d)
	_, err = resolverRepo.SaveBackend(ctx, domainresolver.BackendConfig{
		Backend: domainresolver.AdGuardHomeBackend, BaseURL: "http://adguard.test",
		CredentialRef: "vrooli/adguard-home",
	})
	require.NoError(t, err)
	configured, err := client.ListCapabilities(ctx, connect.NewRequest(&adaptersv1.ListCapabilitiesRequest{}))
	require.NoError(t, err)
	configuredResolverCaps := make(map[string]*adaptersv1.Capability)
	for _, capability := range configured.Msg.Capabilities {
		if capability.GetAdapter() == "adguard-home" {
			configuredResolverCaps[capability.GetAction()] = capability
		}
	}
	for _, action := range []string{"resolver_status", "resolver_client_inventory", "manage_dns_filtering"} {
		capability := configuredResolverCaps[action]
		require.NotNil(t, capability, "configured backend should report %s", action)
		require.True(t, capability.GetSupported(), "configured backend should report %s supported", action)
	}
	require.True(t, configuredResolverCaps["manage_dns_filtering"].GetRollbackSupported())

	explanation, err := client.ExplainUnsupportedAction(ctx, connect.NewRequest(&adaptersv1.ExplainUnsupportedActionRequest{Action: "router_dns_enforcement"}))
	require.NoError(t, err)
	require.NotNil(t, explanation.Msg.GetCapability())
	require.Equal(t, "router_dns_enforcement", explanation.Msg.GetCapability().GetAction())
	require.NotEmpty(t, explanation.Msg.GetManualSteps())

	platform, err := client.GetPlatformSummary(ctx, connect.NewRequest(&adaptersv1.GetPlatformSummaryRequest{}))
	require.NoError(t, err)
	require.NotNil(t, platform.Msg.GetSummary())
	require.NotEmpty(t, platform.Msg.GetSummary().GetOs())
	require.NotEmpty(t, platform.Msg.GetSummary().GetProfile())
	require.NotEmpty(t, platform.Msg.GetSummary().GetNotes())
}
