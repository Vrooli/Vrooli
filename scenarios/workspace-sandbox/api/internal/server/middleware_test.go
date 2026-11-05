package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"workspace-sandbox/internal/logging"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/scheduletest"
)

// fixedClockMW returns a clock that pretends every Now() call advances
// by 50ms. The structuredLogging middleware reads start, then reads
// Since(start) — with this clock the duration is always 50ms.
func fixedClockMW(t *testing.T) *scheduletest.FakeClock {
	t.Helper()
	c := scheduletest.New(time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))
	return c
}

func newTestLogger(t *testing.T, buf *bytes.Buffer) *logging.Logger {
	t.Helper()
	return logging.New("test", logging.WithOutput(buf), logging.WithClock(schedule.System()))
}

// TestMiddleware_Apply_RegistersOrder verifies Apply panics if Logger
// or Clock is missing. Both are required (greenfield: no defaults).
func TestMiddleware_Apply_RequiresLoggerAndClock(t *testing.T) {
	t.Run("nil logger panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Apply with nil Logger should panic")
			}
		}()
		(Middleware{Clock: schedule.System()}).Apply(mux.NewRouter())
	})
	t.Run("nil clock panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("Apply with nil Clock should panic")
			}
		}()
		(Middleware{Logger: logging.New("t", logging.WithClock(schedule.System()))}).Apply(mux.NewRouter())
	})
}

// TestStructuredLogging_RecordsAPIRequest confirms the middleware emits
// exactly one APIRequest log line per served request, with the captured
// status code from the wrapped response writer.
func TestStructuredLogging_RecordsAPIRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(t, &buf)
	mw := Middleware{Logger: logger, Clock: schedule.System()}

	router := mux.NewRouter()
	mw.Apply(router)
	router.HandleFunc("/teapot", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest("GET", "/teapot", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}

	out := buf.String()
	if !strings.Contains(out, `"event":"api.request"`) {
		t.Errorf("log output missing api.request event: %s", out)
	}
	if !strings.Contains(out, `"statusCode":418`) {
		t.Errorf("log output missing captured statusCode 418: %s", out)
	}
	if !strings.Contains(out, `"path":"/teapot"`) {
		t.Errorf("log output missing path /teapot: %s", out)
	}
}

// TestResponseWriter_FlushPassThrough is the regression gate for the
// 2026-04-28 SSE flusher bug. The wrapped responseWriter MUST still
// satisfy http.Flusher so SSE handlers can stream. Without this, every
// fast-failing process loses its `event: exit` frame.
func TestResponseWriter_FlushPassThrough(t *testing.T) {
	logger := newTestLogger(t, &bytes.Buffer{})
	mw := Middleware{Logger: logger, Clock: schedule.System()}

	router := mux.NewRouter()
	mw.Apply(router)

	flushed := false
	router.HandleFunc("/stream", func(w http.ResponseWriter, _ *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("wrapped writer does not satisfy http.Flusher (regression of 2026-04-28 SSE bug)")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: hello\n\n"))
		f.Flush()
		flushed = true
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if !flushed {
		t.Error("handler did not reach Flush() — middleware blocked the writer chain")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// TestResponseWriter_HijackPassThrough exercises the second
// optional-interface forwarding path. WebSocket-style upgrades and the
// interactive-exec endpoint depend on it; if the wrapper drops Hijack,
// they 500 silently.
func TestResponseWriter_HijackPassThrough(t *testing.T) {
	logger := newTestLogger(t, &bytes.Buffer{})
	mw := Middleware{Logger: logger, Clock: schedule.System()}

	router := mux.NewRouter()
	mw.Apply(router)

	hijacked := make(chan bool, 1)
	router.HandleFunc("/upgrade", func(w http.ResponseWriter, _ *http.Request) {
		h, ok := w.(http.Hijacker)
		if !ok {
			hijacked <- false
			return
		}
		conn, _, err := h.Hijack()
		if err != nil {
			hijacked <- false
			return
		}
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n\r\n"))
		_ = conn.Close()
		hijacked <- true
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/upgrade")
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}

	select {
	case ok := <-hijacked:
		if !ok {
			t.Error("hijack failed — wrapper did not forward http.Hijacker")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not return; hijack channel never closed")
	}
}

// TestCORS_AllowlistMatch verifies an exact match in the allowed origins
// reflects the Origin back. Misses are silent (no header written).
func TestCORS_AllowlistMatch(t *testing.T) {
	logger := newTestLogger(t, &bytes.Buffer{})
	mw := Middleware{
		Logger:             logger,
		Clock:              schedule.System(),
		CORSAllowedOrigins: []string{"https://app.example.com"},
	}

	router := mux.NewRouter()
	mw.Apply(router)
	router.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })

	cases := []struct {
		name       string
		origin     string
		wantAllow  string
		wantHeader bool
	}{
		{"allowed", "https://app.example.com", "https://app.example.com", true},
		{"denied", "https://evil.example.com", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/x", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			got := rec.Header().Get("Access-Control-Allow-Origin")
			if got != tc.wantAllow {
				t.Errorf("Allow-Origin = %q, want %q", got, tc.wantAllow)
			}
		})
	}
}

// TestCORS_OptionsShortCircuits confirms preflight requests get a 200
// before the handler chain runs.
func TestCORS_OptionsShortCircuits(t *testing.T) {
	logger := newTestLogger(t, &bytes.Buffer{})
	mw := Middleware{Logger: logger, Clock: schedule.System()}
	router := mux.NewRouter()
	mw.Apply(router)
	called := false
	router.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) { called = true })

	req := httptest.NewRequest("OPTIONS", "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS status = %d, want 200", rec.Code)
	}
	if called {
		t.Error("OPTIONS preflight should not invoke the inner handler")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("Allow-Methods missing POST: %q", got)
	}
}

// TestCORS_EmptyAllowlistFallsBackToUIPort verifies the dev fallback
// path resolves the UI port from the configurable env var, not always
// "UI_PORT", so tests don't depend on operator state.
func TestCORS_EmptyAllowlistFallsBackToUIPort(t *testing.T) {
	logger := newTestLogger(t, &bytes.Buffer{})
	const envName = "WORKSPACE_SANDBOX_TEST_UI_PORT"
	t.Setenv(envName, "5173")
	mw := Middleware{Logger: logger, Clock: schedule.System(), UIPortEnv: envName}

	router := mux.NewRouter()
	mw.Apply(router)
	router.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Allow-Origin = %q, want http://localhost:5173", got)
	}
}

// TestStructuredLogging_DurationFromClock verifies the middleware reads
// duration through Clock, not time.Since directly. Advancing FakeClock
// between Now() and Since() drives the recorded ms value.
func TestStructuredLogging_DurationFromClock(t *testing.T) {
	var buf bytes.Buffer
	fc := fixedClockMW(t)
	logger := logging.New("test", logging.WithOutput(&buf), logging.WithClock(fc))
	mw := Middleware{Logger: logger, Clock: fc}

	router := mux.NewRouter()
	mw.Apply(router)
	router.HandleFunc("/x", func(w http.ResponseWriter, _ *http.Request) {
		fc.Advance(750 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), `"durationMs":750`) {
		t.Errorf("duration not measured through clock seam: %s", buf.String())
	}
}
