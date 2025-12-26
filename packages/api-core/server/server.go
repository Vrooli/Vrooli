// Package server provides HTTP server lifecycle management with graceful shutdown
// for Vrooli scenarios.
//
// The package handles signal-based graceful shutdown, sensible timeouts, and
// cleanup coordination, eliminating boilerplate server lifecycle code.
//
// Basic usage (reads API_PORT from environment):
//
//	if err := server.Run(server.Config{
//	    Handler: router,
//	}); err != nil {
//	    log.Fatalf("Server error: %v", err)
//	}
//
// With cleanup callback:
//
//	if err := server.Run(server.Config{
//	    Handler: router,
//	    Cleanup: func(ctx context.Context) error {
//	        return db.Close()
//	    },
//	}); err != nil {
//	    log.Fatalf("Server error: %v", err)
//	}
//
// With custom port and timeouts:
//
//	if err := server.Run(server.Config{
//	    Handler:         router,
//	    Port:            "9000",
//	    ShutdownTimeout: 30 * time.Second,
//	}); err != nil {
//	    log.Fatalf("Server error: %v", err)
//	}
//
// With custom server (e.g., Fiber):
//
//	app := fiber.New()
//	// ... setup routes ...
//
//	if err := server.Run(server.Config{
//	    StartServer:    func(addr string) error { return app.Listen(addr) },
//	    ShutdownServer: func(ctx context.Context) error { return app.ShutdownWithContext(ctx) },
//	    Cleanup:        func(ctx context.Context) error { return db.Close() },
//	}); err != nil {
//	    log.Fatalf("Server error: %v", err)
//	}
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config controls HTTP server behavior and lifecycle.
type Config struct {
	// Handler is the HTTP handler to serve.
	// Required unless StartServer/ShutdownServer are provided.
	Handler http.Handler

	// StartServer is a custom function to start the server.
	// Use this for non-standard servers like Fiber.
	// When set, Handler is ignored and ShutdownServer must also be set.
	// The addr parameter is in ":port" format (e.g., ":8080").
	// Optional.
	StartServer func(addr string) error

	// ShutdownServer is a custom function to gracefully shutdown the server.
	// Required when StartServer is set.
	// The context has ShutdownTimeout remaining for shutdown operations.
	// Optional.
	ShutdownServer func(ctx context.Context) error

	// Port specifies the port to listen on (without colon prefix).
	// If empty, reads from API_PORT environment variable.
	// If API_PORT is also empty, defaults to "8080".
	Port string

	// ReadTimeout is the maximum duration for reading the entire request.
	// If zero, defaults to 30 seconds.
	// Only used with standard Handler, ignored when StartServer is set.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration for writing the response.
	// If zero, defaults to 30 seconds.
	// Only used with standard Handler, ignored when StartServer is set.
	WriteTimeout time.Duration

	// IdleTimeout is the maximum duration to wait for the next request
	// when keep-alives are enabled.
	// If zero, defaults to 120 seconds.
	// Only used with standard Handler, ignored when StartServer is set.
	IdleTimeout time.Duration

	// ShutdownTimeout is the maximum duration to wait for in-flight
	// requests to complete during graceful shutdown.
	// If zero, defaults to 10 seconds.
	ShutdownTimeout time.Duration

	// Cleanup is called after the HTTP server stops accepting connections.
	// The context has ShutdownTimeout remaining for cleanup operations.
	// Cleanup errors are logged but do not affect the return value.
	// Optional.
	Cleanup func(ctx context.Context) error

	// Logger receives server lifecycle messages (starting, shutdown, stopped).
	// If nil, uses log.Printf.
	Logger func(format string, args ...interface{})

	// EnvGetter overrides os.Getenv for testing.
	// If nil, uses os.Getenv.
	EnvGetter func(key string) string

	// Signals overrides the shutdown signals for testing.
	// If nil, listens for SIGINT and SIGTERM.
	Signals []os.Signal

	// signalChan is used for testing to inject shutdown signals.
	signalChan chan os.Signal
}

// Run starts an HTTP server and blocks until a shutdown signal is received.
//
// The server listens for SIGINT and SIGTERM by default. Upon receiving a
// shutdown signal, it:
//  1. Stops accepting new connections
//  2. Waits for in-flight requests to complete (up to ShutdownTimeout)
//  3. Calls the Cleanup function if provided
//  4. Returns nil on success
//
// For custom servers (e.g., Fiber), provide StartServer and ShutdownServer
// callbacks instead of Handler.
//
// Returns an error if the server fails to start or shutdown fails.
func Run(cfg Config) error {
	cfg = withDefaults(cfg)

	// Validate configuration
	if cfg.StartServer != nil {
		if cfg.ShutdownServer == nil {
			return errors.New("server.Config.ShutdownServer is required when StartServer is set")
		}
		return runCustomServer(cfg)
	}

	if cfg.Handler == nil {
		return errors.New("server.Config.Handler is required (or provide StartServer/ShutdownServer for custom servers)")
	}

	return runStandardServer(cfg)
}

// runStandardServer runs a standard net/http server.
func runStandardServer(cfg Config) error {
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      cfg.Handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Channel for server startup errors
	errCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		cfg.log("Starting server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or startup error
	if err := waitForShutdown(cfg, errCh); err != nil {
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return runCleanup(cfg, ctx)
}

// runCustomServer runs a custom server using the provided callbacks.
func runCustomServer(cfg Config) error {
	addr := ":" + cfg.Port

	// Channel for server startup errors
	errCh := make(chan error, 1)

	// Start server in goroutine
	go func() {
		cfg.log("Starting server on port %s", cfg.Port)
		if err := cfg.StartServer(addr); err != nil {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or startup error
	if err := waitForShutdown(cfg, errCh); err != nil {
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := cfg.ShutdownServer(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return runCleanup(cfg, ctx)
}

// waitForShutdown waits for either a shutdown signal or a startup error.
func waitForShutdown(cfg Config, errCh <-chan error) error {
	sigCh := cfg.signalChan
	if sigCh == nil {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, cfg.Signals...)
	}

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	case sig := <-sigCh:
		cfg.log("Received %v, shutting down gracefully...", sig)
		return nil
	}
}

// runCleanup runs the cleanup function and logs the server stop message.
func runCleanup(cfg Config, ctx context.Context) error {
	if cfg.Cleanup != nil {
		if err := cfg.Cleanup(ctx); err != nil {
			cfg.log("Cleanup error: %v", err)
		}
	}

	cfg.log("Server stopped")
	return nil
}

// withDefaults fills in zero values with sensible defaults.
func withDefaults(cfg Config) Config {
	// Port from environment or default
	if cfg.Port == "" {
		cfg.Port = cfg.getenv("API_PORT")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Timeouts
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 30 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 30 * time.Second
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 120 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 10 * time.Second
	}

	// Signals
	if len(cfg.Signals) == 0 {
		cfg.Signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	// Logger
	if cfg.Logger == nil {
		cfg.Logger = log.Printf
	}

	return cfg
}

// getenv returns environment variable value using custom getter or os.Getenv.
func (cfg Config) getenv(key string) string {
	if cfg.EnvGetter != nil {
		return cfg.EnvGetter(key)
	}
	return os.Getenv(key)
}

// log writes a message using the configured logger.
func (cfg Config) log(format string, args ...interface{}) {
	if cfg.Logger != nil {
		cfg.Logger(format, args...)
	}
}
