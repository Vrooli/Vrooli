// Package server centralizes cross-cutting HTTP concerns shared between
// the production binary and the live-HTTP test harness.
//
// Why this package exists (Round 4 Phase 3):
//
// The 2026-04-28 SSE flusher bug shipped because handler tests used
// httptest.ResponseRecorder, which natively implements http.Flusher and
// http.Hijacker. The custom responseWriter wrapper in main.go was missing
// those forwarders, so SSE collapsed in production for every fast-failing
// process — but no test caught it. The fix was to add the forwarders, but
// the durable lesson is structural: tests must exercise the same middleware
// stack production runs.
//
// Extracting the middleware into a reusable package lets the production
// binary (cmd: main.go) and the live-HTTP test harness
// (internal/testutil/httpx/server.go) share the exact same wrappers. Any
// future bug class in the responseWriter wrapper is now reachable from
// `go test ./...`.
package server

import (
	"bufio"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"workspace-sandbox/internal/logging"

	"github.com/vrooli/api-core/schedule"
)

// Middleware bundles the cross-cutting HTTP concerns wrapping every
// handler. Both main.go and the test harness construct a Middleware and
// call Apply on the same router type (gorilla/mux.Router), so the
// production stack and the test stack are byte-for-byte identical.
type Middleware struct {
	// Logger receives an APIRequest entry for every served request.
	// Required.
	Logger *logging.Logger

	// Clock supplies request-duration measurement. Required (Round 4
	// Phase 2 made the wall-clock seam explicit; nothing in the
	// middleware path may call time.Now directly).
	Clock schedule.Clock

	// CORSAllowedOrigins is the strict allowlist for Access-Control-
	// Allow-Origin. Empty means "fall back to the dev UI port" (see
	// uiOriginFallback). Production binary populates this from
	// config.Server.CORSAllowedOrigins.
	CORSAllowedOrigins []string

	// UIPortEnv is the environment variable whose value supplies the
	// dev UI port for the empty-allowlist fallback. Tests override
	// this so they don't depend on the operator's environment.
	// Default: "UI_PORT".
	UIPortEnv string
}

// Apply registers the middleware on the router in production order:
// structured logging first (so it sees the final status code via the
// wrapped responseWriter) then CORS. Order matters; the test harness
// calls this same method to guarantee parity.
func (m Middleware) Apply(router *mux.Router) {
	if m.Logger == nil {
		panic("server.Middleware.Apply: Logger is required")
	}
	if m.Clock == nil {
		panic("server.Middleware.Apply: Clock is required")
	}
	router.Use(m.structuredLogging)
	router.Use(m.corsMiddleware)
}

// responseWriter wraps http.ResponseWriter to capture the final status
// code while preserving the optional interfaces every SSE/long-running
// handler depends on.
//
// Embedding http.ResponseWriter does NOT propagate the writer's other
// optional interfaces (Flusher, Hijacker, Pusher) to type assertions on
// *responseWriter — Go interface satisfaction looks at the wrapper's own
// method set. SSE handlers in this service do `w.(http.Flusher)`; without
// explicit pass-through methods every SSE response would 500 with
// "streaming not supported", silently breaking the agent-manager log
// stream consumer (see ErrSandboxNoExitInfo). Each method below delegates
// to the underlying writer when it actually supports the interface, and
// no-ops otherwise so middleware composition stays robust.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// structuredLogging records every served request as a single APIRequest
// log entry. Duration is sourced from m.Clock so deterministic tests can
// pin the wall-clock without monkey-patching time.Since.
func (m Middleware) structuredLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := m.Clock.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		ctx := logging.WithLogger(r.Context(), m.Logger)
		r = r.WithContext(ctx)

		next.ServeHTTP(wrapped, r)

		duration := schedule.Since(start)
		m.Logger.APIRequest(r.Method, r.RequestURI, wrapped.statusCode, float64(duration.Milliseconds()))
	})
}

// corsMiddleware enforces the configured CORS allowlist. Empty allowlist
// falls back to the dev UI port (read from m.UIPortEnv, default
// "UI_PORT"). Tests can set m.UIPortEnv to a sentinel and inject the
// expected value via os.Setenv to avoid leaking operator state.
func (m Middleware) corsMiddleware(next http.Handler) http.Handler {
	envName := m.UIPortEnv
	if envName == "" {
		envName = "UI_PORT"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowedOrigins := m.CORSAllowedOrigins
		if len(allowedOrigins) == 0 {
			uiPort := os.Getenv(envName)
			if uiPort != "" {
				allowedOrigins = []string{
					"http://localhost:" + uiPort,
					"http://127.0.0.1:" + uiPort,
				}
			}
		}

		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				originAllowed = true
				break
			}
		}

		if originAllowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
