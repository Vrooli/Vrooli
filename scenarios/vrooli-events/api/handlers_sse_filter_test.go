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

// [REQ:REQ-PS-002] SSE filters events by type glob pattern — matching and non-matching
func TestSSE_TypeFilterGlob(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe?type=app.user.**", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan() // retry line

	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"f1","sourceScenario":"s","eventType":"app.user.created.v1"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"f2","sourceScenario":"s","eventType":"app.order.placed.v1"}`))

	gotMatching := false
	gotNonMatching := false
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			goto check
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: app.user.") {
			gotMatching = true
		}
		if strings.HasPrefix(line, "event: app.order.") {
			gotNonMatching = true
		}
		if gotMatching {
			break
		}
	}
check:
	if !gotMatching {
		t.Fatal("expected to receive matching event (app.user.*)")
	}
	if gotNonMatching {
		t.Fatal("should not receive non-matching event (app.order.*)")
	}
}

// [REQ:REQ-PS-002] SSE with no filter receives all events
func TestSSE_NoFilterReceivesAll(t *testing.T) {
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

	// Ingest events with different types
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"all-1","sourceScenario":"s","eventType":"app.alpha.v1"}`))
	_, _ = http.Post(ts.URL+"/api/v1/events", "application/json",
		strings.NewReader(`{"eventId":"all-2","sourceScenario":"s","eventType":"app.beta.v1"}`))

	received := 0
	deadline := time.After(2 * time.Second)
	for received < 2 {
		select {
		case <-deadline:
			goto done
		default:
		}
		if !scanner.Scan() {
			break
		}
		if strings.HasPrefix(scanner.Text(), "event: app.") {
			received++
		}
	}
done:
	if received < 2 {
		t.Fatalf("expected at least 2 events without filter, got %d", received)
	}
}

// [REQ:REQ-PS-003] Last-Event-ID replays missed events on SSE reconnect
func TestSSE_LastEventIDResume(t *testing.T) {
	_, ts := newTestServer(t)

	for i := 1; i <= 3; i++ {
		body := fmt.Sprintf(`{"eventId":"resume-evt-%d","sourceScenario":"test","eventType":"test.resume.v1"}`, i)
		resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
		resp.Body.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	req.Header.Set("Last-Event-ID", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	replayedCount := 0
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			goto done
		default:
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "event: test.resume.v1") {
			replayedCount++
		}
		if replayedCount >= 2 {
			break
		}
	}
done:
	if replayedCount < 2 {
		t.Fatalf("expected at least 2 replayed events, got %d", replayedCount)
	}
}

// [REQ:REQ-PS-003] SSE with invalid Last-Event-ID still connects successfully
func TestSSE_InvalidLastEventID(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	req.Header.Set("Last-Event-ID", "not-a-number")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	// Connection should succeed despite invalid Last-Event-ID
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	// Should still get the retry directive
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "retry:") {
			t.Fatalf("expected retry directive, got %q", scanner.Text())
		}
	}
}

// [REQ:REQ-PS-003] SSE with Last-Event-ID=0 replays all events
func TestSSE_LastEventIDZero(t *testing.T) {
	_, ts := newTestServer(t)

	// Ingest 2 events
	for i := 1; i <= 2; i++ {
		body := fmt.Sprintf(`{"eventId":"zero-evt-%d","sourceScenario":"test","eventType":"test.zero.v1"}`, i)
		resp, _ := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
		resp.Body.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	req.Header.Set("Last-Event-ID", "0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	replayedCount := 0
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			goto check
		default:
		}
		if !scanner.Scan() {
			break
		}
		if strings.HasPrefix(scanner.Text(), "event: test.zero.v1") {
			replayedCount++
		}
		if replayedCount >= 2 {
			break
		}
	}
check:
	if replayedCount < 2 {
		t.Fatalf("expected 2 replayed events from Last-Event-ID=0, got %d", replayedCount)
	}
}
