package main

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"vrooli-emulator-api/livedesktop"
	"vrooli-emulator-api/procmetrics"
	"vrooli-emulator-api/screenrecording"
)

// shellExec runs a command and returns stdout. Used by procmetrics' window detector.
func shellExec(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return stdout.Bytes(), err
		}
		return stdout.Bytes(), err
	}
	return stdout.Bytes(), nil
}

// processExecutorAdapter satisfies screenrecording.CommandExecutor.
type processExecutorAdapter struct{}

func (a *processExecutorAdapter) ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*screenrecording.ExecutionResult, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, command, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if cmd.ProcessState != nil {
		exit = cmd.ProcessState.ExitCode()
	}
	return &screenrecording.ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exit,
	}, err
}

// loggingMiddleware writes a one-line per-request log.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func main() {
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-emulator",
	}) {
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	router := mux.NewRouter()
	router.Use(loggingMiddleware)

	// Session domain (virtual display + remote access)
	backend := livedesktop.NewLinuxBackend(logger)
	store := livedesktop.NewInMemoryStore()
	sessionSvc := livedesktop.NewService(store, backend, logger)

	// Optional screen recording via FFmpeg resource CLI.
	sessionSvc.WithRecorder(screenrecording.NewRecorder(&processExecutorAdapter{}))

	// Default data directory for session-scoped screenshot/recording outputs.
	dataDir := os.Getenv("EMULATOR_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("creating data directory %q: %v", dataDir, err)
	}
	sessionSvc.WithDataDir(dataDir)

	// Wire process monitoring (used by window detector — not required by
	// the service constructor, but kept here for parity with the platform setup).
	_ = procmetrics.NewXdotoolDetector(procmetrics.ShellFunc(shellExec), logger)

	handler := livedesktop.NewHandler(sessionSvc)
	handler.RegisterRoutes(router)

	// Idle session reaper: 30s check interval, 30m idle timeout.
	livedesktop.StartJanitor(context.Background(), sessionSvc, 30*time.Second, 30*time.Minute)

	healthHandler := health.New().
		Version("1.0.0").
		Handler()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	if err := server.Run(server.Config{
		Handler: handlers.RecoveryHandler()(router),
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
