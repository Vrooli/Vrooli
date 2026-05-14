package metrics

import (
	"context"

	intmetrics "web-console/internal/metrics"
)

// Adapter is the production Service implementation: it bridges the
// in-process *metrics.Metrics counter struct to the transport-neutral
// Snapshot the Connect handler consumes. Constructed in api/main.go
// (and by metrics_test.go) and passed to Module.
type Adapter struct {
	Metrics *intmetrics.Metrics
}

func (a *Adapter) Snapshot(_ context.Context) Snapshot {
	r := a.Metrics.Snapshot()
	return Snapshot{
		Sessions: SessionMetrics{
			Created: r.Sessions.Created,
			Deleted: r.Sessions.Deleted,
			Active:  r.Sessions.Active,
			Resizes: r.Sessions.Resizes,
		},
		Connections: ConnectionMetrics{
			Total:  r.Connections.Total,
			Active: r.Connections.Active,
		},
		Messages: MessageMetrics{
			Sent:     r.Messages.Sent,
			Received: r.Messages.Received,
		},
		Reattach: ReattachMetrics{
			Attempts:  r.Reattach.Attempts,
			Successes: r.Reattach.Successes,
			Failures:  r.Reattach.Failures,
		},
		Recovery: RecoveryMetrics{
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
