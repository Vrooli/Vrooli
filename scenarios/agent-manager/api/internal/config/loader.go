// Package config provides configuration loading and management.
//
// This file handles loading Levers from various sources (environment, files).

package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadLevers loads configuration with the following precedence (highest first):
// 1. Environment variables (override everything)
// 2. Config file (if specified via AGENT_MANAGER_CONFIG)
// 3. Default values
func LoadLevers() (*Levers, error) {
	// Start with defaults
	levers := DefaultLevers()

	// Load from config file if specified
	if configPath := os.Getenv("AGENT_MANAGER_CONFIG"); configPath != "" {
		if err := loadFromFile(&levers, configPath); err != nil {
			return nil, err
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(&levers)

	// Validate final configuration
	if err := levers.Validate(); err != nil {
		return nil, err
	}

	return &levers, nil
}

// loadFromFile reads configuration from a JSON file.
func loadFromFile(levers *Levers, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, levers)
}

// applyEnvOverrides applies environment variable overrides to levers.
// Environment variables use the pattern: AGENT_MANAGER_<CATEGORY>_<LEVER>
func applyEnvOverrides(l *Levers) {
	// Execution levers
	if v := getEnvDuration("AGENT_MANAGER_EXECUTION_DEFAULT_TIMEOUT"); v > 0 {
		l.Execution.DefaultTimeout = v
	}
	if v := getEnvInt("AGENT_MANAGER_EXECUTION_DEFAULT_MAX_TURNS"); v > 0 {
		l.Execution.DefaultMaxTurns = v
	}
	if v := getEnvInt("AGENT_MANAGER_EXECUTION_EVENT_BUFFER_SIZE"); v > 0 {
		l.Execution.EventBufferSize = v
	}
	if v := getEnvDuration("AGENT_MANAGER_EXECUTION_EVENT_FLUSH_INTERVAL"); v > 0 {
		l.Execution.EventFlushInterval = v
	}

	// Safety levers
	if v, ok := envBoolOpt("AGENT_MANAGER_SAFETY_REQUIRE_SANDBOX"); ok {
		l.Safety.RequireSandboxByDefault = v
	}
	if v, ok := envBoolOpt("AGENT_MANAGER_SAFETY_ALLOW_IN_PLACE_OVERRIDE"); ok {
		l.Safety.AllowInPlaceOverride = v
	}
	if v := getEnvInt("AGENT_MANAGER_SAFETY_MAX_FILES_PER_RUN"); v > 0 {
		l.Safety.MaxFilesPerRun = v
	}
	if v := envInt64("AGENT_MANAGER_SAFETY_MAX_BYTES_PER_RUN"); v > 0 {
		l.Safety.MaxBytesPerRun = v
	}
	if v := envStringList("AGENT_MANAGER_SAFETY_DENY_PATH_PATTERNS"); len(v) > 0 {
		l.Safety.DenyPathPatterns = v
	}

	// Concurrency levers
	if v := getEnvInt("AGENT_MANAGER_CONCURRENCY_MAX_CONCURRENT_RUNS"); v > 0 {
		l.Concurrency.MaxConcurrentRuns = v
	}
	if v := getEnvInt("AGENT_MANAGER_CONCURRENCY_MAX_CONCURRENT_PER_SCOPE"); v > 0 {
		l.Concurrency.MaxConcurrentPerScope = v
	}
	if v := getEnvDuration("AGENT_MANAGER_CONCURRENCY_SCOPE_LOCK_TTL"); v > 0 {
		l.Concurrency.ScopeLockTTL = v
	}
	if v := getEnvDuration("AGENT_MANAGER_CONCURRENCY_SCOPE_LOCK_REFRESH_INTERVAL"); v > 0 {
		l.Concurrency.ScopeLockRefreshInterval = v
	}
	if v := getEnvDuration("AGENT_MANAGER_CONCURRENCY_QUEUE_WAIT_TIMEOUT"); v >= 0 {
		l.Concurrency.QueueWaitTimeout = v
	}

	// Approval levers
	if v, ok := envBoolOpt("AGENT_MANAGER_APPROVAL_REQUIRE_APPROVAL"); ok {
		l.Approval.RequireApprovalByDefault = v
	}
	if v := envStringList("AGENT_MANAGER_APPROVAL_AUTO_APPROVE_PATTERNS"); len(v) > 0 {
		l.Approval.AutoApprovePatterns = v
	}
	if v := getEnvInt("AGENT_MANAGER_APPROVAL_REVIEW_TIMEOUT_DAYS"); v > 0 {
		l.Approval.ReviewTimeoutDays = v
	}
	if v, ok := envBoolOpt("AGENT_MANAGER_APPROVAL_ALLOW_PARTIAL"); ok {
		l.Approval.AllowPartialApproval = v
	}

	// Runner levers
	if v := getEnv("AGENT_MANAGER_RUNNERS_CLAUDE_CODE_PATH"); v != "" {
		l.Runners.ClaudeCodePath = v
	}
	if v := getEnv("AGENT_MANAGER_RUNNERS_CODEX_PATH"); v != "" {
		l.Runners.CodexPath = v
	}
	if v := getEnv("AGENT_MANAGER_RUNNERS_OPENCODE_PATH"); v != "" {
		l.Runners.OpenCodePath = v
	}
	if v := getEnvDuration("AGENT_MANAGER_RUNNERS_HEALTH_CHECK_INTERVAL"); v > 0 {
		l.Runners.HealthCheckInterval = v
	}
	if v := getEnvDuration("AGENT_MANAGER_RUNNERS_STARTUP_GRACE_PERIOD"); v >= 0 {
		l.Runners.StartupGracePeriod = v
	}

	// Server levers
	if v := getEnv("AGENT_MANAGER_SERVER_PORT"); v != "" {
		l.Server.Port = v
	}
	// Also check legacy API_PORT for compatibility
	if v := getEnv("API_PORT"); v != "" {
		l.Server.Port = v
	}
	if v := getEnvDuration("AGENT_MANAGER_SERVER_READ_TIMEOUT"); v > 0 {
		l.Server.ReadTimeout = v
	}
	if v := getEnvDuration("AGENT_MANAGER_SERVER_WRITE_TIMEOUT"); v > 0 {
		l.Server.WriteTimeout = v
	}
	if v := getEnvDuration("AGENT_MANAGER_SERVER_IDLE_TIMEOUT"); v > 0 {
		l.Server.IdleTimeout = v
	}
	if v := envInt64("AGENT_MANAGER_SERVER_MAX_REQUEST_BODY_BYTES"); v > 0 {
		l.Server.MaxRequestBodyBytes = v
	}

	// Storage levers
	if v := getEnv("AGENT_MANAGER_STORAGE_DATABASE_URL"); v != "" {
		l.Storage.DatabaseURL = v
	}
	// Also check legacy DATABASE_URL for compatibility
	if v := getEnv("DATABASE_URL"); v != "" {
		l.Storage.DatabaseURL = v
	}
	if v := getEnvInt("AGENT_MANAGER_STORAGE_MAX_OPEN_CONNS"); v > 0 {
		l.Storage.MaxOpenConns = v
	}
	if v := getEnvInt("AGENT_MANAGER_STORAGE_MAX_IDLE_CONNS"); v > 0 {
		l.Storage.MaxIdleConns = v
	}
	if v := getEnvDuration("AGENT_MANAGER_STORAGE_CONN_MAX_LIFETIME"); v > 0 {
		l.Storage.ConnMaxLifetime = v
	}
	if v := getEnvInt("AGENT_MANAGER_STORAGE_EVENT_RETENTION_DAYS"); v > 0 {
		l.Storage.EventRetentionDays = v
	}
	if v := getEnvInt("AGENT_MANAGER_STORAGE_ARTIFACT_RETENTION_DAYS"); v > 0 {
		l.Storage.ArtifactRetentionDays = v
	}
}

// =============================================================================
// ENVIRONMENT HELPERS FOR LEVERS (renamed to avoid conflict with config.go)
// =============================================================================

func getEnv(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func getEnvInt(key string) int {
	if val := getEnv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func envInt64(key string) int64 {
	if val := getEnv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func envBoolOpt(key string) (bool, bool) {
	if val := getEnv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b, true
		}
	}
	return false, false
}

func getEnvDuration(key string) time.Duration {
	if val := getEnv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return 0
}

func envStringList(key string) []string {
	if val := getEnv(key); val != "" {
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return nil
}

// =============================================================================
// PROFILES (Common Configuration Patterns)
// =============================================================================

// Profile represents a named configuration preset.
type Profile string

const (
	// ProfileDevelopment is optimized for local development.
	// More lenient timeouts, faster health checks, smaller limits.
	ProfileDevelopment Profile = "development"

	// ProfileProduction is optimized for production usage.
	// Stricter limits, longer timeouts, full safety defaults.
	ProfileProduction Profile = "production"

	// ProfileTesting is optimized for automated testing.
	// Fast timeouts, minimal concurrency, deterministic behavior.
	ProfileTesting Profile = "testing"
)

// LeversForProfile returns configuration optimized for the given profile.
func LeversForProfile(profile Profile) Levers {
	base := DefaultLevers()

	switch profile {
	case ProfileDevelopment:
		base.Execution.DefaultTimeout = 10 * time.Minute
		base.Concurrency.MaxConcurrentRuns = 3
		base.Runners.HealthCheckInterval = 30 * time.Second
		base.Server.WriteTimeout = 5 * time.Minute // Allow long debug sessions
		base.Storage.EventRetentionDays = 7
		base.Storage.ArtifactRetentionDays = 7

	case ProfileTesting:
		base.Execution.DefaultTimeout = 1 * time.Minute
		base.Execution.DefaultMaxTurns = 10
		base.Concurrency.MaxConcurrentRuns = 2
		base.Concurrency.QueueWaitTimeout = 10 * time.Second
		base.Runners.HealthCheckInterval = 10 * time.Second
		base.Runners.StartupGracePeriod = 5 * time.Second
		base.Server.ReadTimeout = 5 * time.Second
		base.Server.WriteTimeout = 10 * time.Second
		base.Storage.EventRetentionDays = 1
		base.Storage.ArtifactRetentionDays = 1

	case ProfileProduction:
		// Production uses defaults, which are already conservative
		break
	}

	return base
}
