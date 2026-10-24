// Package boottest checks a scenario's real binary wiring in an isolated storage
// tree. It simulates the lifecycle environment; it does not validate control-plane
// supervision, stale-source rebuilds, or domain schema contents.
package boottest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// Config declares the scenario-specific part of a binary boot check.
type Config struct {
	Service string
	// StartupTimeout defaults to ten seconds. Override only for slower startup wiring.
	StartupTimeout time.Duration
	// Env supplies startup inputs such as BRIDGE_BOOTSTRAP_SCRIPT. The harness
	// owns the listener and storage overrides, which cannot be replaced here.
	Env []string
	// WindowsTermination enables boot coverage on Windows using forced process
	// termination. It does not claim graceful shutdown coverage on that platform.
	WindowsTermination bool
}

// Run builds the API in the test's working directory, verifies healthy JSON for
// the expected service, and requires a clean exit after SIGTERM on Unix.
func Run(t *testing.T, cfg Config) {
	t.Helper()
	if cfg.Service == "" {
		t.Fatal("boottest: expected service is required")
	}
	if runtime.GOOS == "windows" && !cfg.WindowsTermination {
		repocontracttest.SkipPlatform(t, "binary boot check requires SIGTERM; Windows termination coverage is not enabled")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "api-bin")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	var buildLogs logTail
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Stdout, build.Stderr = &buildLogs, &buildLogs
	build.WaitDelay = time.Second
	if err := build.Run(); err != nil {
		t.Fatalf("build API: %v\n%s", err, buildLogs.String())
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate API port: %v", err)
	}
	addr := listener.Addr().String()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release API port: %v", err)
	}
	// The kernel allocation must be released before the binary binds. Identity
	// validation and process-exit detection prevent a competing listener passing.
	cmd := exec.Command(binary)
	cmd.Env = environment(os.Environ(), cfg.Env, t.TempDir(), port)
	startup := cfg.StartupTimeout
	if startup <= 0 {
		startup = 10 * time.Second
	}
	if err := check(ctx, cmd, "http://"+addr+"/health", cfg.Service, startup, 15*time.Second); err != nil {
		t.Fatal(err)
	}
}

func environment(parent, extra []string, root string, port int) []string {
	env := envkit.WithOverlay(parent, envkit.ForeignScenario, extra)
	return envkit.WithOverlay(env, envkit.SameScenario, envkit.Env{
		fmt.Sprintf("API_PORT=%d", port), "API_BIND_ADDRESS=127.0.0.1",
		"VROOLI_STORAGE_ROOT=" + root, "VROOLI_STORAGE_NAMESPACE=",
		"VROOLI_LIFECYCLE_MANAGED=true",
	})
}

func check(ctx context.Context, cmd *exec.Cmd, url, service string, startup, shutdown time.Duration) (result error) {
	var logs logTail
	cmd.Stdout, cmd.Stderr = &logs, &logs
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start API: %w", err)
	}
	done := make(chan struct{})
	var exitErr error
	go func() {
		exitErr = cmd.Wait()
		close(done)
	}()
	defer func() {
		select {
		case <-done:
		default:
			_ = cmd.Process.Kill()
			<-done
		}
		if result != nil {
			result = fmt.Errorf("%w\nAPI output (last 32 KiB):\n%s", result, logs.String())
		}
	}()
	readyCtx, cancel := context.WithTimeout(ctx, startup)
	err := waitHealthy(readyCtx, url, service, done)
	cancel()
	if err != nil {
		select {
		case <-done:
			return fmt.Errorf("API exited before health verification: %v (%w)", exitErr, err)
		default:
			return err
		}
	}
	select {
	case <-done:
		return fmt.Errorf("API exited before shutdown request: %v", exitErr)
	default:
	}
	if runtime.GOOS == "windows" {
		err = cmd.Process.Kill()
	} else {
		err = cmd.Process.Signal(syscall.SIGTERM)
	}
	if err != nil {
		return fmt.Errorf("terminate API: %w", err)
	}
	shutdownCtx, stop := context.WithTimeout(ctx, shutdown)
	defer stop()
	select {
	case <-done:
		if exitErr != nil && runtime.GOOS != "windows" {
			return fmt.Errorf("API did not exit cleanly after SIGTERM: %w", exitErr)
		}
		return nil
	case <-shutdownCtx.Done():
		return fmt.Errorf("API shutdown deadline: %w", shutdownCtx.Err())
	}
}

func waitHealthy(ctx context.Context, url, service string, done <-chan struct{}) error {
	client := &http.Client{Timeout: time.Second}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	var lastErr error
	for {
		if err := health(ctx, client, url, service); err == nil {
			return nil
		} else {
			// Preserve the last server response when the overall deadline expires.
			if ctx.Err() == nil || lastErr == nil {
				lastErr = err
			}
		}
		select {
		case <-done:
			return fmt.Errorf("process exited; last health error: %v", lastErr)
		case <-ctx.Done():
			return fmt.Errorf("health deadline: %w; last health error: %v", ctx.Err(), lastErr)
		case <-tick.C:
		}
	}
}

func health(ctx context.Context, client *http.Client, url, service string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health HTTP status %d", resp.StatusCode)
	}
	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return err
	}
	if len(body) > limit {
		return fmt.Errorf("health response exceeds 1 MiB")
	}
	var response struct {
		Status  string `json:"status"`
		Service string `json:"service"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode health JSON: %w", err)
	}
	if response.Status != "healthy" || response.Service != service {
		return fmt.Errorf("health status=%q service=%q; want healthy service=%q", response.Status, response.Service, service)
	}
	return nil
}

// logTail bounds diagnostics even when a failing binary writes continuously.
type logTail struct {
	mu   sync.Mutex
	data []byte
}

func (b *logTail) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	const limit = 32 << 10
	n := len(p)
	if n >= limit {
		b.data = append(b.data[:0], p[n-limit:]...)
	} else {
		if excess := len(b.data) + n - limit; excess > 0 {
			b.data = b.data[excess:]
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}

func (b *logTail) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
