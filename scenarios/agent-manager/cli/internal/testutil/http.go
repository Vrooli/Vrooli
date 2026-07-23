// Package testutil provides shared deterministic fixtures for CLI tests.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Request is the observable HTTP contract emitted by a CLI service call.
type Request struct {
	Method string
	Path   string
	Query  string
}

// RecordingServer serves a JSON response and retains each received request.
// Tests use it to verify the CLI-to-API contract without a running scenario.
type RecordingServer struct {
	mu       sync.Mutex
	requests []Request
	server   *httptest.Server
}

func NewRecordingServer(t testing.TB, response string) *RecordingServer {
	return NewRecordingServerForRequests(t, func(Request) string { return response })
}

// NewRecordingServerForRequests serves a response selected from the received
// request. It keeps contract tests deterministic while allowing a single test
// to model list and detail endpoints with realistic, distinct payloads.
func NewRecordingServerForRequests(t testing.TB, response func(Request) string) *RecordingServer {
	t.Helper()
	recorder := &RecordingServer{}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := Request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, request)
		recorder.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response(request)))
	}))
	t.Cleanup(recorder.server.Close)
	return recorder
}

func (r *RecordingServer) URL() string { return r.server.URL }

func (r *RecordingServer) Requests() []Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Request(nil), r.requests...)
}
