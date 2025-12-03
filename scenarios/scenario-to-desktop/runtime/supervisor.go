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
//	  api/                - HTTP handlers and authentication middleware
//	  service_launcher.go - Service lifecycle management
//	  types.go            - Re-exports and ServiceManager interface
//	  strutil/            - String/map utilities (CopyStringMap, EnvMapToList, Intersection)
//	  fileutil/           - File utilities (TailFile)
//	  deps/               - Dependency resolution (TopoSort, FindService)
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

	"scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/assets"
	"scenario-to-desktop-runtime/config"
	"scenario-to-desktop-runtime/env"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/migrations"
	"scenario-to-desktop-runtime/ports"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/telemetry"
)

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

// Ensure Supervisor implements api.Runtime.
var _ api.Runtime = (*Supervisor)(nil)

// =============================================================================
// Supervisor Configuration
// =============================================================================

// Options is an alias for config.Options for backward compatibility.
// See config.Options for documentation.
type Options = config.Options

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
	secretStore   secrets.Store
	healthChecker HealthChecker
	telemetry     telemetry.Recorder

	// Cached domain objects (created once, reused).
	envRenderer       *env.Renderer
	assetVerifier     *assets.Verifier
	gpuApplier        *gpu.Applier
	migrationExecutor migrations.Runner

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
		appData = filepath.Join(base, config.SanitizeAppName(opts.Manifest.App.Name))
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
		portAllocator = ports.NewManager(opts.Manifest, dialer)
	}

	// Compute paths for secret manager.
	secretsPath := filepath.Join(appData, "secrets.json")

	// Create or use provided SecretStore.
	secretStore := opts.SecretStore
	if secretStore == nil {
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
		// Create envRenderer now since all dependencies are available.
		envRenderer: env.NewRenderer(appData, opts.BundlePath, portAllocator, envReader),
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

	// Create migration executor with tracker.
	tracker := migrations.NewTracker(s.migrationsPath, s.fs)
	migrationState, err := tracker.Load()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	s.migrationExecutor = migrations.NewExecutor(
		migrations.ExecutorConfig{
			BundlePath: s.opts.BundlePath,
			AppVersion: s.opts.Manifest.App.Version,
			Tracker:    tracker,
			ProcRunner: s.procRunner,
			Telemetry:  s.telemetry,
		},
		s.envRenderer,
		s, // Supervisor implements LogProvider
	)
	s.migrationExecutor.SetState(migrationState)
	s.migrations = migrationState

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

	// Create cached domain objects now that telemetry and GPU status are available.
	s.assetVerifier = assets.NewVerifier(s.opts.BundlePath, s.fs, s.telemetry)
	s.gpuApplier = gpu.NewApplier(s.gpuStatus, s.telemetry)

	// Set up HTTP server using the API package.
	apiServer := api.NewServer(s, s.authToken)
	mux := http.NewServeMux()
	apiServer.RegisterHandlers(mux)

	addr := fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, s.opts.Manifest.IPC.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: apiServer.AuthMiddleware(mux),
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
	missing := s.secretStore.MissingRequired()
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

// TelemetryPath returns the telemetry file path.
func (s *Supervisor) TelemetryPath() string {
	return s.telemetryPath
}

// TelemetryUploadURL returns the telemetry upload URL.
func (s *Supervisor) TelemetryUploadURL() string {
	return s.opts.Manifest.Telemetry.UploadTo
}

// Manifest returns the bundle manifest.
func (s *Supervisor) Manifest() *manifest.Manifest {
	return s.opts.Manifest
}

// FileSystem returns the file system abstraction.
func (s *Supervisor) FileSystem() infra.FileSystem {
	return s.fs
}

// SecretStore returns the secret store for API interactions.
func (s *Supervisor) SecretStore() api.SecretStore {
	return s.secretStore
}

// StartServicesIfReady triggers service startup if secrets are ready.
func (s *Supervisor) StartServicesIfReady() {
	if !s.servicesStarted {
		s.startServicesAsync()
	}
}

// RecordTelemetry records a telemetry event (public interface for api package).
func (s *Supervisor) RecordTelemetry(event string, details map[string]interface{}) error {
	return s.recordTelemetry(event, details)
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
// Secret Management
// =============================================================================

// applySecrets injects secrets into the environment for a service.
func (s *Supervisor) applySecrets(env map[string]string, svc manifest.Service) error {
	injector := secrets.NewInjector(s.secretStore, s.fs, s.appData)
	return injector.Apply(env, svc)
}

// UpdateSecrets merges new secrets and persists them.
// Triggers service startup if all required secrets are now available.
func (s *Supervisor) UpdateSecrets(newSecrets map[string]string) error {
	merged := s.secretStore.Merge(newSecrets)

	missing := s.secretStore.MissingRequiredFrom(merged)
	if len(missing) > 0 {
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		return fmt.Errorf("missing required secrets: %s", strings.Join(missing, ", "))
	}

	if err := s.secretStore.Persist(merged); err != nil {
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

// renderEnvMap builds the environment variable map for a service.
func (s *Supervisor) renderEnvMap(svc manifest.Service, bin manifest.Binary) (map[string]string, error) {
	return s.envRenderer.RenderEnvMap(svc, bin)
}

// renderArgs expands template variables in command arguments.
func (s *Supervisor) renderArgs(args []string) []string {
	return s.envRenderer.RenderArgs(args)
}

// renderValue expands template variables in a string.
func (s *Supervisor) renderValue(input string) string {
	return s.envRenderer.RenderValue(input)
}

// GPUStatus returns the current GPU detection status.
func (s *Supervisor) GPUStatus() GPUStatus {
	return s.gpuStatus
}

// PortMap returns allocated ports for all services.
func (s *Supervisor) PortMap() map[string]map[string]int {
	return s.portAllocator.Map()
}

