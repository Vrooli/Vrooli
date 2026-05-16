// Package lpbs (integrations) reports per-operation usage to LPBS.
//
// Async goroutine, 3-attempt exponential backoff (500ms, 1s, 2s),
// idempotent by OperationID. Reports the {UserIdentity, LimitKey,
// Amount, AppBundleKey, OperationID, Metadata} shape LPBS expects.
//
// Status: implementation lands with execute/lpbs-audio-gateway-endpoints.
// This package exists so the seam is named and main.go can wire it
// without churn when the LPBS side ships.
package lpbs

import (
	"context"
	"sync"
	"time"
)

// UsageReport is one operation's accounting payload.
type UsageReport struct {
	UserIdentity string
	LimitKey     string // "ai_credits"
	Amount       int    // credits charged
	AppBundleKey string // "audio-tools"
	OperationID  string // UUID, idempotency key
	Metadata     map[string]string
}

// Reporter queues reports and ships them to LPBS asynchronously.
type Reporter struct {
	baseURL string
	queue   chan UsageReport
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewReporter constructs a reporter; call Start to begin draining the queue.
func NewReporter(baseURL string) *Reporter {
	return &Reporter{
		baseURL: baseURL,
		queue:   make(chan UsageReport, 256),
		done:    make(chan struct{}),
	}
}

// Start drains the queue until Stop. Currently a no-op while the LPBS
// gateway is not implemented; reports are accepted into the channel and
// dropped on shutdown.
func (r *Reporter) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.done:
				return
			case <-ctx.Done():
				return
			case rep := <-r.queue:
				_ = r.deliver(ctx, rep) // Stubbed pending gateway implementation.
			}
		}
	}()
}

// Submit enqueues a report. Non-blocking; drops on full queue (rare in
// practice given the 256-slot buffer).
func (r *Reporter) Submit(rep UsageReport) {
	select {
	case r.queue <- rep:
	default:
		// Drop on backpressure. Future expansion adds a metric for drop count.
	}
}

func (r *Reporter) Stop() {
	close(r.done)
	r.wg.Wait()
}

// deliver implements the 3-attempt 500ms/1s/2s exponential backoff. The
// actual HTTP call is stubbed pending LPBS gateway implementation; the
// retry shape is in place so wiring it later is one function-body change.
func (r *Reporter) deliver(ctx context.Context, rep UsageReport) error {
	backoffs := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	var lastErr error
	for attempt, delay := range backoffs {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		err := r.attemptOnce(ctx, rep)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

// attemptOnce is the unit of HTTP work. Stubbed today.
func (r *Reporter) attemptOnce(ctx context.Context, rep UsageReport) error {
	return nil // gateway not implemented
}
