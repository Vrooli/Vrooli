// Package metrics is the search-hub metrics domain's API surface: the generated
// MetricsService Connect-RPC handler that aggregates per-query telemetry into
// federation-health insights (Phase 7). It sits beside the registry and routing
// domains and reads both the telemetry store and the provider registry (to mark
// registered-but-never-routed leaves as under-utilized).
package metrics

import (
	"log"

	"search-hub/internal/clock"
	"search-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"

	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// Module returns the metrics domain's contribution to the API: the generated
// MetricsService Connect handler backed by the SQLite telemetry store and the
// provider registry store.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	connectPath, connectHandler := metricsconnect.NewMetricsServiceHandler(NewConnectHandler(Deps{
		Insights: internalmetrics.NewSQLiteStore(db, clk),
		Lister:   internalregistry.NewSQLiteStore(db, clk),
		Logger:   logger,
	}))
	return module.Module{
		Name: "metrics",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalmetrics.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalmetrics.Schema() }

// Recorder constructs the telemetry write bridge the routing domain uses. It
// adapts the metrics SQLite store to routing.TelemetryRecorder at the wiring
// edge, so internal/metrics never imports internal/routing. The bridge lives in
// recorder.go.
func Recorder(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) *TelemetryBridge {
	return NewTelemetryBridge(internalmetrics.NewSQLiteStore(db, clk), logger)
}
