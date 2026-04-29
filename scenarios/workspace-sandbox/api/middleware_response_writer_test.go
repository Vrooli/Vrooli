package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/logging"
)

// TestStructuredLoggingMiddleware_PreservesFlusherInterface guards against
// the regression that caused agent-manager runs to fail with
// `ErrSandboxNoExitInfo` on 2026-04-28: every SSE handler does
// `w.(http.Flusher)`; if the responseWriter wrapper doesn't satisfy that
// interface, every stream returns HTTP 500 "streaming not supported".
//
// This is a black-box test — it stands up a live HTTP server with the
// real middleware in the chain rather than calling the handler directly.
// Tests that use httptest.ResponseRecorder bypass the middleware and
// would not have caught the original bug.
func TestStructuredLoggingMiddleware_PreservesFlusherInterface(t *testing.T) {
	t.Parallel()

	srv := &Server{logger: logging.New("test", logging.WithClock(clock.System{}))}

	router := mux.NewRouter()
	router.Use(srv.structuredLoggingMiddleware)

	// Inline SSE handler that mirrors the real StreamProcessLogs
	// shape: assert Flusher, write events, flush between them.
	router.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: chunk-1\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("event: end\ndata: stream closed\n\n"))
		flusher.Flush()
	}).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want 200 (middleware-wrapped writer must satisfy http.Flusher)", resp.StatusCode)
	}

	got, err := readAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(got, "data: chunk-1") || !strings.Contains(got, "event: end") {
		t.Fatalf("body did not contain expected SSE frames: %q", got)
	}
}

// TestStructuredLoggingMiddleware_PreservesHijackerInterface guards the
// same wrapper for Hijacker — used by WebSocket upgrades and other
// long-lived connections that take over the underlying conn.
func TestStructuredLoggingMiddleware_PreservesHijackerInterface(t *testing.T) {
	t.Parallel()

	srv := &Server{logger: logging.New("test", logging.WithClock(clock.System{}))}

	router := mux.NewRouter()
	router.Use(srv.structuredLoggingMiddleware)

	hijackOK := make(chan bool, 1)
	router.HandleFunc("/hijack", func(w http.ResponseWriter, r *http.Request) {
		_, ok := w.(http.Hijacker)
		hijackOK <- ok
		// Don't actually hijack — return a normal response so the test
		// client doesn't hang. The interface assertion is what matters.
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	ts := httptest.NewServer(router)
	defer ts.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL + "/hijack")
	if err != nil {
		t.Fatalf("GET /hijack: %v", err)
	}
	resp.Body.Close()

	select {
	case ok := <-hijackOK:
		if !ok {
			t.Fatal("middleware-wrapped writer does not satisfy http.Hijacker")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not run")
	}
}

func readAll(r interface {
	Read(p []byte) (n int, err error)
},
) (string, error) {
	br := bufio.NewReader(r)
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := br.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			if err.Error() == "EOF" {
				return sb.String(), nil
			}
			return sb.String(), err
		}
	}
}
