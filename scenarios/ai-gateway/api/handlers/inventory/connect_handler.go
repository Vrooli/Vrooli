package inventory

import (
	"context"
	"math"

	domain "ai-gateway/internal/inventory"
	"ai-gateway/internal/providers"

	"connectrpc.com/connect"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
)

type connectHandler struct {
	service *domain.Service
}

type Deps struct {
	Service *domain.Service
	Runner  providers.CommandRunner
}

func NewConnectHandler(deps Deps) *connectHandler {
	service := deps.Service
	if service == nil {
		service = domain.NewService(providers.NewDefaultAdapters(deps.Runner))
	}
	return &connectHandler{service: service}
}

func (h *connectHandler) ListProviderRoles(ctx context.Context, req *connect.Request[inventoryv1.ListProviderRolesRequest]) (*connect.Response[inventoryv1.ListProviderRolesResponse], error) {
	roles, warnings := h.service.ListProviderRoles(ctx, req.Msg.GetProvider())
	out := &inventoryv1.ListProviderRolesResponse{Warnings: warnings}
	for _, role := range roles {
		out.Roles = append(out.Roles, &inventoryv1.ProviderRole{
			Provider:            role.Provider,
			Role:                role.Role,
			Capabilities:        role.Capabilities,
			Locality:            role.Locality,
			Status:              role.Status,
			PolicySchemaVersion: role.PolicySchemaVersion,
		})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SmokeProvider(ctx context.Context, req *connect.Request[inventoryv1.SmokeProviderRequest]) (*connect.Response[inventoryv1.SmokeProviderResponse], error) {
	smoke := h.service.SmokeProvider(ctx, req.Msg.GetProvider())
	return connect.NewResponse(&inventoryv1.SmokeProviderResponse{
		Provider: smoke.Provider,
		Status:   smoke.Status,
		Code:     smoke.Code,
		Message:  smoke.Message,
		ExitCode: safeInt32(smoke.ExitCode),
		Warnings: smoke.Warnings,
	}), nil
}

func safeInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
