package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// Deliver invokes only the pinned Memory finish program with FinishInputs.
// It must return an acknowledged envelope; kernel success alone is not delivery.
type Deliver func(context.Context, Record) (map[string]any, error)

// Runtime has one task owner per process. Serialize its drainer instances,
// even if they were constructed with separate Store wrappers for the same DB.
// The lifecycle remains responsible for preventing concurrent owner processes.
var drainGate = make(chan struct{}, 1)

type Drainer struct {
	ResolveFinishDigest func(context.Context) (string, error)
	Store               *Store
	Deliver             Deliver
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
	select {
	case drainGate <- struct{}{}:
		defer func() { <-drainGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var preliminary error
	requeueState, err := d.Store.loadRequeueScan(ctx)
	if err != nil {
		preliminary = errors.Join(preliminary, err)
	} else if (requeueState.digest != "" || d.ResolveFinishDigest != nil) && (requeueState.lastAttempt.IsZero() || now.Sub(requeueState.lastAttempt) >= time.Minute) {
		requeueState.lastAttempt = now
		if err = d.Store.saveRequeueScan(ctx, requeueState); err != nil {
			preliminary = errors.Join(preliminary, err)
		} else {
			digest := requeueState.digest
			if digest == "" && d.ResolveFinishDigest != nil {
				digest, err = d.ResolveFinishDigest(ctx)
			}
			if err != nil {
				preliminary = errors.Join(preliminary, err)
			} else if digest != "" {
				if err = d.Store.requeueDurable(ctx, digest); err != nil {
					preliminary = errors.Join(preliminary, err)
				}
			}
		}
	}
	if err := d.Store.RouteFindings(ctx); err != nil {
		preliminary = errors.Join(preliminary, err)
	}
	malformed, err := d.Store.malformedPendingCount(ctx)
	if err != nil {
		preliminary = errors.Join(preliminary, err)
	}
	malformedFeedback, err := d.Store.malformedFeedbackCount(ctx)
	if err != nil {
		preliminary = errors.Join(preliminary, err)
	}
	state, err := d.Store.loadOutboxScan(ctx)
	if err != nil {
		return errors.Join(preliminary, err)
	}
	if state.watermark == nil {
		state.cycleUnavailable = false
		state.cutoff = now.UTC().Format(dueTimeFormat)
		var err error
		state.watermark, err = d.Store.outboxWatermark(ctx, state.cutoff)
		if err != nil {
			return err
		}
		if state.watermark == nil {
			state.cutoff = ""
		}
		if err = d.Store.saveOutboxScan(ctx, state); err != nil {
			return errors.Join(preliminary, err)
		}
	}
	pending, err := d.Store.outboxPage(ctx, state.cutoff, state.cursor, state.watermark)
	if err != nil {
		return err
	}
	for _, item := range pending {
		r := item.record
		if err := ctx.Err(); err != nil {
			return err
		}
		if r.receiptUnavailable {
			// The exact immutable intent is no longer available to this process.
			// Preserve its stored ciphertext and never deliver a substituted token.
			state.unavailableObserved = true
			state.cycleUnavailable = true
			cursor := item.cursor
			state.cursor = &cursor
			if err := d.Store.saveOutboxScan(ctx, state); err != nil {
				return errors.Join(preliminary, err)
			}
			continue
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
		cursor := item.cursor
		state.cursor = &cursor
		if err := d.Store.saveOutboxScan(ctx, state); err != nil {
			return errors.Join(preliminary, err)
		}
	}
	if len(pending) < 20 {
		state.unavailableObserved = state.cycleUnavailable
		state.cycleUnavailable = false
		state.cursor, state.watermark, state.cutoff = nil, nil, ""
	}
	var unresolved error
	if (state.unavailableObserved || malformed > 0 || malformedFeedback > 0) && (state.lastWarning.IsZero() || now.Sub(state.lastWarning) >= time.Minute) {
		state.lastWarning = now
		var problems []string
		if state.unavailableObserved {
			problems = append(problems, "one or more tasks contain a receipt unavailable after restart")
		}
		if malformed > 0 || malformedFeedback > 0 {
			problems = append(problems, fmt.Sprintf("%d pending task(s) and %d feedback item(s) have invalid or non-UTC due times", malformed, malformedFeedback))
		}
		unresolved = fmt.Errorf("%s; owner reconciliation required", strings.Join(problems, "; "))
	}
	if err := d.Store.saveOutboxScan(ctx, state); err != nil {
		return errors.Join(preliminary, unresolved, err)
	}
	return errors.Join(preliminary, unresolved, d.Store.DrainFeedback(ctx, now, d.Deliver))
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
