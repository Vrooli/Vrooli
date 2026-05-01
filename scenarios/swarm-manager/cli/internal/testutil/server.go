package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewAPIServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)
	WithAPIBase(tb, server.URL)
	return server
}

func WithAPIBase(tb testing.TB, apiBase string) {
	tb.Helper()
	tb.Setenv("SWARM_MANAGER_API_BASE", apiBase)
}
