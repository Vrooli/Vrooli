// Package portability mounts the capability-availability domain's typed read
// surface. It aggregates the platform declarations authored beside each tool,
// safeguard and scenario manifest; it authors none of them and mutates
// nothing.
package portability

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
	portabilityv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability/portability_v1connect"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"
	internalportability "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Module(root string) module.Module {
	service := internalportability.NewService(root, nil)
	path, handler := portabilityv1connect.NewPortabilityServiceHandler(&connectHandler{service: service})
	return module.Module{
		Name:      "portability",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

type connectHandler struct{ service *internalportability.Service }

// gridError distinguishes an unusable manifest root from a malformed manifest.
// An unresolvable root is FailedPrecondition and names the root that was
// tried; anything else is Internal. Neither is ever answered with an empty
// grid, which would read as "this repository declares no capabilities".
func gridError(err error) error {
	if internalportability.IsUnresolvedRoot(err) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *connectHandler) GetGrid(ctx context.Context, req *connect.Request[portabilityv1.GetGridRequest]) (*connect.Response[portabilityv1.GetGridResponse], error) {
	grid, err := h.service.Grid(ctx)
	if err != nil {
		return nil, gridError(err)
	}
	return connect.NewResponse(&portabilityv1.GetGridResponse{Grid: protoGrid(grid, filterCapabilities(grid, req.Msg.GetCapabilities()))}), nil
}

func (h *connectHandler) GetCapability(ctx context.Context, req *connect.Request[portabilityv1.GetCapabilityRequest]) (*connect.Response[portabilityv1.GetCapabilityResponse], error) {
	name := req.Msg.GetCapability()
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("capability is required"))
	}
	grid, err := h.service.Grid(ctx)
	if err != nil {
		return nil, gridError(err)
	}
	entry, ok := grid.Capability(name)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("capability %q is not in the capability vocabulary at %s", name, grid.ManifestRoot))
	}
	return connect.NewResponse(&portabilityv1.GetCapabilityResponse{
		Capability:   protoEntry(entry),
		ManifestRoot: grid.ManifestRoot,
		ComputedAt:   timestamppb.New(grid.ComputedAt),
	}), nil
}

func (h *connectHandler) ListSituations(ctx context.Context, req *connect.Request[portabilityv1.ListSituationsRequest]) (*connect.Response[portabilityv1.ListSituationsResponse], error) {
	grid, err := h.service.Grid(ctx)
	if err != nil {
		return nil, gridError(err)
	}
	wanted := req.Msg.GetSituation()
	response := &portabilityv1.ListSituationsResponse{
		ManifestRoot: grid.ManifestRoot,
		ComputedAt:   timestamppb.New(grid.ComputedAt),
	}
	for _, entry := range grid.Capabilities {
		if wanted != portabilityv1.CapabilitySituation_CAPABILITY_SITUATION_UNSPECIFIED && protoSituation(entry.Situation) != wanted {
			continue
		}
		response.Capabilities = append(response.Capabilities, protoEntry(entry))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) GetFleet(ctx context.Context, _ *connect.Request[portabilityv1.GetFleetRequest]) (*connect.Response[portabilityv1.GetFleetResponse], error) {
	readout, err := h.service.Fleet(ctx)
	if err != nil {
		return nil, gridError(err)
	}
	return connect.NewResponse(&portabilityv1.GetFleetResponse{Fleet: protoFleet(readout)}), nil
}

// filterCapabilities narrows the grid to the requested names. A requested name
// that is not in the vocabulary is simply absent from the result rather than
// invented as an empty row, and the caller can see the difference by comparing
// against GetCapability's explicit NotFound.
func filterCapabilities(grid internalportability.Grid, requested []string) []internalportability.Entry {
	if len(requested) == 0 {
		return grid.Capabilities
	}
	wanted := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		wanted[name] = struct{}{}
	}
	out := make([]internalportability.Entry, 0, len(requested))
	for _, entry := range grid.Capabilities {
		if _, ok := wanted[entry.Capability]; ok {
			out = append(out, entry)
		}
	}
	return out
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "portability_grid", Path: portabilityv1connect.PortabilityServiceGetGridProcedure, Method: "POST", Summary: "Read the capability portability grid", Category: "portability"},
	{ID: "portability_capability", Path: portabilityv1connect.PortabilityServiceGetCapabilityProcedure, Method: "POST", Summary: "Read one capability's platform row", Category: "portability"},
	{ID: "portability_situations", Path: portabilityv1connect.PortabilityServiceListSituationsProcedure, Method: "POST", Summary: "List capabilities by situation classification", Category: "portability"},
	{ID: "portability_fleet", Path: portabilityv1connect.PortabilityServiceGetFleetProcedure, Method: "POST", Summary: "Read the computed fleet portability view", Category: "portability"},
}
