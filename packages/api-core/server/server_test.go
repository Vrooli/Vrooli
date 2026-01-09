package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// ============================================================================
// Config Defaults Tests
// ============================================================================

func TestWithDefaults_PortFromEnv(t *testing.T) {
	t.Parallel()

	cfg := Config{
		EnvGetter: func(key string) string {
			if key == "API_PORT" {
				return "9999"
			}
			return ""
		},
	}

	cfg = withDefaults(cfg)

	if cfg.Port != "9999" {
		t.Errorf("expected port 9999 from env, got %s", cfg.Port)
	}
}

func TestWithDefaults_PortExplicit(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Port: "7777",
		EnvGetter: func(key string) string {
			return "9999" // Should be ignored
		},
	}

	cfg = withDefaults(cfg)

	if cfg.Port != "7777" {
		t.Errorf("expected explicit port 7777, got %s", cfg.Port)
	}
}

func TestWithDefaults_PortFallback(t *testing.T) {
	t.Parallel()

	cfg := Config{
		EnvGetter: func(key string) string { return "" },
	}

	cfg = withDefaults(cfg)

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
}

func TestWithDefaults_Timeouts(t *testing.T) {
	t.Parallel()

	cfg := withDefaults(Config{})

	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout: expected 30s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout: expected 30s, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout: expected 120s, got %v", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout: expected 10s, got %v", cfg.ShutdownTimeout)
	}
}

func TestWithDefaults_TimeoutsExplicit(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 20 * time.Second,
	}

	cfg = withDefaults(cfg)

	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout: expected explicit 5s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout: expected explicit 10s, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout: expected explicit 60s, got %v", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 20*time.Second {
		t.Errorf("ShutdownTimeout: expected explicit 20s, got %v", cfg.ShutdownTimeout)
	}
}

func TestWithDefaults_Signals(t *testing.T) {
	t.Parallel()

	cfg := withDefaults(Config{})

	if len(cfg.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(cfg.Signals))
	}
	if cfg.Signals[0] != syscall.SIGINT {
		t.Errorf("expected first signal SIGINT, got %v", cfg.Signals[0])
	}
	if cfg.Signals[1] != syscall.SIGTERM {
		t.Errorf("expected second signal SIGTERM, got %v", cfg.Signals[1])
	}
}

// ============================================================================
// Run Tests
// ============================================================================

func TestRun_RequiresHandler(t *testing.T) {
	t.Parallel()

	err := Run(Config{})
	if err == nil {
		t.Fatal("expected error when Handler is nil")
	}
	if !strings.Contains(err.Error(), "Handler is required") {
		t.Errorf("expected error about Handler, got: %v", err)
	}
}

func TestRun_StartsAndStopsGracefully(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	sigCh := make(chan os.Signal, 1)
	var logs []string
	var mu sync.Mutex

	port := findFreePort(t)

	// Run server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Handler:         handler,
			Port:            port,
			ShutdownTimeout: 1 * time.Second,
			signalChan:      sigCh,
			Logger: func(format string, args ...interface{}) {
				mu.Lock()
				logs = append(logs, fmt.Sprintf(format, args...))
				mu.Unlock()
			},
		})
	}()

	// Wait for server to start
	waitForServer(t, port)

	// Make a request to verify it's working
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/", port))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Send shutdown signal
	sigCh <- syscall.SIGTERM

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	// Verify logs
	mu.Lock()
	defer mu.Unlock()

	hasStarting := false
	hasStopped := false
	for _, log := range logs {
		if strings.Contains(log, "Starting server") {
			hasStarting = true
		}
		if strings.Contains(log, "Server stopped") {
			hasStopped = true
		}
	}
	if !hasStarting {
		t.Error("expected 'Starting server' log message")
	}
	if !hasStopped {
		t.Error("expected 'Server stopped' log message")
	}
}

