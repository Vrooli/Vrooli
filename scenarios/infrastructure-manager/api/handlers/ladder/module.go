// Package ladder mounts the device-layer capability ladder's typed read
// surface. It is read-only by contract: the instrument has no controller
// letter, so no verb here may restart, reconcile or mutate anything.
package ladder

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
	ladderv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder/ladder_v1connect"
	internalladder "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/ladder"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// appliedCascade is the rule the ranking applied, stated on every ranked
// response so a reader never has to infer the order from the output.
const appliedCascade = "sensor-channel integrity, host substrate, capability availability, efficiency, measurement improvement"

// Snapshotter is the ladder read the handler needs.
type Snapshotter interface {
	Snapshot(ctx context.Context) internalladder.Snapshot
}

func Module(service Snapshotter) module.Module {
	path, handler := ladderv1connect.NewLadderServiceHandler(&connectHandler{service: service})
	return module.Module{
		Name:      "ladder",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

type connectHandler struct{ service Snapshotter }

func (h *connectHandler) GetLadder(ctx context.Context, _ *connect.Request[ladderv1.GetLadderRequest]) (*connect.Response[ladderv1.GetLadderResponse], error) {
	snapshot := h.service.Snapshot(ctx)
	return connect.NewResponse(&ladderv1.GetLadderResponse{Ladder: protoLadder(snapshot)}), nil
}

func (h *connectHandler) ListCells(ctx context.Context, req *connect.Request[ladderv1.ListCellsRequest]) (*connect.Response[ladderv1.ListCellsResponse], error) {
	snapshot := h.service.Snapshot(ctx)
	response := &ladderv1.ListCellsResponse{ComputedAt: timestamppb.New(snapshot.ComputedAt)}
	for _, cell := range snapshot.Cells {
		if !cellMatches(cell, req.Msg) {
			continue
		}
		response.Cells = append(response.Cells, protoCell(cell))
	}
	return connect.NewResponse(response), nil
}

// ListDevices returns the graded hardware inventory. When the device-graph
// source could not be read the response is explicitly unavailable rather than
// an empty list: an empty list served OK is a claim that the host has no
// hardware.
func (h *connectHandler) ListDevices(ctx context.Context, req *connect.Request[ladderv1.ListDevicesRequest]) (*connect.Response[ladderv1.ListDevicesResponse], error) {
	snapshot := h.service.Snapshot(ctx)
	response := &ladderv1.ListDevicesResponse{ComputedAt: timestamppb.New(snapshot.ComputedAt), Available: true}
	for _, source := range snapshot.Sources {
		if source.ID == sources.DeviceGraphSourceID && !source.Available {
			response.Available = false
			response.UnavailableReason = source.Reason
		}
	}
	class := strings.TrimSpace(req.Msg.GetDeviceClass())
	id := strings.TrimSpace(req.Msg.GetDeviceId())
	for _, device := range snapshot.Devices {
		if class != "" && class != device.Class {
			continue
		}
		if id != "" && id != device.ID {
			continue
		}
		response.Devices = append(response.Devices, protoDevice(device))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListSources(ctx context.Context, _ *connect.Request[ladderv1.ListSourcesRequest]) (*connect.Response[ladderv1.ListSourcesResponse], error) {
	snapshot := h.service.Snapshot(ctx)
	response := &ladderv1.ListSourcesResponse{ComputedAt: timestamppb.New(snapshot.ComputedAt)}
	for _, source := range snapshot.Sources {
		response.Sources = append(response.Sources, protoSource(source))
	}
	for _, coverage := range snapshot.CheckPlatforms {
		response.CheckPlatforms = append(response.CheckPlatforms, protoCheckPlatform(coverage))
	}
	response.Confidence = protoConfidence(snapshot.Confidence)
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RankFindings(ctx context.Context, req *connect.Request[ladderv1.RankFindingsRequest]) (*connect.Response[ladderv1.RankFindingsResponse], error) {
	snapshot := h.service.Snapshot(ctx)
	response := &ladderv1.RankFindingsResponse{
		AppliedCascade: appliedCascade,
		ComputedAt:     timestamppb.New(snapshot.ComputedAt),
	}
	wanted := req.Msg.GetStage()
	for _, finding := range snapshot.Findings {
		item := protoFinding(finding)
		if wanted != ladderv1.CascadeStage_CASCADE_STAGE_UNSPECIFIED && item.GetStage() != wanted {
			continue
		}
		response.Findings = append(response.Findings, item)
	}
	return connect.NewResponse(response), nil
}

func cellMatches(cell internalladder.Cell, filter *ladderv1.ListCellsRequest) bool {
	if class := strings.TrimSpace(filter.GetDeviceClass()); class != "" && class != cell.Key.DeviceClass {
		return false
	}
	if hostOS := strings.TrimSpace(filter.GetHostOs()); hostOS != "" && hostOS != cell.Key.HostOS {
		return false
	}
	if cellRef := strings.TrimSpace(filter.GetCellRef()); cellRef != "" && cellRef != cell.CellRef {
		return false
	}
	if filter.GetRung() != ladderv1.Rung_RUNG_UNSPECIFIED && protoRung(cell.Key.Rung) != filter.GetRung() {
		return false
	}
	return true
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{ID: "ladder_get", Path: ladderv1connect.LadderServiceGetLadderProcedure, Method: "POST", Summary: "Read the device-layer capability ladder", Category: "ladder"},
	{ID: "ladder_cells", Path: ladderv1connect.LadderServiceListCellsProcedure, Method: "POST", Summary: "List ladder cells", Category: "ladder"},
	{ID: "ladder_devices", Path: ladderv1connect.LadderServiceListDevicesProcedure, Method: "POST", Summary: "List graded hardware devices", Category: "ladder"},
	{ID: "ladder_sources", Path: ladderv1connect.LadderServiceListSourcesProcedure, Method: "POST", Summary: "List ladder source availability", Category: "ladder"},
	{ID: "ladder_findings", Path: ladderv1connect.LadderServiceRankFindingsProcedure, Method: "POST", Summary: "Read cascade-ranked ladder findings", Category: "ladder"},
}
