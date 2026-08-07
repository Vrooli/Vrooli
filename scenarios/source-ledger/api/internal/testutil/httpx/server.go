// Package httpx provides a live HTTP harness for handler tests.
//
// LiveServer wraps the production *server.Server in an httptest.Server,
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
package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"source-ledger/internal/server"
)

// LiveServer is the test-side equivalent of main.go's HTTP boot path:
// the production middleware stack wraps the production handler routes,
// behind an httptest.Server. Tests obtain an instance via NewLiveServer
// and exercise it through a real *http.Client.
type LiveServer struct {
	*httptest.Server
	Client *http.Client
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
func NewLiveServer(t *testing.T, s *server.Server) *LiveServer {
	t.Helper()
	return newLiveServer(t, s)
}

func newLiveServer(t failer, s *server.Server) *LiveServer {
	t.Helper()
	if s == nil {
		t.Fatal("httpx.NewLiveServer: server is required")
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
