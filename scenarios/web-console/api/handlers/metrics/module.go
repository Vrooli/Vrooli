// Package metrics is the HTTP-handler home for the metrics domain. It
// exposes the generated Connect-RPC MetricsService (proto schema:
// packages/proto/schemas/web-console/v1/metrics).
//
// RPCs (mounted at /vrooli.web_console.v1.metrics.MetricsService/...):
//
//	Get — point-in-time snapshot of all operational counters and
//	      uptime.
package metrics

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"

	"web-console/internal/module"
)

// Service is the seam the Connect handler depends on. Implemented in
// package main by metricsAdapter, which bridges to the *Metrics counter
// struct.
type Service interface {
	Snapshot(ctx context.Context) Snapshot
}

// Snapshot mirrors the proto GetResponse message field-for-field.
type Snapshot struct {
	Sessions    SessionMetrics
	Connections ConnectionMetrics
	Messages    MessageMetrics
	Reattach    ReattachMetrics
	Recovery    RecoveryMetrics

	AIGenerations              int64
	AISuggestions              int64
	VoiceSkipVerificationTotal int64
	Uptime                     string
}

type SessionMetrics struct {
	Created int64
	Deleted int64
	Active  int64
	Resizes int64
}

type ConnectionMetrics struct {
	Total  int64
	Active int64
}

type MessageMetrics struct {
	Sent     int64
	Received int64
}

type ReattachMetrics struct {
	Attempts  int64
	Successes int64
	Failures  int64
}

type RecoveryMetrics struct {
	Recovered       int64
	OrphanedMeta    int64
	OrphanedTmux    int64
	AttachRetries   int64
	PreservedForNow int64
}

// Module wires the metrics domain into the API server.
func Module(svc Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	connectPath, connectHandler := metricsconnect.NewMetricsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "metrics",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}
