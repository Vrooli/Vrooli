// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// This package orchestrates service lifecycle using domain packages:
//   - infra/     - Infrastructure abstractions (clock, filesystem, network, process, env)
//   - ports/     - Dynamic port allocation
//   - secrets/   - Secret management
//   - health/    - Health and readiness monitoring
//   - gpu/       - GPU detection and requirements
//   - assets/    - Asset verification
//   - env/       - Environment variable templating
//   - migrations/ - Migration state tracking
//   - telemetry/ - Event recording
//   - errors/    - Structured error types
package bundleruntime

import (
	"context"

	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/ports"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/telemetry"
)

// Infrastructure types - re-exported from infra/
type (
	Clock             = infra.Clock
	Ticker            = infra.Ticker
	FileSystem        = infra.FileSystem
	File              = infra.File
	NetworkDialer     = infra.NetworkDialer
	HTTPClient        = infra.HTTPClient
	ProcessRunner     = infra.ProcessRunner
	Process           = infra.Process
	CommandRunner     = infra.CommandRunner
	EnvReader         = infra.EnvReader
)

// Real implementations - re-exported from infra/
type (
	RealClock         = infra.RealClock
	RealFileSystem    = infra.RealFileSystem
	RealNetworkDialer = infra.RealNetworkDialer
	RealHTTPClient    = infra.RealHTTPClient
	RealProcessRunner = infra.RealProcessRunner
	RealCommandRunner = infra.RealCommandRunner
	RealEnvReader     = infra.RealEnvReader
)

// Port allocation - re-exported from ports/
type (
	PortAllocator = ports.Allocator
	PortRange     = ports.Range
)

// Secret management - re-exported from secrets/
type (
	SecretStore = secrets.Store
)

// Health monitoring - re-exported from health/
type (
	HealthChecker = health.Checker
	ServiceStatus = health.Status
)

// GPU detection - re-exported from gpu/
type (
	GPUDetector = gpu.Detector
	GPUStatus   = gpu.Status
)

// Telemetry - re-exported from telemetry/
type (
	TelemetryRecorder = telemetry.Recorder
)

// Environment rendering - re-exported from env/
type (
	EnvRenderer = env.Renderer
)

// MigrationsState tracks applied migrations per service and the current app version.
type MigrationsState struct {
	AppVersion string              `json:"app_version"`
	Applied    map[string][]string `json:"applied"` // service ID -> list of applied migration versions
}

// ServiceManager provides a high-level interface for service lifecycle management.
// This interface can be used by external code (like Electron) to interact with the runtime.
type ServiceManager interface {
	// Start initializes the runtime and begins service orchestration.
	Start(ctx context.Context) error
	// Shutdown gracefully stops all services.
	Shutdown(ctx context.Context) error
	// IsStarted returns whether the runtime has been started.
	IsStarted() bool
	// AllServicesReady returns true if all services are ready.
	AllServicesReady() bool
	// ServiceStatuses returns the current status of all services.
	ServiceStatuses() map[string]ServiceStatus
	// UpdateSecrets merges new secrets and triggers service startup if ready.
	UpdateSecrets(secrets map[string]string) error
	// GPUStatus returns GPU detection results.
	GPUStatus() GPUStatus
	// PortMap returns allocated ports for all services.
	PortMap() map[string]map[string]int
	// AppDataDir returns the application data directory.
	AppDataDir() string
	// AuthToken returns the control API authentication token.
	AuthToken() string
}

// Signal constants for cross-platform compatibility - re-exported from infra/
var (
	Interrupt = infra.Interrupt
	Kill      = infra.Kill
)

// DefaultPortRange is the fallback range when not specified in manifest.
var DefaultPortRange = ports.DefaultRange

// Factory functions

// NewPortManager creates a new PortManager with the given dependencies.
func NewPortManager(m *Manifest, dialer NetworkDialer) *ports.Manager {
	return ports.NewManager(m, dialer)
}

// NewSecretManager creates a new SecretManager with the given dependencies.
func NewSecretManager(m *Manifest, fs FileSystem, secretsPath string) *secrets.Manager {
	return secrets.NewManager(m, fs, secretsPath)
}

// NewHealthMonitor creates a new HealthMonitor with the given dependencies.
func NewHealthMonitor(cfg health.MonitorConfig) *health.Monitor {
	return health.NewMonitor(cfg)
}

// NewEnvRenderer creates a new environment variable renderer.
func NewEnvRenderer(appData, bundlePath string, portAlloc PortAllocator, envReader EnvReader) *env.Renderer {
	return env.NewRenderer(appData, bundlePath, portAlloc, envReader)
}

// NewTelemetryRecorder creates a new telemetry file recorder.
func NewTelemetryRecorder(path string, clock Clock, fs FileSystem) *telemetry.FileRecorder {
	return telemetry.NewFileRecorder(path, clock, fs)
}

// NewGPUDetector creates a new GPU detector.
func NewGPUDetector(cmdRunner CommandRunner, envReader EnvReader) *gpu.RealDetector {
	return gpu.NewDetector(cmdRunner, envReader)
}
