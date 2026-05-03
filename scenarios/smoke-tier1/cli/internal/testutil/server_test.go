package testutil

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
}

func TestNewHTTPServer_ServesAndCleansUp(t *testing.T) {
	server := NewHTTPServer(t, okHandler())
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", server.URL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
}

func TestNewAPIServer_SetsAPIBaseURL(t *testing.T) {
	server := NewAPIServer(t, okHandler())
	if got := os.Getenv("API_BASE_URL"); got != server.URL {
		t.Fatalf("API_BASE_URL = %q, want %q", got, server.URL)
	}
}

// TestNewAPIServer_RestoresEnvOnCleanup proves t.Setenv's auto-restore
// is what's actually cleaning up the env var. The outer test sets a
// sentinel; the inner subtest invokes NewAPIServer which clobbers it;
// when the inner exits the sentinel must be back. If a future refactor
// switches to bare os.Setenv, this test fails.
func TestNewAPIServer_RestoresEnvOnCleanup(t *testing.T) {
	const sentinel = "outer-sentinel-value"
	t.Setenv("API_BASE_URL", sentinel)

	t.Run("inner", func(tt *testing.T) {
		_ = NewAPIServer(tt, okHandler())
		if got := os.Getenv("API_BASE_URL"); got == sentinel {
			tt.Fatal("inner subtest did not override API_BASE_URL")
		}
	})

	if got := os.Getenv("API_BASE_URL"); got != sentinel {
		t.Fatalf("API_BASE_URL after inner cleanup = %q, want %q", got, sentinel)
	}
}

func TestCaptureStdout_CapturesWrites(t *testing.T) {
	got := CaptureStdout(t, func() error {
		fmt.Print("hello")
		return nil
	})
	if got != "hello" {
		t.Fatalf("captured = %q, want hello", got)
	}
}

// TestCaptureStdout_RestoresStdoutViaCleanup guards against the
// os.Stdout swap leaking past the capture window. Restoration happens
// via tb.Cleanup, so the test drives a recording stub and asserts the
// cleanup callback rewinds os.Stdout. Without this wiring, every
// subsequent test would print into a closed pipe.
func TestCaptureStdout_RestoresStdoutViaCleanup(t *testing.T) {
	original := os.Stdout
	r := &recordingTB{}
	_ = captureStdout(r, func() error {
		if os.Stdout == original {
			return errors.New("stdout not swapped during capture")
		}
		return nil
	})
	if len(r.cleanups) != 1 {
		t.Fatalf("expected 1 Cleanup registered, got %d", len(r.cleanups))
	}
	r.runCleanups()
	if os.Stdout != original {
		t.Fatalf("after cleanup, os.Stdout = %p, want %p", os.Stdout, original)
	}
}

// recordingTB spies on the failer interface CaptureStdout drives. Used
// to verify Fatalf paths without failing the parent test.
type recordingTB struct {
	helperCalls  int
	cleanups     []func()
	fatalCalled  bool
	fatalMessage string
}

func (r *recordingTB) Helper() { r.helperCalls++ }
func (r *recordingTB) Cleanup(fn func()) {
	r.cleanups = append(r.cleanups, fn)
}
func (r *recordingTB) Fatalf(format string, args ...any) {
	r.fatalCalled = true
	r.fatalMessage = format
}
func (r *recordingTB) runCleanups() {
	for i := len(r.cleanups) - 1; i >= 0; i-- {
		r.cleanups[i]()
	}
	r.cleanups = nil
}

func TestCaptureStdout_PropagatesErrViaFatalf(t *testing.T) {
	r := &recordingTB{}
	got := captureStdout(r, func() error {
		return errors.New("boom")
	})
	r.runCleanups()
	if !r.fatalCalled {
		t.Fatal("expected captureStdout to call Fatalf when fn returns an error")
	}
	if !strings.Contains(r.fatalMessage, "command returned error") {
		t.Errorf("Fatalf format = %q, want to mention 'command returned error'", r.fatalMessage)
	}
	if got != "" {
		t.Errorf("captureStdout returned %q on error path; expected empty", got)
	}
}

// TestCaptureStdout_PreservesMultilineOutput is a guard against
// truncation — long output spanning many lines must round-trip
// verbatim. Catches a regression where the capture pipe was closed
// before draining.
func TestCaptureStdout_PreservesMultilineOutput(t *testing.T) {
	want := strings.Repeat("line\n", 50)
	got := CaptureStdout(t, func() error {
		fmt.Print(want)
		return nil
	})
	if got != want {
		t.Fatalf("captured output mismatch (len got=%d want=%d)", len(got), len(want))
	}
}
