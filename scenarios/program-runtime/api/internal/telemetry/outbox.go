package telemetry

import (
	"context"
	"log"
	"time"
)

const (
	DeadLetterWindow     = 7 * 24 * time.Hour
	defaultDrainInterval = time.Second
	maxBackoff           = time.Hour
)

// Drainer turns pending local facts into acknowledged platform events. It
// owns no event history itself; all state transitions are persisted through
// Repository so a process restart cannot lose the delivery ledger.
type Drainer struct {
	repo      Repository
	publisher Publisher
	clock     func() time.Time
	interval  time.Duration
	stop      chan struct{}
	done      chan struct{}
}

func NewDrainer(repo Repository, publisher Publisher, clock func() time.Time) *Drainer {
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &Drainer{repo: repo, publisher: publisher, clock: clock, interval: defaultDrainInterval, stop: make(chan struct{}), done: make(chan struct{})}
}

func (d *Drainer) DrainOnce(ctx context.Context) error {
	if d.repo == nil || d.publisher == nil {
		return nil
	}
	now := d.clock().UTC()
	rows, err := d.repo.Pending(ctx, now, 100)
	if err != nil {
		return err
	}
	for _, row := range rows {
		occurred, parseErr := time.Parse(time.RFC3339Nano, row.event.GetOccurredAt())
		if parseErr == nil && now.Sub(occurred) >= DeadLetterWindow {
			if err := d.repo.MarkDead(ctx, row.eventID, row.attempts, "dead-letter window exceeded"); err != nil {
				return err
			}
			continue
		}
		if err := d.publisher.Publish(ctx, row.event); err == nil {
			if err := d.repo.MarkDelivered(ctx, row.eventID); err != nil {
				return err
			}
			continue
		} else {
			attempts := row.attempts + 1
			backoff := time.Duration(1<<uint(min(attempts-1, 16))) * time.Second
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			if err := d.repo.MarkFailed(ctx, row.eventID, attempts, err.Error(), now.Add(backoff)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *Drainer) Start(ctx context.Context) {
	go func() {
		defer close(d.done)
		ticker := time.NewTicker(d.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := d.DrainOnce(ctx); err != nil {
					log.Printf("telemetry outbox drain failed: %v", err)
				}
			case <-d.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (d *Drainer) Stop() { close(d.stop); <-d.done }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
