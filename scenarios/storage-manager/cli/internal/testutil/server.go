// Package testutil provides shared CLI test scaffolding.
//
// This package is test-only — production CLI code must not import it. The
// contract is enforced by no_prod_import_test.go (an AST guardrail mirroring
// the API side at api/internal/testutil/no_prod_import_test.go).
//
// What lives here:
//
//   - NewHTTPServer / NewAPIServer: real httptest.Server fakes, t-cleanup wired.
//   - WithAPIBase: sets the API_BASE_URL env var so cli-core's resolver picks
//     up the test server. Restored at end-of-test via t.Setenv.
//   - NewTestApp: wires a *cliapp.ScenarioApp pointed at a test server so
//     domain handler tests can drive core.Get / core.Request through one
//     line of construction. See app.go.
//
// Scenario-agnostic helpers live in cli-core/cliapptest:
//
//   - cliapptest.MustMarshalProto: serialise a proto.Message as the wire
//     shape the API would emit (UseProtoNames=true).
//   - cliapptest.CaptureStdout: capture writes to os.Stdout for the duration
//     of fn.
//   - cliapptest.NewTestRunContext / NewTestRunContextFromArgs / NewCapturedRunContext:
//     test-only RunContext constructors.
//
// Patterns to extend (not implemented yet — add when the first real consumer
// exists):
//
//   - A typed APIClientFake that records request/response pairs for tests
//     that want to assert on the request shape, not just the response.
//   - A canned-response helper for table-driven endpoint tests.
//
// Resist over-generalising. Add helpers when the third caller appears.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewAPIServer wraps NewHTTPServer and additionally points the CLI at the
// test server via WithAPIBase. Use this when the test exercises a command
// that resolves the API base from env (the common case).
func NewAPIServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := NewHTTPServer(tb, handler)
	WithAPIBase(tb, server.URL)
	return server
}

// NewHTTPServer constructs a real httptest.Server (not a Recorder) and
// registers Close as a t.Cleanup. Real-socket transport is a deliberate
// choice: Recorder fakes http.Flusher and http.Hijacker, masking SSE flush
// and websocket-upgrade bugs. The cost is microseconds; the payoff is
// catching a class of bug that has shipped to production from Recorder-only
// tests before.
func NewHTTPServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)
	return server
}

// WithAPIBase sets API_BASE_URL for the test (auto-restored on cleanup).
// API_BASE_URL is the env var declared in cli/app.go's ExtraAPIEnvVars, so
// cli-core's APIBase resolver picks it up. If a scenario adds another API
// env var (e.g. an upper-snake-case variant of the scenario id), it can wrap
// or replace this helper.
func WithAPIBase(tb testing.TB, apiBase string) {
	tb.Helper()
	tb.Setenv("API_BASE_URL", apiBase)
}
