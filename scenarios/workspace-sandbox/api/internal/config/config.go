// Package config provides unified configuration for the workspace-sandbox service.
//
// # Control Surface Design
//
// This package defines the tunable levers for workspace-sandbox, organized into
// coherent groups that reflect how operators think about the system:
//
//   - Server: HTTP server behavior (timeouts, address)
//   - Limits: Capacity constraints (max sandboxes, sizes)
//   - Lifecycle: Sandbox TTL and GC behavior
//   - Policy: Approval and attribution rules
//   - Driver: Filesystem driver settings
//
// # Design Principles
//
//  1. Levers have clear, intention-revealing names
//  2. Each lever has a single, obvious responsibility
//  3. Defaults work well for common usage
//  4. Extreme values degrade gracefully, not catastrophically
//  5. Environment variables follow WORKSPACE_SANDBOX_ prefix convention
package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the unified configuration for workspace-sandbox.
// All tunable levers are centralized here.
type Config struct {
	// Server controls HTTP server behavior.
	Server ServerConfig

	// Limits defines capacity constraints.
	Limits LimitsConfig

	// Lifecycle controls sandbox TTL and GC behavior.
	Lifecycle LifecycleConfig

	// Policy controls approval and attribution rules.
	Policy PolicyConfig

	// Driver controls filesystem driver settings.
	Driver DriverConfig

	// Execution controls sandbox execution defaults and constraints.
	Execution ExecutionConfig

	// Database connection settings.
	Database DatabaseConfig

	// Integration settings for external scenario sync.
	Integration IntegrationConfig

	// Retention controls the diff-archive retention reconciler.
	Retention RetentionConfig
}

// ServerConfig controls HTTP server behavior.
// Higher timeouts = more tolerance for slow clients but more resource usage.
type ServerConfig struct {
	// Port is the HTTP server port.
	// Default: from API_PORT env var (required).
	Port string

	// ReadTimeout is the maximum duration for reading the entire request.
	// Higher = more tolerance for slow uploads.
	// Default: 30s
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration for writing the response.
	// Higher = more tolerance for large diffs.
	// Default: 30s
	WriteTimeout time.Duration

	// IdleTimeout is the maximum duration for keep-alive connections.
	// Higher = better connection reuse but more memory.
	// Default: 120s
	IdleTimeout time.Duration

	// ShutdownTimeout is the maximum duration for graceful shutdown.
	// Default: 10s
	ShutdownTimeout time.Duration

	// CORSAllowedOrigins controls CORS. Empty means allow all (*).
	// Default: empty (allow all)
	CORSAllowedOrigins []string
}

// LimitsConfig defines capacity constraints.
// These prevent resource exhaustion and ensure predictable behavior.
type LimitsConfig struct {
	// MaxSandboxes is the maximum number of active sandboxes.
	// Higher = more parallel agents but more resource usage.
	// Default: 1000
	MaxSandboxes int

	// MaxSandboxSizeMB is the maximum size per sandbox in megabytes.
	// Higher = larger projects but more disk usage.
	// Default: 10240 (10 GB)
	MaxSandboxSizeMB int64

	// MaxTotalSizeMB is the maximum total storage for all sandboxes.
	// When exceeded, GC should be triggered.
	// Default: 102400 (100 GB)
	MaxTotalSizeMB int64

	// DefaultListLimit is the default page size for list operations.
	// Default: 100
	DefaultListLimit int

	// MaxListLimit is the maximum allowed page size for list operations.
	// Default: 1000
	MaxListLimit int
}

// LifecycleConfig controls sandbox TTL and GC behavior.
type LifecycleConfig struct {
	// DefaultTTL is the default time-to-live for sandboxes.
	// Sandboxes older than this may be garbage collected.
	// Default: 24h
	DefaultTTL time.Duration

	// IdleTimeout is how long a sandbox can be unused before GC eligibility.
	// Default: 4h
	IdleTimeout time.Duration

	// GCInterval is how often the GC process runs.
	// Default: 15m
	GCInterval time.Duration

	// AutoCleanupTerminal controls whether approved/rejected sandboxes
	// are automatically cleaned up after a delay.
	// Default: true
	AutoCleanupTerminal bool

	// TerminalCleanupDelay is how long to wait before cleaning up
	// approved/rejected sandboxes.
	// Default: 1h
	TerminalCleanupDelay time.Duration

	// ProcessGracePeriod is how long to wait after SIGTERM before sending SIGKILL.
	// Higher = more time for graceful shutdown but slower cleanup.
	// Default: 100ms
	ProcessGracePeriod time.Duration

	// ProcessKillWait is how long to wait after SIGKILL for process to die.
	// Default: 50ms
	ProcessKillWait time.Duration

	// AutoHealIdleGrace is the minimum time since LastUsedAt before
	// a stale sandbox mount is eligible for automatic remount. Default: 30s.
	AutoHealIdleGrace time.Duration

	// AutoHealMaxRetries is the maximum consecutive remount failures
	// before marking a sandbox as Error. Default: 5.
	AutoHealMaxRetries int

	// AutoHealBaseBackoff is the initial backoff after a failed remount.
	// Doubled on each subsequent failure, capped at 1h. Default: 30s.
	AutoHealBaseBackoff time.Duration

	// ManualReviewTTL is the maximum time a manualReview=true sandbox may
	// remain in pending-review state past its run-end timestamp before the
	// GC reconciler auto-denies all pending-review file changes and tears
	// the sandbox down. Set to 0 to disable expiry (sandbox persists
	// indefinitely until explicit operator action).
	//
	// Per Decision D1 in
	// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md
	// the locked default is 7 days. Operators with longer review cycles
	// can raise WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL accordingly.
	//
	// Consumed by the existing LifecycleReconciler (no new ticker) — see
	// scenarios/workspace-sandbox/api/internal/gc/lifecycle.go.
	ManualReviewTTL time.Duration
}

