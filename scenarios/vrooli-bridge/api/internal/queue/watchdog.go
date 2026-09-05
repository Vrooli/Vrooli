package queue

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// WatchdogConfig contains bounded delivery and execution safety limits. The
// defaults are intentionally short enough to recover a lost desktop channel,
// while remaining configurable for slower links and long-running scenarios.
type WatchdogConfig struct {
	DeliveryLease      time.Duration
	Interval           time.Duration
	MaxAttempts        int
	StartDeadline      time.Duration
	DeadlineGrace      time.Duration
	PresenceStaleAfter time.Duration
}

func DefaultWatchdogConfig() WatchdogConfig {
	return WatchdogConfig{DeliveryLease: 10 * time.Second, Interval: 2 * time.Second, MaxAttempts: 3, StartDeadline: 30 * time.Second, DeadlineGrace: 5 * time.Second, PresenceStaleAfter: 45 * time.Second}
}

// WatchdogConfigFromEnv reads the scenario's documented bounds. Invalid or
// unsafe values fall back to defaults rather than disabling a safety control.
func WatchdogConfigFromEnv() WatchdogConfig {
	c := DefaultWatchdogConfig()
	c.DeliveryLease = envDuration("BRIDGE_DELIVERY_LEASE_SECONDS", c.DeliveryLease)
	c.Interval = envDuration("BRIDGE_WATCHDOG_INTERVAL_SECONDS", c.Interval)
	c.StartDeadline = envDuration("BRIDGE_START_DEADLINE_SECONDS", c.StartDeadline)
	c.DeadlineGrace = envDuration("BRIDGE_DEADLINE_GRACE_SECONDS", c.DeadlineGrace)
	c.PresenceStaleAfter = envDuration("BRIDGE_PRESENCE_STALE_SECONDS", c.PresenceStaleAfter)
	if n, err := strconv.Atoi(os.Getenv("BRIDGE_MAX_DELIVERY_ATTEMPTS")); err == nil && n > 0 {
		c.MaxAttempts = n
	}
	return c
}

func envDuration(name string, fallback time.Duration) time.Duration {
	n, err := strconv.ParseInt(os.Getenv(name), 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// Watchdog is the one server-owned reconciliation loop for delivery leases
// and start/deadline safety. Sweep is deterministic and is exposed for tests;
// Start only supplies the production ticker.
type Watchdog struct {
	store     DurableStore
	scheduler *Scheduler
	aborter   Aborter
	clock     schedule.Clock
	config    WatchdogConfig
	onOutcome func(Reconciliation)
}

func NewWatchdog(store DurableStore, scheduler *Scheduler, aborter Aborter, clk schedule.Clock, config WatchdogConfig, onOutcome func(Reconciliation)) *Watchdog {
	if config.DeliveryLease <= 0 || config.Interval <= 0 || config.MaxAttempts <= 0 || config.StartDeadline <= 0 {
		defaults := DefaultWatchdogConfig()
		if config.DeliveryLease <= 0 {
			config.DeliveryLease = defaults.DeliveryLease
		}
		if config.Interval <= 0 {
			config.Interval = defaults.Interval
		}
		if config.MaxAttempts <= 0 {
			config.MaxAttempts = defaults.MaxAttempts
		}
		if config.StartDeadline <= 0 {
			config.StartDeadline = defaults.StartDeadline
		}
		if config.DeadlineGrace <= 0 {
			config.DeadlineGrace = defaults.DeadlineGrace
		}
		if config.PresenceStaleAfter <= 0 {
			config.PresenceStaleAfter = defaults.PresenceStaleAfter
		}
	}
	return &Watchdog{store: store, scheduler: scheduler, aborter: aborter, clock: clk, config: config, onOutcome: onOutcome}
}

func (w *Watchdog) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = w.Sweep(ctx)
			}
		}
	}()
}

func (w *Watchdog) Sweep(ctx context.Context) error {
	entries, err := w.store.Load(ctx)
	if err != nil {
		return err
	}
	now := w.clock.Now().UTC()
	for _, entry := range entries {
		if entry.State == StateQueued {
			if entry.Job.TimeoutSeconds > 0 && !entry.EnqueuedAt.IsZero() && !entry.EnqueuedAt.Add(time.Duration(entry.Job.TimeoutSeconds)*time.Second+w.config.DeadlineGrace).After(now) {
				if err := w.fail(ctx, entry, "deadline_exceeded", now); err != nil {
					return err
				}
			}
			continue
		}
		if !entry.Acked && !entry.LeaseExpiresAt.IsZero() && !entry.LeaseExpiresAt.After(now) {
			if entry.DeliveryAttempts < w.config.MaxAttempts {
				if err := w.scheduler.Requeue(ctx, entry.Job); err != nil {
					return err
				}
				w.emit(Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "delivery_lease_expired_redelivery"})
				continue
			}
			if err := w.fail(ctx, entry, "node_channel_lost", now); err != nil {
				return err
			}
			continue
		}
		if entry.Acked && entry.StartedAt.IsZero() && !entry.AckedAt.IsZero() && !entry.AckedAt.Add(w.config.StartDeadline).After(now) {
			if err := w.fail(ctx, entry, "no_start_after_ack", now); err != nil {
				return err
			}
			continue
		}
		if !entry.StartedAt.IsZero() && entry.Job.TimeoutSeconds > 0 && !entry.StartedAt.Add(time.Duration(entry.Job.TimeoutSeconds)*time.Second+w.config.DeadlineGrace).After(now) {
			if err := w.aborter.Abort(ctx, entry.Job.RunID, "deadline_exceeded"); err != nil {
				return err
			}
			w.scheduler.Remove(entry.Job.NodeID, entry.Job.RunID)
			w.emit(Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: "deadline_exceeded", Terminal: true})
		}
	}
	return nil
}

func (w *Watchdog) fail(ctx context.Context, entry DurableEntry, reason string, at time.Time) error {
	w.scheduler.Remove(entry.Job.NodeID, entry.Job.RunID)
	if err := w.store.MarkFailedDelivery(ctx, entry.Job.RunID, reason, at); err != nil {
		return err
	}
	w.emit(Reconciliation{RunID: entry.Job.RunID, NodeID: entry.Job.NodeID, Reason: reason, Terminal: true})
	return nil
}

func (w *Watchdog) emit(outcome Reconciliation) {
	if w.onOutcome != nil {
		w.onOutcome(outcome)
	}
}
