// Package fleet mounts structure-health's FleetService — deterministic,
// structured offender queries over the whole fleet's structure conformance.
package fleet

import (
	"context"
	"log"

	"connectrpc.com/connect"

	internalfleet "structure-health/internal/fleet"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet/fleet_v1connect"
)

// Handler implements the generated FleetServiceHandler.
type Handler struct {
	fleetconnect.UnimplementedFleetServiceHandler
	scanner *internalfleet.Scanner
	logger  *log.Logger
}

// NewHandler builds a fleet Handler.
func NewHandler(scanner *internalfleet.Scanner, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{scanner: scanner, logger: logger}
}

var _ fleetconnect.FleetServiceHandler = (*Handler)(nil)

// ScanFleet grades the requested scenarios (or every discovered scenario) and
// returns the aggregated structure rollup.
func (h *Handler) ScanFleet(ctx context.Context, req *connect.Request[fleetv1.ScanFleetRequest]) (*connect.Response[fleetv1.ScanFleetResponse], error) {
	var result internalfleet.Result
	var err error
	if len(req.Msg.GetTargets()) > 0 {
		targets := make([]internalfleet.Target, 0, len(req.Msg.GetTargets()))
		for _, target := range req.Msg.GetTargets() {
			targets = append(targets, internalfleet.Target{Kind: target.GetKind(), ID: target.GetId(), Root: target.GetPath()})
		}
		result, err = h.scanner.ScanTargets(ctx, targets)
	} else {
		result, err = h.scanner.Scan(ctx, req.Msg.GetScenarios())
	}
	if err != nil {
		h.logger.Printf("fleet.ScanFleet: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resultToProto(result)), nil
}

func resultToProto(in internalfleet.Result) *fleetv1.ScanFleetResponse {
	out := &fleetv1.ScanFleetResponse{
		ScenarioCount:      int32(in.ScenarioCount),
		PassingCount:       int32(in.PassingCount),
		AutofixableTotal:   int32(in.AutofixableTotal),
		TargetCount:        int32(in.TargetCount),
		PassingTargetCount: int32(in.PassingTargetCount),
	}
	for _, e := range in.Entries {
		out.Entries = append(out.Entries, &fleetv1.FleetScenarioEntry{
			Scenario:          e.Scenario,
			Passed:            e.Passed,
			ProfileId:         e.ProfileID,
			ProfileRecognized: e.ProfileRecognized,
			ErrorCount:        int32(e.ErrorCount),
			WarningCount:      int32(e.WarningCount),
			TotalFindings:     int32(e.TotalFindings),
			AutofixableCount:  int32(e.AutofixableCount),
			Surfaces:          e.Surfaces,
			DegradedReason:    e.DegradedReason,
			TargetKind:        e.TargetKind,
			TargetId:          e.TargetID,
			TargetPath:        e.TargetRoot,
		})
	}
	for _, rc := range in.RuleConformance {
		out.RuleConformance = append(out.RuleConformance, &fleetv1.RuleConformance{
			Code:               rc.Code,
			OffendingScenarios: int32(rc.OffendingScenarios),
			TotalFindings:      int32(rc.TotalFindings),
			Autofixable:        int32(rc.Autofixable),
			WorstSeverity:      rc.WorstSeverity,
		})
	}
	for _, pc := range in.ProfileDistribution {
		out.ProfileDistribution = append(out.ProfileDistribution, &fleetv1.ProfileDistribution{
			ProfileId:     pc.ProfileID,
			ScenarioCount: int32(pc.ScenarioCount),
			Recognized:    pc.Recognized,
		})
	}
	for _, e := range in.Errors {
		out.Errors = append(out.Errors, &fleetv1.FleetScanError{Scenario: e.Scenario, Reason: e.Reason})
	}
	return out
}
