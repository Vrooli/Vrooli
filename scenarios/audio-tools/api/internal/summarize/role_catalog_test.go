package summarize

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type fakeSummaryInventoryClient struct {
	response *inventoryv1.ListProviderRolesResponse
	err      error
}

func (f fakeSummaryInventoryClient) ListProviderRoles(_ context.Context, req *connect.Request[inventoryv1.ListProviderRolesRequest]) (*connect.Response[inventoryv1.ListProviderRolesResponse], error) {
	if req.Msg.GetProvider() != "ai-gateway" {
		return nil, errors.New("unexpected provider")
	}
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(f.response), nil
}

type fakeSummaryCatalogClient struct {
	response *commonv1.AIGatewayCapabilityCatalog
}

func (f fakeSummaryCatalogClient) GetCapabilityCatalog(context.Context, *connect.Request[catalogv1.GetCapabilityCatalogRequest]) (*connect.Response[commonv1.AIGatewayCapabilityCatalog], error) {
	return connect.NewResponse(f.response), nil
}

func TestListAudioToolsSummarizeRolesUsesInventoryForRolesAndCatalogForDisplay(t *testing.T) {
	roles, err := ListAudioToolsSummarizeRolesFromClients(context.Background(), fakeSummaryInventoryClient{response: &inventoryv1.ListProviderRolesResponse{Roles: []*inventoryv1.ProviderRole{
		{Provider: "ai-gateway", Role: "summarize.audio_tools.experimental"},
		{Provider: "ai-gateway", Role: "summarize.audio_tools"},
		{Provider: "ai-gateway", Role: "summarize.faithful"},
		{Provider: "ollama", Role: "summarize.audio_tools.evil"},
	}}}, fakeSummaryCatalogClient{response: &commonv1.AIGatewayCapabilityCatalog{Roles: []*commonv1.AIGatewayRole{
		{Id: "summarize.audio_tools.experimental", Description: "Display-only declared summary description."},
	}}})
	require.NoError(t, err)
	require.Equal(t, []DeclaredSummaryRole{
		{ID: "summarize.audio_tools", Description: ""},
		{ID: "summarize.audio_tools.experimental", Description: "Display-only declared summary description."},
	}, roles)
	require.True(t, ContainsDeclaredAudioToolsSummarizeRole(roles, "summarize.audio_tools.experimental"), "new inventory roles list and validate without Go role enumeration")
	require.False(t, ContainsDeclaredAudioToolsSummarizeRole(roles, "summarize.faithful"))
}

func TestListAudioToolsSummarizeRolesFailsClosedOnInventoryFailure(t *testing.T) {
	_, err := ListAudioToolsSummarizeRolesFromClients(context.Background(), fakeSummaryInventoryClient{err: errors.New("inventory unavailable")}, fakeSummaryCatalogClient{})
	require.ErrorContains(t, err, "inventory unavailable")
}
