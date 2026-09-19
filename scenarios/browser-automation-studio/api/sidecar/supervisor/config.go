// Package supervisor provides process lifecycle management for the playwright-driver sidecar.
//
// The supervisor spawns, monitors, and restarts the sidecar process with exponential
// backoff and configurable restart limits. It integrates with the health monitoring
// system to determine when restarts are needed.
package supervisor

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds configuration for the process supervisor.
type Config struct {
	// Enabled determines whether sidecar management is active.
	// Automatically disabled if PLAYWRIGHT_DRIVER_URL is set (external driver).
	Enabled bool

	// DriverDir is the directory containing the playwright-driver.
	// Relative to the API working directory.
	DriverDir string

	// DriverScript is the script to run within DriverDir.
	DriverScript string

	// NodePath is the path to the node binary.
	// For Electron, this points to the bundled node executable.
	NodePath string

	// DriverPort is the port the sidecar listens on.
	DriverPort int

	// MaxRestarts is the maximum number of restarts allowed within RestartWindow.
	// After this limit, the supervisor enters an unrecoverable state.
	MaxRestarts int

	// RestartWindow is the time window for counting restarts.
	// Restarts outside this window don't count toward MaxRestarts.
	RestartWindow time.Duration

	// InitialBackoff is the initial delay before the first restart attempt.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between restart attempts.
	MaxBackoff time.Duration

	// BackoffMultiplier is the exponential factor for increasing backoff.
	// delay = InitialBackoff * (BackoffMultiplier ^ attemptInWindow)
	BackoffMultiplier float64

	// StartupTimeout is how long to wait for the sidecar to become healthy after start.
	StartupTimeout time.Duration

	// GracefulStop is the grace period for SIGTERM before sending SIGKILL.
	GracefulStop time.Duration
}

// DefaultConfig returns a Config with all default values.
func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		DriverDir:         "playwright-driver",
		DriverScript:      "dist/server.js",
		NodePath:          "node",
		DriverPort:        39400,
		MaxRestarts:       5,
		RestartWindow:     5 * time.Minute,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		StartupTimeout:    10 * time.Second,
		GracefulStop:      5 * time.Second,
	}
}

func LoadConfig(settings map[string]any) Config {
	cfg := DefaultConfig()
	if value, ok := settings["sidecar_enabled"].(bool); ok {
		cfg.Enabled = value
	}
	if value, ok := settings["sidecar_driver_dir"].(string); ok {
		cfg.DriverDir = value
	}
	if value, ok := settings["sidecar_driver_script"].(string); ok {
		cfg.DriverScript = value
	}
	if value, ok := settings["sidecar_node_path"].(string); ok {
		cfg.NodePath = value
	}
	// The control plane allocates the sidecar port and exposes it to every
	// component as PLAYWRIGHT_DRIVER_PORT. It must override the development
	// default, otherwise the API probes the allocated port while the child
	// listens on 39400.
	if raw := strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_PORT")); raw != "" {
		if port, err := strconv.Atoi(raw); err == nil && port > 0 && port <= 65535 {
			cfg.DriverPort = port
		}
	}
	cfg.MaxRestarts = integer(settings, "sidecar_max_restarts", cfg.MaxRestarts)
	cfg.RestartWindow = duration(settings, "sidecar_restart_window_ms", cfg.RestartWindow)
	cfg.InitialBackoff = duration(settings, "sidecar_initial_backoff_ms", cfg.InitialBackoff)
	cfg.MaxBackoff = duration(settings, "sidecar_max_backoff_ms", cfg.MaxBackoff)
	cfg.StartupTimeout = duration(settings, "sidecar_startup_timeout_ms", cfg.StartupTimeout)
	cfg.GracefulStop = duration(settings, "sidecar_graceful_stop_ms", cfg.GracefulStop)
	if value, ok := settings["sidecar_backoff_multiplier"].(float64); ok && value > 0 {
		cfg.BackoffMultiplier = value
	}
	return cfg
}

func integer(settings map[string]any, key string, fallback int) int {
	if value, ok := settings[key].(float64); ok && value >= 0 {
		return int(value)
	}
	return fallback
}

func duration(settings map[string]any, key string, fallback time.Duration) time.Duration {
	if value, ok := settings[key].(float64); ok && value > 0 {
		return time.Duration(value) * time.Millisecond
	}
	return fallback
}

// parseBool parses a string as a boolean, returning defaultVal on failure.
// isLocalURL checks if a URL points to localhost or a local address.
// Local addresses are assumed to be sidecar-managed, not external.
func isLocalURL(urlStr string) bool {
	// Check for common localhost patterns
	localPatterns := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"[::1]",
	}
	for _, pattern := range localPatterns {
		if strings.Contains(urlStr, pattern) {
			return true
		}
	}
	return false
}
