//go:build e2e && !windows

// Package main e2e gate. Build-tag-isolated so it never runs under
// `go test ./...` — the canonical entry point is `go test -tags=e2e .`,
// invoked from the .github/workflows/test.yml `api-e2e` step and
// available locally for hand-runs.
//
// # What this test catches that handler tests don't
//
// Handler tests wire `*server.Server` from the inside, with fake
// dependencies and `httpx.NewLiveServer` over `httptest.Server`. They
// verify the HTTP surface of the server, but they never exercise:
//
//   - main.go's `server.Deps` wiring (a forgotten field surfaces as a
//     handler nil-deref at boot, not a compile error)
//   - api-core's `preflight.Run` flow (the binary re-exec dance after a
//     stale-source rebuild)
//   - api-core's `apiserver.Run` listener config (port resolution, the
//     SIGTERM cleanup hook)
//   - storage.NewResolver path selection on the actual host
//   - schema bootstrap against a real on-disk SQLite file
//
// This test boots the real binary, polls /health on a real socket,
// then sends SIGTERM and waits for clean exit. If any of the above
// regress, this test fails — and crucially, fails *before* the first
// scenario adopts the broken template.
package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestE2E_BinaryBootsAndServesHealth(t *testing.T) {
	binary := buildBinary(t)
	port := pickFreePort(t)
	dbPath := filepath.Join(t.TempDir(), "e2e.db")

	cmd := exec.Command(binary)
	cmd.Env = append(os.Environ(),
		// api-core's apiserver.Config reads API_PORT (see
		// packages/api-core/server/server.go::config.getenv).
		"API_PORT="+strconv.Itoa(port),
		"SQLITE_PATH="+dbPath,
		// Pass the preflight lifecycle guard. Without this the binary
		// errors out with "must be run through the Vrooli lifecycle
		// system" before opening any listener. preflight.LifecycleManagedEnvVar
		// is the canonical signal that the parent (in production: the
		// lifecycle daemon; in this test: us) is taking responsibility
		// for env/port allocation.
		"VROOLI_LIFECYCLE_MANAGED=true",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start binary: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGKILL)
			_, _ = cmd.Process.Wait()
		}
	})

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	if err := waitForHealth(url, 10*time.Second); err != nil {
		t.Fatalf("binary failed to serve /health within 10s: %v", err)
	}

	// Verify the body is the proto wire shape, not a generic 200 from
	// some other listener that happened to grab the port.
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health = %d, want 200; body=%s", resp.StatusCode, body)
	}
	bodyStr := string(body)
	for _, want := range []string{`"status"`, `"healthy"`} {
		if !strings.Contains(bodyStr, want) {
			t.Fatalf("GET /health body lacks %q; got %s", want, bodyStr)
		}
	}

	// Graceful shutdown: SIGTERM and assert the process exits cleanly.
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			// SIGTERM-induced exit may surface as a non-nil error
			// depending on api-core's signal handling. Accept the
			// signal-terminated exit class but flag any unexpected
			// failure mode.
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == -1 {
					return // terminated by signal; expected
				}
			}
			t.Fatalf("binary exited with unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("binary did not exit within 5s of SIGTERM (cleanup hook hung?)")
	}
}

// buildBinary compiles the api package into a test-scoped temp file.
// Uses `go build .` from the test's working dir (api/). The build
// failing is itself a useful signal — main.go imports must resolve.
func buildBinary(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "api-bin")
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build .: %v", err)
	}
	return out
}

// pickFreePort asks the kernel for an unused TCP port, then closes the
// listener and returns the port number. There is a small TOCTOU window
// (port could be claimed before the binary opens it), but the
// alternative — a hardcoded port — would flap whenever CI parallelism
// collided.
func pickFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// waitForHealth polls /health until it returns 200 or the deadline
// passes. The poll interval is intentionally short (50ms) — startup
// latency for the bare API in CI is on the order of 200ms.
func waitForHealth(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s; last error: %v", timeout, lastErr)
}
