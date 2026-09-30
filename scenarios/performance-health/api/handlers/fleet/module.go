package fleet

import (
	"context"
	"database/sql"
	"log"

	internalbudgets "performance-health/internal/budgets"
	"performance-health/internal/fleet"
	"performance-health/internal/module"
	"performance-health/internal/readiness"
	"performance-health/internal/trend"

	"github.com/gorilla/mux"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet/fleet_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted FleetService.
var ProtoFile = fleetv1.File_performance_health_v1_fleet_fleet_proto

// Module mounts the FleetService backed by the real grader (composing the
// declarative budget config + the persisted trend + readiness tier) and the
// real enumerator (the typed `vrooli scenario list` contract).
func Module(logger *log.Logger, repoRoot string, db *sql.DB) module.Module {
	budgetStore := internalbudgets.NewConfigStore(repoRoot, nil)
	var trendStore trendReader
	if db != nil {
		trendStore = trend.NewStore(db)
	} else {
		trendStore = emptyTrend{}
	}
	readinessSvc := readiness.NewService(readiness.NewCodeFactsClient(repoRoot))

	g := newGrader(budgetStore, trendStore, readinessSvc)
	svc := fleet.NewService(g, newCLIEnumerator())
	handler := NewHandler(svc, logger)
	path, connectHandler := fleetconnect.NewFleetServiceHandler(handler)
	return module.Module{
		Name: "fleet",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// emptyTrend is the no-DB fallback: every scenario reads back as having no
// persisted sample (so build times are unknown and nothing is "regressed").
type emptyTrend struct{}

func (emptyTrend) Latest(context.Context, string) (trend.Sample, bool, error) {
	return trend.Sample{}, false, nil
}

// Schema returns the empty schema: fleet owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "fleet_scan_fleet",
		Path:        fleetconnect.FleetServiceScanFleetProcedure,
		Method:      "POST",
		Summary:     "Scan the fleet for performance offenders",
		Description: "Grades the requested scenarios (or every discovered scenario) and rolls up offender lists (no budget, slow build, regressed) plus the capture-tier distribution.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"entries": "array<FleetScenarioEntry>", "tier_distribution": "array<TierDistribution>", "scenario_count": "int32", "no_budget_count": "int32", "regressed_count": "int32", "errors": "array<FleetScanError>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Fleet scan failure"}},
	},
}
