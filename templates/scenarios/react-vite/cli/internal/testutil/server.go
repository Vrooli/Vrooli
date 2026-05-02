// Package testutil provides shared CLI test scaffolding.
//
// This package is test-only — production CLI code must not import it.
// The contract is enforced by no_prod_import_test.go (an AST guardrail
// mirroring the API side at api/internal/testutil/no_prod_import_test.go).
//
// What lives here:
//
//   - NewHTTPServer / NewAPIServer: real httptest.Server fakes, t-cleanup wired.
//   - WithAPIBase: sets the API_BASE_URL env var so cli-core's resolver picks
//     up the test server. Restored at end-of-test via t.Setenv.
//   - CaptureStdout: capture writes to os.Stdout for the duration of fn.
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
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// NewAPIServer wraps NewHTTPServer and additionally points the CLI at
// the test server via WithAPIBase. Use this when the test exercises a
// command that resolves the API base from env (the common case).
func NewAPIServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := NewHTTPServer(tb, handler)
	WithAPIBase(tb, server.URL)
	return server
}

// NewHTTPServer constructs a real httptest.Server (not a Recorder) and
// registers Close as a t.Cleanup. Real-socket transport is a deliberate
// choice: Recorder fakes http.Flusher and http.Hijacker, masking SSE
// flush and websocket-upgrade bugs. The cost is microseconds; the
// payoff is catching a class of bug that has shipped to production
// from Recorder-only tests before.
func NewHTTPServer(tb testing.TB, handler http.Handler) *httptest.Server {
	tb.Helper()
	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)
	return server
}

// WithAPIBase sets API_BASE_URL for the test (auto-restored on cleanup).
// API_BASE_URL is the env var declared in cli/app.go's ExtraAPIEnvVars,
// so cli-core's APIBase resolver picks it up. If a scenario adds another
// API env var (e.g. an upper-snake-case variant of the scenario id),
// it can wrap or replace this helper.
func WithAPIBase(tb testing.TB, apiBase string) {
	tb.Helper()
	tb.Setenv("API_BASE_URL", apiBase)
}

// failer is the minimum surface CaptureStdout needs from testing.TB.
// Carved out so this package's own tests can spy on Fatalf with a
// recording stub — testing.TB cannot be implemented externally because
// of its private() gate.
type failer interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}

// CaptureStdout redirects os.Stdout to a pipe for the duration of fn,
// then returns everything written. Useful for asserting on the human
// output produced by cli-core's RenderOperationalReport et al.
//
// The original stdout is restored via t.Cleanup; tests that fail
// mid-capture still get their stdout back for the test runner.
func CaptureStdout(tb testing.TB, fn func() error) string {
	tb.Helper()
	return captureStdout(tb, fn)
}

func captureStdout(tb failer, fn func() error) string {
	tb.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		tb.Fatalf("pipe: %v", err)
		return ""
	}
	os.Stdout = w
	tb.Cleanup(func() { os.Stdout = orig })

	runErr := fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		tb.Fatalf("copy stdout: %v", err)
		return ""
	}
	if runErr != nil {
		tb.Fatalf("command returned error: %v", runErr)
		return ""
	}
	return buf.String()
}
