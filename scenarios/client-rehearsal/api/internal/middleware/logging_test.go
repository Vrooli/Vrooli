package middleware_test

import (
	"client-rehearsal/internal/middleware"
	"client-rehearsal/internal/testutil/mocks"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	logger := &mocks.FakeLogger{}

	mw := middleware.NewLoggingMiddleware(clk, logger)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clk.Advance(150 * time.Millisecond) // simulate work
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, logger.Lines, 1)
	require.True(t, strings.Contains(logger.Lines[0], "[GET]"), "method should appear in log line")
	require.True(t, strings.Contains(logger.Lines[0], "/probe"), "URI should appear in log line")
	require.True(t, strings.Contains(logger.Lines[0], "150ms"), "deterministic duration must appear (150 * time.Millisecond)")
}

func TestLoggingMiddlewareRequiresLogger(t *testing.T) {
	clk := scheduletest.New(time.Time{})
	require.Panics(t, func() { middleware.NewLoggingMiddleware(clk, nil) })
}
