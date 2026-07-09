// Package fleet mounts storage-health's FleetService — deterministic,
// structured storage-inventory queries across the whole fleet.
package fleet

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"

	internalfleet "storage-health/internal/fleet"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet/fleet_v1connect"
)

// Handler implements the generated FleetServiceHandler.
type Handler struct {
	fleetconnect.UnimplementedFleetServiceHandler
	svc    *internalfleet.Service
	logger *log.Logger
}

// NewHandler builds a fleet Handler.
func NewHandler(svc *internalfleet.Service, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

var _ fleetconnect.FleetServiceHandler = (*Handler)(nil)

// ScanFleet classifies the requested scenarios (or every enumerated one), rolls
// up the inventory + offender counts, persists the snapshot, and returns it.
func (h *Handler) ScanFleet(ctx context.Context, req *connect.Request[fleetv1.ScanFleetRequest]) (*connect.Response[fleetv1.ScanFleetResponse], error) {
	res, err := h.svc.Scan(ctx, req.Msg.GetScenarios())
	if err != nil {
		h.logger.Printf("fleet.ScanFleet: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProto(res)), nil
}

// GetInventory returns the latest persisted snapshot without re-scanning.
func (h *Handler) GetInventory(ctx context.Context, _ *connect.Request[fleetv1.GetInventoryRequest]) (*connect.Response[fleetv1.ScanFleetResponse], error) {
	res, err := h.svc.Inventory(ctx)
	if err != nil {
		h.logger.Printf("fleet.GetInventory: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toProto(res)), nil
}

// toProto projects the engine Result onto the wire response.
func toProto(res internalfleet.Result) *fleetv1.ScanFleetResponse {
	out := &fleetv1.ScanFleetResponse{
		ScenarioCount:          int32(res.ScenarioCount),
		IsolationUnreadyCount:  int32(res.IsolationUnreadyCount),
		NoBackupCount:          int32(res.NoBackupCount),
		FindingCount:           int32(res.FindingCount),
		DataDirOverBudgetCount: int32(res.DataDirOverBudgetCount),
	}
	if !res.ScannedAt.IsZero() {
		out.ScannedAt = res.ScannedAt.UTC().Format(time.RFC3339)
	}
	for _, e := range res.Entries {
		out.Entries = append(out.Entries, &fleetv1.FleetScenarioEntry{
			Scenario:           e.Scenario,
			Engines:            e.Engines,
			PrimaryEngine:      e.PrimaryEngine,
			Language:           e.Language,
			StorageStage:       e.StorageStage,
			IsolationReady:     e.IsolationReady,
			IsolationReason:    e.IsolationReason,
			NamespaceAdopted:   e.NamespaceAdopted,
			HasBackupTarget:    e.HasBackupTarget,
			FindingCount:       int32(e.FindingCount),
			ErrorCount:         int32(e.ErrorCount),
			AutofixableCount:   int32(e.AutofixableCount),
			DataDirBytes:       e.DataDirBytes,
			DataDirBudgetBytes: e.DataDirBudget,
			DataDirUtilization: e.DataDirUtil,
			DataDirOverBudget:  e.DataDirOverBudget,
			DataDirSeverity:    e.DataDirSeverity,
			DataDirPaths:       e.DataDirPaths,
		})
	}
	for _, d := range res.EngineDistribution {
		out.EngineDistribution = append(out.EngineDistribution, &fleetv1.EngineCount{Engine: d.Engine, ScenarioCount: int32(d.ScenarioCount)})
	}
	for _, d := range res.StageDistribution {
		out.StageDistribution = append(out.StageDistribution, &fleetv1.StageCount{Stage: d.Stage, ScenarioCount: int32(d.ScenarioCount)})
	}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &fleetv1.FleetScanError{Scenario: e.Scenario, Reason: e.Reason})
	}
	return out
}
