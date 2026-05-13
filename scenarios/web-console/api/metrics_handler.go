package main

import (
	"context"

	metricsH "web-console/handlers/metrics"
)

// metricsAdapter bridges the in-process *Metrics counter struct to
// handlers/metrics.Service so the Connect handler can be mounted from
// main without crossing a package boundary the wrong way.
type metricsAdapter struct {
	srv *Server
}

func newMetricsAdapter(s *Server) *metricsAdapter {
	return &metricsAdapter{srv: s}
}

func (a *metricsAdapter) Snapshot(_ context.Context) metricsH.Snapshot {
	r := a.srv.metrics.Snapshot()
	return metricsH.Snapshot{
		Sessions: metricsH.SessionMetrics{
			Created: r.Sessions.Created,
			Deleted: r.Sessions.Deleted,
			Active:  r.Sessions.Active,
			Resizes: r.Sessions.Resizes,
		},
		Connections: metricsH.ConnectionMetrics{
			Total:  r.Connections.Total,
			Active: r.Connections.Active,
		},
		Messages: metricsH.MessageMetrics{
			Sent:     r.Messages.Sent,
			Received: r.Messages.Received,
		},
		Reattach: metricsH.ReattachMetrics{
			Attempts:  r.Reattach.Attempts,
			Successes: r.Reattach.Successes,
			Failures:  r.Reattach.Failures,
		},
		Recovery: metricsH.RecoveryMetrics{
			Recovered:       r.Recovery.Recovered,
			OrphanedMeta:    r.Recovery.OrphanedMeta,
			OrphanedTmux:    r.Recovery.OrphanedTmux,
			AttachRetries:   r.Recovery.AttachRetries,
			PreservedForNow: r.Recovery.PreservedForNow,
		},
		AIGenerations:              r.AIGenerations,
		AISuggestions:              r.AISuggestions,
		StdinBeforeReadyTotal:      r.StdinBeforeReadyTotal,
		VoiceSkipVerificationTotal: r.VoiceSkipVerificationTotal,
		Uptime:                     r.Uptime,
	}
}
