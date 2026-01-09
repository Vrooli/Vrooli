// Package config provides configuration types for the bundle runtime.
package config

import (
	"strings"

	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/ports"
	"scenario-to-desktop-runtime/secrets"
)

// Options configures the Supervisor.
type Options struct {
	AppDataDir string             // Override for app data directory (default: user config dir)
	Manifest   *manifest.Manifest // Bundle manifest (required)
	BundlePath string             // Root path of the unpacked bundle
	DryRun     bool               // Skip actual service launches (for testing)

	// Injectable dependencies (nil = use real implementations)
	Clock         infra.Clock         // Time operations (default: RealClock)
	FileSystem    infra.FileSystem    // File operations (default: RealFileSystem)
	NetworkDialer infra.NetworkDialer // Network operations (default: RealNetworkDialer)
	ProcessRunner infra.ProcessRunner // Process execution (default: RealProcessRunner)
	CommandRunner infra.CommandRunner // Command execution (default: RealCommandRunner)
	GPUDetector   gpu.Detector        // GPU detection (default: RealGPUDetector)
	EnvReader     infra.EnvReader     // Environment variable access (default: RealEnvReader)
	PortAllocator ports.Allocator     // Port allocation (default: PortManager)
	SecretStore   secrets.Store       // Secret management (default: SecretManager)
	HealthChecker health.Checker      // Health monitoring (default: HealthMonitor)
}

// SanitizeAppName normalizes an application name for filesystem use.
func SanitizeAppName(name string) string {
	out := strings.TrimSpace(name)
	if out == "" {
		return "desktop-app"
	}
	out = strings.ReplaceAll(out, " ", "-")
	out = strings.ToLower(out)
	return out
}