// PolicyConfig controls attribution and validation rules.
type PolicyConfig struct {
	// CommitMessageTemplate is the template for auto-generated commit messages.
	// Supports placeholders: {{.SandboxID}}, {{.FileCount}}, {{.Actor}}
	// Default: "Apply sandbox changes ({{.FileCount}} files)"
	CommitMessageTemplate string

	// CommitAuthorMode controls how commit authors are attributed.
	// Options: "agent", "reviewer", "coauthored"
	// Default: "agent"
	CommitAuthorMode string

	// ValidationHooks defines pre-commit validation hooks to run before
	// applying changes. Each hook is a command that must exit 0 to pass.
	// [OT-P1-005] Pre-commit Validation Hooks
	ValidationHooks []ValidationHookConfig

	// ValidationTimeout is the maximum time to wait for all validation hooks.
	// Default: 5m
	ValidationTimeout time.Duration

	// BinaryDetectionThreshold is the number of bytes to scan when detecting
	// binary files. Files with null bytes in this range are treated as binary.
	// Higher = more accurate detection but slower for large files.
	// Default: 8000
	BinaryDetectionThreshold int

	// DefaultNoLock controls whether new sandboxes skip scope locking by default.
	// When true, sandboxes are created without mutual exclusion checks unless
	// the caller explicitly requests locking (noLock: false).
	// When false, sandboxes enforce mutual exclusion unless the caller
	// explicitly disables it (noLock: true).
	// Default: true
	DefaultNoLock bool

	// TeardownHooks defines pre-teardown hooks to run before unmounting or
	// deleting a sandbox. Each hook runs best-effort; failures are logged
	// but never block teardown. This enables external systems to gracefully
	// evacuate processes from the sandbox's merged directory before the
	// filesystem disappears.
	//
	// In the Vrooli ecosystem, this is typically configured to call
	// "vrooli scenario heal-from-sandbox" which detects scenarios running
	// from the sandbox path and restarts them from the canonical repo.
	TeardownHooks []TeardownHookConfig

	// TeardownTimeout is the maximum time to wait for all teardown hooks.
	// Shorter than validation timeout since teardown should be fast.
	// Default: 30s
	TeardownTimeout time.Duration
}

// ValidationHookConfig defines a single validation hook.
type ValidationHookConfig struct {
	// Name is a human-readable identifier for the hook.
	Name string

	// Description explains what the hook validates.
	Description string

	// Command is the executable to run.
	Command string

	// Args are arguments to pass to the command.
	Args []string

	// Required determines if a failure blocks the commit.
	// If false, failure is logged but approval proceeds.
	Required bool

	// Timeout is the maximum time for this specific hook.
	// If zero, uses the global ValidationTimeout.
	Timeout time.Duration
}

// TeardownHookConfig defines a single pre-teardown hook.
// Unlike ValidationHookConfig, there is no Required field — all teardown
// hooks are best-effort because teardown must never be blocked.
type TeardownHookConfig struct {
	// Name is a human-readable identifier for the hook.
	Name string

	// Description explains what the hook does.
	Description string

	// Command is the executable to run.
	Command string

	// Args are arguments to pass to the command.
	Args []string

	// Timeout is the maximum time for this specific hook.
	// If zero, uses the global TeardownTimeout.
	Timeout time.Duration
}

// DriverConfig controls filesystem driver settings.
type DriverConfig struct {
	// BaseDir is the root directory for sandbox artifacts.
	// Default: ~/.local/share/workspace-sandbox (XDG-compliant, user-writable)
	BaseDir string

	// UseFuseOverlayfs enables fuse-overlayfs instead of kernel overlayfs.
	// Enables unprivileged operation but may be slower.
	// Default: false
	UseFuseOverlayfs bool

	// HomeOverlayBaseDir is the directory that holds per-sandbox
	// home-{upper,work,merged} dirs for the host-$HOME overlay. MUST be
	// outside $HOME — placing the upper layer inside $HOME (the lower
	// layer) creates a self-referential overlayfs mount whose behavior is
	// undefined per kernel docs and triggers intermittent EBUSY/EINVAL.
	//
	// DOC: home-overlay storage seam. See docs/internal/SEAMS.md.
	// Default: ${XDG_RUNTIME_DIR}/workspace-sandbox or
	// /var/tmp/workspace-sandbox-$UID. Validated fatally at startup if it
	// resolves under $HOME.
	HomeOverlayBaseDir string

	// ProjectRoot is the default project root for sandboxes.
	// Set via PROJECT_ROOT env var, otherwise resolved from repo-contract-aware
	// root discovery (VROOLI_SOURCE_ROOT/VROOLI_ROOT/CWD).
	// If root discovery fails, it must be specified per-request.
	ProjectRoot string
}

