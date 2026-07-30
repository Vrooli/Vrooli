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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/assets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/config"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/env"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/gpu"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/health"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/migrations"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/ports"
	resourceplan "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/resources"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/telemetry"
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
	instanceID     string
	manifestHash   string
	startedAt      time.Time
	resourcePlan   *resourceplan.Plan
	resourceServer *resourceplan.ServiceSupervisor

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
	resourcePlan, err := loadResourcePlan(opts.BundlePath)
	if err != nil {
		return nil, err
	}
	appData, err := resolveAppDataDir(opts)
	if err != nil {
		return nil, err
	}
	deps := resolveDependencies(opts, appData)

	s := &Supervisor{
		opts:          opts,
		clock:         deps.clock,
		fs:            deps.fileSystem,
		dialer:        deps.dialer,
		procRunner:    deps.procRunner,
		cmdRunner:     deps.cmdRunner,
		gpuDetector:   deps.gpuDetector,
		envReader:     deps.envReader,
		portAllocator: deps.portAllocator,
		secretStore:   deps.secretStore,
		appData:       appData,
		serviceStatus: make(map[string]ServiceStatus),
		procs:         make(map[string]*serviceProcess),
		instanceID:    newInstanceID(),
		manifestHash:  hashManifest(opts.Manifest),
		resourcePlan:  resourcePlan,
		// Create envRenderer now since all dependencies are available.
		envRenderer: env.NewRenderer(appData, opts.BundlePath, deps.portAllocator, deps.envReader),
	}

	s.healthChecker = resolveHealthChecker(opts, deps, appData, s.getStatus)

	return s, nil
}

type runtimeDependencies struct {
	clock         Clock
	fileSystem    FileSystem
	dialer        NetworkDialer
	procRunner    ProcessRunner
	cmdRunner     CommandRunner
	gpuDetector   GPUDetector
	envReader     EnvReader
	portAllocator PortAllocator
	secretStore   secrets.Store
}

func loadResourcePlan(bundlePath string) (*resourceplan.Plan, error) {
	if bundlePath == "" {
		return nil, nil
	}
	plan, err := resourceplan.Load(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("validate resolved resource deployment plan: %w", err)
	}
	return plan, nil
}

func resolveAppDataDir(opts Options) (string, error) {
	if opts.AppDataDir != "" {
		return opts.AppDataDir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve app data dir: %w", err)
	}
	return filepath.Join(base, config.SanitizeAppName(opts.Manifest.App.Name)), nil
}

func resolveDependencies(opts Options, appData string) runtimeDependencies {
	deps := runtimeDependencies{clock: opts.Clock, fileSystem: opts.FileSystem, dialer: opts.NetworkDialer, procRunner: opts.ProcessRunner, cmdRunner: opts.CommandRunner, gpuDetector: opts.GPUDetector, envReader: opts.EnvReader, portAllocator: opts.PortAllocator, secretStore: opts.SecretStore}
	if deps.clock == nil {
		deps.clock = RealClock{}
	}
	if deps.fileSystem == nil {
		deps.fileSystem = RealFileSystem{}
	}
	if deps.dialer == nil {
		deps.dialer = RealNetworkDialer{}
	}
	if deps.procRunner == nil {
		deps.procRunner = infra.RealProcessRunner{}
	}
	if deps.cmdRunner == nil {
		deps.cmdRunner = infra.RealCommandRunner{}
	}
	if deps.envReader == nil {
		deps.envReader = infra.RealEnvReader{}
	}
	if deps.gpuDetector == nil {
		deps.gpuDetector = gpu.NewDetector(deps.cmdRunner, deps.envReader)
	}
	if deps.portAllocator == nil {
		deps.portAllocator = ports.NewManager(opts.Manifest, deps.dialer)
	}
	if deps.secretStore == nil {
		nativeStore, err := secrets.NewNativeManager(opts.Manifest)
		if err != nil {
			deps.secretStore = secrets.NewUnavailableManager(opts.Manifest, err)
		} else {
			deps.secretStore = nativeStore
		}
	}
	return deps
}

func resolveHealthChecker(opts Options, deps runtimeDependencies, appData string, statusGetter func(string) (health.Status, bool)) HealthChecker {
	if opts.HealthChecker != nil {
		return opts.HealthChecker
	}
	return health.NewMonitor(health.MonitorConfig{Manifest: opts.Manifest, Ports: deps.portAllocator, Dialer: deps.dialer, CmdRunner: deps.cmdRunner, FS: deps.fileSystem, Clock: deps.clock, AppData: appData, StatusGetter: statusGetter})
}

