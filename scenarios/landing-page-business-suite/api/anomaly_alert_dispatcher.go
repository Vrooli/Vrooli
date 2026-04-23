package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// AnomalyDispatchUserAgent is set on every outbound webhook request so
// integrators can filter their access logs.
const AnomalyDispatchUserAgent = "lpbs-anomaly-dispatcher/1"

// anomalyDispatchPayload is what the dispatcher serialises to the webhook body.
type anomalyDispatchPayload struct {
	ID          int64                  `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Email       string                 `json:"email,omitempty"`
	CustomerID  string                 `json:"customer_id,omitempty"`
	SubjectID   string                 `json:"-"`
	SubjectKind string                 `json:"-"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	WebhookURL  string                 `json:"-"`
}

// anomalyDispatchWireFormat is the exact JSON shape sent to integrators.
// Kept separate from the in-memory payload so the wire format stays stable
// even if we refactor the in-memory struct.
type anomalyDispatchWireFormat struct {
	ID         int64                  `json:"id"`
	Type       string                 `json:"type"`
	Severity   string                 `json:"severity"`
	Email      string                 `json:"email,omitempty"`
	CustomerID string                 `json:"customer_id,omitempty"`
	Subject    *anomalySubjectWire    `json:"subject,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	Scenario   string                 `json:"scenario"`
}

type anomalySubjectWire struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// httpDoer is the subset of http.Client the dispatcher uses. Tests swap in a
// stub with UseHTTPClient.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// AnomalyAlertDispatcher dispatches anomaly webhooks with retry, rate
// limiting, and per-row dispatch-state tracking.
type AnomalyAlertDispatcher struct {
	db          *sql.DB
	httpClient  httpDoer
	mu          sync.Mutex
	buckets     map[string]*tokenBucket
	maxAttempts int
	perAttempt  time.Duration
	backoffs    []time.Duration
}

// NewAnomalyAlertDispatcher returns a dispatcher with production defaults:
// 3 attempts, 5s per attempt, 1s/2s/4s backoff between failed attempts.
func NewAnomalyAlertDispatcher(db *sql.DB) *AnomalyAlertDispatcher {
	return &AnomalyAlertDispatcher{
		db: db,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		buckets:     map[string]*tokenBucket{},
		maxAttempts: 3,
		perAttempt:  5 * time.Second,
		backoffs:    []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// UseHTTPClient swaps the HTTP client (tests).
func (d *AnomalyAlertDispatcher) UseHTTPClient(client httpDoer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.httpClient = client
}

// UseBackoff overrides the per-attempt backoff (tests).
func (d *AnomalyAlertDispatcher) UseBackoff(backoffs []time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.backoffs = backoffs
}

// Dispatch POSTs the payload to the configured webhook with retry. On final
// failure it updates the row's dispatch_status/attempts/error; on success
// it marks the row as sent with a dispatched_at timestamp.
func (d *AnomalyAlertDispatcher) Dispatch(ctx context.Context, p anomalyDispatchPayload) {
	if p.WebhookURL == "" {
		return
	}
	body, err := json.Marshal(anomalyDispatchWireFormat{
		ID:         p.ID,
		Type:       p.Type,
		Severity:   p.Severity,
		Email:      p.Email,
		CustomerID: p.CustomerID,
		Subject:    subjectOrNil(p.SubjectKind, p.SubjectID),
		Details:    p.Details,
		CreatedAt:  p.CreatedAt.UTC().Format(time.RFC3339),
		Scenario:   paymentAnomalyScenarioName,
	})
	if err != nil {
		d.recordFailure(ctx, p.ID, 0, fmt.Sprintf("marshal payload: %v", err))
		return
	}

	var lastErr string
	attempts := 0
	for attempts < d.maxAttempts {
		attempts++
		code, errStr, retryable := d.sendOnce(ctx, p.WebhookURL, body)
		if code >= 200 && code < 300 {
			d.recordSuccess(ctx, p.ID, attempts)
			return
		}
		lastErr = errStr
		if !retryable || attempts >= d.maxAttempts {
			break
		}
		idx := attempts - 1
		if idx >= len(d.backoffs) {
			idx = len(d.backoffs) - 1
		}
		select {
		case <-ctx.Done():
			d.recordFailure(ctx, p.ID, attempts, ctx.Err().Error())
			return
		case <-time.After(d.backoffs[idx]):
		}
	}
	d.recordFailure(ctx, p.ID, attempts, lastErr)
}

// sendOnce performs one HTTP attempt. Returns (statusCode, errString,
// retryable). retryable is true for 5xx and transport errors, false for 2xx
// (success) and 4xx (caller bug — do not amplify).
func (d *AnomalyAlertDispatcher) sendOnce(ctx context.Context, url string, body []byte) (int, string, bool) {
	attemptCtx, cancel := context.WithTimeout(ctx, d.perAttempt)
	defer cancel()
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err.Error(), false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", AnomalyDispatchUserAgent)

	d.mu.Lock()
	client := d.httpClient
	d.mu.Unlock()

	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error(), true
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, "", false
	}
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	errStr := fmt.Sprintf("http %d: %s", resp.StatusCode, string(snippet))
	retryable := resp.StatusCode >= 500
	return resp.StatusCode, errStr, retryable
}

func (d *AnomalyAlertDispatcher) recordSuccess(ctx context.Context, rowID int64, attempts int) {
	if _, err := d.db.ExecContext(ctx, `
		UPDATE payment_anomaly_log
		SET dispatch_status = $1, dispatch_attempts = $2, dispatched_at = NOW(), dispatch_error = NULL
		WHERE id = $3
	`, anomalyDispatchSent, attempts, rowID); err != nil {
		logStructuredError("payment_anomaly_dispatch_record_success_failed", map[string]interface{}{
			"id":    rowID,
			"error": err.Error(),
		})
	}
}

func (d *AnomalyAlertDispatcher) recordFailure(ctx context.Context, rowID int64, attempts int, lastErr string) {
	trimmed := lastErr
	if len(trimmed) > 512 {
		trimmed = trimmed[:512]
	}
	// Use Background context for the record so a cancelled parent ctx still
	// persists the terminal state — otherwise a shutdown mid-retry would
	// leave the row stuck in "pending" forever.
	if _, err := d.db.ExecContext(context.Background(), `
		UPDATE payment_anomaly_log
		SET dispatch_status = $1, dispatch_attempts = $2, dispatch_error = NULLIF($3, '')
		WHERE id = $4
	`, anomalyDispatchFailed, attempts, trimmed, rowID); err != nil {
		logStructuredError("payment_anomaly_dispatch_record_failure_failed", map[string]interface{}{
			"id":    rowID,
			"error": err.Error(),
		})
	}
}

func subjectOrNil(kind, id string) *anomalySubjectWire {
	if kind == "" && id == "" {
		return nil
	}
	return &anomalySubjectWire{Kind: kind, ID: id}
}
