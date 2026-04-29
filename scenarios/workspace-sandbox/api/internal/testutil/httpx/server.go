package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/logging"
	"workspace-sandbox/internal/metrics"
	"workspace-sandbox/internal/server"
)

// LiveServer is the test-side equivalent of main.go's HTTP boot path:
// the production middleware stack wraps the production handler routes,
// behind an httptest.Server. Tests obtain an instance via NewLiveServer
// and exercise it through a real http.Client — just like agent-manager
// does.
//
// Why not httptest.ResponseRecorder + handler.Method(rr, req)?
// httptest.ResponseRecorder natively implements http.Flusher and
// http.Hijacker, so a wrapper that drops those interfaces still passes
// recorder-based tests. The 2026-04-28 SSE flusher bug shipped because
// of exactly that gap. LiveServer eliminates the gap by running over a
// real TCP socket through the real middleware chain.
type LiveServer struct {
	// Server is the underlying httptest.Server. Tests can reach the
	// raw URL via Server.URL or use the wrappers below.
	Server *httptest.Server

	// Client is a sane default *http.Client that follows redirects
	// transparently and does not buffer SSE streams. Tests can replace
	// it for special cases (e.g., disabling redirects).
	Client *http.Client

	// LogBuffer captures every structured log line emitted by the
	// middleware. Useful for asserting that the API request actually
	// flowed through the production middleware chain.
	LogBuffer *bytes.Buffer
}

// LiveServerOption customizes the harness. Most tests use the defaults.
type LiveServerOption func(*liveServerCfg)

type liveServerCfg struct {
	clock              clock.Clock
	corsAllowedOrigins []string
	uiPortEnv          string
	logger             *logging.Logger
	logBuf             *bytes.Buffer
	metricsCollector   *metrics.Collector
	registerExtra      []func(*mux.Router)
}

// WithClock injects a clock into the middleware (Round 4 Phase 2 seam).
// Defaults to clock.System{} so most tests don't have to think about it.
func WithClock(c clock.Clock) LiveServerOption {
	return func(cfg *liveServerCfg) { cfg.clock = c }
}

// WithCORSAllowedOrigins seeds the strict allowlist used by the CORS
// middleware. Empty means "fall back to UI port env" — the production
// default; rarely useful in tests.
func WithCORSAllowedOrigins(origins []string) LiveServerOption {
	return func(cfg *liveServerCfg) {
		cfg.corsAllowedOrigins = append([]string{}, origins...)
	}
}

// WithUIPortEnv overrides the env-var name used by the CORS dev fallback.
// Tests use this to avoid leaking the operator's UI_PORT into the harness.
func WithUIPortEnv(name string) LiveServerOption {
	return func(cfg *liveServerCfg) { cfg.uiPortEnv = name }
}

// WithMetricsCollector wires a metrics collector through to handlers.
// Optional; handlers that do not use it ignore the field.
func WithMetricsCollector(c *metrics.Collector) LiveServerOption {
	return func(cfg *liveServerCfg) { cfg.metricsCollector = c }
}

// WithExtraRoutes lets a test register additional routes on the router
// after the production handler routes are in place. Useful when a test
// needs a probe endpoint (e.g., a forced 500 to exercise the recovery
// handler) without touching the handlers package.
func WithExtraRoutes(fn func(*mux.Router)) LiveServerOption {
	return func(cfg *liveServerCfg) {
		cfg.registerExtra = append(cfg.registerExtra, fn)
	}
}

// NewLiveServer spins up a live HTTP server wired with the production
// middleware stack and the production handler routes. The server is
// torn down via t.Cleanup; tests do not need to call Close.
//
// The returned LiveServer carries a default *http.Client and a buffer
// containing every middleware log line — enough surface for assertions
// on status codes, response bodies, headers, SSE frames, and the fact
// that the middleware itself actually ran.
func NewLiveServer(t *testing.T, h *handlers.Handlers, opts ...LiveServerOption) *LiveServer {
	t.Helper()
	if h == nil {
		t.Fatal("httpx.NewLiveServer: Handlers is required")
	}

	cfg := &liveServerCfg{
		clock:     h.Clock,
		uiPortEnv: "WORKSPACE_SANDBOX_TESTUTIL_UI_PORT",
		logBuf:    &bytes.Buffer{},
	}
	if cfg.clock == nil {
		cfg.clock = clock.System{}
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logger == nil {
		cfg.logger = logging.New("workspace-sandbox-testutil",
			logging.WithOutput(cfg.logBuf),
			logging.WithClock(cfg.clock),
		)
	}

	router := mux.NewRouter()
	server.Middleware{
		Logger:             cfg.logger,
		Clock:              cfg.clock,
		CORSAllowedOrigins: cfg.corsAllowedOrigins,
		UIPortEnv:          cfg.uiPortEnv,
	}.Apply(router)

	// Register the production handler routes. Tool-discovery routes are
	// intentionally not wired here; handler tests don't use them, and
	// they would require additional fakes (toolregistry, toolexecution).
	h.RegisterRoutes(router, cfg.metricsCollector)

	for _, fn := range cfg.registerExtra {
		fn(router)
	}

	srv := httptest.NewServer(gorillahandlers.RecoveryHandler()(router))
	t.Cleanup(srv.Close)

	return &LiveServer{
		Server:    srv,
		Client:    srv.Client(),
		LogBuffer: cfg.logBuf,
	}
}

// URL returns the absolute URL for `path`. Path may be absolute
// ("/api/v1/health") or already-resolved (returned untouched if it
// parses as absolute).
func (l *LiveServer) URL(path string) string {
	if u, err := url.Parse(path); err == nil && u.IsAbs() {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return l.Server.URL + path
}

// Do issues an HTTP request through the live server. Body may be nil
// for GET-style requests. Headers can be supplied via opts; the helper
// always closes the body once the response has been fully consumed.
//
// Tests almost always want this over Server.Client.Do because it handles
// URL resolution, body reading, and cleanup uniformly.
func (l *LiveServer) Do(t *testing.T, method, path string, body io.Reader, opts ...RequestOption) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, l.URL(path), body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for _, opt := range opts {
		opt(req)
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

// DoJSON issues a request whose body is the JSON encoding of `payload`.
// For convenience: tests rarely care about manually wiring the
// Content-Type header.
func (l *LiveServer) DoJSON(t *testing.T, method, path, payload string, opts ...RequestOption) (*http.Response, []byte) {
	t.Helper()
	r := strings.NewReader(payload)
	opts = append(opts, WithHeader("Content-Type", "application/json"))
	return l.Do(t, method, path, r, opts...)
}

// RequestOption mutates a request before it is dispatched. Use With*
// helpers; do not return new requests.
type RequestOption func(*http.Request)

// WithHeader sets a single header on the outgoing request.
func WithHeader(name, value string) RequestOption {
	return func(r *http.Request) { r.Header.Set(name, value) }
}
