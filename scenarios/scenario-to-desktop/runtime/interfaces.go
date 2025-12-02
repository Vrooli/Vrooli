package bundleruntime

import (
	"context"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
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
	Signal(sig os.Signal) error
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
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// MkdirAll creates a directory and all parent directories.
	MkdirAll(path string, perm fs.FileMode) error
	// Stat returns file info for the given path.
	Stat(path string) (fs.FileInfo, error)
	// OpenFile opens a file with the given flags and permissions.
	OpenFile(path string, flag int, perm fs.FileMode) (File, error)
	// Remove removes a file or empty directory.
	Remove(path string) error
}

// File abstracts os.File for testing.
type File interface {
	io.Writer
	io.Closer
	// Sync commits the current contents of the file to stable storage.
	Sync() error
}

// NetworkDialer abstracts network connections for testing.
type NetworkDialer interface {
	// Dial connects to the address on the named network.
	Dial(network, address string) (net.Conn, error)
	// DialTimeout connects to the address with a timeout.
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
	// Listen creates a listener on the given address.
	Listen(network, address string) (net.Listener, error)
}

// HTTPClient abstracts HTTP client operations for testing.
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// CommandRunner abstracts command execution for health checks.
type CommandRunner interface {
	// Run executes a command and returns its exit status.
	Run(ctx context.Context, name string, args []string) error
	// LookPath searches for an executable in the system PATH.
	LookPath(file string) (string, error)
	// Output runs the command and returns its standard output.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// GPUDetector abstracts GPU detection for testing.
type GPUDetector interface {
	// Detect probes the system for GPU availability.
	Detect() GPUStatus
}

// GPUStatus holds the result of GPU detection.
type GPUStatus struct {
	Available bool   // Whether a usable GPU was detected
	Method    string // Detection method used (env_override, nvidia-smi, system_profiler, wmic, probe)
	Reason    string // Human-readable explanation
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
	State() MigrationsState
}

// MigrationsState tracks applied migrations per service and the current app version.
type MigrationsState struct {
	AppVersion string              `json:"app_version"`
	Applied    map[string][]string `json:"applied"` // service ID -> list of applied migration versions
}

// TelemetryRecorder abstracts telemetry recording for testing.
type TelemetryRecorder interface {
	// Record records a telemetry event.
	Record(event string, details map[string]interface{}) error
}

// EnvReader abstracts environment variable access for testing.
type EnvReader interface {
	// Getenv retrieves the value of the environment variable named by the key.
	Getenv(key string) string
	// Environ returns a copy of strings representing the environment.
	Environ() []string
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

// Ensure Supervisor implements ServiceManager.
var _ ServiceManager = (*Supervisor)(nil)

// =============================================================================
// Real Implementations
// =============================================================================

// RealProcessRunner implements ProcessRunner using os/exec.
type RealProcessRunner struct{}

// Start starts a process with the given command and arguments.
func (RealProcessRunner) Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	c.Env = env
	c.Dir = dir
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Start(); err != nil {
		return nil, err
	}
	return &realProcess{cmd: c}, nil
}

// realProcess wraps exec.Cmd to implement Process interface.
type realProcess struct {
	cmd *exec.Cmd
}

func (p *realProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *realProcess) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Signal(sig)
}

func (p *realProcess) Kill() error {
	if p.cmd.Process == nil {
		return os.ErrProcessDone
	}
	return p.cmd.Process.Kill()
}

func (p *realProcess) Pid() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// Ensure RealProcessRunner implements ProcessRunner.
var _ ProcessRunner = RealProcessRunner{}

// RealFileSystem implements FileSystem using the os package.
type RealFileSystem struct{}

func (RealFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (RealFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (RealFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (RealFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (RealFileSystem) OpenFile(path string, flag int, perm fs.FileMode) (File, error) {
	return os.OpenFile(path, flag, perm)
}

func (RealFileSystem) Remove(path string) error {
	return os.Remove(path)
}

// Ensure RealFileSystem implements FileSystem.
var _ FileSystem = RealFileSystem{}

// RealNetworkDialer implements NetworkDialer using the net package.
type RealNetworkDialer struct{}

func (RealNetworkDialer) Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func (RealNetworkDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

func (RealNetworkDialer) Listen(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

// Ensure RealNetworkDialer implements NetworkDialer.
var _ NetworkDialer = RealNetworkDialer{}

// RealHTTPClient implements HTTPClient using http.Client.
type RealHTTPClient struct {
	Client *http.Client
}

func (c *RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.Client == nil {
		return http.DefaultClient.Do(req)
	}
	return c.Client.Do(req)
}

// Ensure RealHTTPClient implements HTTPClient.
var _ HTTPClient = (*RealHTTPClient)(nil)

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

func (RealCommandRunner) Run(ctx context.Context, name string, args []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

func (RealCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (RealCommandRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// Ensure RealCommandRunner implements CommandRunner.
var _ CommandRunner = RealCommandRunner{}

// RealEnvReader implements EnvReader using the os package.
type RealEnvReader struct{}

func (RealEnvReader) Getenv(key string) string {
	return os.Getenv(key)
}

func (RealEnvReader) Environ() []string {
	return os.Environ()
}

// Ensure RealEnvReader implements EnvReader.
var _ EnvReader = RealEnvReader{}

// RealGPUDetector implements GPUDetector using system commands.
type RealGPUDetector struct {
	CommandRunner CommandRunner
	EnvReader     EnvReader
}

// Detect is implemented in gpu.go to avoid circular references.
// This struct is provided for dependency injection.

// Ensure RealGPUDetector implements GPUDetector (implementation in gpu.go).
var _ GPUDetector = (*RealGPUDetector)(nil)

// =============================================================================
// Signal constants for cross-platform compatibility
// =============================================================================

// Interrupt is os.Interrupt for use in Process.Signal calls.
var Interrupt = os.Interrupt

// Kill signal for forceful termination.
var Kill = syscall.SIGKILL
