package tunnel

import (
	"log"
	"net/http"
	"os"
	"time"

	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	tunnelconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/tunnel/tunnel_v1connect"

	internaltunnel "tunnel-manager/internal/tunnel"
)

// Module returns the tunnel domain's contribution to the API: the generated
// Connect-RPC TunnelService handler. The center (server.New) does not change —
// adding a domain is one Module() call in main.go plus one registration row in
// modules.AllEndpoints/AllSchemas/AllProtoFiles.
//
// The metrics endpoint is read from TUNNEL_METRICS_URL (falling back to the
// service default) so deployments can point at a non-default cloudflared
// Prometheus listener without a rebuild.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	repo := internaltunnel.NewSQLiteRepository(db, clk)
	doer := &http.Client{Timeout: 5 * time.Second}
	endpoint := os.Getenv("TUNNEL_METRICS_URL")
	svc := internaltunnel.NewService(repo, cmdrunner.Default, doer, clk, endpoint)
	connectPath, connectHandler := tunnelconnect.NewTunnelServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "tunnel",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internaltunnel.Schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaltunnel.Schema() }

// Endpoints is the machine-readable description of the tunnel module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants from tunnelconnect, so adding or renaming an RPC in tunnel.proto
// breaks this file at compile time. TestProtoConnectParity
// (api/internal/modules/registry_test.go) asserts every rpc has exactly one
// entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "tunnel_get_status",
		Path:        tunnelconnect.TunnelServiceGetStatusProcedure,
		Method:      "POST",
		Summary:     "Get composite tunnel health",
		Description: "Returns the current composite tunnel health (cloudflared systemd unit state + /ready probe) plus the most recent scraped metrics sample, through the generated Connect-RPC Tunnel service.",
		Category:    "tunnel",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"status":         "TunnelStatus",
				"latest_metrics": "MetricsSample (null when none scraped)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get status", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.tunnel.TunnelService/GetStatus -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "tunnel_list_metrics",
		Path:        tunnelconnect.TunnelServiceListMetricsProcedure,
		Method:      "POST",
		Summary:     "List scraped metrics samples",
		Description: "Returns persisted Prometheus metrics samples scraped within the optional [from, to] window, ordered by scraped_at ascending, through the generated Connect-RPC Tunnel service.",
		Category:    "tunnel",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"from": "google.protobuf.Timestamp (optional window start)",
				"to":   "google.protobuf.Timestamp (optional window end)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"samples": "array<MetricsSample>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List metrics", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.tunnel.TunnelService/ListMetrics -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "tunnel_scrape",
		Path:        tunnelconnect.TunnelServiceScrapeProcedure,
		Method:      "POST",
		Summary:     "Force a metrics scrape",
		Description: "Fetches the cloudflared /metrics endpoint once, parses the key gauges/counters, persists the sample, and returns it, through the generated Connect-RPC Tunnel service.",
		Category:    "tunnel",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"sample": "MetricsSample"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Scrape fetch/parse or repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Scrape now", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.tunnel.TunnelService/Scrape -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