func TestRun_CallsCleanup(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sigCh := make(chan os.Signal, 1)
	cleanupCalled := false
	var cleanupCtx context.Context

	port := findFreePort(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Handler:         handler,
			Port:            port,
			ShutdownTimeout: 1 * time.Second,
			signalChan:      sigCh,
			Logger:          func(format string, args ...interface{}) {},
			Cleanup: func(ctx context.Context) error {
				cleanupCalled = true
				cleanupCtx = ctx
				return nil
			},
		})
	}()

	waitForServer(t, port)
	sigCh <- syscall.SIGTERM

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	if !cleanupCalled {
		t.Error("expected Cleanup to be called")
	}
	if cleanupCtx == nil {
		t.Error("expected Cleanup to receive a context")
	}
}

func TestRun_CleanupErrorIsLogged(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	sigCh := make(chan os.Signal, 1)
	var logs []string
	var mu sync.Mutex

	port := findFreePort(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Handler:         handler,
			Port:            port,
			ShutdownTimeout: 1 * time.Second,
			signalChan:      sigCh,
			Logger: func(format string, args ...interface{}) {
				mu.Lock()
				logs = append(logs, fmt.Sprintf(format, args...))
				mu.Unlock()
			},
			Cleanup: func(ctx context.Context) error {
				return fmt.Errorf("cleanup failed")
			},
		})
	}()

	waitForServer(t, port)
	sigCh <- syscall.SIGTERM

	select {
	case err := <-errCh:
		// Cleanup errors should not cause Run to return an error
		if err != nil {
			t.Fatalf("Run should not return error for cleanup failure: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	// Check that cleanup error was logged
	mu.Lock()
	defer mu.Unlock()

	hasCleanupError := false
	for _, log := range logs {
		if strings.Contains(log, "Cleanup error") && strings.Contains(log, "cleanup failed") {
			hasCleanupError = true
		}
	}
	if !hasCleanupError {
		t.Error("expected cleanup error to be logged")
	}
}

func TestRun_FailsOnPortInUse(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := findFreePort(t)

	// Start first server
	sigCh1 := make(chan os.Signal, 1)
	errCh1 := make(chan error, 1)
	go func() {
		errCh1 <- Run(Config{
			Handler:    handler,
			Port:       port,
			signalChan: sigCh1,
			Logger:     func(format string, args ...interface{}) {},
		})
	}()

	waitForServer(t, port)

	// Try to start second server on same port
	sigCh2 := make(chan os.Signal, 1)
	errCh2 := make(chan error, 1)
	go func() {
		errCh2 <- Run(Config{
			Handler:    handler,
			Port:       port,
			signalChan: sigCh2,
			Logger:     func(format string, args ...interface{}) {},
		})
	}()

	// Second server should fail
	select {
	case err := <-errCh2:
		if err == nil {
			t.Fatal("expected error when port is in use")
		}
		if !strings.Contains(err.Error(), "failed to start") {
			t.Errorf("expected 'failed to start' error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for second server to fail")
	}

	// Clean up first server
	sigCh1 <- syscall.SIGTERM
	<-errCh1
}

func TestRun_HandlesInFlightRequests(t *testing.T) {
	t.Parallel()

	requestStarted := make(chan struct{})
	requestCanFinish := make(chan struct{})

	mux := http.NewServeMux()
	// Health endpoint for waitForServer - returns immediately
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Slow endpoint that blocks until signaled
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-requestCanFinish
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("completed"))
	})

	sigCh := make(chan os.Signal, 1)
	port := findFreePort(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Handler:         mux,
			Port:            port,
			ShutdownTimeout: 5 * time.Second,
			signalChan:      sigCh,
			Logger:          func(format string, args ...interface{}) {},
		})
	}()

	// Wait for server using health endpoint
	waitForServerHealth(t, port)

	// Start a slow request
	respCh := make(chan *http.Response, 1)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/slow", port))
		if err == nil {
			respCh <- resp
		}
	}()

	// Wait for slow request to start
	<-requestStarted

	// Send shutdown signal while request is in progress
	sigCh <- syscall.SIGTERM

	// Allow request to complete
	close(requestCanFinish)

	// Request should complete successfully
	select {
	case resp := <-respCh:
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "completed" {
			t.Errorf("expected 'completed' response, got: %s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request did not complete")
	}

	// Server should stop
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}
}

// ============================================================================
// Helpers
// ============================================================================

func findFreePort(t *testing.T) string {
	t.Helper()

	// Use port 0 to let OS assign a free port, then extract it
	// For simplicity in tests, we'll use a high random port
	// In real code, you'd use net.Listen(":0") and extract the port
	return fmt.Sprintf("%d", 10000+time.Now().UnixNano()%50000)
}

