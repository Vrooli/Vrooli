package emitter

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// [REQ:DI-002] sendOne retries on HTTP failure and eventually succeeds
func TestSendOne_RetriesOnFailureThenSucceeds(t *testing.T) {
	var attempts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    3,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{EventID: "retry-1", EventType: "test.retry"})
	e.Close()

	sent, dropped := e.Stats()
	if sent != 1 {
		t.Fatalf("expected 1 sent after retries, got %d (dropped=%d)", sent, dropped)
	}
	if got := attempts.Load(); got < 3 {
		t.Fatalf("expected at least 3 attempts, got %d", got)
	}
}

// [REQ:DI-002] sendOne drops event after exhausting all retries
func TestSendOne_DropsAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    2,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{EventID: "fail-1", EventType: "test.fail"})
	e.Close()

	sent, dropped := e.Stats()
	if sent != 0 {
		t.Fatalf("expected 0 sent, got %d", sent)
	}
	if dropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", dropped)
	}
}

// [REQ:DI-002] sendOne handles connection refused (unreachable server)
func TestSendOne_ConnectionRefused(t *testing.T) {
	e := NewEmitter(Config{
		EventsURL:     "http://127.0.0.1:1", // port 1 should be unreachable
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{EventID: "connrefused-1", EventType: "test.connrefused"})
	e.Close()

	sent, dropped := e.Stats()
	if sent != 0 {
		t.Fatalf("expected 0 sent with unreachable server, got %d", sent)
	}
	if dropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", dropped)
	}
}

// [REQ:DI-002] sendOne sends correct Content-Type and body
func TestSendOne_RequestFormat(t *testing.T) {
	var gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		gotBody = buf[:n]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    10,
		BatchSize:     1,
		FlushInterval: time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{
		EventID:        "fmt-1",
		EventType:      "test.format",
		SourceScenario: "src",
	})
	e.Close()

	if gotContentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", gotContentType)
	}
	if len(gotBody) == 0 {
		t.Fatal("expected non-empty body")
	}
}

// [REQ:DI-002] Batch flush triggered by batch size threshold
func TestDrain_BatchSizeFlush(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    256,
		BatchSize:     3,
		FlushInterval: time.Hour, // very long, so only batch size triggers flush
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	for i := 0; i < 3; i++ {
		e.Emit(EventPayload{EventID: "batch", EventType: "test.batch"})
	}

	// Wait for batch flush
	time.Sleep(100 * time.Millisecond)

	got := received.Load()
	if got != 3 {
		t.Fatalf("expected 3 events flushed by batch size, got %d", got)
	}

	e.Close()
}

// [REQ:DI-002] Timer-based flush sends events before batch is full
func TestDrain_TimerFlush(t *testing.T) {
	var received atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := NewEmitter(Config{
		EventsURL:     srv.URL,
		BufferSize:    256,
		BatchSize:     100, // large batch so only timer triggers
		FlushInterval: 20 * time.Millisecond,
		MaxRetries:    1,
		RetryBackoff:  time.Millisecond,
	})

	e.Emit(EventPayload{EventID: "timer-1", EventType: "test.timer"})

	// Wait for timer-based flush
	time.Sleep(100 * time.Millisecond)

	got := received.Load()
	if got != 1 {
		t.Fatalf("expected 1 event flushed by timer, got %d", got)
	}

	e.Close()
}
