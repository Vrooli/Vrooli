package summarize

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog/catalog_v1connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory/inventory_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const AudioToolsSummarizeRoleNamespace = "summarize.audio_tools"

type DeclaredSummaryRole struct {
	ID          string
	Description string // descriptive only; never used to validate or dispatch
}

func IsAudioToolsSummarizeRole(id string) bool {
	id = strings.TrimSpace(id)
	return id == AudioToolsSummarizeRoleNamespace || strings.HasPrefix(id, AudioToolsSummarizeRoleNamespace+".")
}

func FilterAudioToolsSummarizeRoles(rows []*inventoryv1.ProviderRole) []DeclaredSummaryRole {
	out := make([]DeclaredSummaryRole, 0)
	for _, row := range rows {
		if row == nil || row.GetProvider() != "ai-gateway" || !IsAudioToolsSummarizeRole(row.GetRole()) {
			continue
		}
		out = append(out, DeclaredSummaryRole{ID: strings.TrimSpace(row.GetRole())})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ListAudioToolsSummarizeRoles uses AI Gateway inventory as the authority for
// which app-scoped task roles exist. The capability catalog contributes only
// human-readable descriptions; it is never used for validation or dispatch.
func ListAudioToolsSummarizeRoles(ctx context.Context) ([]DeclaredSummaryRole, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	return ListAudioToolsSummarizeRolesFromClients(ctx,
		inventoryconnect.NewInventoryServiceClient(client, url),
		catalogconnect.NewCatalogServiceClient(client, url))
}

type summaryInventoryClient interface {
	ListProviderRoles(context.Context, *connect.Request[inventoryv1.ListProviderRolesRequest]) (*connect.Response[inventoryv1.ListProviderRolesResponse], error)
}

type summaryCatalogClient interface {
	GetCapabilityCatalog(context.Context, *connect.Request[catalogv1.GetCapabilityCatalogRequest]) (*connect.Response[commonv1.AIGatewayCapabilityCatalog], error)
}

func ListAudioToolsSummarizeRolesFromClients(ctx context.Context, inventoryClient summaryInventoryClient, catalogClient summaryCatalogClient) ([]DeclaredSummaryRole, error) {
	inventory, err := inventoryClient.ListProviderRoles(ctx, connect.NewRequest(&inventoryv1.ListProviderRolesRequest{Provider: "ai-gateway"}))
	if err != nil {
		return nil, err
	}
	roles := FilterAudioToolsSummarizeRoles(inventory.Msg.GetRoles())
	if len(roles) == 0 {
		return nil, errors.New("AI Gateway inventory declared no Audio Tools summarize roles")
	}
	if catalog, err := catalogClient.GetCapabilityCatalog(ctx, connect.NewRequest(&catalogv1.GetCapabilityCatalogRequest{})); err == nil && catalog != nil && catalog.Msg != nil {
		descriptions := make(map[string]string, len(catalog.Msg.GetRoles()))
		for _, row := range catalog.Msg.GetRoles() {
			if row != nil {
				descriptions[row.GetId()] = row.GetDescription()
			}
		}
		for i := range roles {
			roles[i].Description = descriptions[roles[i].ID]
		}
	}
	return roles, nil
}

func IsDeclaredLocalSummarizeRole(ctx context.Context, id string) (bool, error) {
	if !IsAudioToolsSummarizeRole(id) {
		return false, nil
	}
	roles, err := ListAudioToolsSummarizeRoles(ctx)
	if err != nil {
		return false, err
	}
	return ContainsDeclaredAudioToolsSummarizeRole(roles, id), nil
}

func ContainsDeclaredAudioToolsSummarizeRole(roles []DeclaredSummaryRole, id string) bool {
	id = strings.TrimSpace(id)
	for _, role := range roles {
		if role.ID == id {
			return true
		}
	}
	return false
}
