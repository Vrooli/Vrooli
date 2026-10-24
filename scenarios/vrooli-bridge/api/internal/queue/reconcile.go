package queue

import (
	"context"

	"github.com/vrooli/api-core/schedule"
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

// Reconcile restores durable delivery state before traffic is accepted. Only a
// pre-start, unacknowledged delivery is safe to redrive: queued work must pass
// through Scheduler.Promote so the per-node bound and FIFO order remain in
// force, while acknowledged or already-started work may have an external
// effect and must not be blindly replayed.
func Reconcile(ctx context.Context, store DurableStore, presence ReconcilePresence, pusher Pusher, clk schedule.Clock, cancellationSenders ...CancellationSender) ([]Reconciliation, error) {
	entries, err := store.Load(ctx)
	if err != nil {
		return nil, err
	}
	var outcomes []Reconciliation
	for _, entry := range entries {
		now := clk.Now().UTC()
		if !entry.CancelRequestedAt.IsZero() && !entry.CancellationConfirmed {
			if len(cancellationSenders) == 0 || !presence.Dispatchable(entry.Job.NodeID) {
				outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "cancellation_preserved"})
				continue
			}
			reason := entry.CancelReason
			if reason == "" {
				reason = "reconcile cancellation"
			}
			delivered, sendErr := cancellationSenders[0].CancelJob(ctx, entry.Job.NodeID, entry.Job.RunID, reason)
			if sendErr != nil || delivered == 0 {
				if err := store.MarkUncertain(ctx, entry.Job.RunID, "cancellation delivery unconfirmed", now); err != nil {
					return nil, err
				}
				outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "cancellation_unconfirmed"})
				continue
			}
			outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "cancellation_redriven"})
			continue
		}
		if entry.State == StateQueued {
			if !presence.Dispatchable(entry.Job.NodeID) {
				const reason = "node_channel_lost"
				if err := store.MarkFailedDelivery(ctx, entry.Job.RunID, reason, now); err != nil {
					return nil, err
				}
				outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: reason, Terminal: true})
				continue
			}
			outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "queued_restored"})
			continue
		}
		if entry.Acked || !entry.StartedAt.IsZero() {
			reason := "execution_state_preserved"
			if entry.Acked {
				reason = "acknowledged_delivery_preserved"
			}
			outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: reason})
			continue
		}
		if presence.Dispatchable(entry.Job.NodeID) {
			delivered, pushErr := pusher.Push(ctx, entry.Job)
			if pushErr == nil && delivered > 0 {
				if err := store.MarkPushed(ctx, entry.Job.RunID, now, now.Add(DefaultDeliveryLease)); err != nil {
					return nil, err
				}
				outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "delivery_redriven"})
				continue
			}
		}
		reason := "node_channel_lost"
		if err := store.MarkFailedDelivery(ctx, entry.Job.RunID, reason, now); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: reason, Terminal: true})
	}
	return outcomes, nil
}
