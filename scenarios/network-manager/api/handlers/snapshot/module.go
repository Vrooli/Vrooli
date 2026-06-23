package snapshot

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	snapshotv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot"
	snapshotconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot/snapshot_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := snapshotconnect.NewSnapshotServiceHandler(&handler{})
	return module.Module{
		Name: "snapshot",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

func (h *handler) RunSnapshot(_ context.Context, req *connect.Request[snapshotv1.RunSnapshotRequest]) (*connect.Response[snapshotv1.RunSnapshotResponse], error) {
	profile := req.Msg.GetProfile()
	if profile == "" {
		profile = "home"
	}
	return connect.NewResponse(&snapshotv1.RunSnapshotResponse{Snapshot: sampleSnapshot("snapshot-preview", profile)}), nil
}

func (h *handler) ListSnapshots(context.Context, *connect.Request[snapshotv1.ListSnapshotsRequest]) (*connect.Response[snapshotv1.ListSnapshotsResponse], error) {
	return connect.NewResponse(&snapshotv1.ListSnapshotsResponse{Snapshots: []*snapshotv1.Snapshot{sampleSnapshot("snapshot-preview", "home")}}), nil
}

func (h *handler) GetSnapshot(_ context.Context, req *connect.Request[snapshotv1.GetSnapshotRequest]) (*connect.Response[snapshotv1.GetSnapshotResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		id = "snapshot-preview"
	}
	return connect.NewResponse(&snapshotv1.GetSnapshotResponse{Snapshot: sampleSnapshot(id, "home")}), nil
}

func (h *handler) ExportSnapshotReport(_ context.Context, req *connect.Request[snapshotv1.ExportSnapshotReportRequest]) (*connect.Response[snapshotv1.ExportSnapshotReportResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		id = "snapshot-preview"
	}
	format := req.Msg.GetFormat()
	if format == "" {
		format = "markdown"
	}
	return connect.NewResponse(&snapshotv1.ExportSnapshotReportResponse{
		Id:     id,
		Format: format,
		Report: "Network Manager snapshot reporting is wired; real probes are implemented in the snapshot domain next.",
	}), nil
}

func sampleSnapshot(id, profile string) *snapshotv1.Snapshot {
	return &snapshotv1.Snapshot{
		Id:        id,
		Status:    "preview",
		Profile:   profile,
		Summary:   "Read-only snapshot contract is available; live probes are not implemented yet.",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Metrics: []*snapshotv1.Metric{
			{Name: "gateway_reachability", Value: "pending", Unit: "status", Status: "not_measured"},
			{Name: "dns_latency", Value: "pending", Unit: "ms", Status: "not_measured"},
		},
		Findings: []string{"No network mutations are performed by this scaffolded snapshot."},
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("snapshot_run", snapshotconnect.SnapshotServiceRunSnapshotProcedure, "Run network health snapshot", "snapshot"),
	connectEndpoint("snapshot_list", snapshotconnect.SnapshotServiceListSnapshotsProcedure, "List network health snapshots", "snapshot"),
	connectEndpoint("snapshot_get", snapshotconnect.SnapshotServiceGetSnapshotProcedure, "Get network health snapshot", "snapshot"),
	connectEndpoint("snapshot_export", snapshotconnect.SnapshotServiceExportSnapshotReportProcedure, "Export network health report", "snapshot"),
}

func connectEndpoint(id, path, summary, category string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:       id,
		Path:     path,
		Method:   "POST",
		Summary:  summary,
		Category: category,
		Request:  &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"status": "proto response"}},
	}
}
