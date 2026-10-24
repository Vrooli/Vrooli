// Package fleet mounts the FleetService over the compute-on-read sweep
// (internal/fleet): every discovered scenario graded through the same
// check engine as per-scenario validation, worst-first with as-of stamps.
package fleet

import (
	"context"
	"log"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"google.golang.org/protobuf/types/known/timestamppb"

	localautofix "business-health/internal/autofix"
	internalfleet "business-health/internal/fleet"
	"business-health/internal/module"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/fleet/fleet_v1connect"
)

var ProtoFile = fleetv1.File_business_health_v1_fleet_fleet_proto

type handler struct {
	logger  *log.Logger
	sweeper *internalfleet.Sweeper
}

// Module wires the fleet service over the shared check engine.
func Module(logger *log.Logger, repoRoot string, engine internalfleet.Engine) module.Module {
	h := &handler{
		logger:  logger,
		sweeper: internalfleet.NewSweeper(repoRoot, engine, localautofix.RuleIDs(), nil),
	}
	path, svc := fleetconnect.NewFleetServiceHandler(h)
	return module.Module{
		Name: "fleet",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

func (h *handler) ScanFleet(ctx context.Context, req *connect.Request[fleetv1.ScanFleetRequest]) (*connect.Response[fleetv1.ScanFleetResponse], error) {
	result, err := h.sweeper.Scan(ctx, req.Msg.GetScenarios())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &fleetv1.ScanFleetResponse{
		AsOf:                 timestamppb.New(result.AsOf),
		ScenarioCount:        int32(result.ScenarioCount),
		PassingCount:         int32(result.PassingCount),
		StarterRegistryCount: int32(result.StarterRegistryCount),
		TemplateLaggardCount: int32(result.TemplateLaggardCount),
	}
	for _, e := range result.Entries {
		resp.Entries = append(resp.Entries, &fleetv1.FleetScenarioEntry{
			Scenario:         e.Scenario,
			Passed:           e.Passed,
			ErrorCount:       int32(e.ErrorCount),
			WarningCount:     int32(e.WarningCount),
			TotalFindings:    int32(e.TotalFindings),
			AutofixableCount: int32(e.AutofixableCount),
			StarterRegistry:  e.StarterRegistry,
			TemplateVersion:  e.TemplateVersion,
			TemplateLaggard:  e.TemplateLaggard,
			OrphanedTargets:  int32(e.OrphanedTargets),
			UnprovenClaims:   int32(e.UnprovenClaims),
			DebtScore:        int32(e.DebtScore),
			DegradedReason:   e.DegradedReason,
		})
	}
	for _, se := range result.Errors {
		resp.Errors = append(resp.Errors, &fleetv1.FleetScanError{Scenario: se.Scenario, Reason: se.Reason})
	}
	return connect.NewResponse(resp), nil
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "fleet_scan",
		Path:        fleetconnect.FleetServiceScanFleetProcedure,
		Method:      "POST",
		Summary:     "Fleet-wide business-contract debt sweep",
		Description: "Grades every discovered scenario's business contract (starter registries, template laggards, orphaned targets, unproven claims) through the same check engine as per-scenario validation, worst-first with as-of stamps. Compute-on-read; nothing cached.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string> (optional subset; empty = every discovered scenario)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"entries": "array<FleetScenarioEntry> (worst-first)", "as_of": "timestamp", "scenario_count": "int", "passing_count": "int", "errors": "array<FleetScanError>"}},
		Examples: []module.Example{
			{Name: "Scan the fleet", Curl: "curl http://localhost:${API_PORT}/vrooli.business_health.v1.fleet.FleetService/ScanFleet -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
