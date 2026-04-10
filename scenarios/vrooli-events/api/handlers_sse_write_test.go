package main

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
)

// [REQ:REQ-PS-001A] SSE message format compliance
// [REQ:REQ-PS-001A1] Heartbeat comment format

func TestWriteSSEMessage_Heartbeat_WithData(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEMessage(w, broker.SSEMessage{Event: "heartbeat", Data: `{"subs":3}`})
	got := w.Body.String()
	want := ": heartbeat {\"subs\":3}\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteSSEMessage_Heartbeat_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEMessage(w, broker.SSEMessage{Event: "heartbeat"})
	got := w.Body.String()
	want := ": heartbeat\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteSSEMessage_DataEvent(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEMessage(w, broker.SSEMessage{ID: 42, Event: "test.v1", Data: `{"key":"val"}`})
	got := w.Body.String()
	want := "id: 42\nevent: test.v1\ndata: {\"key\":\"val\"}\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWriteSSEMessage_ZeroID(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEMessage(w, broker.SSEMessage{ID: 0, Event: "init.v1", Data: "{}"})
	got := w.Body.String()
	want := "id: 0\nevent: init.v1\ndata: {}\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// [REQ:REQ-PS-001B] SSE headers and retry directive

func TestWriteSSEHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEHeaders(w, w, 5000)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cc)
	}
	if xa := w.Header().Get("X-Accel-Buffering"); xa != "no" {
		t.Fatalf("X-Accel-Buffering = %q, want no", xa)
	}
	body := w.Body.String()
	if body != "retry: 5000\n\n" {
		t.Fatalf("body = %q, want retry directive", body)
	}
}

func TestWriteSSEHeaders_CustomRetry(t *testing.T) {
	w := httptest.NewRecorder()
	writeSSEHeaders(w, w, 3000)
	if !bytes.Contains(w.Body.Bytes(), []byte("retry: 3000")) {
		t.Fatalf("expected retry: 3000 in %q", w.Body.String())
	}
}
