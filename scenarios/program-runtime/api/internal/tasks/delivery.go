package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
)

// Deliver invokes only the pinned Memory finish program with FinishInputs.
// It must return an acknowledged envelope; kernel success alone is not delivery.
type Deliver func(context.Context, Record) (map[string]any, error)
type Drainer struct {
	ResolveFinishDigest func(context.Context) (string, error)
	lastResolve         time.Time
	Store               *Store
	Deliver             Deliver
	mu                  sync.Mutex
}

// Run is the server-owned outbox loop. Cancellation stops observation and
// delivery; pending intents remain durable for the next process.
func (d *Drainer) Run(ctx context.Context, report func(error)) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := d.DrainOnce(ctx, time.Now()); err != nil && ctx.Err() == nil && report != nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *Drainer) DrainOnce(ctx context.Context, now time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ResolveFinishDigest != nil && now.Sub(d.lastResolve) >= time.Minute {
		d.lastResolve = now
		if digest, err := d.ResolveFinishDigest(ctx); err == nil && digest != "" {
			if err = d.Store.RequeueUnpinned(ctx, digest); err != nil {
				return err
			}
		}
	}
	if err := d.Store.RouteFindings(ctx); err != nil {
		return err
	}
	pending, err := d.Store.Pending(ctx, now)
	if err != nil {
		return err
	}
	for _, r := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, deliveryErr := d.Deliver(ctx, r)
		permanent := permanentDeliveryFailure(result, deliveryErr)
		if deliveryErr == nil {
			signals, _ := result["signals"].(map[string]any)
			if result["status"] != "ok" || signals["capture_status"] != "complete" || signals["entry_id"] == nil || signals["entry_id"] == "" {
				deliveryErr = fmt.Errorf("finish program did not acknowledge capture")
			}
		}
		_, err = d.Store.Update(ctx, r.AttemptID, func(current *Record) error {
			current.DeliveryAttempts++
			if deliveryErr == nil {
				current.Delivery = "delivered"
				current.DeliveryResult = result
				current.LastError = ""
				return nil
			}
			current.LastError = fmt.Sprint(deliveryErr)
			current.DeliveryResult = result // Preserve partial acknowledgements.
			if len(current.LastError) > 512 {
				current.LastError = current.LastError[:512]
			}
			delay := time.Second * time.Duration(1<<min(current.DeliveryAttempts, 12))
			current.NextAttemptAt = now.Add(delay).UTC().Format(time.RFC3339Nano)
			// Keep exact intent indefinitely. Operator-visible blocked state bounds
			// automatic retries; no domain action is ever retried by this worker.
			if permanent || current.DeliveryAttempts >= 20 {
				current.Delivery = "blocked"
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return d.Store.DrainFeedback(ctx, now, d.Deliver)
}

// Invalid immutable inputs and denied authority require a new decision, not
// twenty identical retries. Transient transport failures retain backoff.
func permanentDeliveryFailure(result map[string]any, err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument, connect.CodeAlreadyExists, connect.CodePermissionDenied, connect.CodeUnauthenticated, connect.CodeFailedPrecondition:
		return true
	}
	errors, _ := result["errors"].([]any)
	for _, raw := range errors {
		item, _ := raw.(map[string]any)
		for _, key := range []string{"class", "cause"} {
			switch item[key] {
			case "invalid_input", "immutable_conflict", "no_grant", "not_run_eligible":
				return true
			}
		}
	}
	return false
}
