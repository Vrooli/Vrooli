// Package servertest provides a live HTTP harness for api-core/server consumers.
//
// LiveServer wraps any production server exposing Handler in an httptest.Server,
// so handler tests exercise the full middleware chain over a real TCP
// socket — the same code path main.go ships.
//
// # Why not httptest.ResponseRecorder
//
// ResponseRecorder natively implements http.Flusher and http.Hijacker.
// A wrapper that accidentally drops those interfaces still passes
// recorder-based tests but breaks streaming endpoints in production. A
// real httptest.Server eliminates that gap by running over a real
// socket through the real handler chain.
package servertest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// LiveServer is the test-side equivalent of main.go's HTTP boot path:
// the production middleware stack wraps the production handler routes,
// behind an httptest.Server. Tests obtain an instance via NewLiveServer
// and exercise it through a real *http.Client.
type LiveServer struct {
	*httptest.Server
	Client *http.Client
}

// HandlerProvider is the narrow production surface required by the harness.
// Keeping the dependency at one method lets every server implementation adopt
// the companion without importing a scenario's internal package.
type HandlerProvider interface {
	Handler() http.Handler
}

// failer is the minimum surface NewLiveServer needs. It exists so the
// nil-server guard can be exercised by this package's own tests with a
// recording stub — testing.TB cannot be implemented externally because
// of its private() gate.
type failer interface {
	Helper()
	Cleanup(func())
	Fatal(args ...any)
}

// NewLiveServer spins up a live HTTP server wired with the production
// middleware stack and the production handler routes (whatever
// s.Handler() returns). The server is torn down via t.Cleanup; tests
// do not need to call Close.
func NewLiveServer(t *testing.T, s HandlerProvider) *LiveServer {
	t.Helper()
	return newLiveServer(t, s)
}

// NewHandlerServer is the compatibility spelling used by older scenario
// tests. It retains the same cleanup and real-socket behavior as
// NewLiveServer, without requiring a wrapper type for callers that only need
// the httptest.Server URL.
func NewHandlerServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// NewServer is the compatibility spelling for a path-handler fixture.
func NewServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range routes {
		mux.HandleFunc(path, handler)
	}
	return NewHandlerServer(t, mux)
}

// TestClient returns a bounded client for compatibility fixtures. Callers
// that need the exact transport from an httptest.Server should use its
// Client method instead.
func TestClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// AssertMethod is the legacy fixture assertion retained for consumers of the
// servertest compatibility surface.
func AssertMethod(t *testing.T, req *http.Request, want string) {
	t.Helper()
	if req == nil {
		t.Fatalf("AssertMethod: request is nil (want %s)", want)
	}
	if req.Method != want {
		t.Fatalf("AssertMethod: got %s, want %s", req.Method, want)
	}
}

// WriteJSON writes a JSON fixture response with an explicit status.
func WriteJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
}

// DecodeJSON decodes a request body for legacy HTTP fixture handlers.
func DecodeJSON[T any](t *testing.T, r *http.Request) T {
	t.Helper()
	var value T
	if r == nil || r.Body == nil {
		t.Fatalf("DecodeJSON: request body is nil")
		return value
	}
	if err := json.NewDecoder(r.Body).Decode(&value); err != nil {
		t.Fatalf("DecodeJSON: %v", err)
	}
	return value
}

func newLiveServer(t failer, s HandlerProvider) *LiveServer {
	t.Helper()
	if s == nil {
		t.Fatal("servertest.NewLiveServer: server is required")
		return nil
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return &LiveServer{Server: srv, Client: srv.Client()}
}

// Do issues an HTTP request through the live server. body may be nil
// for GET-style requests. The helper reads and closes the body before
// returning, so tests can hold the returned bytes without leaking.
func (l *LiveServer) Do(t *testing.T, method, path string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	req, err := http.NewRequest(method, l.URL+path, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := l.Client.Do(req)
	if err != nil {
		t.Fatalf("client.Do %s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, respBody
}
