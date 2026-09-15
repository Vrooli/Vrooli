// Package config provides configuration types for the bundle runtime.
package config

import (
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/gpu"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/health"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/ports"
	resourceplan "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/resources"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/strutil"
)

// Options configures the Supervisor.
type Options struct {
	AppDataDir  string             // Override for app data directory (default: user config dir)
	PeerHomeDir string             // Override for the shared ~/.vrooli/peers registry root (tests only)
	Manifest    *manifest.Manifest // Bundle manifest (required)
	BundlePath  string             // Root path of the unpacked bundle
	DryRun      bool               // Skip actual service launches (for testing)
	// SharedResourceResolver is optional and may be supplied only after explicit
	// user consent. Nil preserves the private bundled-service default.
	SharedResourceResolver resourceplan.SharedServiceResolver

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
	return strutil.SanitizeAppName(name)
}
