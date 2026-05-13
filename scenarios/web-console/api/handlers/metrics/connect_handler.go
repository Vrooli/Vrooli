package metrics

import (
	"context"
	"log"

	"connectrpc.com/connect"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
)

// Deps wires the seams the Connect metrics handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// MetricsServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Get(ctx context.Context, _ *connect.Request[metricsv1.GetRequest]) (*connect.Response[metricsv1.GetResponse], error) {
	snap := h.deps.Service.Snapshot(ctx)
	return connect.NewResponse(&metricsv1.GetResponse{
		Sessions: &metricsv1.SessionMetrics{
			Created: snap.Sessions.Created,
			Deleted: snap.Sessions.Deleted,
			Active:  snap.Sessions.Active,
			Resizes: snap.Sessions.Resizes,
		},
		Connections: &metricsv1.ConnectionMetrics{
			Total:  snap.Connections.Total,
			Active: snap.Connections.Active,
		},
		Messages: &metricsv1.MessageMetrics{
			Sent:     snap.Messages.Sent,
			Received: snap.Messages.Received,
		},
		Reattach: &metricsv1.ReattachMetrics{
			Attempts:  snap.Reattach.Attempts,
			Successes: snap.Reattach.Successes,
			Failures:  snap.Reattach.Failures,
		},
		Recovery: &metricsv1.RecoveryMetrics{
			Recovered:                  snap.Recovery.Recovered,
			OrphanedMetadata:           snap.Recovery.OrphanedMeta,
			OrphanedTmux:               snap.Recovery.OrphanedTmux,
			AttachRetries:              snap.Recovery.AttachRetries,
			PreservedForFutureRecovery: snap.Recovery.PreservedForNow,
		},
		AiGenerations:              snap.AIGenerations,
		AiSuggestions:              snap.AISuggestions,
		StdinBeforeReadyTotal:      snap.StdinBeforeReadyTotal,
		VoiceSkipVerificationTotal: snap.VoiceSkipVerificationTotal,
		Uptime:                     snap.Uptime,
	}), nil
}
