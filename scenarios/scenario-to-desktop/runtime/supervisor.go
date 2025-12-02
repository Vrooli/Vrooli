// Package bundleruntime provides the desktop bundle runtime supervisor.
//
// The supervisor is responsible for:
//   - Loading and validating the bundle manifest
//   - Allocating ports for services
//   - Managing secrets and migrations
//   - Starting services in dependency order
//   - Monitoring service health and readiness
//   - Exposing a control API for Electron/CLI interaction
//   - Recording telemetry events
//
// Architecture (Screaming Architecture):
//
//	Domain Packages:
//	  infra/       - Infrastructure abstractions (clock, filesystem, network, process)
//	  ports/       - Dynamic port allocation
//	  secrets/     - Secret management and injection
//	  health/      - Health and readiness monitoring
//	  gpu/         - GPU detection and requirements
//	  assets/      - Asset verification and Playwright conventions
//	  env/         - Environment variable templating
//	  migrations/  - Migration state tracking
//	  telemetry/   - Event recording
//	  errors/      - Structured error types
//	  manifest/    - Bundle manifest parsing
//
//	Orchestration Layer (this package):
//	  supervisor.go       - Core Supervisor struct, Start, Shutdown
//	  control_api.go      - HTTP handlers and authentication middleware
//	  service_launcher.go - Service lifecycle management
//	  interfaces.go       - Re-exports and ServiceManager interface
//	  utils.go            - Shared utilities (topoSort, findService, etc.)
//
// See README.md for detailed documentation and architecture diagrams.
package bundleruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/ports"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/telemetry"
)

// =============================================================================
// Type Aliases (re-exports from domain packages)
// =============================================================================

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

// Signal constants for cross-platform compatibility - re-exported from infra/
var (
	Interrupt = infra.Interrupt
	Kill      = infra.Kill
)

// DefaultPortRange is the fallback range when not specified in manifest.
var DefaultPortRange = ports.DefaultRange

// =============================================================================
// ServiceManager Interface
// =============================================================================

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

// Ensure Supervisor implements ServiceManager.
var _ ServiceManager = (*Supervisor)(nil)

// =============================================================================
// Supervisor Configuration
// =============================================================================

// Options configures the Supervisor.
type Options struct {
	AppDataDir string             // Override for app data directory (default: user config dir)
	Manifest   *manifest.Manifest // Bundle manifest (required)
	BundlePath string             // Root path of the unpacked bundle
	DryRun     bool               // Skip actual service launches (for testing)

	// Injectable dependencies (nil = use real implementations)
	Clock         Clock         // Time operations (default: RealClock)
	FileSystem    FileSystem    // File operations (default: RealFileSystem)
	NetworkDialer NetworkDialer // Network operations (default: RealNetworkDialer)
	ProcessRunner ProcessRunner // Process execution (default: RealProcessRunner)
	CommandRunner CommandRunner // Command execution (default: RealCommandRunner)
	GPUDetector   GPUDetector   // GPU detection (default: RealGPUDetector)
	EnvReader     EnvReader     // Environment variable access (default: RealEnvReader)
	PortAllocator PortAllocator // Port allocation (default: PortManager)
	SecretStore   SecretStore   // Secret management (default: SecretManager)
	HealthChecker HealthChecker // Health monitoring (default: HealthMonitor)
}

// Supervisor manages the desktop bundle runtime.
// It orchestrates service startup, health monitoring, and exposes a control API.
type Supervisor struct {
	opts Options

	// Injected dependencies.
	clock         Clock
	fs            FileSystem
	dialer        NetworkDialer
	procRunner    ProcessRunner
	cmdRunner     CommandRunner
	gpuDetector   GPUDetector
	envReader     EnvReader
	portAllocator PortAllocator
	secretStore   *secrets.Manager // Using concrete type for extended methods
	healthChecker HealthChecker
	telemetry     telemetry.Recorder

	// Paths and auth.
	authToken      string
	appData        string
	telemetryPath  string
	migrationsPath string

	// Runtime state.
	serviceStatus   map[string]ServiceStatus
	procs           map[string]*serviceProcess
	migrations      MigrationsState
	gpuStatus       GPUStatus
	servicesStarted bool
	started         bool

	// HTTP server.
	server *http.Server

	// Concurrency control.
	mu         sync.RWMutex
	wg         sync.WaitGroup
	cancel     context.CancelFunc
	runtimeCtx context.Context
}

