package bundleruntime

import (
	"context"
	"io"
	"net"
)

// ProcessRunner abstracts process execution for testing.
type ProcessRunner interface {
	// Start starts a process with the given command and arguments.
	Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error)
}

// Process represents a running process.
type Process interface {
	// Wait waits for the process to exit and returns any error.
	Wait() error
	// Signal sends a signal to the process.
	Signal(sig interface{}) error
	// Kill forcefully terminates the process.
	Kill() error
	// Pid returns the process ID.
	Pid() int
}

// FileSystem abstracts file system operations for testing.
type FileSystem interface {
	// ReadFile reads the entire contents of a file.
	ReadFile(path string) ([]byte, error)
	// WriteFile writes data to a file with the given permissions.
	WriteFile(path string, data []byte, perm int) error
	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string, perm int) error
	// Stat returns file info for the given path.
	Stat(path string) (FileInfo, error)
	// OpenFile opens a file with the given flags and permissions.
	OpenFile(path string, flag int, perm int) (io.WriteCloser, error)
}

// FileInfo abstracts os.FileInfo for testing.
type FileInfo interface {
	Size() int64
	IsDir() bool
}

// NetworkDialer abstracts network connections for testing.
type NetworkDialer interface {
	// Dial connects to the address on the named network.
	Dial(network, address string) (net.Conn, error)
	// Listen creates a listener on the given address.
	Listen(network, address string) (net.Listener, error)
}

// HTTPClient abstracts HTTP client operations for testing.
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req interface{}) (interface{}, error)
}

// TelemetryRecorder abstracts telemetry recording for testing.
type TelemetryRecorder interface {
	// Record records a telemetry event.
	Record(event string, details map[string]interface{}) error
}

// GPUDetector abstracts GPU detection for testing.
type GPUDetector interface {
	// Detect probes the system for GPU availability.
	Detect() gpuStatus
}

// SecretStore abstracts secret storage for testing.
type SecretStore interface {
	// Load reads secrets from persistent storage.
	Load() (map[string]string, error)
	// Persist saves secrets to persistent storage.
	Persist(secrets map[string]string) error
	// Get returns a copy of the current secrets.
	Get() map[string]string
	// MissingRequired returns IDs of required secrets that are missing.
	MissingRequired() []string
}

// PortAllocator abstracts port allocation for testing.
type PortAllocator interface {
	// Allocate assigns ports to all services based on manifest requirements.
	Allocate() error
	// Resolve looks up an allocated port for a service.
	Resolve(serviceID, portName string) (int, error)
	// Map returns a copy of all allocated ports.
	Map() map[string]map[string]int
}

// HealthChecker abstracts health checking for testing.
type HealthChecker interface {
	// WaitForReadiness waits for a service to become ready.
	WaitForReadiness(ctx context.Context, serviceID string) error
	// CheckOnce performs a single health check.
	CheckOnce(ctx context.Context, serviceID string) bool
}

// MigrationRunner abstracts migration execution for testing.
type MigrationRunner interface {
	// Run executes pending migrations for a service.
	Run(ctx context.Context, serviceID string, env map[string]string) error
	// State returns the current migration state.
	State() migrationsState
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
	GPUStatus() gpuStatus
	// PortMap returns allocated ports for all services.
	PortMap() map[string]map[string]int
	// AppDataDir returns the application data directory.
	AppDataDir() string
	// AuthToken returns the control API authentication token.
	AuthToken() string
}

// Ensure Supervisor implements ServiceManager.
var _ ServiceManager = (*Supervisor)(nil)
