package graph

// DOC: docs/reference/retention.md

import (
	"context"
	"errors"

	"architecture-cartographer/internal/graph"

	"connectrpc.com/connect"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
)

// PreviewSnapshotRetention reports reclaimable snapshot storage without
// deleting anything.
func (h *Handler) PreviewSnapshotRetention(ctx context.Context, req *connect.Request[graphv1.PreviewSnapshotRetentionRequest]) (*connect.Response[graphv1.PreviewSnapshotRetentionResponse], error) {
	preview, err := h.svc.PreviewSnapshotRetention(ctx, int(req.Msg.GetKeepPerScenario()))
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}

	out := &graphv1.PreviewSnapshotRetentionResponse{
		ReclaimableBytes: preview.ReclaimableBytes,
		ReclaimableRows:  int32(preview.ReclaimableRows),
		KeepPerScenario:  int32(preview.KeepPerScenario),
		TotalSnapshots:   int32(preview.TotalSnapshots),
	}
	for _, s := range preview.Scenarios {
		out.Scenarios = append(out.Scenarios, &graphv1.ScenarioSnapshotCount{
			Scenario:         s.Scenario,
			SnapshotCount:    int32(s.SnapshotCount),
			ReclaimableCount: int32(s.ReclaimableCount),
		})
	}
	return connect.NewResponse(out), nil
}

// ApplySnapshotRetention prunes snapshots beyond the retention floor.
//
// Confirmation is required rather than implied. Deletion is irreversible, so
// an empty request must never mean "delete": a caller that forgets to set the
// field gets an error, not a prune.
func (h *Handler) ApplySnapshotRetention(ctx context.Context, req *connect.Request[graphv1.ApplySnapshotRetentionRequest]) (*connect.Response[graphv1.ApplySnapshotRetentionResponse], error) {
	if !req.Msg.GetConfirm() {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("confirm must be true: applying retention permanently deletes snapshots"))
	}

	result, err := h.svc.ApplySnapshotRetention(ctx, int(req.Msg.GetKeepPerScenario()))
	if err != nil {
		return nil, connect.NewError(graph.ErrorToConnectCode(err), err)
	}
	return connect.NewResponse(&graphv1.ApplySnapshotRetentionResponse{
		RowsRemoved:      int32(result.RowsRemoved),
		BytesReclaimed:   result.BytesReclaimed,
		PagesFreed:       result.PagesFreed,
		ScenariosScanned: int32(result.ScenariosScanned),
		KeepPerScenario:  int32(req.Msg.GetKeepPerScenario()),
	}), nil
}
