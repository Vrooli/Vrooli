package emitter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// [REQ:DI-002] Fire-and-forget event emission tests

func TestEmit_AcceptsWhenBufferHasSpace(t *testing.T) {
	// [REQ:DI-002] Emit returns true when buffer has space
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:    srv.URL,
		BufferSize:   10,
		BatchSize:    10,
		MaxRetries:   1,
		RetryBackoff: time.Millisecond,
	})
	defer e.Close()

	ok := e.Emit(EventPayload{EventID: "1", EventType: "test"})
	if !ok {
		t.Fatal("expected Emit to return true when buffer has space")
	}
}

func TestEmit_DropsWhenBufferFull(t *testing.T) {
	// [REQ:DI-002] Emit returns false (drops) when buffer is full
	// Create an emitter with buffer size 1 and a very slow flush interval.
	// We pause the drain by using a blocking HTTP server so the buffer stays full.
	blocker := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocker
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    1,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	// First emit fills the buffer
	e.Emit(EventPayload{EventID: "1", EventType: "fill"})

	// Wait for drain to pick up the event and block on the HTTP call
	time.Sleep(50 * time.Millisecond)

	// Second emit fills the buffer again
	e.Emit(EventPayload{EventID: "2", EventType: "fill2"})

	// Third emit should be dropped
	ok := e.Emit(EventPayload{EventID: "3", EventType: "overflow"})
	if ok {
		t.Fatal("expected Emit to return false when buffer is full")
	}

	_, dropped := e.Stats()
	if dropped < 1 {
		t.Fatalf("expected at least 1 dropped event, got %d", dropped)
	}

	// Unblock the server so Close can finish
	close(blocker)
	e.Close()
}

func TestEmit_DeliversToServer(t *testing.T) {
	// [REQ:DI-002] Events are delivered to a test HTTP server
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var evt EventPayload
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			t.Errorf("failed to decode event: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    256,
		BatchSize:     5,
		FlushInterval: 10 * time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	for i := 0; i < 5; i++ {
		e.Emit(EventPayload{EventID: "evt", EventType: "test"})
	}

	e.Close()

	if got := received.Load(); got != 5 {
		t.Fatalf("expected 5 events delivered, got %d", got)
	}
}

func TestStats_ReturnsCorrectCounts(t *testing.T) {
	// [REQ:DI-002] Stats() returns correct counts
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    256,
		BatchSize:     10,
		FlushInterval: 10 * time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	for i := 0; i < 3; i++ {
		e.Emit(EventPayload{EventID: "s", EventType: "test"})
	}

	e.Close()

	sent, dropped := e.Stats()
	if sent != 3 {
		t.Fatalf("expected 3 sent, got %d", sent)
	}
	if dropped != 0 {
		t.Fatalf("expected 0 dropped, got %d", dropped)
	}
}

func TestClose_FlushesRemainingEvents(t *testing.T) {
	// [REQ:DI-002] Close() flushes remaining events
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    256,
		BatchSize:     100, // large batch so timer flush is needed
		FlushInterval: time.Hour,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{EventID: "flush1", EventType: "test"})
	e.Emit(EventPayload{EventID: "flush2", EventType: "test"})

	// Close should flush without waiting for the timer
	e.Close()

	if got := received.Load(); got != 2 {
		t.Fatalf("expected 2 events flushed on Close, got %d", got)
	}
}
