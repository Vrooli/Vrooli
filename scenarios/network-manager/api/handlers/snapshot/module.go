package snapshot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainsnapshot "network-manager/internal/snapshot"

	snapshotv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot"
	snapshotconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot/snapshot_v1connect"
)

type handler struct {
	service *domainsnapshot.Service
}

func Module(db domainsnapshot.SQLExecutor) module.Module {
	service := domainsnapshot.NewService(domainsnapshot.Config{
		Repo:   domainsnapshot.NewSQLiteRepository(db),
		Runner: domainsnapshot.RealProbeRunner{},
	})
	path, h := snapshotconnect.NewSnapshotServiceHandler(&handler{service: service})
	return module.Module{
		Name: "snapshot",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return domainsnapshot.Schema() }

func (h *handler) RunSnapshot(ctx context.Context, req *connect.Request[snapshotv1.RunSnapshotRequest]) (*connect.Response[snapshotv1.RunSnapshotResponse], error) {
	s, err := h.service.Run(ctx, req.Msg.GetProfile(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&snapshotv1.RunSnapshotResponse{Snapshot: toProto(s)}), nil
}

func (h *handler) ListSnapshots(ctx context.Context, _ *connect.Request[snapshotv1.ListSnapshotsRequest]) (*connect.Response[snapshotv1.ListSnapshotsResponse], error) {
	snapshots, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*snapshotv1.Snapshot, 0, len(snapshots))
	for _, s := range snapshots {
		out = append(out, toProto(s))
	}
	return connect.NewResponse(&snapshotv1.ListSnapshotsResponse{Snapshots: out}), nil
}

func (h *handler) GetSnapshot(ctx context.Context, req *connect.Request[snapshotv1.GetSnapshotRequest]) (*connect.Response[snapshotv1.GetSnapshotResponse], error) {
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot id is required"))
	}
	s, err := h.service.Get(ctx, id)
	if errors.Is(err, domainsnapshot.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("snapshot %q not found", id))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&snapshotv1.GetSnapshotResponse{Snapshot: toProto(s)}), nil
}

func (h *handler) ExportSnapshotReport(ctx context.Context, req *connect.Request[snapshotv1.ExportSnapshotReportRequest]) (*connect.Response[snapshotv1.ExportSnapshotReportResponse], error) {
	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot id is required"))
	}
	id, format, report, err := h.service.Export(ctx, req.Msg.GetId(), req.Msg.GetFormat())
	if errors.Is(err, domainsnapshot.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("snapshot %q not found", req.Msg.GetId()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&snapshotv1.ExportSnapshotReportResponse{
		Id:     id,
		Format: format,
		Report: report,
	}), nil
}

func toProto(s domainsnapshot.Snapshot) *snapshotv1.Snapshot {
	metrics := make([]*snapshotv1.Metric, 0, len(s.Metrics))
	for _, m := range s.Metrics {
		metrics = append(metrics, &snapshotv1.Metric{Name: m.Name, Value: m.Value, Unit: m.Unit, Status: m.Status})
	}
	return &snapshotv1.Snapshot{
		Id:        s.ID,
		Status:    s.Status,
		Profile:   s.Profile,
		Summary:   s.Summary,
		Metrics:   metrics,
		Findings:  s.Findings,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
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