// Manifest is an alias for the manifest package type for convenience.
type Manifest = manifest.Manifest

// sanitizeAppName normalizes an application name for filesystem use.
func sanitizeAppName(name string) string {
	out := strings.TrimSpace(name)
	if out == "" {
		return "desktop-app"
	}
	out = strings.ReplaceAll(out, " ", "-")
	out = strings.ToLower(out)
	return out
}

// NewSupervisor creates a new Supervisor with the given options.
// Injectable dependencies are set to real implementations when nil.
func NewSupervisor(opts Options) (*Supervisor, error) {
	if opts.Manifest == nil {
		return nil, errors.New("manifest is required")
	}

	appData := opts.AppDataDir
	if appData == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("resolve app data dir: %w", err)
		}
		appData = filepath.Join(base, sanitizeAppName(opts.Manifest.App.Name))
	}

	// Set default implementations for nil dependencies.
	clock := opts.Clock
	if clock == nil {
		clock = RealClock{}
	}

	fileSystem := opts.FileSystem
	if fileSystem == nil {
		fileSystem = RealFileSystem{}
	}

	dialer := opts.NetworkDialer
	if dialer == nil {
		dialer = RealNetworkDialer{}
	}

	procRunner := opts.ProcessRunner
	if procRunner == nil {
		procRunner = infra.RealProcessRunner{}
	}

	cmdRunner := opts.CommandRunner
	if cmdRunner == nil {
		cmdRunner = infra.RealCommandRunner{}
	}

	envReader := opts.EnvReader
	if envReader == nil {
		envReader = infra.RealEnvReader{}
	}

	gpuDetector := opts.GPUDetector
	if gpuDetector == nil {
		gpuDetector = gpu.NewDetector(cmdRunner, envReader)
	}

	portAllocator := opts.PortAllocator
	if portAllocator == nil {
		portAllocator = NewPortManager(opts.Manifest, dialer)
	}

	// Compute paths for secret manager.
	secretsPath := filepath.Join(appData, "secrets.json")

	// Create or use provided SecretStore.
	var secretStore *secrets.Manager
	if opts.SecretStore != nil {
		// If a SecretStore was provided, it must be a *secrets.Manager for extended methods.
		var ok bool
		secretStore, ok = opts.SecretStore.(*secrets.Manager)
		if !ok {
			return nil, errors.New("SecretStore must be a *secrets.Manager")
		}
	} else {
		secretStore = secrets.NewManager(opts.Manifest, fileSystem, secretsPath)
	}

	s := &Supervisor{
		opts:          opts,
		clock:         clock,
		fs:            fileSystem,
		dialer:        dialer,
		procRunner:    procRunner,
		cmdRunner:     cmdRunner,
		gpuDetector:   gpuDetector,
		envReader:     envReader,
		portAllocator: portAllocator,
		secretStore:   secretStore,
		appData:       appData,
		serviceStatus: make(map[string]ServiceStatus),
		procs:         make(map[string]*serviceProcess),
	}

	// Create or use provided HealthChecker.
	if opts.HealthChecker != nil {
		s.healthChecker = opts.HealthChecker
	} else {
		s.healthChecker = health.NewMonitor(health.MonitorConfig{
			Manifest:     opts.Manifest,
			Ports:        portAllocator,
			Dialer:       dialer,
			CmdRunner:    cmdRunner,
			FS:           fileSystem,
			Clock:        clock,
			AppData:      appData,
			StatusGetter: s.getStatus, // Closure captures supervisor
		})
	}

	return s, nil
}

