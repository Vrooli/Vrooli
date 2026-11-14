// Package fleet mounts performance-health's FleetService — deterministic,
// structured offender queries about scenario performance across the fleet.
package fleet

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internalfleet "performance-health/internal/fleet"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet/fleet_v1connect"
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

// ScanFleet grades the requested scenarios (or every enumerated one) and rolls
// up offender lists and the tier distribution.
func (h *Handler) ScanFleet(ctx context.Context, req *connect.Request[fleetv1.ScanFleetRequest]) (*connect.Response[fleetv1.ScanFleetResponse], error) {
	res, err := h.svc.Scan(ctx, req.Msg.GetScenarios())
	if err != nil {
		h.logger.Printf("fleet.ScanFleet: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &fleetv1.ScanFleetResponse{
		ScenarioCount:  int32(res.ScenarioCount),
		NoBudgetCount:  int32(res.NoBudgetCount),
		RegressedCount: int32(res.RegressedCount),
	}
	for _, e := range res.Entries {
		out.Entries = append(out.Entries, &fleetv1.FleetScenarioEntry{
			Scenario:       e.Scenario,
			Tier:           e.Tier,
			HasBudget:      e.HasBudget,
			GoBuildMs:      e.GoBuildMs,
			UiBuildMs:      e.UIBuildMs,
			Regressed:      e.Regressed,
			DegradedReason: e.DegradedReason,
		})
	}
	for _, d := range res.TierDistribution {
		out.TierDistribution = append(out.TierDistribution, &fleetv1.TierDistribution{
			Tier:          d.Tier,
			ScenarioCount: int32(d.ScenarioCount),
		})
	}
	for _, e := range res.Errors {
		out.Errors = append(out.Errors, &fleetv1.FleetScanError{Scenario: e.Scenario, Reason: e.Reason})
	}
	return connect.NewResponse(out), nil
}
