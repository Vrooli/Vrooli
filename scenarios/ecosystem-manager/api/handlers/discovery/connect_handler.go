// Package discovery is the Connect-RPC transport edge for the discovery domain.
// The handler is thin: it translates between the generated proto messages and
// the transport-agnostic internal/discovery service, and maps the service's
// sentinel errors via ToConnectError. All behaviour lives in
// github.com/ecosystem-manager/api/internal/discovery.
package discovery

import (
	"context"

	"connectrpc.com/connect"

	internaldiscovery "github.com/ecosystem-manager/api/internal/discovery"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/tasks"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery"
)

// Deps are the live dependencies the discovery Connect handler needs.
type Deps struct {
	// Assembler supplies the operation catalogue for ListOperations.
	Assembler *prompts.Assembler
}

// ConnectHandler implements the generated DiscoveryServiceHandler interface.
type ConnectHandler struct {
	svc *internaldiscovery.Service
}

// NewConnectHandler builds the handler over a fresh internal/discovery service.
func NewConnectHandler(deps Deps) *ConnectHandler {
	return &ConnectHandler{svc: internaldiscovery.NewService(deps.Assembler)}
}

// ListResources returns every discovered local resource.
func (h *ConnectHandler) ListResources(
	_ context.Context, req *connect.Request[discoveryv1.ListResourcesRequest],
) (*connect.Response[discoveryv1.ListResourcesResponse], error) {
	resources, _, err := h.svc.Resources(req.Msg.GetRefresh())
	if err != nil && len(resources) == 0 {
		// Discovery failed with no last-good data to fall back on: report it
		// explicitly (Unavailable) rather than returning an empty list that
		// looks like "zero resources".
		return nil, internaldiscovery.ToConnectError(err)
	}
	protos := make([]*discoveryv1.Resource, 0, len(resources))
	for _, r := range resources {
		protos = append(protos, toProtoResource(r))
	}
	return connect.NewResponse(&discoveryv1.ListResourcesResponse{
		Resources: protos,
		Count:     int32(len(protos)),
	}), nil
}

// ListScenarios returns every discovered scenario.
func (h *ConnectHandler) ListScenarios(
	_ context.Context, req *connect.Request[discoveryv1.ListScenariosRequest],
) (*connect.Response[discoveryv1.ListScenariosResponse], error) {
	scenarios, _, err := h.svc.Scenarios(req.Msg.GetRefresh())
	if err != nil && len(scenarios) == 0 {
		return nil, internaldiscovery.ToConnectError(err)
	}
	protos := make([]*discoveryv1.Scenario, 0, len(scenarios))
	for _, s := range scenarios {
		protos = append(protos, toProtoScenario(s))
	}
	return connect.NewResponse(&discoveryv1.ListScenariosResponse{
		Scenarios: protos,
		Count:     int32(len(protos)),
	}), nil
}

// GetResource returns one discovered resource by name (NotFound if absent).
func (h *ConnectHandler) GetResource(
	_ context.Context, req *connect.Request[discoveryv1.GetResourceRequest],
) (*connect.Response[discoveryv1.Resource], error) {
	resource, _, err := h.svc.Resource(req.Msg.GetName(), req.Msg.GetRefresh())
	if err != nil {
		return nil, internaldiscovery.ToConnectError(err)
	}
	return connect.NewResponse(toProtoResource(resource)), nil
}

// GetScenario returns one discovered scenario by name (NotFound if absent).
func (h *ConnectHandler) GetScenario(
	_ context.Context, req *connect.Request[discoveryv1.GetScenarioRequest],
) (*connect.Response[discoveryv1.Scenario], error) {
	scenario, _, err := h.svc.Scenario(req.Msg.GetName(), req.Msg.GetRefresh())
	if err != nil {
		return nil, internaldiscovery.ToConnectError(err)
	}
	return connect.NewResponse(toProtoScenario(scenario)), nil
}

// ListOperations returns the configured task operation types.
func (h *ConnectHandler) ListOperations(
	_ context.Context, _ *connect.Request[discoveryv1.ListOperationsRequest],
) (*connect.Response[discoveryv1.ListOperationsResponse], error) {
	ops := h.svc.Operations()
	protos := make([]*discoveryv1.Operation, 0, len(ops))
	for _, op := range ops {
		protos = append(protos, &discoveryv1.Operation{Name: op.Name, Description: op.Description})
	}
	return connect.NewResponse(&discoveryv1.ListOperationsResponse{Operations: protos}), nil
}

// ListCategories returns the resource/scenario category groupings.
func (h *ConnectHandler) ListCategories(
	_ context.Context, _ *connect.Request[discoveryv1.ListCategoriesRequest],
) (*connect.Response[discoveryv1.ListCategoriesResponse], error) {
	return connect.NewResponse(&discoveryv1.ListCategoriesResponse{
		ResourceCategories: h.svc.ResourceCategories(),
		ScenarioCategories: h.svc.ScenarioCategories(),
	}), nil
}

func toProtoResource(r tasks.ResourceInfo) *discoveryv1.Resource {
	return &discoveryv1.Resource{
		Name:        r.Name,
		Path:        r.Path,
		Port:        int32(r.Port),
		Category:    r.Category,
		Description: r.Description,
		Healthy:     r.Healthy,
		Version:     r.Version,
		Status:      r.Status,
	}
}

func toProtoScenario(s tasks.ScenarioInfo) *discoveryv1.Scenario {
	return &discoveryv1.Scenario{
		Name:        s.Name,
		Path:        s.Path,
		Category:    s.Category,
		Description: s.Description,
		Version:     s.Version,
		Status:      s.Status,
	}
}
