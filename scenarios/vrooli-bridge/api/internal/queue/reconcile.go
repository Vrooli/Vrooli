package queue

import (
	"context"
	"time"

	"vrooli-bridge/internal/clock"
)

// ReconcilePresence is the minimal boot-time eligibility projection.
type ReconcilePresence interface {
	Dispatchable(nodeID string) bool
}

type Reconciliation struct {
	RunID    string
	NodeID   string
	Reason   string
	Terminal bool
}

// Reconcile restores every durable non-terminal delivery before traffic is
// accepted. Eligible nodes are re-driven; runs whose node is not currently
// dispatchable are terminalized with an accountable typed reason rather than
// left stranded in memory.
func Reconcile(ctx context.Context, store DurableStore, presence ReconcilePresence, pusher Pusher, clk clock.Clock) ([]Reconciliation, error) {
	entries, err := store.Load(ctx)
	if err != nil {
		return nil, err
	}
	var outcomes []Reconciliation
	for _, entry := range entries {
		if presence.Dispatchable(entry.Job.NodeID) {
			delivered, pushErr := pusher.Push(ctx, entry.Job)
			if pushErr == nil && delivered > 0 {
				if err := store.MarkPushed(ctx, entry.Job.RunID, clk.Now().UTC(), time.Time{}); err != nil {
					return nil, err
				}
				continue
			}
		}
		reason := "node_channel_lost"
		if err := store.MarkFailedDelivery(ctx, entry.Job.RunID, reason, clk.Now().UTC()); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: reason, Terminal: true})
	}
	return outcomes, nil
}
