package fleet

import (
	"log"

	internalfleet "structure-health/internal/fleet"
	"structure-health/internal/module"
	"structure-health/internal/profile"
	internalvalidation "structure-health/internal/validation"

	"github.com/gorilla/mux"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet/fleet_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted FleetService; the
// global parity test walks it against the Endpoints slice.
var ProtoFile = fleetv1.File_structure_health_v1_fleet_fleet_proto

// Module mounts the FleetService, backed by a fleet scanner that grades each
// scenario through structure-health's own validation engine.
func Module(logger *log.Logger, repoRoot string) module.Module {
	engine := internalvalidation.New()
	engine.Facts = profile.CodeFactsClient{Locator: profile.DefaultLocator{RepoRoot: repoRoot}}
	scanner := internalfleet.New(engine, internalfleet.FilesystemLister{RepoRoot: repoRoot})
	handler := NewHandler(scanner, logger)
	path, connectHandler := fleetconnect.NewFleetServiceHandler(handler)
	return module.Module{
		Name: "fleet",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: fleet owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "fleet_scan_fleet",
		Path:        fleetconnect.FleetServiceScanFleetProcedure,
		Method:      "POST",
		Summary:     "Roll up structure conformance across the fleet",
		Description: "Statically grades every discovered scenario (or the requested subset) through structure-health's engine and returns per-scenario rollups, profile/surface distributions, per-rule conformance, and auto-fixable coverage.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"entries": "array<FleetScenarioEntry>", "rule_conformance": "array<RuleConformance>", "profile_distribution": "array<ProfileDistribution>", "scenario_count": "int32", "passing_count": "int32", "autofixable_total": "int32", "errors": "array<FleetScanError>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Scenario enumeration or grading failure"}},
	},
}