// ExecutionConfig controls sandbox execution defaults and constraints.
// These levers allow operators to set sensible defaults for resource limits
// and isolation behavior, with enforcement ceilings to prevent abuse.
type ExecutionConfig struct {
	// DefaultResourceLimits are applied when no limits are specified per-request.
	// Zero values mean unlimited.
	DefaultResourceLimits ResourceLimitsConfig `json:"defaultResourceLimits"`

	// MaxResourceLimits are the maximum values users can request.
	// Zero values mean no maximum (unlimited allowed).
	MaxResourceLimits ResourceLimitsConfig `json:"maxResourceLimits"`

	// DefaultIsolationProfile is the profile used when none is specified.
	// Default: "full"
	DefaultIsolationProfile string `json:"defaultIsolationProfile"`
}

// ResourceLimitsConfig defines resource limit settings.
// Zero values mean unlimited (no limit applied).
type ResourceLimitsConfig struct {
	// MemoryLimitMB sets the virtual memory limit in megabytes.
	MemoryLimitMB int `json:"memoryLimitMB"`

	// CPUTimeSec sets the CPU time limit in seconds.
	CPUTimeSec int `json:"cpuTimeSec"`

	// MaxProcesses sets the maximum number of child processes.
	MaxProcesses int `json:"maxProcesses"`

	// MaxOpenFiles sets the maximum number of open file descriptors.
	MaxOpenFiles int `json:"maxOpenFiles"`

	// TimeoutSec sets the wall-clock timeout in seconds.
	TimeoutSec int `json:"timeoutSec"`
}

// DatabaseConfig controls database connection settings for the embedded
// SQLite store.
type DatabaseConfig struct {
	// Path is an explicit override for the SQLite file location. When empty,
	// the API derives the path through api-core/storage. Honors SQLITE_PATH.
	Path string
}

// RetentionConfig controls how long diff archives (sandbox_diff_archives
// rows + their on-disk blobs) are kept before the archive-retention
// reconciler evicts them. The three levers compose: any archive that
// trips ANY of them is eligible for eviction. Zero values disable a
// lever (subject to the per-field rules below).
//
// Snapshot rules and their reason:
//
//   - MaxArchiveAgeDays: archives whose snapshot_at is older than now -
//     MaxAge are unconditionally evicted. 0 disables age-based eviction.
//
//   - MaxArchiveSizeBytes: when the sum of total_blob_bytes across all
//     archives exceeds this budget, the oldest archives are evicted
//     until the running total falls below the budget. 0 disables
//     size-based eviction. Counted across ALL archives — not per
//     project — because the disk is the shared resource.
//
//   - MaxArchivesPerProject: when more than this many archives exist
//     for a given project_root, the oldest within that project are
//     evicted to bring the count to the cap. 0 disables the cap.
//
// Defaults (90 days, 10 GiB total, no per-project cap) are conservative:
// the goal is to make Git Control Tower's audit story durable without
// turning a forgotten dev box into a disk-full incident.
//
// Persistence: defaults flow from Default() and may be overridden by
// the documented WORKSPACE_SANDBOX_RETENTION_MAX_AGE_DAYS,
// WORKSPACE_SANDBOX_RETENTION_MAX_SIZE_BYTES, and
// WORKSPACE_SANDBOX_RETENTION_MAX_PER_PROJECT env vars at startup. The
// PUT /config/retention endpoint persists runtime updates to a JSON
// file under ClassConfig so they survive restart; on next boot the
// file is the source of truth and overrides env-derived defaults.
type RetentionConfig struct {
	// MaxArchiveAgeDays evicts archives older than this many days.
	// Default: 90. 0 disables age-based eviction.
	MaxArchiveAgeDays int `json:"maxArchiveAgeDays"`

	// MaxArchiveSizeBytes is the total disk budget for all archive
	// blobs combined (the sum of total_blob_bytes across rows). When
	// the sum exceeds this, the oldest archives are evicted oldest-
	// first. Default: 10 GiB (10737418240). 0 disables size eviction.
	MaxArchiveSizeBytes int64 `json:"maxArchiveSizeBytes"`

	// MaxArchivesPerProject caps how many archives may exist per
	// project_root. Excess archives within a project are evicted
	// oldest-first. Default: 0 (no cap).
	MaxArchivesPerProject int `json:"maxArchivesPerProject"`
}

// IntegrationConfig controls cross-scenario callbacks.
type IntegrationConfig struct {
	// AgentManagerURL is the base URL for agent-manager API (optional).
	AgentManagerURL string

	// AgentManagerSyncEnabled enables workspace-sandbox -> agent-manager sync.
	AgentManagerSyncEnabled bool

	// AgentManagerSyncTimeout bounds outbound sync requests.
	AgentManagerSyncTimeout time.Duration
}

// DefaultBaseDir returns the default sandbox base directory.
// Uses XDG data directory (~/.local/share/workspace-sandbox) for unprivileged operation.
// Falls back to /var/lib/workspace-sandbox if home directory cannot be determined.
func DefaultBaseDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "workspace-sandbox")
	}
	// Fallback for edge cases (e.g., running in containers without HOME set)
	return "/var/lib/workspace-sandbox"
}

