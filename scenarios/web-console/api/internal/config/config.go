// Package config holds the web-console API's tunable levers.
//
// DOC: docs/reference/configuration.md#api-levers-environment-variables
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// DefaultTerminalScrollbackLines is the one production scrollback depth used
// by the emulator and persistent backend. UI clients must retain at least
// this many lines when replaying a snapshot.
const DefaultTerminalScrollbackLines = 50_000

// DefaultMaxSnapshotBytes bounds one reconnect/resync replay.
const DefaultMaxSnapshotBytes = 8 * 1024 * 1024

// Config holds all tunable levers for the web-console API.
// Each field maps to an environment variable with a sane default.
// See docs/reference/configuration.md for full documentation.
type Config struct {
	// BridgeURL is the optional server-side Bridge endpoint. Empty means
	// resolve the local vrooli-bridge scenario through discovery.
	// Env: WC_BRIDGE_URL | Default: empty (local slug discovery)
	BridgeURL string

	// TerminalScrollbackLines is the number of decoded scrollback lines the
	// per-session terminal emulator retains. Replayed via the snapshot
	// stream on every (re)connect.
	// Env: WC_TERMINAL_SCROLLBACK_LINES | Default: fifty-thousand | Range: 100–100 thousand
	TerminalScrollbackLines int

	// MaxSnapshotBytes is the maximum ANSI byte stream sent for one replay.
	// Env: WC_MAX_SNAPSHOT_BYTES | Default: 8388608 | Range: 65536–16777216
	MaxSnapshotBytes int

	// PTYReadBuffer is the byte size of the buffer used to read PTY output.
	// Env: WC_PTY_READ_BUFFER | Default: 4096 | Range: 512–65536
	PTYReadBuffer int

	// WSBufferSize is the read and write buffer size for WebSocket connections.
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
	// Env: WC_DEFAULT_SHELL | Default: $SHELL or /bin/sh
	DefaultShell string

	// MaxSessions is the maximum number of concurrent PTY sessions allowed.
	// Zero means unlimited.
	// Env: WC_MAX_SESSIONS | Default: 0 (unlimited) | Range: 0–1000
	MaxSessions int

	// ArchiveMessageLessAgeDays permanently deletes explicitly archived rows
	// with no conversation messages after this age. Zero means unlimited.
	// Env: WC_ARCHIVE_MESSAGELESS_AGE_DAYS | Default: 0 | Range: 0–36500
	ArchiveMessageLessAgeDays int

	// ArchiveAgentHomeAgeDays prunes resumability state while preserving the
	// transcript after this age. Zero means unlimited.
	// Env: WC_ARCHIVE_AGENT_HOME_AGE_DAYS | Default: 0 | Range: 0–36500
	ArchiveAgentHomeAgeDays int

	// ArchiveMaxBytes is a soft ceiling across archived transcripts and agent
	// history. Retention only removes agent history to meet it; message-bearing
	// transcripts remain protected. Zero means unlimited.
	// Env: WC_ARCHIVE_MAX_BYTES | Default: 0 (unlimited)
	ArchiveMaxBytes int64

	// ConversationRetentionDays removes conversation events older than this
	// many days. Zero means unlimited.
	// Env: WC_CONVERSATION_RETENTION_DAYS | Default: 180 | Range: 0–36500
	ConversationRetentionDays int

	// ConversationMaxEventsPerSession bounds retained events per session. The
	// newest events are preserved. Zero means unlimited.
	// Env: WC_CONVERSATION_MAX_EVENTS_PER_SESSION | Default: 5000 | Range: 0–1000000
	ConversationMaxEventsPerSession int

	// ClientChannelBuffer is the capacity of the per-client output channel.
	// Env: WC_CLIENT_CHANNEL_BUFFER | Default: 256 | Range: 8–1024
	ClientChannelBuffer int

	// CoalesceNotifyThreshold is the number of coalesced output frames per
	// client before a sync_warning notification is sent.
	// Env: WC_COALESCE_NOTIFY_THRESHOLD | Default: 5 | Range: 1–1000
	CoalesceNotifyThreshold int

	// InputQueueSize bounds ordered PTY input waiting for the writer worker.
	// Env: WC_INPUT_QUEUE_SIZE | Default: 256 | Range: 16–4096
	InputQueueSize int

	// DefaultCWD is the working directory used for newly spawned shell sessions.
	// Env: WC_DEFAULT_CWD | Default: derived from environment/runtime
	DefaultCWD string

	// DefaultBackend is the session backend used when the client does not
	// specify one. "standard" uses a raw PTY; "persistent" uses tmux;
	// "auto" (the default) resolves to "persistent" when tmux is available.
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

// Default returns the default configuration with all sane defaults.
func Default() Config {
	shell, _ := resolveShell()
	return Config{
		BridgeURL:                        "",
		TerminalScrollbackLines:         DefaultTerminalScrollbackLines,
		MaxSnapshotBytes:                DefaultMaxSnapshotBytes,
		PTYReadBuffer:                   4096,
		WSBufferSize:                    4096,
		DefaultCols:                     80,
		DefaultRows:                     24,
		DefaultShell:                    shell,
		MaxSessions:                     0,
		ArchiveMessageLessAgeDays:       0,
		ArchiveAgentHomeAgeDays:         0,
		ArchiveMaxBytes:                 0,
		ConversationRetentionDays:       180,
		ConversationMaxEventsPerSession: 5000,
		ClientChannelBuffer:             256,
		CoalesceNotifyThreshold:         5,
		InputQueueSize:                  256,
		DefaultCWD:                      resolveWorkingDir(),
		DefaultBackend:                  "auto",
		DefaultPolicyMode:               "never",
		DefaultPolicyDuration:           "",
	}
}

// Load reads configuration from environment variables, falling back
// to Default() values for anything not set or invalid.
func Load() Config {
	cfg := Default()
	cfg.BridgeURL = os.Getenv("WC_BRIDGE_URL")

	cfg.TerminalScrollbackLines = envInt("WC_TERMINAL_SCROLLBACK_LINES", cfg.TerminalScrollbackLines, 100, 100_000)
	cfg.MaxSnapshotBytes = envInt("WC_MAX_SNAPSHOT_BYTES", cfg.MaxSnapshotBytes, 64*1024, 16*1024*1024)
	cfg.PTYReadBuffer = envInt("WC_PTY_READ_BUFFER", cfg.PTYReadBuffer, 512, 65536)
	cfg.WSBufferSize = envInt("WC_WS_BUFFER_SIZE", cfg.WSBufferSize, 512, 65536)
	cfg.DefaultCols = uint16(envInt("WC_DEFAULT_COLS", int(cfg.DefaultCols), 20, 500))
	cfg.DefaultRows = uint16(envInt("WC_DEFAULT_ROWS", int(cfg.DefaultRows), 5, 200))
	cfg.MaxSessions = envInt("WC_MAX_SESSIONS", cfg.MaxSessions, 0, 1000)
	cfg.ArchiveMessageLessAgeDays = envInt("WC_ARCHIVE_MESSAGELESS_AGE_DAYS", cfg.ArchiveMessageLessAgeDays, 0, 36_500)
	cfg.ArchiveAgentHomeAgeDays = envInt("WC_ARCHIVE_AGENT_HOME_AGE_DAYS", cfg.ArchiveAgentHomeAgeDays, 0, 36_500)
	cfg.ArchiveMaxBytes = envInt64("WC_ARCHIVE_MAX_BYTES", cfg.ArchiveMaxBytes, 0, 1<<62)
	cfg.ConversationRetentionDays = envInt("WC_CONVERSATION_RETENTION_DAYS", cfg.ConversationRetentionDays, 0, 36_500)
	cfg.ConversationMaxEventsPerSession = envInt("WC_CONVERSATION_MAX_EVENTS_PER_SESSION", cfg.ConversationMaxEventsPerSession, 0, 1_000_000)
	cfg.ClientChannelBuffer = envInt("WC_CLIENT_CHANNEL_BUFFER", cfg.ClientChannelBuffer, 8, 1024)
	cfg.CoalesceNotifyThreshold = envInt("WC_COALESCE_NOTIFY_THRESHOLD", cfg.CoalesceNotifyThreshold, 1, 1000)
	cfg.InputQueueSize = envInt("WC_INPUT_QUEUE_SIZE", cfg.InputQueueSize, 16, 4096)

	shell, _ := resolveShell()
	cfg.DefaultShell = shell
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

// LoadChecked is the startup-safe form of Load. It preserves the historical
// Load API for callers that cannot return an error while exposing the typed
// host-configuration failure to lifecycle owners.
func LoadChecked() (Config, error) {
	cfg := Load()
	shell, err := resolveShell()
	if err != nil {
		cfg.DefaultShell = ""
		return cfg, err
	}
	cfg.DefaultShell = shell
	return cfg, nil
}

// ResolveWorkingDir is the exported entry point for callers outside this
// package that need the same working-directory resolution logic.
func ResolveWorkingDir() string { return resolveWorkingDir() }

// ResolveShell is the exported entry point for callers that need the same
// shell-resolution logic.
func ResolveShell() string {
	shell, _ := resolveShell()
	return shell
}

var ErrShellUnavailable = fmt.Errorf("web-console shell is unavailable")

// resolveShell determines which shell binary to use:
//
//	WC_DEFAULT_SHELL  →  powershell.exe (Windows) or $SHELL → /bin/sh
var shellLookPath = exec.LookPath

func resolveShell() (string, error) {
	return resolveShellForOS(runtime.GOOS)
}

func resolveShellForOS(goos string) (string, error) {
	requested := ""
	if v := os.Getenv("WC_DEFAULT_SHELL"); v != "" {
		requested = v
	} else if goos == "windows" {
		// Git Bash commonly exports a POSIX SHELL path that cannot be resolved
		// by Windows exec.LookPath. Prefer the native shell unless the caller
		// explicitly selected WC_DEFAULT_SHELL.
		requested = "powershell.exe"
	} else if v := os.Getenv("SHELL"); v != "" {
		requested = v
	} else {
		requested = "/bin/sh"
	}
	resolved, err := shellLookPath(requested)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %v", ErrShellUnavailable, requested, err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("%w: cannot make %q absolute: %v", ErrShellUnavailable, resolved, err)
	}
	return resolved, nil
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
//  1. WC_DEFAULT_CWD
//  2. PROJECT_ROOT
//  3. project root inferred from SCENARIO_DIR path
//  4. SCENARIO_DIR
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

func envInt64(key string, defaultVal, min, max int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(raw, 10, 64)
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
