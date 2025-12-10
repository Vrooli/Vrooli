// Package containment provides OS-level sandboxing for agent execution.
package containment

import (
	"time"

	"test-genie/internal/shared"
)

// Config holds tunable levers for containment behavior.
// These control the tradeoff between security isolation and execution flexibility.
//
// # Environment Variables
//
// These can be set to override defaults without code changes:
//
//   - CONTAINMENT_DOCKER_IMAGE: Docker image for agent execution (default: ubuntu:22.04)
//   - CONTAINMENT_MAX_MEMORY_MB: Memory limit in MB (default: 2048)
//   - CONTAINMENT_MAX_CPU_PERCENT: CPU limit as percentage (default: 200 = 2 cores)
//   - CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS: Docker check timeout (default: 5)
//   - CONTAINMENT_PREFER_DOCKER: Whether to prefer Docker when available (default: true)
type Config struct {
	// --- Container Image ---

	// DockerImage specifies the container image for Docker-based containment.
	// Should be a minimal image with development tools the agents need.
	// Use a specific tag (not 'latest') for reproducibility.
	// Default: "ubuntu:22.04"
	DockerImage string

	// --- Resource Limits ---

	// MaxMemoryMB limits memory available to contained processes.
	// Higher = agents can handle larger codebases/operations.
	// Lower = better isolation, prevents runaway agents from exhausting memory.
	// Range: 256-16384 MB. Default: 2048 MB (2GB).
	MaxMemoryMB int

	// MaxCPUPercent limits CPU usage as a percentage of one core.
	// 100 = 1 core, 200 = 2 cores, etc.
	// Higher = faster execution but more resource pressure.
	// Lower = gentler on resources but slower execution.
	// Range: 50-800 (0.5 to 8 cores). Default: 200 (2 cores).
	MaxCPUPercent int

	// --- Availability & Selection ---

	// AvailabilityTimeoutSeconds controls how long to wait when checking
	// if Docker or other containment methods are available.
	// Higher = more tolerant of slow Docker daemons.
	// Lower = faster startup when Docker is unavailable.
	// Range: 1-30 seconds. Default: 5 seconds.
	AvailabilityTimeoutSeconds int

	// PreferDocker controls whether Docker is preferred over other providers
	// when multiple are available. Set to false to prefer bubblewrap on Linux.
	// Default: true.
	PreferDocker bool

	// AllowFallback controls whether to allow execution without containment
	// when no sandbox is available. Set to false to fail instead of running unsandboxed.
	// Default: true (for backward compatibility).
	AllowFallback bool

	// --- Security Hardening ---

	// DropAllCapabilities controls whether to drop all Linux capabilities in Docker.
	// Provides stronger isolation but may break some operations.
	// Default: true.
	DropAllCapabilities bool

	// NoNewPrivileges prevents privilege escalation inside containers.
	// Should almost always be true for security.
	// Default: true.
	NoNewPrivileges bool

	// ReadOnlyRootFS makes the container root filesystem read-only.
	// Provides extra security but requires writable working directory.
	// Default: false (agents need to write test files).
	ReadOnlyRootFS bool
}

// DefaultConfig returns a Config with production-ready defaults.
func DefaultConfig() Config {
	return Config{
		DockerImage:                "ubuntu:22.04",
		MaxMemoryMB:                2048,
		MaxCPUPercent:              200,
		AvailabilityTimeoutSeconds: 5,
		PreferDocker:               true,
		AllowFallback:              true,
		DropAllCapabilities:        true,
		NoNewPrivileges:            true,
		ReadOnlyRootFS:             false,
	}
}

// LoadConfigFromEnv loads configuration from environment variables,
// falling back to defaults for unset values.
func LoadConfigFromEnv() Config {
	cfg := DefaultConfig()

	cfg.DockerImage = shared.EnvString("CONTAINMENT_DOCKER_IMAGE", cfg.DockerImage)
	cfg.MaxMemoryMB = shared.ClampInt(shared.EnvInt("CONTAINMENT_MAX_MEMORY_MB", cfg.MaxMemoryMB), 256, 16384)
	cfg.MaxCPUPercent = shared.ClampInt(shared.EnvInt("CONTAINMENT_MAX_CPU_PERCENT", cfg.MaxCPUPercent), 50, 800)
	cfg.AvailabilityTimeoutSeconds = shared.ClampInt(shared.EnvInt("CONTAINMENT_AVAILABILITY_TIMEOUT_SECONDS", cfg.AvailabilityTimeoutSeconds), 1, 30)
	cfg.PreferDocker = shared.EnvBool("CONTAINMENT_PREFER_DOCKER", cfg.PreferDocker)
	cfg.AllowFallback = shared.EnvBool("CONTAINMENT_ALLOW_FALLBACK", cfg.AllowFallback)
	cfg.DropAllCapabilities = shared.EnvBool("CONTAINMENT_DROP_ALL_CAPABILITIES", cfg.DropAllCapabilities)
	cfg.NoNewPrivileges = shared.EnvBool("CONTAINMENT_NO_NEW_PRIVILEGES", cfg.NoNewPrivileges)
	cfg.ReadOnlyRootFS = shared.EnvBool("CONTAINMENT_READ_ONLY_ROOT_FS", cfg.ReadOnlyRootFS)

	return cfg
}

// AvailabilityTimeout returns the availability timeout as a duration.
func (c Config) AvailabilityTimeout() time.Duration {
	return time.Duration(c.AvailabilityTimeoutSeconds) * time.Second
}

// Validate checks for configuration consistency.
func (c Config) Validate() error {
	// Currently all constraints are enforced via clamping in LoadConfigFromEnv
	return nil
}

// ValidationResult captures what adjustments were made during config loading.
// This provides visibility into implicit changes operators should be aware of.
type ValidationResult struct {
	// Adjustments contains human-readable descriptions of any clamping or corrections.
	Adjustments []string
	// Warnings contains non-fatal issues operators should be aware of.
	Warnings []string
}

// HasChanges returns true if any adjustments or warnings were recorded.
func (r ValidationResult) HasChanges() bool {
	return len(r.Adjustments) > 0 || len(r.Warnings) > 0
}

// ValidateWithReport checks configuration and reports any issues.
// This provides visibility into security posture and potential concerns.
func (c Config) ValidateWithReport() ValidationResult {
	result := ValidationResult{}

	// Security warnings
	if c.AllowFallback {
		result.Warnings = append(result.Warnings,
			"AllowFallback is enabled. Agents may run without containment if Docker is unavailable.")
	}

	if !c.DropAllCapabilities {
		result.Warnings = append(result.Warnings,
			"DropAllCapabilities is disabled. Containers retain default Linux capabilities.")
	}

	if !c.NoNewPrivileges {
		result.Warnings = append(result.Warnings,
			"NoNewPrivileges is disabled. Privilege escalation inside containers is possible.")
	}

	// Resource warnings
	if c.MaxMemoryMB > 8192 {
		result.Warnings = append(result.Warnings,
			"MaxMemoryMB is very high (>8GB). Agents can consume substantial memory.")
	}

	if c.MaxCPUPercent > 400 {
		result.Warnings = append(result.Warnings,
			"MaxCPUPercent is very high (>4 cores). Agents can consume substantial CPU.")
	}

	// Image warnings
	if c.DockerImage == "ubuntu:latest" || c.DockerImage == "alpine:latest" {
		result.Warnings = append(result.Warnings,
			"Using 'latest' tag for Docker image. Consider using a specific tag for reproducibility.")
	}

	return result
}