// ResolveHomeOverlayBaseDir returns the directory that holds per-sandbox
// home-{upper,work,merged} dirs. The result MUST be outside $HOME to
// avoid a self-referential overlayfs mount (lower=$HOME, upper=$HOME/...).
//
// Resolution order:
//  1. WORKSPACE_SANDBOX_HOME_OVERLAY_BASE env var (operator override).
//  2. ${XDG_RUNTIME_DIR}/workspace-sandbox.
//  3. /var/tmp/workspace-sandbox-$UID (created mode 0700 if missing).
//
// Validated to be outside $HOME; returns an error otherwise. The directory
// is created (mode 0700) if missing.
//
// DOC: home-overlay storage seam. See docs/internal/SEAMS.md.
func ResolveHomeOverlayBaseDir() (string, error) {
	chosen := strings.TrimSpace(os.Getenv("WORKSPACE_SANDBOX_HOME_OVERLAY_BASE"))
	if chosen == "" {
		if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
			chosen = filepath.Join(runtimeDir, "workspace-sandbox")
		}
	}
	if chosen == "" {
		uid := os.Getuid()
		chosen = filepath.Join("/var/tmp", fmt.Sprintf("workspace-sandbox-%d", uid))
	}

	abs, err := filepath.Abs(chosen)
	if err != nil {
		return "", fmt.Errorf("resolve home overlay base dir %q: %w", chosen, err)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		homeAbs, err := filepath.Abs(home)
		if err == nil {
			rel, err := filepath.Rel(homeAbs, abs)
			if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
				return "", fmt.Errorf("home overlay base dir %q is inside $HOME (%q); pick a path outside $HOME via WORKSPACE_SANDBOX_HOME_OVERLAY_BASE to avoid a self-referential overlayfs mount", abs, homeAbs)
			}
			if rel == "." {
				return "", fmt.Errorf("home overlay base dir %q equals $HOME; pick a path outside $HOME via WORKSPACE_SANDBOX_HOME_OVERLAY_BASE", abs)
			}
		}
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", fmt.Errorf("create home overlay base dir %q: %w", abs, err)
	}
	return abs, nil
}

// Default returns a Config with sensible defaults.
// These defaults are safe for development and small deployments.
func Default() Config {
	return Config{
		Server: ServerConfig{
			ReadTimeout: 30 * time.Second,
			// 24h effectively disables the per-response write deadline
			// for the lifetime of an agent run. /processes/{pid}/logs/stream
			// is a long-lived SSE connection whose duration matches the
			// agent process lifetime; the previous 30s killed any
			// sandboxed agent run that exceeded that budget, surfacing
			// as SANDBOX_NO_EXIT_INFO upstream.
			//
			// We use 24h rather than 0 because the upstream api-core
			// server treats WriteTimeout=0 as "unset" and substitutes
			// its own 30s default. Per-handler context cancellation
			// (client disconnect, shutdown) still tears requests down
			// promptly. Long-running runs hitting 24h is not a real
			// concern for a single-tenant local dev service.
			WriteTimeout:       24 * time.Hour,
			IdleTimeout:        120 * time.Second,
			ShutdownTimeout:    10 * time.Second,
			CORSAllowedOrigins: nil, // Allow all
		},
		Limits: LimitsConfig{
			MaxSandboxes:     1000,
			MaxSandboxSizeMB: 10240,  // 10 GB
			MaxTotalSizeMB:   102400, // 100 GB
			DefaultListLimit: 100,
			MaxListLimit:     1000,
		},
		Lifecycle: LifecycleConfig{
			DefaultTTL:           24 * time.Hour,
			IdleTimeout:          4 * time.Hour,
			GCInterval:           15 * time.Minute,
			AutoCleanupTerminal:  true,
			TerminalCleanupDelay: 1 * time.Hour,
			ProcessGracePeriod:   100 * time.Millisecond,
			ProcessKillWait:      50 * time.Millisecond,
			AutoHealIdleGrace:    30 * time.Second,
			AutoHealMaxRetries:   5,
			AutoHealBaseBackoff:  30 * time.Second,
			ManualReviewTTL:      7 * 24 * time.Hour, // Decision D1: 7-day TTL from run end
		},
		Policy: PolicyConfig{
			DefaultNoLock:            true,
			CommitMessageTemplate:    "Apply sandbox changes ({{.FileCount}} files)",
			CommitAuthorMode:         "agent",
			ValidationHooks:          nil, // No hooks by default
			ValidationTimeout:        5 * time.Minute,
			BinaryDetectionThreshold: 8000,
			TeardownHooks:            nil, // No hooks by default
			// TeardownTimeout caps ALL pre-teardown hooks combined. Set to 90s to
			// give the per-hook budget (60s) room plus overhead for hook startup,
			// process metadata scanning, and logging. In teardown.go's nested
			// timeout model, each hook gets min(global, per-hook) time.
			//
			// If this timeout fires, the process.Starter context cancellation sends SIGKILL to the hook
			// process, which means affected scenarios may not be stopped before
			// unmount — they become orphaned with no filesystem. This is why the
			// timeout must be generous.
			TeardownTimeout: 90 * time.Second,
		},
		Driver: DriverConfig{
			BaseDir:          DefaultBaseDir(),
			UseFuseOverlayfs: false,
			// HomeOverlayBaseDir is resolved at LoadFromEnv() time so the
			// validation error surfaces during startup rather than as a
			// silent default difference between Default() and the running
			// service. Default() leaves it empty.
			HomeOverlayBaseDir: "",
		},
		Execution: ExecutionConfig{
			DefaultResourceLimits: ResourceLimitsConfig{
				// All zeros = unlimited by default
			},
			MaxResourceLimits: ResourceLimitsConfig{
				MemoryLimitMB: 16384, // 16 GB max
				CPUTimeSec:    3600,  // 1 hour max
				MaxProcesses:  1000,  // 1000 processes max
				MaxOpenFiles:  65536, // 64K files max
				TimeoutSec:    7200,  // 2 hours max
			},
			DefaultIsolationProfile: "full",
		},
		Database: DatabaseConfig{},
		Integration: IntegrationConfig{
			AgentManagerURL:         "",
			AgentManagerSyncEnabled: true,
			AgentManagerSyncTimeout: 5 * time.Second,
		},
		Retention: RetentionConfig{
			MaxArchiveAgeDays:     90,
			MaxArchiveSizeBytes:   10 * 1024 * 1024 * 1024, // 10 GiB
			MaxArchivesPerProject: 0,                       // 0 = unlimited
		},
	}
}

