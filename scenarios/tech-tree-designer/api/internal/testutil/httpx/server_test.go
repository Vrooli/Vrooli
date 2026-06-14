package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"tech-tree-designer/handlers/health"
	"tech-tree-designer/internal/clock"
	"tech-tree-designer/internal/server"
	"tech-tree-designer/internal/testutil/mocks"
)

// TestNewLiveServer_RoutesRegistered confirms the harness wires the
// production middleware + handler routes correctly. Hits the real
// /health endpoint through a real HTTP client and expects a JSON body
// with the canonical service identity.
func TestNewLiveServer_RoutesRegistered(t *testing.T) {
	srv := newHarnessServer(t, "harness-test")
	live := NewLiveServer(t, srv)

	resp, body := live.Do(t, http.MethodGet, "/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.StatusCode, string(body))
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if got["service"] != "harness-test" {
		t.Errorf("service = %v, want harness-test", got["service"])
	}
}

// TestLiveServer_DoReturnsBodyBytes is a smoke test for the Do
// convenience: bytes returned must match what the server actually
// wrote. Catches regressions where Do silently swallows or truncates
// the body (e.g. forgetting to read before Close).
func TestLiveServer_DoReturnsBodyBytes(t *testing.T) {
	srv := newHarnessServer(t, "body-test")
	live := NewLiveServer(t, srv)

	resp, body := live.Do(t, http.MethodGet, "/health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if len(body) == 0 {
		t.Fatal("Do returned empty body for /health")
	}
	if !strings.Contains(string(body), `"service"`) {
		t.Errorf("body missing service field: %s", string(body))
	}
}

// TestNewLiveServer_NormalisesPathPrefix proves the slash-prefix
// helper inside Do — callers can pass either "/health" or "health" and
// reach the same endpoint. Documenting via test guards against a
// future refactor that drops the normalisation.
func TestNewLiveServer_NormalisesPathPrefix(t *testing.T) {
	srv := newHarnessServer(t, "prefix-test")
	live := NewLiveServer(t, srv)

	resp, _ := live.Do(t, http.MethodGet, "health", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d (path without leading slash should still resolve)", resp.StatusCode)
	}
}

// newHarnessServer wires a server with just the health module so the
// harness tests have a real route to hit. The harness itself is what's
// under test; the module choice is incidental.
func newHarnessServer(t *testing.T, service string) *server.Server {
	t.Helper()
	mod := health.Module(&mocks.FakePinger{}, service, "0.0.1")
	return server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		mod,
	)
}

// recordingT spies on the failer surface NewLiveServer drives. Used to
// exercise the nil-server guard without failing the parent test (which
// is what an inverted t.Run subtest would do).
type recordingT struct {
	helperCalls  int
	cleanups     []func()
	fatalCalled  bool
	fatalMessage string
}

func (r *recordingT) Helper() { r.helperCalls++ }
func (r *recordingT) Cleanup(fn func()) {
	r.cleanups = append(r.cleanups, fn)
}

func (r *recordingT) Fatal(args ...any) {
	r.fatalCalled = true
	r.fatalMessage = fmt.Sprint(args...)
}

func TestNewLiveServer_NilServerFatal(t *testing.T) {
	r := &recordingT{}
	got := newLiveServer(r, nil)
	if !r.fatalCalled {
		t.Fatal("expected NewLiveServer(nil) to call t.Fatal")
	}
	if !strings.Contains(r.fatalMessage, "server is required") {
		t.Errorf("Fatal message %q should explain the failure", r.fatalMessage)
	}
	if got != nil {
		t.Errorf("NewLiveServer(nil) returned non-nil LiveServer: %+v", got)
	}
}
