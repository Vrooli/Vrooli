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
// All values can be overridden via environment variables with BAS_SIDECAR_ prefix.
type Config struct {
	// Enabled determines whether sidecar management is active.
	// Automatically disabled if PLAYWRIGHT_DRIVER_URL is set (external driver).
	// Env: BAS_SIDECAR_ENABLED (default: true if PLAYWRIGHT_DRIVER_URL not set)
	Enabled bool

	// DriverDir is the directory containing the playwright-driver.
	// Relative to the API working directory.
	// Env: BAS_SIDECAR_DRIVER_DIR (default: playwright-driver)
	DriverDir string

	// DriverScript is the script to run within DriverDir.
	// Env: BAS_SIDECAR_DRIVER_SCRIPT (default: dist/server.js)
	DriverScript string

	// NodePath is the path to the node binary.
	// For Electron, this points to the bundled node executable.
	// Env: BAS_SIDECAR_NODE_PATH (default: node, uses PATH)
	NodePath string

	// DriverPort is the port the sidecar listens on.
	// Env: PLAYWRIGHT_DRIVER_PORT (default: 39400)
	DriverPort int

	// MaxRestarts is the maximum number of restarts allowed within RestartWindow.
	// After this limit, the supervisor enters an unrecoverable state.
	// Env: BAS_SIDECAR_MAX_RESTARTS (default: 5)
	MaxRestarts int

	// RestartWindow is the time window for counting restarts.
	// Restarts outside this window don't count toward MaxRestarts.
	// Env: BAS_SIDECAR_RESTART_WINDOW_MS (default: 300000, 5 minutes)
	RestartWindow time.Duration

	// InitialBackoff is the initial delay before the first restart attempt.
	// Env: BAS_SIDECAR_INITIAL_BACKOFF_MS (default: 1000)
	InitialBackoff time.Duration

	// MaxBackoff is the maximum delay between restart attempts.
	// Env: BAS_SIDECAR_MAX_BACKOFF_MS (default: 30000)
	MaxBackoff time.Duration

	// BackoffMultiplier is the exponential factor for increasing backoff.
	// delay = InitialBackoff * (BackoffMultiplier ^ attemptInWindow)
	// Env: BAS_SIDECAR_BACKOFF_MULTIPLIER (default: 2.0)
	BackoffMultiplier float64

	// StartupTimeout is how long to wait for the sidecar to become healthy after start.
	// Env: BAS_SIDECAR_STARTUP_TIMEOUT_MS (default: 10000)
	StartupTimeout time.Duration

	// GracefulStop is the grace period for SIGTERM before sending SIGKILL.
	// Env: BAS_SIDECAR_GRACEFUL_STOP_MS (default: 5000)
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

// LoadConfig loads configuration from environment variables.
// Values not set in the environment use defaults from DefaultConfig.
func LoadConfig() Config {
	cfg := DefaultConfig()

	// Check for explicit sidecar enable/disable setting FIRST
	// BAS_SIDECAR_ENABLED=true takes precedence over PLAYWRIGHT_DRIVER_URL
	if v := os.Getenv("BAS_SIDECAR_ENABLED"); v != "" {
		cfg.Enabled = parseBool(v, cfg.Enabled)
		// If explicitly set, skip the PLAYWRIGHT_DRIVER_URL check
	} else if externalURL := os.Getenv("PLAYWRIGHT_DRIVER_URL"); externalURL != "" {
		// Only treat as external driver if BAS_SIDECAR_ENABLED is NOT explicitly set
		// AND the URL points to a non-local address
		if !isLocalURL(externalURL) {
			cfg.Enabled = false
		}
		// Local URLs (localhost, 127.0.0.1) are assumed to be sidecar-managed
	}

	if v := os.Getenv("BAS_SIDECAR_DRIVER_DIR"); v != "" {
		cfg.DriverDir = v
	}

	if v := os.Getenv("BAS_SIDECAR_DRIVER_SCRIPT"); v != "" {
		cfg.DriverScript = v
	}

	if v := os.Getenv("BAS_SIDECAR_NODE_PATH"); v != "" {
		cfg.NodePath = v
	}

	if v := os.Getenv("PLAYWRIGHT_DRIVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			cfg.DriverPort = port
		}
	}

	if v := os.Getenv("BAS_SIDECAR_MAX_RESTARTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.MaxRestarts = n
		}
	}

	if v := os.Getenv("BAS_SIDECAR_RESTART_WINDOW_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.RestartWindow = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_INITIAL_BACKOFF_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.InitialBackoff = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_MAX_BACKOFF_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.MaxBackoff = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_BACKOFF_MULTIPLIER"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			cfg.BackoffMultiplier = f
		}
	}

	if v := os.Getenv("BAS_SIDECAR_STARTUP_TIMEOUT_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.StartupTimeout = time.Duration(ms) * time.Millisecond
		}
	}

	if v := os.Getenv("BAS_SIDECAR_GRACEFUL_STOP_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.GracefulStop = time.Duration(ms) * time.Millisecond
		}
	}

	return cfg
}

// parseBool parses a string as a boolean, returning defaultVal on failure.
func parseBool(s string, defaultVal bool) bool {
	switch s {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}

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