// LoadFromEnv loads configuration from environment variables.
// Environment variables override defaults where set.
// Uses WORKSPACE_SANDBOX_ prefix for clarity.
func LoadFromEnv() (Config, error) {
	cfg := Default()
	var errs []string

	// Server config
	cfg.Server.Port = requireEnv("API_PORT", &errs)
	cfg.Server.ReadTimeout = envDuration("WORKSPACE_SANDBOX_READ_TIMEOUT", cfg.Server.ReadTimeout)
	cfg.Server.WriteTimeout = envDuration("WORKSPACE_SANDBOX_WRITE_TIMEOUT", cfg.Server.WriteTimeout)
	cfg.Server.IdleTimeout = envDuration("WORKSPACE_SANDBOX_IDLE_TIMEOUT", cfg.Server.IdleTimeout)
	cfg.Server.ShutdownTimeout = envDuration("WORKSPACE_SANDBOX_SHUTDOWN_TIMEOUT", cfg.Server.ShutdownTimeout)

	if origins := os.Getenv("WORKSPACE_SANDBOX_CORS_ORIGINS"); origins != "" {
		cfg.Server.CORSAllowedOrigins = strings.Split(origins, ",")
	}

	// Limits config
	cfg.Limits.MaxSandboxes = envInt("WORKSPACE_SANDBOX_MAX_SANDBOXES", cfg.Limits.MaxSandboxes)
	cfg.Limits.MaxSandboxSizeMB = int64(envInt("WORKSPACE_SANDBOX_MAX_SIZE_MB", int(cfg.Limits.MaxSandboxSizeMB)))
	cfg.Limits.MaxTotalSizeMB = int64(envInt("WORKSPACE_SANDBOX_MAX_TOTAL_SIZE_MB", int(cfg.Limits.MaxTotalSizeMB)))
	cfg.Limits.DefaultListLimit = envInt("WORKSPACE_SANDBOX_DEFAULT_LIST_LIMIT", cfg.Limits.DefaultListLimit)
	cfg.Limits.MaxListLimit = envInt("WORKSPACE_SANDBOX_MAX_LIST_LIMIT", cfg.Limits.MaxListLimit)

	// Lifecycle config
	cfg.Lifecycle.DefaultTTL = envDuration("WORKSPACE_SANDBOX_DEFAULT_TTL", cfg.Lifecycle.DefaultTTL)
	cfg.Lifecycle.IdleTimeout = envDuration("WORKSPACE_SANDBOX_IDLE_TTL", cfg.Lifecycle.IdleTimeout)
	cfg.Lifecycle.GCInterval = envDuration("WORKSPACE_SANDBOX_GC_INTERVAL", cfg.Lifecycle.GCInterval)
	cfg.Lifecycle.AutoCleanupTerminal = envBool("WORKSPACE_SANDBOX_AUTO_CLEANUP_TERMINAL", cfg.Lifecycle.AutoCleanupTerminal)
	cfg.Lifecycle.TerminalCleanupDelay = envDuration("WORKSPACE_SANDBOX_TERMINAL_CLEANUP_DELAY", cfg.Lifecycle.TerminalCleanupDelay)
	cfg.Lifecycle.ProcessGracePeriod = envDuration("WORKSPACE_SANDBOX_PROCESS_GRACE_PERIOD", cfg.Lifecycle.ProcessGracePeriod)
	cfg.Lifecycle.ProcessKillWait = envDuration("WORKSPACE_SANDBOX_PROCESS_KILL_WAIT", cfg.Lifecycle.ProcessKillWait)
	cfg.Lifecycle.AutoHealIdleGrace = envDuration("WORKSPACE_SANDBOX_AUTOHEAL_IDLE_GRACE", cfg.Lifecycle.AutoHealIdleGrace)
	cfg.Lifecycle.AutoHealMaxRetries = envInt("WORKSPACE_SANDBOX_AUTOHEAL_MAX_RETRIES", cfg.Lifecycle.AutoHealMaxRetries)
	cfg.Lifecycle.AutoHealBaseBackoff = envDuration("WORKSPACE_SANDBOX_AUTOHEAL_BASE_BACKOFF", cfg.Lifecycle.AutoHealBaseBackoff)
	cfg.Lifecycle.ManualReviewTTL = envDuration("WORKSPACE_SANDBOX_MANUAL_REVIEW_TTL", cfg.Lifecycle.ManualReviewTTL)

	// Policy config
	cfg.Policy.BinaryDetectionThreshold = envInt("WORKSPACE_SANDBOX_BINARY_THRESHOLD", cfg.Policy.BinaryDetectionThreshold)
	cfg.Policy.DefaultNoLock = envBool("WORKSPACE_SANDBOX_DEFAULT_NO_LOCK", cfg.Policy.DefaultNoLock)
	if tmpl := os.Getenv("WORKSPACE_SANDBOX_COMMIT_TEMPLATE"); tmpl != "" {
		cfg.Policy.CommitMessageTemplate = tmpl
	}
	if mode := os.Getenv("WORKSPACE_SANDBOX_COMMIT_AUTHOR_MODE"); mode != "" {
		cfg.Policy.CommitAuthorMode = mode
	}

	// Teardown hook config
	//
	// Pre-teardown hooks run before sandbox unmount/delete to give external
	// systems a chance to evacuate processes from the merged directory. Hooks
	// are always best-effort: failures are logged but never block teardown.
	//
	// Priority:
	//   1. Explicit WORKSPACE_SANDBOX_TEARDOWN_HOOK_CMD env var
	//   2. Auto-detect: if "vrooli" is on PATH, use the built-in heal command
	//   3. No hooks (no-op)
	if hookCmd := os.Getenv("WORKSPACE_SANDBOX_TEARDOWN_HOOK_CMD"); hookCmd != "" {
		cfg.Policy.TeardownHooks = []TeardownHookConfig{{
			Name:        "pre-teardown",
			Description: "Configured via WORKSPACE_SANDBOX_TEARDOWN_HOOK_CMD",
			Command:     hookCmd,
		}}
	} else if vrooliPath, err := exec.LookPath("vrooli"); err == nil {
		// Auto-enable: when running inside a Vrooli environment, automatically
		// configure the heal-from-sandbox hook. This ensures scenarios running
		// from a sandbox's merged directory are gracefully restarted before the
		// overlay is torn down.
		//
		// The hook reads SANDBOX_MERGED_DIR from the environment (set by the
		// teardown policy) to find and restart affected scenarios.
		cfg.Policy.TeardownHooks = []TeardownHookConfig{{
			Name:        "vrooli-heal-from-sandbox",
			Description: "Stops scenarios running from sandbox merged path before teardown, then restarts them from the canonical repo in the background",
			Command:     vrooliPath,
			Args:        []string{"scenario", "heal-from-sandbox"},
			// Timeout budget: each scenario stop takes ~4s (SIGTERM + 2s grace +
			// SIGKILL + 1s cleanup in lifecycle.sh), so 60s supports ~15 scenarios
			// stopping sequentially. Restarts are backgrounded by heal.sh and don't
			// count against this budget.
			Timeout: 60 * time.Second,
		}}
	}
	cfg.Policy.TeardownTimeout = envDuration("WORKSPACE_SANDBOX_TEARDOWN_TIMEOUT", cfg.Policy.TeardownTimeout)

	// Driver config
	cfg.Driver.ProjectRoot = ResolveDefaultProjectRoot()
	if baseDir := os.Getenv("SANDBOX_BASE_DIR"); baseDir != "" {
		cfg.Driver.BaseDir = baseDir
	}
	cfg.Driver.UseFuseOverlayfs = envBool("WORKSPACE_SANDBOX_USE_FUSE", cfg.Driver.UseFuseOverlayfs)
	homeOverlayBase, err := ResolveHomeOverlayBaseDir()
	if err != nil {
		return cfg, err
	}
	cfg.Driver.HomeOverlayBaseDir = homeOverlayBase

	// Integration config
	cfg.Integration.AgentManagerURL = envString("WORKSPACE_SANDBOX_AGENT_MANAGER_URL", cfg.Integration.AgentManagerURL)
	cfg.Integration.AgentManagerSyncEnabled = envBool("WORKSPACE_SANDBOX_AGENT_MANAGER_SYNC_ENABLED", cfg.Integration.AgentManagerSyncEnabled)
	cfg.Integration.AgentManagerSyncTimeout = envDuration("WORKSPACE_SANDBOX_AGENT_MANAGER_SYNC_TIMEOUT", cfg.Integration.AgentManagerSyncTimeout)

	// Execution config - defaults
	cfg.Execution.DefaultResourceLimits.MemoryLimitMB = envInt("WORKSPACE_SANDBOX_DEFAULT_MEMORY_MB", cfg.Execution.DefaultResourceLimits.MemoryLimitMB)
	cfg.Execution.DefaultResourceLimits.CPUTimeSec = envInt("WORKSPACE_SANDBOX_DEFAULT_CPU_SEC", cfg.Execution.DefaultResourceLimits.CPUTimeSec)
	cfg.Execution.DefaultResourceLimits.MaxProcesses = envInt("WORKSPACE_SANDBOX_DEFAULT_MAX_PROCS", cfg.Execution.DefaultResourceLimits.MaxProcesses)
	cfg.Execution.DefaultResourceLimits.MaxOpenFiles = envInt("WORKSPACE_SANDBOX_DEFAULT_MAX_FILES", cfg.Execution.DefaultResourceLimits.MaxOpenFiles)
	cfg.Execution.DefaultResourceLimits.TimeoutSec = envInt("WORKSPACE_SANDBOX_DEFAULT_TIMEOUT_SEC", cfg.Execution.DefaultResourceLimits.TimeoutSec)

	// Execution config - maximums
	cfg.Execution.MaxResourceLimits.MemoryLimitMB = envInt("WORKSPACE_SANDBOX_MAX_MEMORY_MB", cfg.Execution.MaxResourceLimits.MemoryLimitMB)
	cfg.Execution.MaxResourceLimits.CPUTimeSec = envInt("WORKSPACE_SANDBOX_MAX_CPU_SEC", cfg.Execution.MaxResourceLimits.CPUTimeSec)
	cfg.Execution.MaxResourceLimits.MaxProcesses = envInt("WORKSPACE_SANDBOX_MAX_PROCS", cfg.Execution.MaxResourceLimits.MaxProcesses)
	cfg.Execution.MaxResourceLimits.MaxOpenFiles = envInt("WORKSPACE_SANDBOX_MAX_FILES", cfg.Execution.MaxResourceLimits.MaxOpenFiles)
	cfg.Execution.MaxResourceLimits.TimeoutSec = envInt("WORKSPACE_SANDBOX_MAX_TIMEOUT_SEC", cfg.Execution.MaxResourceLimits.TimeoutSec)

	if profile := os.Getenv("WORKSPACE_SANDBOX_DEFAULT_PROFILE"); profile != "" {
		cfg.Execution.DefaultIsolationProfile = profile
	}

	// Database config (SQLite path; falls back to api-core/storage resolver
	// when unset).
	cfg.Database.Path = os.Getenv("SQLITE_PATH")

	// Retention config (diff-archive retention reconciler).
	cfg.Retention.MaxArchiveAgeDays = envInt("WORKSPACE_SANDBOX_RETENTION_MAX_AGE_DAYS", cfg.Retention.MaxArchiveAgeDays)
	cfg.Retention.MaxArchiveSizeBytes = envInt64("WORKSPACE_SANDBOX_RETENTION_MAX_SIZE_BYTES", cfg.Retention.MaxArchiveSizeBytes)
	cfg.Retention.MaxArchivesPerProject = envInt("WORKSPACE_SANDBOX_RETENTION_MAX_PER_PROJECT", cfg.Retention.MaxArchivesPerProject)

	if len(errs) > 0 {
		return cfg, fmt.Errorf("missing required environment variables: %s", strings.Join(errs, ", "))
	}

	return cfg, cfg.Validate()
}

