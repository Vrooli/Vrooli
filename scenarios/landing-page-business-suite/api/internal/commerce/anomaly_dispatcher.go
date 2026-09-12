// Package commerce owns subscription, payment, and payment-operations domain
// behavior. It deliberately depends on infrastructure through narrow ports.
package commerce

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

const AnomalyDispatchUserAgent = "lpbs-anomaly-dispatcher/1"

const (
	DispatchPending = "pending"
	DispatchSent    = "sent"
	DispatchSkipped = "skipped"
	DispatchFailed  = "failed"
)

// PaymentAnomalyStore is the persistence port used by the payment-operations
// service and its dispatcher.
//
// seam: Payment anomaly persistence remains independent of API composition.
type PaymentAnomalyStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type AnomalyRateLimit struct {
	Burst         int `json:"burst"`
	RefillSeconds int `json:"refill_seconds"`
}

// AnomalyDispatchPayload is the internal dispatch intent. Subject and webhook
// fields are intentionally excluded from the integrator wire format.
type AnomalyDispatchPayload struct {
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

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type DispatcherRuntime struct {
	ScenarioName string
	LogError     func(string, map[string]interface{})
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

// AnomalyAlertDispatcher dispatches anomaly webhooks with retry, rate
// limiting, and per-row dispatch-state tracking.
type AnomalyAlertDispatcher struct {
	db          PaymentAnomalyStore
	httpClient  httpDoer
	logError    func(string, map[string]interface{})
	scenario    string
	mu          sync.Mutex
	buckets     map[string]*tokenBucket
	maxAttempts int
	perAttempt  time.Duration
	backoffs    []time.Duration
}

func NewAnomalyAlertDispatcher(db PaymentAnomalyStore, runtime DispatcherRuntime) *AnomalyAlertDispatcher {
	return &AnomalyAlertDispatcher{
		db:          db,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		logError:    runtime.LogError,
		scenario:    runtime.ScenarioName,
		buckets:     map[string]*tokenBucket{},
		maxAttempts: 3,
		perAttempt:  5 * time.Second,
		backoffs:    []time.Duration{time.Second, 2 * time.Second, 4 * time.Second},
	}
}

func (d *AnomalyAlertDispatcher) UseHTTPClient(client httpDoer) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.httpClient = client
}

func (d *AnomalyAlertDispatcher) UseBackoff(backoffs []time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.backoffs = append([]time.Duration(nil), backoffs...)
}

func (d *AnomalyAlertDispatcher) UseMaxAttempts(attempts int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxAttempts = attempts
}

func (d *AnomalyAlertDispatcher) UsePerAttempt(timeout time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.perAttempt = timeout
}

func (d *AnomalyAlertDispatcher) Dispatch(ctx context.Context, p AnomalyDispatchPayload) {
	if p.WebhookURL == "" {
		return
	}
	body, err := json.Marshal(anomalyDispatchWireFormat{ID: p.ID, Type: p.Type, Severity: p.Severity, Email: p.Email, CustomerID: p.CustomerID, Subject: subjectOrNil(p.SubjectKind, p.SubjectID), Details: p.Details, CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339), Scenario: d.scenario})
	if err != nil {
		d.recordFailure(ctx, p.ID, 0, fmt.Sprintf("marshal payload: %v", err))
		return
	}
	d.mu.Lock()
	maxAttempts, backoffs := d.maxAttempts, append([]time.Duration(nil), d.backoffs...)
	d.mu.Unlock()
	var lastErr string
	attempts := 0
	for attempts < maxAttempts {
		attempts++
		code, errStr, retryable := d.sendOnce(ctx, p.WebhookURL, body)
		if code >= 200 && code < 300 {
			d.recordSuccess(ctx, p.ID, attempts)
			return
		}
		lastErr = errStr
		if !retryable || attempts >= maxAttempts {
			break
		}
		idx := attempts - 1
		if idx >= len(backoffs) {
			idx = len(backoffs) - 1
		}
		if idx < 0 {
			break
		}
		select {
		case <-ctx.Done():
			d.recordFailure(ctx, p.ID, attempts, ctx.Err().Error())
			return
		case <-time.After(backoffs[idx]):
		}
	}
	d.recordFailure(ctx, p.ID, attempts, lastErr)
}

func (d *AnomalyAlertDispatcher) Allow(anomalyType string, limits map[string]AnomalyRateLimit, defaultBurst, defaultRefillSeconds int) bool {
	if d == nil {
		return true
	}
	burst, refillSeconds := defaultBurst, defaultRefillSeconds
	if override, ok := limits[anomalyType]; ok {
		burst, refillSeconds = override.Burst, override.RefillSeconds
	}
	if burst <= 0 || refillSeconds <= 0 {
		return true
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	bucket, ok := d.buckets[anomalyType]
	if !ok {
		bucket = &tokenBucket{tokens: burst, lastRefill: now}
		d.buckets[anomalyType] = bucket
	}
	if elapsed := now.Sub(bucket.lastRefill); elapsed > 0 {
		if add := int(elapsed / (time.Duration(refillSeconds) * time.Second)); add > 0 {
			bucket.tokens += add
			if bucket.tokens > burst {
				bucket.tokens = burst
			}
			bucket.lastRefill = bucket.lastRefill.Add(time.Duration(add) * time.Duration(refillSeconds) * time.Second)
		}
	}
	if bucket.tokens <= 0 {
		return false
	}
	bucket.tokens--
	return true
}

func (d *AnomalyAlertDispatcher) sendOnce(ctx context.Context, url string, body []byte) (int, string, bool) {
	d.mu.Lock()
	perAttempt, client := d.perAttempt, d.httpClient
	d.mu.Unlock()
	attemptCtx, cancel := context.WithTimeout(ctx, perAttempt)
	defer cancel()
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err.Error(), false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", AnomalyDispatchUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error(), true
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, "", false
	}
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return resp.StatusCode, fmt.Sprintf("http %d: %s", resp.StatusCode, string(snippet)), resp.StatusCode >= 500
}

func (d *AnomalyAlertDispatcher) recordSuccess(ctx context.Context, rowID int64, attempts int) {
	if _, err := d.db.ExecContext(ctx, `UPDATE payment_anomaly_log SET dispatch_status = $1, dispatch_attempts = $2, dispatched_at = NOW(), dispatch_error = NULL WHERE id = $3`, DispatchSent, attempts, rowID); err != nil {
		d.log("payment_anomaly_dispatch_record_success_failed", map[string]interface{}{"id": rowID, "error": err.Error()})
	}
}

func (d *AnomalyAlertDispatcher) recordFailure(ctx context.Context, rowID int64, attempts int, lastErr string) {
	if len(lastErr) > 512 {
		lastErr = lastErr[:512]
	}
	if _, err := d.db.ExecContext(context.WithoutCancel(ctx), `UPDATE payment_anomaly_log SET dispatch_status = $1, dispatch_attempts = $2, dispatch_error = NULLIF($3, '') WHERE id = $4`, DispatchFailed, attempts, lastErr, rowID); err != nil {
		d.log("payment_anomaly_dispatch_record_failure_failed", map[string]interface{}{"id": rowID, "error": err.Error()})
	}
}

func (d *AnomalyAlertDispatcher) log(event string, fields map[string]interface{}) {
	if d.logError != nil {
		d.logError(event, fields)
	}
}

func subjectOrNil(kind, id string) *anomalySubjectWire {
	if kind == "" && id == "" {
		return nil
	}
	return &anomalySubjectWire{Kind: kind, ID: id}
}
