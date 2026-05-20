package inventory

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/module"
	"react-component-library/internal/uimanifest"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory/inventory_v1connect"
)

// ProtoFile exposes the inventory domain's proto FileDescriptor so the
// global parity test can walk it.
var ProtoFile = inventoryv1.File_ui_health_v1_inventory_inventory_proto

// Module wires the inventory domain. AdoptionsReader is the slice of
// the adoptions service used by ScanScenario; tests inject a fake.
func Module(logger *log.Logger, scenariosRoot string, reader AdoptionsReader, loader uimanifest.Loader) module.Module {
	connectPath, connectHandler := inventoryconnect.NewInventoryServiceHandler(NewConnectHandler(Deps{
		Logger:        logger,
		Adoptions:     reader,
		ManifestLoad:  loader,
		ScenariosRoot: scenariosRoot,
	}))
	return module.Module{
		Name: "inventory",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

// AdoptionsServiceAdapter wraps the adoptions service to expose only the
// AdoptionsReader slice the inventory handler needs.
type AdoptionsServiceAdapter struct {
	Service adoptions.Service
}

func (a AdoptionsServiceAdapter) List(ctx context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error) {
	return a.Service.List(ctx, q)
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "inventory_scan_scenario",
		Path:        inventoryconnect.InventoryServiceScanScenarioProcedure,
		Method:      "POST",
		Summary:     "Scan a scenario's UI tree for provenance + widgets + surfaces",
		Description: "Implements ui-health's InventoryService.ScanScenario for React scenarios. SQLite adoptions store is authoritative; JSDoc blocks are the heal-from signal.",
		Category:    "inventory",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"provenance": "array<ComponentProvenance>",
				"widgets":    "array<WidgetDeclaration>",
				"surfaces":   "array<SurfaceRecord>",
			},
		},
		Examples: []module.Example{
			{Name: "Scan scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.ui_health.v1.inventory.InventoryService/ScanScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"ui-health\"}'"},
		},
	},
}