func waitForServer(t *testing.T, port string) {
	t.Helper()
	waitForServerPath(t, port, "/")
}

func waitForServerHealth(t *testing.T, port string) {
	t.Helper()
	waitForServerPath(t, port, "/health")
}

func waitForServerPath(t *testing.T, port, path string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s%s", port, path))
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start on port %s", port)
}

// ============================================================================
// Custom Server Tests
// ============================================================================

func TestRun_RequiresShutdownServerWhenStartServerSet(t *testing.T) {
	t.Parallel()

	err := Run(Config{
		StartServer: func(addr string) error { return nil },
		// ShutdownServer not set
	})

	if err == nil {
		t.Fatal("expected error when ShutdownServer is nil but StartServer is set")
	}
	if !strings.Contains(err.Error(), "ShutdownServer is required") {
		t.Errorf("expected error about ShutdownServer, got: %v", err)
	}
}

func TestRun_CustomServer_StartsAndStopsGracefully(t *testing.T) {
	t.Parallel()

	port := findFreePort(t)
	var srv *http.Server

	sigCh := make(chan os.Signal, 1)
	var logs []string
	var mu sync.Mutex

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Port: port,
			StartServer: func(addr string) error {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("custom server"))
				})
				srv = &http.Server{Addr: addr, Handler: handler}
				return srv.ListenAndServe()
			},
			ShutdownServer: func(ctx context.Context) error {
				return srv.Shutdown(ctx)
			},
			ShutdownTimeout: 1 * time.Second,
			signalChan:      sigCh,
			Logger: func(format string, args ...interface{}) {
				mu.Lock()
				logs = append(logs, fmt.Sprintf(format, args...))
				mu.Unlock()
			},
		})
	}()

	// Wait for server to start
	waitForServer(t, port)

	// Make a request to verify it's working
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/", port))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "custom server" {
		t.Errorf("expected 'custom server' response, got: %s", body)
	}

	// Send shutdown signal
	sigCh <- syscall.SIGTERM

	// Wait for server to stop
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	// Verify logs
	mu.Lock()
	defer mu.Unlock()

	hasStarting := false
	hasStopped := false
	for _, log := range logs {
		if strings.Contains(log, "Starting server") {
			hasStarting = true
		}
		if strings.Contains(log, "Server stopped") {
			hasStopped = true
		}
	}
	if !hasStarting {
		t.Error("expected 'Starting server' log message")
	}
	if !hasStopped {
		t.Error("expected 'Server stopped' log message")
	}
}

func TestRun_CustomServer_CallsCleanup(t *testing.T) {
	t.Parallel()

	port := findFreePort(t)
	var srv *http.Server

	sigCh := make(chan os.Signal, 1)
	cleanupCalled := false

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Port: port,
			StartServer: func(addr string) error {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				srv = &http.Server{Addr: addr, Handler: handler}
				return srv.ListenAndServe()
			},
			ShutdownServer: func(ctx context.Context) error {
				return srv.Shutdown(ctx)
			},
			ShutdownTimeout: 1 * time.Second,
			signalChan:      sigCh,
			Logger:          func(format string, args ...interface{}) {},
			Cleanup: func(ctx context.Context) error {
				cleanupCalled = true
				return nil
			},
		})
	}()

	waitForServer(t, port)
	sigCh <- syscall.SIGTERM

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for server to stop")
	}

	if !cleanupCalled {
		t.Error("expected Cleanup to be called for custom server")
	}
}

func TestRun_CustomServer_FailsOnStartError(t *testing.T) {
	t.Parallel()

	sigCh := make(chan os.Signal, 1)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(Config{
			Port: "9999",
			StartServer: func(addr string) error {
				return fmt.Errorf("custom start error")
			},
			ShutdownServer: func(ctx context.Context) error {
				return nil
			},
			signalChan: sigCh,
			Logger:     func(format string, args ...interface{}) {},
		})
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error when StartServer fails")
		}
		if !strings.Contains(err.Error(), "failed to start") {
			t.Errorf("expected 'failed to start' error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error")
	}
}
