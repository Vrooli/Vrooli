package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// DOC: docs/reference/configuration.md#api-levers-environment-variables
// Config holds all tunable levers for the web-console API.
// Each field maps to an environment variable with a sane default.
// See docs/reference/configuration.md for full documentation.
type Config struct {
	// TerminalScrollbackLines is the number of decoded scrollback lines the
	// per-session terminal emulator retains. Replayed via the snapshot
	// stream on every (re)connect.
	// Env: WC_TERMINAL_SCROLLBACK_LINES | Default: 10000 | Range: 100–100000
	TerminalScrollbackLines int

	// PTYReadBuffer is the byte size of the buffer used to read PTY output.
	// Larger values reduce syscall overhead for high-throughput terminals;
	// smaller values reduce per-session memory.
	// Env: WC_PTY_READ_BUFFER | Default: 4096 | Range: 512–65536
	PTYReadBuffer int

	// WSBufferSize is the read and write buffer size for WebSocket connections.
	// Larger buffers improve throughput for terminals with heavy output.
	// Env: WC_WS_BUFFER_SIZE | Default: 4096 | Range: 512–65536
	WSBufferSize int

	// DefaultCols is the default terminal column count when the client does
	// not specify one.
	// CROSS-LANGUAGE COUPLING: Must match DEFAULT_COLS in ui/src/consts/config.ts.
	// Env: WC_DEFAULT_COLS | Default: 80 | Range: 20–500
	DefaultCols uint16

	// DefaultRows is the default terminal row count when the client does
	// not specify one.
	// CROSS-LANGUAGE COUPLING: Must match DEFAULT_ROWS in ui/src/consts/config.ts.
	// Env: WC_DEFAULT_ROWS | Default: 24 | Range: 5–200
	DefaultRows uint16

	// DefaultShell is the shell binary to launch when no shell is requested.
	// Falls back to $SHELL, then /bin/sh.
	// Env: WC_DEFAULT_SHELL | Default: $SHELL or /bin/sh
	DefaultShell string

	// MaxSessions is the maximum number of concurrent PTY sessions allowed.
	// Zero means unlimited. This is a safety guardrail to prevent resource
	// exhaustion on constrained systems.
	// Env: WC_MAX_SESSIONS | Default: 0 (unlimited) | Range: 0–1000
	MaxSessions int

	// ClientChannelBuffer is the capacity of the per-client output channel.
	// Higher values absorb output bursts from fast-producing PTYs before
	// frame coalescing kicks in. With coalescing, frames are merged into a
	// per-client pending buffer rather than dropped. When the pending
	// buffer exceeds pendingBufferMax (broadcast.go), the oldest data is
	// truncated; the next snapshot replay restores correct state.
	// Env: WC_CLIENT_CHANNEL_BUFFER | Default: 256 | Range: 8–1024
	ClientChannelBuffer int

	// CoalesceNotifyThreshold is the number of coalesced output frames per
	// client before a sync_warning notification is sent via the WebSocket.
	// Lower values alert the user sooner; higher values reduce noise.
	// Env: WC_COALESCE_NOTIFY_THRESHOLD | Default: 5 | Range: 1–1000
	CoalesceNotifyThreshold int

	// SIGWINCHCooldownMs is the minimum interval in milliseconds between
	// SIGWINCH-based coalesce-trim recoveries per session. A cooldown
	// prevents rapid trim events from storming the TUI with SIGWINCH
	// while it is redrawing. The recovery path is also suppressed entirely
	// while the PTY is in the alternate screen buffer; the cooldown only
	// applies outside alt-buffer mode.
	// Env: WC_SIGWINCH_COOLDOWN_MS | Default: 1000 | Range: 0–30000
	SIGWINCHCooldownMs int

	// DefaultCWD is the working directory used for newly spawned shell sessions.
	// Fallback chain:
	//   WC_DEFAULT_CWD -> PROJECT_ROOT -> SCENARIO_DIR -> inferred scenario dir -> current process cwd
	// Env: WC_DEFAULT_CWD | Default: derived from environment/runtime
	DefaultCWD string

	// DefaultBackend is the session backend used when the client does not
	// specify one. "standard" uses a raw PTY; "persistent" uses tmux;
	// "auto" (the default) resolves to "persistent" when tmux is available,
	// falling back to "standard" otherwise. Resolved at server startup.
	// CROSS-LANGUAGE COUPLING: Must match BackendID constants in backend_registry.go.
	// Env: WC_DEFAULT_BACKEND | Default: "auto"
	DefaultBackend string

	// DefaultPolicyMode is the default expiration policy mode for new sessions.
	// Env: WC_DEFAULT_POLICY_MODE | Default: "never"
	DefaultPolicyMode string

	// DefaultPolicyDuration is the default expiration policy duration for new sessions.
	// Only used when DefaultPolicyMode is "preset" or "custom".
	// Env: WC_DEFAULT_POLICY_DURATION | Default: ""
	DefaultPolicyDuration string
}

