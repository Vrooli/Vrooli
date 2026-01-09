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
}

// PolicyConfig controls approval and attribution rules.
type PolicyConfig struct {
	// RequireHumanApproval controls whether human approval is required
	// before applying sandbox changes to the canonical repo.
	// Default: true
	RequireHumanApproval bool

	// AutoApproveThresholdFiles is the maximum number of changed files
	// that can be auto-approved (when RequireHumanApproval is false).
	// Set to 0 for no limit.
	// Default: 10
	AutoApproveThresholdFiles int

	// AutoApproveThresholdLines is the maximum number of changed lines
	// that can be auto-approved (when RequireHumanApproval is false).
	// Set to 0 for no limit.
	// Default: 500
	AutoApproveThresholdLines int

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

// DriverConfig controls filesystem driver settings.
type DriverConfig struct {
	// BaseDir is the root directory for sandbox artifacts.
	// Default: ~/.local/share/workspace-sandbox (XDG-compliant, user-writable)
	BaseDir string

	// UseFuseOverlayfs enables fuse-overlayfs instead of kernel overlayfs.
	// Enables unprivileged operation but may be slower.
	// Default: false
	UseFuseOverlayfs bool

	// ProjectRoot is the default project root for sandboxes.
	// Set via PROJECT_ROOT env var, falls back to VROOLI_ROOT if not set.
	// If both are empty, must be specified per-request.
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

// DatabaseConfig controls database connection settings.
type DatabaseConfig struct {
	// URL is the full database connection URL.
	// If set, overrides individual host/port/user/password/name settings.
	URL string

	// Host is the database server hostname.
	Host string

	// Port is the database server port.
	Port string

	// User is the database username.
	User string

	// Password is the database password.
	Password string

	// Name is the database name.
	Name string

	// Schema is the PostgreSQL schema to use.
	// Default: workspace-sandbox
	Schema string

	// SSLMode controls SSL connection mode.
	// Default: disable
	SSLMode string
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

// Default returns a Config with sensible defaults.
// These defaults are safe for development and small deployments.
func Default() Config {
	return Config{
		Server: ServerConfig{
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       30 * time.Second,
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
		},
		Policy: PolicyConfig{
			RequireHumanApproval:      true,
			AutoApproveThresholdFiles: 10,
			AutoApproveThresholdLines: 500,
			CommitMessageTemplate:     "Apply sandbox changes ({{.FileCount}} files)",
			CommitAuthorMode:          "agent",
			ValidationHooks:           nil, // No hooks by default
			ValidationTimeout:         5 * time.Minute,
			BinaryDetectionThreshold:  8000,
		},
		Driver: DriverConfig{
			BaseDir:          DefaultBaseDir(),
			UseFuseOverlayfs: false,
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
		Database: DatabaseConfig{
			Schema:  "workspace-sandbox",
			SSLMode: "disable",
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

	// Policy config
	cfg.Policy.RequireHumanApproval = envBool("WORKSPACE_SANDBOX_REQUIRE_HUMAN_APPROVAL", cfg.Policy.RequireHumanApproval)
	cfg.Policy.AutoApproveThresholdFiles = envInt("WORKSPACE_SANDBOX_AUTO_APPROVE_FILES", cfg.Policy.AutoApproveThresholdFiles)
	cfg.Policy.AutoApproveThresholdLines = envInt("WORKSPACE_SANDBOX_AUTO_APPROVE_LINES", cfg.Policy.AutoApproveThresholdLines)
	cfg.Policy.BinaryDetectionThreshold = envInt("WORKSPACE_SANDBOX_BINARY_THRESHOLD", cfg.Policy.BinaryDetectionThreshold)
	if tmpl := os.Getenv("WORKSPACE_SANDBOX_COMMIT_TEMPLATE"); tmpl != "" {
		cfg.Policy.CommitMessageTemplate = tmpl
	}
	if mode := os.Getenv("WORKSPACE_SANDBOX_COMMIT_AUTHOR_MODE"); mode != "" {
		cfg.Policy.CommitAuthorMode = mode
	}

	// Driver config
	// PROJECT_ROOT takes precedence, falls back to VROOLI_ROOT if not set
	cfg.Driver.ProjectRoot = os.Getenv("PROJECT_ROOT")
	if cfg.Driver.ProjectRoot == "" {
		cfg.Driver.ProjectRoot = os.Getenv("VROOLI_ROOT")
	}
	if baseDir := os.Getenv("SANDBOX_BASE_DIR"); baseDir != "" {
		cfg.Driver.BaseDir = baseDir
	}
	cfg.Driver.UseFuseOverlayfs = envBool("WORKSPACE_SANDBOX_USE_FUSE", cfg.Driver.UseFuseOverlayfs)

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

	// Database config
	cfg.Database.URL = os.Getenv("DATABASE_URL")
	cfg.Database.Host = os.Getenv("POSTGRES_HOST")
	cfg.Database.Port = os.Getenv("POSTGRES_PORT")
	cfg.Database.User = os.Getenv("POSTGRES_USER")
	cfg.Database.Password = os.Getenv("POSTGRES_PASSWORD")
	cfg.Database.Name = os.Getenv("POSTGRES_DB")
	if schema := os.Getenv("POSTGRES_SCHEMA"); schema != "" {
		cfg.Database.Schema = schema
	}

	if len(errs) > 0 {
		return cfg, fmt.Errorf("missing required environment variables: %s", strings.Join(errs, ", "))
	}

	return cfg, cfg.Validate()
}

// Validate checks that all configuration values are within acceptable ranges.
func (c *Config) Validate() error {
	var errs []string

	// Server validation
	if c.Server.Port == "" {
		errs = append(errs, "server.port is required")
	}
	if c.Server.ReadTimeout < time.Second {
		errs = append(errs, "server.readTimeout must be at least 1s")
	}
	if c.Server.WriteTimeout < time.Second {
		errs = append(errs, "server.writeTimeout must be at least 1s")
	}

	// Limits validation
	if c.Limits.MaxSandboxes < 1 {
		errs = append(errs, "limits.maxSandboxes must be at least 1")
	}
	if c.Limits.MaxSandboxes > 100000 {
		errs = append(errs, "limits.maxSandboxes exceeds safe limit (100000)")
	}
	if c.Limits.MaxSandboxSizeMB < 1 {
		errs = append(errs, "limits.maxSandboxSizeMB must be at least 1")
	}
	if c.Limits.DefaultListLimit < 1 || c.Limits.DefaultListLimit > c.Limits.MaxListLimit {
		errs = append(errs, "limits.defaultListLimit must be between 1 and maxListLimit")
	}

	// Lifecycle validation
	if c.Lifecycle.DefaultTTL < time.Minute {
		errs = append(errs, "lifecycle.defaultTTL must be at least 1 minute")
	}
	if c.Lifecycle.GCInterval < time.Minute {
		errs = append(errs, "lifecycle.gcInterval must be at least 1 minute")
	}

	// Policy validation
	validAuthorModes := map[string]bool{"agent": true, "reviewer": true, "coauthored": true}
	if !validAuthorModes[c.Policy.CommitAuthorMode] {
		errs = append(errs, fmt.Sprintf("policy.commitAuthorMode must be one of: agent, reviewer, coauthored (got: %s)", c.Policy.CommitAuthorMode))
	}

	// Driver validation
	if c.Driver.BaseDir == "" {
		errs = append(errs, "driver.baseDir is required")
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