// Start initializes the supervisor and begins service orchestration.
// It sets up the control API, loads secrets and migrations, allocates ports,
// and starts services asynchronously once all required secrets are available.
func (s *Supervisor) Start(ctx context.Context) error {
	// Create app data directory.
	if err := s.fs.MkdirAll(s.appData, 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}

	// Set up paths.
	s.telemetryPath = manifest.ResolvePath(s.appData, s.opts.Manifest.Telemetry.File)
	s.migrationsPath = filepath.Join(s.appData, "migrations.json")

	if err := s.fs.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return fmt.Errorf("create telemetry dir: %w", err)
	}

	// Initialize telemetry recorder.
	s.telemetry = telemetry.NewFileRecorder(s.telemetryPath, s.clock, s.fs)

	// Load persisted state.
	loadedSecrets, err := s.secretStore.Load()
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}
	s.secretStore.Set(loadedSecrets)

	migrations, err := s.loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	s.migrations = migrations

	// Load or create auth token.
	tokenPath := manifest.ResolvePath(s.appData, s.opts.Manifest.IPC.AuthTokenRel)
	token, err := s.loadOrCreateToken(tokenPath)
	if err != nil {
		return fmt.Errorf("load auth token: %w", err)
	}
	s.authToken = token

	// Allocate ports.
	if err := s.portAllocator.Allocate(); err != nil {
		return err
	}

	// Initialize service status.
	for _, svc := range s.opts.Manifest.Services {
		s.serviceStatus[svc.ID] = ServiceStatus{Ready: false, Message: "pending start"}
	}

	// Record startup telemetry.
	if err := s.recordTelemetry("runtime_start", nil); err != nil {
		return fmt.Errorf("write telemetry: %w", err)
	}

	// Detect GPU.
	s.gpuStatus = s.gpuDetector.Detect()
	_ = s.recordTelemetry("gpu_status", map[string]interface{}{
		"available": s.gpuStatus.Available,
		"method":    s.gpuStatus.Method,
		"reason":    s.gpuStatus.Reason,
	})

	// Set up HTTP server.
	mux := http.NewServeMux()
	s.registerHandlers(mux)

	addr := fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, s.opts.Manifest.IPC.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: s.authMiddleware(mux),
	}
	s.server = server

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("start control API on %s: %w", addr, err)
	}

	s.started = true

	// Create runtime context.
	runtimeCtx, cancel := context.WithCancel(ctx)
	s.runtimeCtx = runtimeCtx
	s.cancel = cancel

	// Check for missing secrets.
	missing := s.missingRequiredSecrets()
	if len(missing) > 0 {
		msg := fmt.Sprintf("waiting for secrets: %s", strings.Join(missing, ", "))
		for _, svc := range s.opts.Manifest.Services {
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg})
		}
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
	} else {
		s.startServicesAsync()
	}

	// Watch for context cancellation.
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	// Start HTTP server.
	go func() {
		if serveErr := server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			_ = s.recordTelemetry("runtime_error", map[string]interface{}{"error": serveErr.Error()})
		}
	}()

	return nil
}

// Shutdown gracefully stops all services and the control API.
func (s *Supervisor) Shutdown(ctx context.Context) error {
	if !s.started {
		return nil
	}
	s.started = false

	if s.cancel != nil {
		s.cancel()
	}

	// Stop services in reverse dependency order.
	s.stopServices(ctx)
	s.wg.Wait()

	_ = s.recordTelemetry("runtime_shutdown", nil)

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// setStatus updates the status for a service.
func (s *Supervisor) setStatus(id string, status ServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceStatus[id] = status
}

// getStatus retrieves the status for a service.
func (s *Supervisor) getStatus(id string) (ServiceStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.serviceStatus[id]
	return st, ok
}

// setProc stores a service process reference.
func (s *Supervisor) setProc(id string, proc *serviceProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.procs[id] = proc
}

// getProc retrieves a service process reference.
func (s *Supervisor) getProc(id string) *serviceProcess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.procs[id]
}

// loadOrCreateToken loads an existing auth token or creates a new one.
func (s *Supervisor) loadOrCreateToken(path string) (string, error) {
	if data, err := s.fs.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}

	if err := s.fs.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := s.fs.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

