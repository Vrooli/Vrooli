package probes

import (
	"log"
	"net/http"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	probesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/probes/probes_v1connect"

	internalprobes "tunnel-manager/internal/probes"
	internalroutes "tunnel-manager/internal/routes"
)

// probeHTTPTimeout bounds each outbound liveness GET so a hung upstream
// cannot stall the probe cycle. Redirects are followed by the default
// client; a route that 30x's still resolves to its final status code.
const probeHTTPTimeout = 5 * time.Second

// Module returns the probes domain's contribution to the API: the
// generated Connect-RPC ProbesService handler. The center (server.New)
// does not change — adding a domain is one Module() call in main.go plus
// one registration row in modules.AllEndpoints/AllSchemas/AllProtoFiles.
//
// The service reads the routes manifest through a routes.Service built
// from the same RoutedDB, and probes upstreams through a timeout-bounded
// *http.Client (the httpc.Doer seam) so a slow scenario cannot wedge the
// cycle.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	return ModuleWithService(NewProductionService(db, clk), logger)
}

// NewProductionService wires the probes service with the same production
// seams used by the Connect handler and the background scheduler.
func NewProductionService(db *database.RoutedDB, clk clock.Clock) internalprobes.Service {
	routesReader := internalroutes.NewService(internalroutes.NewSQLiteRepository(db, clk))
	repo := internalprobes.NewSQLiteRepository(db, clk)
	httpClient := &http.Client{Timeout: probeHTTPTimeout}
	return internalprobes.NewService(routesReader, repo, httpClient, clk)
}

func ModuleWithService(svc internalprobes.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := probesconnect.NewProbesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "probes",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func probeResultsProperty() map[string]string {
	return map[string]string{"results": "array<ProbeResult>"}
}

func internalProbeErrors(description string) []module.ErrorDesc {
	return []module.ErrorDesc{{Status: 500, Code: "internal", Description: description}}
}

// Schema re-exports internalprobes.Schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalprobes.Schema() }

// Endpoints is the machine-readable description of the probes module's
// public surface. Connect-RPC method paths reference the generated
// *Procedure constants from probesconnect, so adding or renaming an RPC in
// probes.proto breaks this file at compile time. TestProtoConnectParity
// (api/internal/modules/registry_test.go) asserts every rpc has exactly
// one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "probes_run",
		Path:        probesconnect.ProbesServiceRunProbesProcedure,
		Method:      "POST",
		Summary:     "Run a probe cycle",
		Description: "Probes every enabled route's local port (internal) and public URL (external), persists the results, and returns them, through the generated Connect-RPC Probes service.",
		Category:    "probes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: probeResultsProperty(),
		},
		Errors: internalProbeErrors("Route manifest read failure"),
		Examples: []module.Example{
			{Name: "Run probes", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.probes.ProbesService/RunProbes -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "probes_list",
		Path:        probesconnect.ProbesServiceListProbesProcedure,
		Method:      "POST",
		Summary:     "List probe history",
		Description: "Returns recent probe history newest-first, optionally filtered by subdomain, through the generated Connect-RPC Probes service.",
		Category:    "probes",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"subdomain": "string (optional filter)",
				"limit":     "int32 (0 = default)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: probeResultsProperty(),
		},
		Errors: internalProbeErrors("Repository read failure"),
		Examples: []module.Example{
			{Name: "List probes", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.probes.ProbesService/ListProbes -H 'Content-Type: application/json' -d '{\"subdomain\":\"agent-manager\",\"limit\":20}'"},
		},
	},
	{
		ID:          "probes_classify",
		Path:        probesconnect.ProbesServiceClassifyProcedure,
		Method:      "POST",
		Summary:     "Classify route reachability",
		Description: "Derives the per-route reachability diagnosis (healthy / tunnel-down / scenario-down / config-drift) from the latest stored internal+external probes.",
		Category:    "probes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"classifications": "array<RouteClassification>"},
		},
		Errors: internalProbeErrors("Repository read failure"),
		Examples: []module.Example{
			{Name: "Classify routes", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.probes.ProbesService/Classify -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
