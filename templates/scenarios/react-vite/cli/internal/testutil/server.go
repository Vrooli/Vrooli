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
//   - MustMarshalProto: serialise a proto.Message as the wire shape the API
//     would emit (snake_case via UseProtoNames=true), or fail the test.
//   - NewTestApp: wires a *cliapp.ScenarioApp pointed at a test server so
//     domain handler tests can drive core.Get / core.Request through one
//     line of construction. See app.go.
//
// Patterns to extend (not implemented yet — add when the first real consumer
// exists):
//
//   - A typed APIClientFake that records request/response pairs for tests
//     that want to assert on the request shape, not just the response.
//   - A canned-response helper for table-driven endpoint tests.
//
// Resist over-generalising. Add helpers when the third caller appears.
//
// # Failer-seam asymmetry
//
// CaptureStdout exposes a `failer` interface so its fail paths can be
// exercised in this package's own tests. NewHTTPServer / NewAPIServer /
// WithAPIBase don't — they call into testing.TB directly. The
// asymmetry is intentional: those helpers can only fail in the
// environment (port exhaustion, etc.), not in shapes a unit test would
// usefully spy on. CaptureStdout's failure modes (pipe creation, IO
// copy, command error) are all worth pinning. Apply the same judgment
// when adding new helpers — add a failer seam only when the failure
// mode rewards in-process verification.
package testutil

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

// MustMarshalProto serialises msg as the wire shape the API would emit
// (snake_case via UseProtoNames=true), or fails the test. The CLI
// counterpart to assertx.MustUnmarshalProto on the API side: every CLI
// handler test that needs to feed a fake response should reach for
// this rather than hand-writing JSON literals. Hand-rolled JSON drifts
// silently when the proto schema grows or renames fields; a typed
// proto.Marshal call breaks at compile time, surfacing the schema
// change at the test that depends on it.
//
// Canonical usage in a fake API server:
//
//	body := testutil.MustMarshalProto(t, &notesv1.ListNotesResponse{
//	    Notes: []*notesv1.Note{{Id: "a", Title: "first"}},
//	})
//	w.Write(body)
//
// UseProtoNames=true mirrors the production handler's
// `(protojson.MarshalOptions{UseProtoNames: true}).Marshal(...)` so
// the wire shape the test feeds matches what production sends —
// including snake_case keys like `created_at` that the CLI's
// protojson.Unmarshal accepts on the read side.
func MustMarshalProto(tb testing.TB, msg proto.Message) []byte {
	tb.Helper()
	body, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(msg)
	if err != nil {
		tb.Fatalf("MustMarshalProto: %v", err)
		return nil
	}
	return body
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