// Start initializes the supervisor and begins service orchestration.
// It sets up the control API, loads secrets and migrations, allocates ports,
// and starts services asynchronously once all required secrets are available.
func (s *Supervisor) Start(ctx context.Context) error {
	if err := s.initPaths(); err != nil {
		return err
	}

	if err := s.initSecrets(); err != nil {
		return err
	}

	if err := s.initMigrations(); err != nil {
		return err
	}

	if err := s.initAuthAndPorts(); err != nil {
		return err
	}

	// Initialize service status.
	for _, svc := range s.opts.Manifest.Services {
		s.serviceStatus[svc.ID] = ServiceStatus{Ready: false, Message: "pending start"}
	}

	if err := s.recordTelemetry("runtime_start", nil); err != nil {
		return fmt.Errorf("write telemetry: %w", err)
	}

	s.initGPUAndDomainObjects()

	ln, err := s.startControlAPI()
	if err != nil {
		return err
	}

	s.started = true
	s.startedAt = s.clock.Now()

	runtimeCtx, cancel := context.WithCancel(ctx)
	s.runtimeCtx = runtimeCtx
	s.cancel = cancel
	if err := s.startBundledResources(runtimeCtx); err != nil {
		cancel()
		return err
	}

	s.triggerServicesOrWait()

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer shutdownCancel()
		_ = s.Shutdown(shutdownCtx)
	}()
	go func() {
		if serveErr := s.server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			_ = s.recordTelemetry("runtime_error", map[string]interface{}{"error": serveErr.Error()})
		}
	}()

	return nil
}

// initPaths creates directories and sets up telemetry.
func (s *Supervisor) initPaths() error {
	if err := s.fs.MkdirAll(s.appData, 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}

	s.telemetryPath = manifest.ResolvePath(s.appData, s.opts.Manifest.Telemetry.File)
	s.migrationsPath = filepath.Join(s.appData, "migrations.json")

	if err := s.fs.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return fmt.Errorf("create telemetry dir: %w", err)
	}

	s.telemetry = telemetry.NewFileRecorder(s.telemetryPath, s.clock, s.fs)
	return nil
}

// initSecrets loads persisted secrets and auto-generates missing ones.
func (s *Supervisor) initSecrets() error {
	loadedSecrets, err := s.secretStore.Load()
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}

	generated, genErr := s.secretStore.GenerateMissing(loadedSecrets)
	if genErr != nil {
		_ = s.recordTelemetry("secrets_generation_failed", map[string]interface{}{"error": genErr.Error()})
		return fmt.Errorf("generate secrets: %w", genErr)
	}
	if len(generated) > 0 {
		generatedIDs := make([]string, 0, len(generated))
		for k, v := range generated {
			loadedSecrets[k] = v
			generatedIDs = append(generatedIDs, k)
		}
		if persistErr := s.secretStore.Persist(loadedSecrets); persistErr != nil {
			return fmt.Errorf("persist generated secrets: %w", persistErr)
		}
		_ = s.recordTelemetry("secrets_generated", map[string]interface{}{
			"count": len(generated),
			"ids":   generatedIDs,
		})
	}

	s.secretStore.Set(loadedSecrets)
	return nil
}

// initMigrations sets up the migration executor and loads migration state.
func (s *Supervisor) initMigrations() error {
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
		s,
	)
	s.migrationExecutor.SetState(migrationState)
	s.migrations = migrationState
	return nil
}

// initAuthAndPorts loads the auth token and allocates ports.
func (s *Supervisor) initAuthAndPorts() error {
	tokenPath := manifest.ResolvePath(s.appData, s.opts.Manifest.IPC.AuthTokenRel)
	token, err := s.loadOrCreateToken(tokenPath)
	if err != nil {
		return fmt.Errorf("load auth token: %w", err)
	}
	s.authToken = token

	return s.portAllocator.Allocate()
}

// initGPUAndDomainObjects detects GPU and creates cached domain objects.
func (s *Supervisor) initGPUAndDomainObjects() {
	s.gpuStatus = s.gpuDetector.Detect()
	_ = s.recordTelemetry("gpu_status", map[string]interface{}{
		"available": s.gpuStatus.Available,
		"method":    s.gpuStatus.Method,
		"reason":    s.gpuStatus.Reason,
	})

	s.assetVerifier = assets.NewVerifier(s.opts.BundlePath, s.fs, s.telemetry)
	s.gpuApplier = gpu.NewApplier(s.gpuStatus, s.telemetry)
}

// startControlAPI sets up and binds the HTTP control API server.
func (s *Supervisor) startControlAPI() (net.Listener, error) {
	apiServer := api.NewServer(s, s.authToken)
	mux := http.NewServeMux()
	apiServer.RegisterHandlers(mux)

	addr := fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, s.opts.Manifest.IPC.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		ln, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, 0))
		if err != nil {
			return nil, fmt.Errorf("start control API on %s: %w", addr, err)
		}
	}
	actualAddr := ln.Addr().(*net.TCPAddr)
	s.opts.Manifest.IPC.Port = actualAddr.Port

	portPath := filepath.Join(s.appData, "runtime", "ipc_port")
	if err := s.fs.MkdirAll(filepath.Dir(portPath), 0o700); err == nil {
		_ = s.fs.WriteFile(portPath, []byte(fmt.Sprintf("%d", actualAddr.Port)), 0o600)
	}

	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.opts.Manifest.IPC.Host, actualAddr.Port),
		Handler:           apiServer.AuthMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return ln, nil
}

// triggerServicesOrWait starts services if all secrets are available, otherwise waits.
func (s *Supervisor) triggerServicesOrWait() {
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
	if s.resourceServer != nil {
		_ = s.resourceServer.Stop(ctx)
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}

	_ = s.recordTelemetry("runtime_shutdown", nil)

	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
