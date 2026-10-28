// Package testutil provides agent-manager-specific HTTP contract fixtures.
package testutil

import (
	"github.com/vrooli/cli-core/cliutil"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// Request is the observable HTTP contract emitted by a CLI service call.
type Request struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// RecordingServer serves a JSON response and retains each received request.
// It remains scenario-local because its response contract is specific to the
// agent-manager CLI's endpoint suite.
type RecordingServer struct {
	mu       sync.Mutex
	requests []Request
	server   *httptest.Server
}

func NewRecordingServer(t testing.TB, response string) *RecordingServer {
	return NewRecordingServerForRequests(t, func(Request) string { return response })
}

// NewRecordingServerForRequests selects a response from each received request.
func NewRecordingServerForRequests(t testing.TB, response func(Request) string) *RecordingServer {
	t.Helper()
	recorder := &RecordingServer{}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read test request: %v", err)
		}
		request := Request{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)}
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

// IsolateCredentialBroker gives disposable HTTP fixtures a process-local owner
// seam. The broker explicitly preserves the fixture's inherited credential;
// tests never consult a developer's live identity authority or socket.
func IsolateCredentialBroker(t *testing.T) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "am-cli-credentials-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", dir)
	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/credential" {
			t.Errorf("unexpected credential request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); _ = os.RemoveAll(dir) })
}
