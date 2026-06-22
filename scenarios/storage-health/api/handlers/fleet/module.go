package fleet

import (
	"log"

	"storage-health/internal/clock"
	"storage-health/internal/fleet"
	"storage-health/internal/module"
	"storage-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/storage-health/v1/fleet/fleet_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted FleetService.
var ProtoFile = fleetv1.File_storage_health_v1_fleet_fleet_proto

// Module mounts the FleetService backed by the real classifier (the storage
// validation engine projected onto inventory fields), the real enumerator (the
// typed `vrooli scenario list` contract), and the SQLite snapshot store.
func Module(logger *log.Logger, repoRoot string, db *database.RoutedDB, clk clock.Clock) module.Module {
	// Fleet scans classify every discovered scenario, so detection must be fast
	// and dependency-free: the filesystem detector (language from the api/ build
	// manifest, domains from api/internal/<domain> layout) is used instead of the
	// per-call code-facts parse the single-scenario producer can afford.
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Detector: validation.FilesystemDetector{}, Logger: logger})
	svc := fleet.NewService(newClassifier(validator), newCLIEnumerator(), fleet.NewSQLStore(db), clk)
	handler := NewHandler(svc, logger)
	path, connectHandler := fleetconnect.NewFleetServiceHandler(handler)
	return module.Module{
		Name: "fleet",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "fleet_scan_fleet",
		Path:        fleetconnect.FleetServiceScanFleetProcedure,
		Method:      "POST",
		Summary:     "Scan the fleet's storage inventory",
		Description: "Classifies the requested scenarios (or every discovered scenario) by engine, isolation-readiness, namespace adoption, deploy stage, and backup readiness, persists the snapshot, and rolls up fleet-wide distributions + offender counts.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<string>"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"entries": "array<FleetScenarioEntry>", "engine_distribution": "array<EngineCount>", "stage_distribution": "array<StageCount>",
			"scenario_count": "int32", "isolation_unready_count": "int32", "no_backup_count": "int32", "finding_count": "int32",
			"errors": "array<FleetScanError>", "scanned_at": "string",
		}},
		Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Fleet scan failure"}},
		Examples: []module.Example{
			{Name: "Scan whole fleet", Curl: "curl http://localhost:${API_PORT}/vrooli.storage_health.v1.fleet.FleetService/ScanFleet -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "fleet_get_inventory",
		Path:        fleetconnect.FleetServiceGetInventoryProcedure,
		Method:      "POST",
		Summary:     "Read the latest persisted fleet snapshot",
		Description: "Returns the most recent persisted fleet inventory snapshot without re-scanning. An empty snapshot (scenario_count 0) means ScanFleet has not run yet.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"entries": "array<FleetScenarioEntry>", "scenario_count": "int32", "scanned_at": "string",
		}},
		Errors: []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Snapshot read failure"}},
		Examples: []module.Example{
			{Name: "Read snapshot", Curl: "curl http://localhost:${API_PORT}/vrooli.storage_health.v1.fleet.FleetService/GetInventory -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
