// Package config provides configuration loading and management.
//
// Configuration follows a hierarchical loading pattern:
// 1. Built-in defaults
// 2. Configuration file (if present)
// 3. Environment variables (override file values)
//
// This package is a seam that isolates configuration mechanics from
// business logic, enabling testing with controlled configurations.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for agent-manager.
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Sandbox configuration
	Sandbox SandboxConfig

	// Runner configuration
	Runners RunnerConfig

	// Policy defaults
	Policy PolicyDefaults
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// SandboxConfig holds workspace-sandbox integration settings.
type SandboxConfig struct {
	URL             string
	DefaultTimeout  time.Duration
	ProjectRoot     string
}

// RunnerConfig holds runner settings.
type RunnerConfig struct {
	ClaudeCodePath string
	CodexPath      string
	OpenCodePath   string
	DefaultTimeout time.Duration
	MaxTurns       int
}

// PolicyDefaults holds default policy values.
type PolicyDefaults struct {
	RequireSandbox       bool
	RequireApproval      bool
	MaxConcurrentRuns    int
	MaxConcurrentPerScope int
	DefaultLockTTL       time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         envString("API_PORT", "8080"),
			ReadTimeout:  envDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: envDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  envDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			URL:             envString("DATABASE_URL", ""),
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Sandbox: SandboxConfig{
			URL:            envString("WORKSPACE_SANDBOX_URL", "http://localhost:8081"),
			DefaultTimeout: envDuration("SANDBOX_DEFAULT_TIMEOUT", 30*time.Minute),
			ProjectRoot:    envString("PROJECT_ROOT", ""),
		},
		Runners: RunnerConfig{
			ClaudeCodePath: envString("CLAUDE_CODE_PATH", "claude"),
			CodexPath:      envString("CODEX_PATH", "codex"),
			OpenCodePath:   envString("OPENCODE_PATH", "opencode"),
			DefaultTimeout: envDuration("RUNNER_DEFAULT_TIMEOUT", 30*time.Minute),
			MaxTurns:       envInt("RUNNER_MAX_TURNS", 100),
		},
		Policy: PolicyDefaults{
			RequireSandbox:        envBool("POLICY_REQUIRE_SANDBOX", true),
			RequireApproval:       envBool("POLICY_REQUIRE_APPROVAL", true),
			MaxConcurrentRuns:     envInt("POLICY_MAX_CONCURRENT_RUNS", 10),
			MaxConcurrentPerScope: envInt("POLICY_MAX_CONCURRENT_PER_SCOPE", 1),
			DefaultLockTTL:        envDuration("POLICY_DEFAULT_LOCK_TTL", 30*time.Minute),
		},
	}
}

// envString returns environment variable or default.
func envString(key, defaultVal string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultVal
}

// envInt returns environment variable as int or default.
func envInt(key string, defaultVal int) int {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// envBool returns environment variable as bool or default.
func envBool(key string, defaultVal bool) bool {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

// envDuration returns environment variable as duration or default.
func envDuration(key string, defaultVal time.Duration) time.Duration {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
