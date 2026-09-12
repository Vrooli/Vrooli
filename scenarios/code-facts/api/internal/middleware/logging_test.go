package middleware_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code-facts/internal/middleware"

	"github.com/vrooli/api-core/scheduletest"

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
	clk := scheduletest.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
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

// TestLoggingMiddleware_RequiresLogger proves a composition error cannot hide
// behind a package-global logger that tests cannot substitute.
func TestLoggingMiddleware_RequiresLogger(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	require.PanicsWithValue(t, "middleware.NewLoggingMiddleware requires a logger", func() {
		middleware.NewLoggingMiddleware(clk, nil)
	})
}
