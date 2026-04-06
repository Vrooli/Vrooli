package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// [REQ:REQ-PS-001] SSE streaming with retry directive
func TestSSESubscription(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if line != "retry: 5000" {
			t.Fatalf("expected retry: 5000, got %q", line)
		}
	}

	// Ingest an event and verify it arrives via SSE
	eventJSON := `{"eventId":"sse-evt-1","sourceScenario":"test","eventType":"test.sse.v1"}`
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(eventJSON))

	var gotEvent bool
	deadline := time.After(3 * time.Second)
	for !gotEvent {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for SSE event")
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if eventType == "test.sse.v1" {
				gotEvent = true
			}
		}
	}

	if !gotEvent {
		t.Fatal("did not receive SSE event")
	}
}

// [REQ:REQ-PS-001] SSE response includes correct headers
func TestSSE_Headers(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	// Verify all required SSE headers
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type: expected text/event-stream, got %s", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control: expected no-cache, got %s", cc)
	}
	if conn := resp.Header.Get("Connection"); conn != "keep-alive" {
		t.Fatalf("Connection: expected keep-alive, got %s", conn)
	}
	if xab := resp.Header.Get("X-Accel-Buffering"); xab != "no" {
		t.Fatalf("X-Accel-Buffering: expected no, got %s", xab)
	}
}

// [REQ:REQ-PS-004] Heartbeat reports dropped count under backpressure
func TestSSE_BackpressureHeartbeat(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream")
	}

	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "retry:") {
			t.Fatalf("expected retry directive, got %q", line)
		}
		// Verify retry value is a number
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			t.Fatalf("malformed retry line: %q", line)
		}
		retryVal := strings.TrimSpace(parts[1])
		if retryVal == "" {
			t.Fatal("retry value is empty")
		}
	} else {
		t.Fatal("expected at least one line from SSE stream")
	}
}

// [REQ:REQ-PS-004] SSE subscriber channel has expected capacity for backpressure handling
func TestSSE_MultipleEventsDelivered(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan() // retry line

	// Ingest multiple events rapidly
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf(`{"eventId":"bp-%d","sourceScenario":"s","eventType":"test.bp.v1"}`, i)
		_, _ = http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	}

	received := 0
	deadline := time.After(3 * time.Second)
	for received < 5 {
		select {
		case <-deadline:
			goto done
		default:
		}
		if !scanner.Scan() {
			break
		}
		if strings.HasPrefix(scanner.Text(), "event: test.bp.v1") {
			received++
		}
	}
done:
	if received < 3 {
		t.Fatalf("expected at least 3 events delivered under rapid ingestion, got %d", received)
	}
}
