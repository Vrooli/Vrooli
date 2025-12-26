package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "web-console",
	}) {
		return // Process was re-exec'd after rebuild
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	workspacePath := cfg.storagePath + "/workspace.json"
	ws, err := newWorkspace(workspacePath)
	if err != nil {
		logger.Error("failed to initialize workspace", "error", err)
		os.Exit(1)
	}
	logger.Info("workspace initialized", "path", workspacePath)

	metrics := newMetricsRegistry()
	manager := newSessionManager(cfg, metrics, ws)
	manager.updateIdleTimeout(ws.idleTimeoutDuration())

	mux := http.NewServeMux()
	registerRoutes(mux, manager, metrics, ws)

	logger.Info(
		"web console api starting",
		"defaultCommand", cfg.defaultCommand,
		"defaultArgs", cfg.defaultArgs,
	)

	if err := server.Run(server.Config{
		Handler:      loggingMiddleware(logger, mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(lrw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.status = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("loggingResponseWriter: underlying writer does not implement http.Hijacker")
}

func (lrw *loggingResponseWriter) Flush() {
	if f, ok := lrw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (lrw *loggingResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if rf, ok := lrw.ResponseWriter.(io.ReaderFrom); ok {
		lrw.status = http.StatusOK
		return rf.ReadFrom(r)
	}
	return io.Copy(lrw.ResponseWriter, r)
}

func (lrw *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := lrw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
