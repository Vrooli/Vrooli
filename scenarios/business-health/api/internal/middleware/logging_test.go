package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"business-health/internal/middleware"
	"business-health/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// TestLoggingMiddleware_LogsDuration proves the FakeClock seam works:
// the middleware reads time twice (request start, request end), so
// advancing the fake clock inside the inner handler produces a
// deterministic duration string in the log output.
//
// Without the Clock seam this test would have to time.Sleep(150ms)
// and assert on a fuzzy match — flaky on loaded CI, undefined on
// fast hardware.
func TestLoggingMiddleware_LogsDuration(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	buf := &bytes.Buffer{}
	logger := log.New(buf, "", 0)

	mw := middleware.NewLoggingMiddleware(clk, logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clk.Advance(150 * time.Millisecond) // simulate work
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, buf.String(), "[GET]", "method should appear in log line")
	require.Contains(t, buf.String(), "/probe", "URI should appear in log line")
	require.Contains(t, buf.String(), "150ms", "deterministic duration must appear (150 * time.Millisecond)")
}

// TestLoggingMiddleware_NilLoggerDefaults verifies the nil-logger
// fallback documented on NewLoggingMiddleware. Production callers
// always pass a logger; this guard exists for the case where a
// scenario forgets to wire it during early bring-up.
func TestLoggingMiddleware_NilLoggerDefaults(t *testing.T) {
	clk := mocks.NewFakeClock(time.Time{})
	mw := middleware.NewLoggingMiddleware(clk, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	require.NotPanics(t, func() { handler.ServeHTTP(rec, req) })
	require.Equal(t, http.StatusOK, rec.Code)
}