// AppDataDir returns the resolved app data directory.
func (s *Supervisor) AppDataDir() string {
	return s.appData
}

// AuthToken returns the current auth token for the control API.
func (s *Supervisor) AuthToken() string {
	return s.authToken
}

// IsStarted returns whether the supervisor has been started.
func (s *Supervisor) IsStarted() bool {
	return s.started
}

// AllServicesReady returns true if all services are ready.
func (s *Supervisor) AllServicesReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, st := range s.serviceStatus {
		if !st.Ready {
			return false
		}
	}
	return len(s.serviceStatus) > 0
}

// ServiceStatuses returns a copy of all service statuses.
func (s *Supervisor) ServiceStatuses() map[string]ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]ServiceStatus)
	for id, st := range s.serviceStatus {
		out[id] = st
	}
	return out
}

// recordTelemetry writes a telemetry event if the recorder is initialized.
func (s *Supervisor) recordTelemetry(event string, details map[string]interface{}) error {
	if s.telemetry == nil {
		return nil
	}
	return s.telemetry.Record(event, details)
}

// =============================================================================
// Secret Management (delegates to secrets package)
// =============================================================================

// secretsCopy returns a thread-safe copy of the current secrets.
func (s *Supervisor) secretsCopy() map[string]string {
	return s.secretStore.Get()
}

// missingRequiredSecrets returns a list of required secrets that are missing.
func (s *Supervisor) missingRequiredSecrets() []string {
	return s.secretStore.MissingRequired()
}

// missingRequiredSecretsFrom checks a secrets map for missing required values.
func (s *Supervisor) missingRequiredSecretsFrom(secretsMap map[string]string) []string {
	return s.secretStore.MissingRequiredFrom(secretsMap)
}

// persistSecrets saves secrets to the secrets file.
func (s *Supervisor) persistSecrets(secretsMap map[string]string) error {
	return s.secretStore.Persist(secretsMap)
}

// findSecret looks up a secret definition by ID.
func (s *Supervisor) findSecret(id string) *manifest.Secret {
	return s.secretStore.FindSecret(id)
}

// applySecrets injects secrets into the environment for a service.
func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	injector := secrets.NewInjector(s.secretStore, s.fs, s.appData)
	return injector.Apply(env, svc)
}

// UpdateSecrets merges new secrets and persists them.
// Triggers service startup if all required secrets are now available.
func (s *Supervisor) UpdateSecrets(newSecrets map[string]string) error {
	merged := s.secretStore.Merge(newSecrets)

	missing := s.missingRequiredSecretsFrom(merged)
	if len(missing) > 0 {
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}

	if err := s.persistSecrets(merged); err != nil {
		return err
	}

	s.secretStore.Set(merged)

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})

	// Trigger service startup if not already started.
	if !s.servicesStarted {
		s.startServicesAsync()
	}
	return nil
}

// =============================================================================
// Template Expansion (delegates to env package)
// =============================================================================

// envRenderer returns an environment renderer for the current supervisor state.
func (s *Supervisor) envRenderer() *env.Renderer {
	return env.NewRenderer(s.appData, s.opts.BundlePath, s.portAllocator, s.envReader)
}

// renderEnvMap builds the environment variable map for a service.
func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	return s.envRenderer().RenderEnvMap(svc, bin)
}

// renderArgs expands template variables in command arguments.
func (s *Supervisor) renderArgs(args []string) []string {
	return s.envRenderer().RenderArgs(args)
}

// renderValue expands template variables in a string.
func (s *Supervisor) renderValue(input string) string {
	return s.envRenderer().RenderValue(input)
}

// GPUStatus returns the current GPU detection status.
func (s *Supervisor) GPUStatus() GPUStatus {
	return s.gpuStatus
}

// PortMap returns allocated ports for all services.
func (s *Supervisor) PortMap() map[string]map[string]int {
	return s.portAllocator.Map()
}

// =============================================================================
// Factory Functions
// =============================================================================

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