// Validate checks that all configuration values are within acceptable
// ranges, that mutually-exclusive options aren't both set, and that
// dependent options are coherent. Called from main.go after Load() so
// startup fails loudly when the operator hands us a self-contradictory
// configuration. Round 4 Phase 8 (2026-04-29) tightened the rule set
// to cover the dependencies the runtime quietly assumed before.
func (c *Config) Validate() error {
	var errs []string

	// --- Server ---
	if c.Server.Port == "" {
		errs = append(errs, "server.port is required")
	} else if p, err := strconv.Atoi(c.Server.Port); err != nil || p < 1 || p > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be a number in 1..65535 (got: %q)", c.Server.Port))
	}
	if c.Server.ReadTimeout < time.Second {
		errs = append(errs, "server.readTimeout must be at least 1s")
	}
	// server.writeTimeout=0 is allowed (and is the recommended setting
	// for this service) because /processes/{pid}/logs/stream needs to
	// stay open for the lifetime of long-running agent processes. Any
	// non-zero value must still clear 1s to avoid pathological configs.
	if c.Server.WriteTimeout != 0 && c.Server.WriteTimeout < time.Second {
		errs = append(errs, "server.writeTimeout must be 0 (disabled) or at least 1s")
	}
	if c.Server.IdleTimeout < time.Second {
		errs = append(errs, "server.idleTimeout must be at least 1s")
	}
	if c.Server.ShutdownTimeout < time.Second {
		errs = append(errs, "server.shutdownTimeout must be at least 1s")
	}

	// --- Limits ---
	if c.Limits.MaxSandboxes < 1 {
		errs = append(errs, "limits.maxSandboxes must be at least 1")
	}
	if c.Limits.MaxSandboxes > 100000 {
		errs = append(errs, "limits.maxSandboxes exceeds safe limit (100000)")
	}
	if c.Limits.MaxSandboxSizeMB < 1 {
		errs = append(errs, "limits.maxSandboxSizeMB must be at least 1")
	}
	if c.Limits.MaxTotalSizeMB < c.Limits.MaxSandboxSizeMB {
		errs = append(errs, "limits.maxTotalSizeMB must be >= maxSandboxSizeMB")
	}
	if c.Limits.MaxListLimit < 1 {
		errs = append(errs, "limits.maxListLimit must be at least 1")
	}
	if c.Limits.DefaultListLimit < 1 || c.Limits.DefaultListLimit > c.Limits.MaxListLimit {
		errs = append(errs, "limits.defaultListLimit must be between 1 and maxListLimit")
	}

	// --- Lifecycle ---
	if c.Lifecycle.DefaultTTL < time.Minute {
		errs = append(errs, "lifecycle.defaultTTL must be at least 1 minute")
	}
	if c.Lifecycle.IdleTimeout <= 0 {
		errs = append(errs, "lifecycle.idleTimeout must be greater than 0")
	}
	if c.Lifecycle.IdleTimeout > c.Lifecycle.DefaultTTL {
		errs = append(errs, "lifecycle.idleTimeout must be <= defaultTTL (idle reclaim cannot outlive the absolute TTL)")
	}
	if c.Lifecycle.GCInterval < time.Minute {
		errs = append(errs, "lifecycle.gcInterval must be at least 1 minute")
	}
	if c.Lifecycle.AutoHealMaxRetries < 1 {
		errs = append(errs, "lifecycle.autoHealMaxRetries must be at least 1")
	}
	if c.Lifecycle.AutoHealBaseBackoff <= 0 {
		errs = append(errs, "lifecycle.autoHealBaseBackoff must be greater than 0")
	}
	if c.Lifecycle.AutoHealIdleGrace < 0 {
		errs = append(errs, "lifecycle.autoHealIdleGrace must be >= 0")
	}
	if c.Lifecycle.ProcessGracePeriod <= 0 {
		errs = append(errs, "lifecycle.processGracePeriod must be greater than 0")
	}
	if c.Lifecycle.ProcessKillWait <= 0 {
		errs = append(errs, "lifecycle.processKillWait must be greater than 0")
	}
	if c.Lifecycle.ManualReviewTTL < 0 {
		errs = append(errs, "lifecycle.manualReviewTTL must be >= 0 (0 disables expiry)")
	}
	// Mutual exclusion: AutoCleanupTerminal=false makes TerminalCleanupDelay
	// meaningless. Allow zero (the no-op value); reject non-zero so
	// operators can't be misled into thinking the delay is honored.
	if !c.Lifecycle.AutoCleanupTerminal && c.Lifecycle.TerminalCleanupDelay > 0 {
		errs = append(errs, "lifecycle.terminalCleanupDelay must be 0 when autoCleanupTerminal is false (delay is unused)")
	}

	// --- Policy ---
	validAuthorModes := map[string]bool{"agent": true, "reviewer": true, "coauthored": true}
	if !validAuthorModes[c.Policy.CommitAuthorMode] {
		errs = append(errs, fmt.Sprintf("policy.commitAuthorMode must be one of: agent, reviewer, coauthored (got: %s)", c.Policy.CommitAuthorMode))
	}
	if c.Policy.CommitMessageTemplate == "" {
		errs = append(errs, "policy.commitMessageTemplate is required")
	}
	if c.Policy.BinaryDetectionThreshold < 1 {
		errs = append(errs, "policy.binaryDetectionThreshold must be at least 1")
	}
	if c.Policy.ValidationTimeout <= 0 {
		errs = append(errs, "policy.validationTimeout must be greater than 0")
	}
	if c.Policy.TeardownTimeout <= 0 {
		errs = append(errs, "policy.teardownTimeout must be greater than 0")
	}

	// --- Driver ---
	if c.Driver.BaseDir == "" {
		errs = append(errs, "driver.baseDir is required")
	}
	// HomeOverlayBaseDir is resolved at LoadFromEnv() time via
	// ResolveHomeOverlayBaseDir, which fails fatally if the path lands
	// inside $HOME. Default() leaves it empty so structural validation
	// here doesn't require it; the operator-facing failure point is
	// LoadFromEnv, not Validate.

	// --- Execution ---
	// Per-resource: when both default and max are set, the default
	// cannot exceed the max — otherwise zero-valued requests get
	// clamped down by ApplyResourceLimitDefaults to a value below the
	// configured default, which is operator-confusing.
	def := c.Execution.DefaultResourceLimits
	mx := c.Execution.MaxResourceLimits
	if mx.MemoryLimitMB > 0 && def.MemoryLimitMB > mx.MemoryLimitMB {
		errs = append(errs, "execution.defaultResourceLimits.memoryLimitMB must be <= maxResourceLimits.memoryLimitMB")
	}
	if mx.CPUTimeSec > 0 && def.CPUTimeSec > mx.CPUTimeSec {
		errs = append(errs, "execution.defaultResourceLimits.cpuTimeSec must be <= maxResourceLimits.cpuTimeSec")
	}
	if mx.MaxProcesses > 0 && def.MaxProcesses > mx.MaxProcesses {
		errs = append(errs, "execution.defaultResourceLimits.maxProcesses must be <= maxResourceLimits.maxProcesses")
	}
	if mx.MaxOpenFiles > 0 && def.MaxOpenFiles > mx.MaxOpenFiles {
		errs = append(errs, "execution.defaultResourceLimits.maxOpenFiles must be <= maxResourceLimits.maxOpenFiles")
	}
	if mx.TimeoutSec > 0 && def.TimeoutSec > mx.TimeoutSec {
		errs = append(errs, "execution.defaultResourceLimits.timeoutSec must be <= maxResourceLimits.timeoutSec")
	}
	if c.Execution.DefaultIsolationProfile == "" {
		errs = append(errs, "execution.defaultIsolationProfile is required (typically \"full\")")
	}

	// --- Integration ---
	// AgentManagerURL is allowed to be empty when sync is enabled —
	// the Service falls back to discovery.ResolveScenarioURLDefault at
	// request time (see internal/sandbox/service_audit.go::resolveAgentManagerURL).
	// We therefore do NOT require URL when sync is enabled; the
	// runtime fallback is the documented contract.
	if c.Integration.AgentManagerSyncTimeout < 0 {
		errs = append(errs, "integration.agentManagerSyncTimeout must be >= 0")
	}

	// --- Retention ---
	// Each lever is independently optional (0 disables). Negatives are
	// always invalid because they have no meaningful interpretation.
	if c.Retention.MaxArchiveAgeDays < 0 {
		errs = append(errs, "retention.maxArchiveAgeDays must be >= 0 (0 disables age-based eviction)")
	}
	if c.Retention.MaxArchiveSizeBytes < 0 {
		errs = append(errs, "retention.maxArchiveSizeBytes must be >= 0 (0 disables size-based eviction)")
	}
	if c.Retention.MaxArchivesPerProject < 0 {
		errs = append(errs, "retention.maxArchivesPerProject must be >= 0 (0 disables the per-project cap)")
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errs, "; "))
	}

	return nil
}

// --- Helper functions ---

func requireEnv(key string, errs *[]string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		*errs = append(*errs, key)
	}
	return value
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func envInt64(key string, defaultVal int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func envString(key string, defaultVal string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultVal
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultVal
}
