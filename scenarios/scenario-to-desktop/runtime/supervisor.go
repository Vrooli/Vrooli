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
// Architecture:
//
//	supervisor.go       - Core Supervisor struct, Start, Shutdown
//	control_api.go      - HTTP handlers and authentication middleware
//	service_launcher.go - Service lifecycle management (includes topoSort, findService)
//	health.go           - Health and readiness checking
//	secrets.go          - Secret loading, persistence, and injection
//	migrations.go       - Migration tracking and execution
//	ports.go            - Port allocation and resolution
//	telemetry.go        - Telemetry event recording
//	gpu.go              - GPU detection and requirement enforcement
//	playwright.go       - Playwright environment setup
//	assets.go           - Asset verification
//	env.go              - Environment variable rendering
//	errors.go           - Structured error types
//	interfaces.go       - Interfaces for testing and external integration
//	utils.go            - Shared utilities (copyStringMap, envMapToList)
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

	"scenario-to-desktop-runtime/manifest"
)

// Options configures the Supervisor.
type Options struct {
	AppDataDir string             // Override for app data directory (default: user config dir)
	Manifest   *manifest.Manifest // Bundle manifest (required)
	BundlePath string             // Root path of the unpacked bundle
	DryRun     bool               // Skip actual service launches (for testing)
}

// Supervisor manages the desktop bundle runtime.
// It orchestrates service startup, health monitoring, and exposes a control API.
type Supervisor struct {
	opts Options

	// Paths and auth.
	authToken      string
	appData        string
	secretsPath    string
	telemetryPath  string
	migrationsPath string

	// Runtime state.
	portMap         map[string]map[string]int // service ID -> port name -> port
	serviceStatus   map[string]ServiceStatus
	procs           map[string]*serviceProcess
	secrets         map[string]string
	migrations      migrationsState
	gpuStatus       gpuStatus
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

// ServiceStatus represents the current state of a service.
type ServiceStatus struct {
	Ready    bool   `json:"ready"`
	Message  string `json:"message,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
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

	return &Supervisor{
		opts:          opts,
		appData:       appData,
		portMap:       make(map[string]map[string]int),
		serviceStatus: make(map[string]ServiceStatus),
		procs:         make(map[string]*serviceProcess),
	}, nil
}

// Start initializes the supervisor and begins service orchestration.
// It sets up the control API, loads secrets and migrations, allocates ports,
// and starts services asynchronously once all required secrets are available.
func (s *Supervisor) Start(ctx context.Context) error {
	// Create app data directory.
	if err := os.MkdirAll(s.appData, 0o755); err != nil {
		return fmt.Errorf("create app data dir: %w", err)
	}

	// Set up paths.
	s.secretsPath = filepath.Join(s.appData, "secrets.json")
	s.telemetryPath = manifest.ResolvePath(s.appData, s.opts.Manifest.Telemetry.File)
	s.migrationsPath = filepath.Join(s.appData, "migrations.json")

	if err := os.MkdirAll(filepath.Dir(s.telemetryPath), 0o755); err != nil {
		return fmt.Errorf("create telemetry dir: %w", err)
	}

	// Load persisted state.
	loadedSecrets, err := s.loadSecrets()
	if err != nil {
		return fmt.Errorf("load secrets: %w", err)
	}
	s.secrets = loadedSecrets

	migrations, err := s.loadMigrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	s.migrations = migrations

	// Load or create auth token.
	tokenPath := manifest.ResolvePath(s.appData, s.opts.Manifest.IPC.AuthTokenRel)
	token, err := loadOrCreateToken(tokenPath)
	if err != nil {
		return fmt.Errorf("load auth token: %w", err)
	}
	s.authToken = token

	// Allocate ports.
	if err := s.allocatePorts(); err != nil {
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
	s.gpuStatus = detectGPU()
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
func loadOrCreateToken(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data)), nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
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
