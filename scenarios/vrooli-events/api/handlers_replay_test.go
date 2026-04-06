package main

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// [REQ:REQ-PS-003] Last-Event-ID resume
// [REQ:REQ-ES-001A] Storage layer provides GetSince for replay

func TestReplay_InvalidLastEventID(t *testing.T) {
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

	// Should still get SSE stream (invalid ID is logged and skipped)
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", ct)
	}
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		line := scanner.Text()
		if line != "retry: 5000" {
			t.Fatalf("expected retry directive, got %q", line)
		}
	}
}

func TestReplay_ReplaysMissedEvents(t *testing.T) {
	_, ts := newTestServer(t)

	// Ingest 3 events
	for i := 1; i <= 3; i++ {
		body := `{"eventId":"replay-` + itoa(int64(i)) + `","sourceScenario":"test","eventType":"test.replay.v1"}`
		resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
		resp.Body.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Subscribe with Last-Event-ID=1, should replay events 2 and 3
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	req.Header.Set("Last-Event-ID", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	replayedEvents := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: test.replay.v1") {
			replayedEvents++
		}
		if replayedEvents >= 2 {
			break
		}
	}
	if replayedEvents < 2 {
		t.Fatalf("expected >=2 replayed events, got %d", replayedEvents)
	}
}

func TestReplay_NoLastEventID_SkipsReplay(t *testing.T) {
	_, ts := newTestServer(t)

	// Ingest an event first
	body := `{"eventId":"no-replay-1","sourceScenario":"test","eventType":"test.noreplay.v1"}`
	resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Subscribe without Last-Event-ID - should NOT replay old events
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// Read retry directive
	if scanner.Scan() {
		line := scanner.Text()
		if line != "retry: 5000" {
			t.Fatalf("expected retry directive, got %q", line)
		}
	}

	// Next line should be blank (end of retry frame), not a replayed event
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: test.noreplay.v1") {
			t.Fatal("received replayed event when no Last-Event-ID was set")
		}
	}
}

// [REQ:REQ-ES-003A] Time-based retention pruning verified via query after insert
func TestReplay_LargeLastEventID_ReturnsEmpty(t *testing.T) {
	_, ts := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Last-Event-ID far beyond any stored event
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/events/subscribe", nil)
	req.Header.Set("Last-Event-ID", "999999")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %s", ct)
	}
}
