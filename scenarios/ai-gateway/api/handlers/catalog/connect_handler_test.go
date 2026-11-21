package catalog

import (
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"ai-gateway/internal/gatewaycatalog"
	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog/catalog_v1connect"
)

func TestGetCapabilityCatalogConnectRPC(t *testing.T) {
	profiles := &gatewaycatalog.ProfileCatalog{SchemaVersion: 1, Profiles: []gatewaycatalog.Profile{
		{Proto: "PROFILE_LOCAL_ONLY", Key: "local-only", LabelKey: "profiles.localOnly", DescriptionKey: "profiles.localOnlyDesc", Order: 1},
	}}
	roles := rolecatalog.RoleCatalog{Roles: map[string]rolecatalog.InferenceRole{
		"chat.default": {Description: "Default chat", Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "chat.default", Reason: "local candidate"}}},
	}}
	router := mux.NewRouter()
	Module(profiles, roles).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()

	client := catalogconnect.NewCatalogServiceClient(server.Client(), server.URL)
	response, err := client.GetCapabilityCatalog(t.Context(), connect.NewRequest(&catalogv1.GetCapabilityCatalogRequest{}))
	require.NoError(t, err)
	require.Equal(t, "1", response.Msg.GetContractVersion())
	require.Equal(t, "ai-gateway", response.Msg.GetScenario())
	require.Equal(t, []string{"PROFILE_LOCAL_ONLY"}, []string{response.Msg.GetProfiles()[0].GetProtoName()})
	require.Contains(t, response.Msg.GetRequestKinds()[0].GetRequiredInputModalities(), "text")
	require.Equal(t, "chat.default", response.Msg.GetRoles()[0].GetId())
	require.NotEmpty(t, response.Msg.GetProviders())
}

func TestUnavailableProfileCatalogFailsClosed(t *testing.T) {
	handler := NewConnectHandler(nil, rolecatalog.RoleCatalog{}, []providers.ProviderSpec{})
	_, err := handler.GetCapabilityCatalog(t.Context(), connect.NewRequest(&catalogv1.GetCapabilityCatalogRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
