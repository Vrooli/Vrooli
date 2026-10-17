// Package opsalert owns the bounded outbound transport shared by operational
// alerts. Callers provide the alert payload; this package owns retry, timeout,
// backoff, and token-bucket admission.
package opsalert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type Transport struct {
	client   HTTPDoer
	mu       sync.Mutex
	buckets  map[string]*bucket
	attempts int
	timeout  time.Duration
	backoff  []time.Duration
}
type bucket struct {
	tokens   int
	refilled time.Time
}

func New() *Transport {
	return &Transport{client: &http.Client{Timeout: 5 * time.Second}, buckets: map[string]*bucket{}, attempts: 3, timeout: 5 * time.Second, backoff: []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}}
}
func (t *Transport) UseHTTPClient(client HTTPDoer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.client = client
}
func (t *Transport) SetTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = timeout
}
func (t *Transport) Allow(key string, burst int, refill time.Duration, now time.Time) bool {
	if burst <= 0 || refill <= 0 {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	b := t.buckets[key]
	if b == nil {
		b = &bucket{tokens: burst, refilled: now}
		t.buckets[key] = b
	}
	if add := int(now.Sub(b.refilled) / refill); add > 0 {
		b.tokens += add
		if b.tokens > burst {
			b.tokens = burst
		}
		b.refilled = b.refilled.Add(time.Duration(add) * refill)
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}
func (t *Transport) Send(ctx context.Context, url string, body []byte) error {
	if url == "" {
		return nil
	}
	t.mu.Lock()
	attempts, backoff := t.attempts, append([]time.Duration(nil), t.backoff...)
	t.mu.Unlock()
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		_, message, retryable := t.SendOnce(ctx, url, body, "opsalert")
		if message == "" {
			return nil
		}
		last = &transportError{message: message}
		if !retryable {
			break
		}
		if attempt+1 >= attempts {
			break
		}
		delay := backoff[min(attempt, len(backoff)-1)]
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return last
}

// SendOnce performs one bounded HTTP attempt and returns status text and
// whether the caller should retry. It is also used by payment anomaly alerts
// to keep their transport behavior identical to other operational alerts.
func (t *Transport) SendOnce(ctx context.Context, url string, body []byte, userAgent string) (int, string, bool) {
	t.mu.Lock()
	client, timeout := t.client, t.timeout
	t.mu.Unlock()
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err.Error(), false
	}
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
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

type httpError struct{ status int }

func (e *httpError) Error() string { return http.StatusText(e.status) }

type transportError struct{ message string }

func (e *transportError) Error() string { return e.message }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