// DefaultConfig returns the default configuration with all sane defaults.
func DefaultConfig() Config {
	return Config{
		TerminalScrollbackLines: 10_000,
		PTYReadBuffer:           4096,
		WSBufferSize:            4096,
		DefaultCols:             80,
		DefaultRows:             24,
		DefaultShell:            resolveShell(),
		MaxSessions:             0,
		ClientChannelBuffer:     256,
		CoalesceNotifyThreshold: 5,
		SIGWINCHCooldownMs:      1000,
		DefaultCWD:              resolveWorkingDir(),
		DefaultBackend:          "auto",
		DefaultPolicyMode:       "never",
		DefaultPolicyDuration:   "",
	}
}

// LoadConfig reads configuration from environment variables, falling back
// to DefaultConfig values for anything not set or invalid.
func LoadConfig() Config {
	cfg := DefaultConfig()

	cfg.TerminalScrollbackLines = envInt("WC_TERMINAL_SCROLLBACK_LINES", cfg.TerminalScrollbackLines, 100, 100_000)
	cfg.PTYReadBuffer = envInt("WC_PTY_READ_BUFFER", cfg.PTYReadBuffer, 512, 65536)
	cfg.WSBufferSize = envInt("WC_WS_BUFFER_SIZE", cfg.WSBufferSize, 512, 65536)
	cfg.DefaultCols = uint16(envInt("WC_DEFAULT_COLS", int(cfg.DefaultCols), 20, 500))
	cfg.DefaultRows = uint16(envInt("WC_DEFAULT_ROWS", int(cfg.DefaultRows), 5, 200))
	cfg.MaxSessions = envInt("WC_MAX_SESSIONS", cfg.MaxSessions, 0, 1000)
	cfg.ClientChannelBuffer = envInt("WC_CLIENT_CHANNEL_BUFFER", cfg.ClientChannelBuffer, 8, 1024)
	cfg.CoalesceNotifyThreshold = envInt("WC_COALESCE_NOTIFY_THRESHOLD", cfg.CoalesceNotifyThreshold, 1, 1000)
	cfg.SIGWINCHCooldownMs = envInt("WC_SIGWINCH_COOLDOWN_MS", cfg.SIGWINCHCooldownMs, 0, 30000)

	cfg.DefaultShell = resolveShell()
	cfg.DefaultCWD = resolveWorkingDir()

	if v := os.Getenv("WC_DEFAULT_BACKEND"); v != "" {
		cfg.DefaultBackend = v
	}
	if v := os.Getenv("WC_DEFAULT_POLICY_MODE"); v != "" {
		cfg.DefaultPolicyMode = v
	}
	if v := os.Getenv("WC_DEFAULT_POLICY_DURATION"); v != "" {
		cfg.DefaultPolicyDuration = v
	}

	return cfg
}

// resolveShell determines which shell binary to use. The full fallback chain
// (from highest to lowest priority) is:
//
//	WC_DEFAULT_SHELL  →  $SHELL  →  /bin/sh
//
// This is the single place where the "which shell to launch?" decision is made.
func resolveShell() string {
	if v := os.Getenv("WC_DEFAULT_SHELL"); v != "" {
		return v
	}
	if v := os.Getenv("SHELL"); v != "" {
		return v
	}
	return "/bin/sh"
}

func validDirectory(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func inferScenarioDirFromWD(wd string) string {
	if wd == "" {
		return ""
	}

	cleaned := filepath.Clean(wd)
	if filepath.Base(cleaned) == "api" {
		parent := filepath.Dir(cleaned)
		if validDirectory(parent) {
			return parent
		}
	}
	return ""
}

func inferProjectRootFromScenarioPath(path string) string {
	if path == "" {
		return ""
	}

	current := filepath.Clean(path)
	for {
		if filepath.Base(current) == "scenarios" {
			root := filepath.Dir(current)
			if validDirectory(root) {
				return root
			}
			return ""
		}

		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// resolveWorkingDir determines the default PTY working directory.
// Priority:
//  1. WC_DEFAULT_CWD (explicit override)
//  2. PROJECT_ROOT (workspace root for cross-device parity)
//  3. project root inferred from SCENARIO_DIR path
//  4. SCENARIO_DIR (scenario lifecycle hint)
//  5. project root inferred from current working directory path
//  6. parent of cwd when running from scenario/api
//  7. current process working directory
func resolveWorkingDir() string {
	if v := os.Getenv("WC_DEFAULT_CWD"); validDirectory(v) {
		return v
	}
	if v := os.Getenv("PROJECT_ROOT"); validDirectory(v) {
		return v
	}
	if v := os.Getenv("SCENARIO_DIR"); validDirectory(v) {
		if inferred := inferProjectRootFromScenarioPath(v); inferred != "" {
			return inferred
		}
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if inferred := inferProjectRootFromScenarioPath(wd); inferred != "" {
		return inferred
	}
	if scenarioDir := inferScenarioDirFromWD(wd); scenarioDir != "" {
		return scenarioDir
	}
	return wd
}

// envInt reads an integer environment variable, clamping it to [min, max].
// Returns defaultVal if the variable is unset or unparseable.
func envInt(key string, defaultVal, min, max int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "web-console: ignoring invalid %s=%q: %v\n", key, raw, err)
		return defaultVal
	}
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
