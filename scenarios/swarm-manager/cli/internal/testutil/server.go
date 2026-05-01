package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func NewAPIServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := NewHTTPServer(tb, handler)
	WithAPIBase(tb, server.URL)
	return server
}

func NewHTTPServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)
	return server
}

func WithAPIBase(tb testing.TB, apiBase string) {
	tb.Helper()
	tb.Setenv("SWARM_MANAGER_API_BASE", apiBase)
}

func CaptureStdout(tb testing.TB, fn func() error) string {
	tb.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		tb.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	tb.Cleanup(func() { os.Stdout = orig })

	runErr := fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		tb.Fatalf("copy stdout: %v", err)
	}
	if runErr != nil {
		tb.Fatalf("command returned error: %v", runErr)
	}
	return buf.String()
}
