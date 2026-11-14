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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"audio-tools/internal/httpc"
)

// UsageReport is one operation's accounting payload.
type UsageReport struct {
	LPBSToken    string `json:"-"` // verified LPBS access token; never serialized into the body
	UserIdentity string
	LimitKey     string // "ai_credits"
	Amount       int    // credits charged
	AppBundleKey string // "audio-tools"
	OperationID  string // UUID, idempotency key
	Metadata     map[string]string
}

// RemoteReporter queues reports and ships them to LPBS asynchronously.
type RemoteReporter struct {
	baseURL  string
	doer     httpc.Doer
	queue    chan UsageReport
	done     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

// NewRemoteReporter constructs a reporter; call Start to begin draining the queue.
func NewRemoteReporter(baseURL string) *RemoteReporter {
	return &RemoteReporter{
		baseURL: baseURL,
		doer:    httpc.DefaultDoer(),
		queue:   make(chan UsageReport, 256),
		done:    make(chan struct{}),
	}
}

// Start drains the queue until Stop.
func (r *RemoteReporter) Start(ctx context.Context) {
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
				_ = r.deliver(ctx, rep)
			}
		}
	}()
}

// Submit enqueues a report. Non-blocking; drops on full queue (rare in
// practice given the 256-slot buffer).
func (r *RemoteReporter) Submit(rep UsageReport) {
	select {
	case r.queue <- rep:
	default:
		// Drop on backpressure. Future expansion adds a metric for drop count.
	}
}

func (r *RemoteReporter) Stop() {
	r.stopOnce.Do(func() { close(r.done) })
	r.wg.Wait()
}

// deliver implements the 3-attempt 500ms/1s/2s exponential backoff.
func (r *RemoteReporter) deliver(ctx context.Context, rep UsageReport) error {
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

// attemptOnce is the unit of HTTP work. LPBS derives the billable identity
// from the verified bearer token; the caller-supplied identity is metadata
// only and is deliberately not trusted by the LPBS handler.
func (r *RemoteReporter) attemptOnce(ctx context.Context, rep UsageReport) error {
	if strings.TrimSpace(rep.LPBSToken) == "" {
		return fmt.Errorf("lpbs usage report: access token is required")
	}
	if strings.TrimSpace(rep.OperationID) == "" {
		return fmt.Errorf("lpbs usage report: operation id is required")
	}
	if rep.Amount <= 0 {
		return fmt.Errorf("lpbs usage report: amount must be positive")
	}
	endpoint := strings.TrimRight(r.baseURL, "/") + "/api/v1/usage/report"
	body, err := json.Marshal([]UsageReport{rep})
	if err != nil {
		return fmt.Errorf("marshal usage report: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build usage report request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(rep.LPBSToken))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", rep.OperationID)
	resp, err := r.doer.Do(request)
	if err != nil {
		return fmt.Errorf("send usage report: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("usage report returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}
