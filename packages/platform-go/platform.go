// Package platform owns operating-system-specific lifecycle behavior for the
// Vrooli control plane and scenario modules. Callers should depend on this
// package instead of importing syscall, x/sys, or selecting an OS themselves.
package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ErrLockUnavailable indicates that a non-blocking lock attempt found another
// process holding the lock.
var ErrLockUnavailable = errors.New("platform: lock unavailable")

// ProcessOptions describes the lifecycle semantics requested from a command.
// The build-tagged backend translates it into native process attributes.
type ProcessOptions struct {
	Detached bool
}

// ConfigureCommand applies native process attributes to cmd.
func ConfigureCommand(cmd *exec.Cmd, options ProcessOptions) error {
	if cmd == nil {
		return fmt.Errorf("platform: nil command")
	}
	if options.Detached {
		cmd.SysProcAttr = DetachedProcessAttrs()
	}
	return nil
}

// AssignProcessContainment attaches a started process to the native process
// tree containment primitive. Unix backends have no extra handle to retain;
// detached process groups already provide the tree boundary.
func AssignProcessContainment(process *os.Process) (func(), error) {
	return assignProcessContainment(process)
}

// GracefulStopProcess requests the native graceful shutdown mechanism for a
// started process. Callers remain responsible for bounded waiting and forceful
// escalation through KillProcess.
func GracefulStopProcess(process *os.Process) error {
	return gracefulStopProcess(process)
}

// DetachedProcessAttrs returns the native attributes used for a long-lived
// child process. The returned value is intentionally typed by the standard
// library so callers never need to import an OS package.
func DetachedProcessAttrs() *syscall.SysProcAttr { return detachedProcessAttrs() }

// SignalPID requests graceful or forced termination of one process.
func SignalPID(pid int, force bool) error { return signalPID(pid, force) }

// SignalPIDWithSignal sends an explicit graceful signal to one process. The
// platform backend translates the signal into the native process-control
// mechanism; callers should use this seam instead of importing syscall.
func SignalPIDWithSignal(pid int, signal os.Signal) error {
	return signalPIDWithSignal(pid, signal)
}

// SignalProcessGroup requests graceful or forced termination of a process tree.
func SignalProcessGroup(groupID int, force bool) error {
	return signalProcessGroup(groupID, force)
}

// ProcessGroupID returns the native process-group identity where the host
// exposes one. Windows returns the process identity because Job Objects own
// containment there rather than numeric process groups.
func ProcessGroupID(pid int) (int, error) { return processGroupID(pid) }

// ReraiseSignal sends signal to the current process using the native backend.
func ReraiseSignal(signal os.Signal) error { return reraiseSignal(signal) }

// TerminationSignals returns the signals a long-running Vrooli process should
// treat as a graceful shutdown request on this host.
func TerminationSignals() []os.Signal { return terminationSignals() }

// KillProcess terminates a process using the platform's best available tree
// semantics. On Windows this is backed by the process/job backend.
func KillProcess(pid int, force bool) error { return killProcess(pid, force) }

// ReplaceProcess re-executes a process image using native platform
// semantics, behind the same lifecycle seam as process control.
func ReplaceProcess(argv0 string, argv []string, env []string) error {
	return replaceProcess(argv0, argv, env)
}

// IsPIDRunning reports whether pid can be observed as alive by this user.
func IsPIDRunning(pid int) bool { return pidIsAlive(pid) }

// ReadProcessEnvironment returns the environment observable for pid on hosts
// that expose it. Unsupported hosts return a typed error from their backend.
func ReadProcessEnvironment(pid int) (map[string]string, error) {
	return readProcessEnvironment(pid)
}

// AcquireFileLock takes an exclusive advisory lock and returns its release
// function. The file remains on disk so lock ownership can be diagnosed.
func AcquireFileLock(path string) (func(), error) { return acquireFileLock(path) }

// LockFile applies the native lock primitive to an already-open file. When
// nonBlocking is true, ErrLockUnavailable is returned instead of waiting.
func LockFile(file *os.File, nonBlocking bool) (func(), error) {
	if file == nil {
		return nil, fmt.Errorf("platform: nil lock file")
	}
	return lockFile(file, nonBlocking)
}

// HomeDir resolves the current user's home directory.
func HomeDir() (string, error) { return os.UserHomeDir() }

// ResolveHomePath resolves a path below home while rejecting an empty path and
// path traversal outside home.
func ResolveHomePath(home, relative string) (string, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return "", fmt.Errorf("platform: home directory is empty")
	}
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return filepath.Clean(home), nil
	}
	resolved := filepath.Join(home, relative)
	cleanHome, err := filepath.Abs(filepath.Clean(home))
	if err != nil {
		return "", fmt.Errorf("platform: resolve home: %w", err)
	}
	cleanResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("platform: resolve path: %w", err)
	}
	if cleanResolved != cleanHome && !strings.HasPrefix(cleanResolved, cleanHome+string(filepath.Separator)) {
		return "", fmt.Errorf("platform: path %q escapes home", relative)
	}
	return cleanResolved, nil
}

// ServiceInstallOptions describes a per-user lifecycle supervisor service.
type ServiceInstallOptions struct {
	HomeDir    string
	Executable string
	SourceRoot string
	User       bool
}

// ServiceInstallResult is the platform-neutral service outcome.
type ServiceInstallResult struct {
	UnitName string `json:"unit_name"`
	UnitPath string `json:"unit_path"`
	Scope    string `json:"scope"`
	Active   bool   `json:"active"`
}

// InstallService installs and starts the native runtime supervisor service.
func InstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	return installService(options)
}

// UninstallService stops and removes the native runtime supervisor service.
func UninstallService(options ServiceInstallOptions) (ServiceInstallResult, error) {
	return uninstallService(options)
}

// SupportsService reports whether this host has a supported native service
// backend for the requested user scope.
func SupportsService(user bool) bool { return supportsService(user) }

// ServiceStartHint returns the native operator command for starting the
// installed supervisor service, or an empty string when no native hint exists.
func ServiceStartHint() string { return serviceStartHint() }

// ServiceManagerCommand returns the native service-manager executable name
// for the current target. Callers should use the unit-addressable service
// methods for lifecycle work; this value is retained for injected command
// adapters and operator-facing diagnostics.
func ServiceManagerCommand() string { return serviceManagerCommand() }

// ServiceManagerCommandPath returns the absolute command path used by a
// setup-provisioned privilege grant on the current target, when one exists.
func ServiceManagerCommandPath() string { return serviceManagerCommandPath() }

// NativeServiceOptions describes a service owned by a caller. The platform
// backend owns the lifecycle verbs and file placement; callers provide only
// the platform-neutral identity and rendered service definition.
type NativeServiceOptions struct {
	Name       string
	Path       string
	Content    string
	User       bool
	Executable string
	Args       []string
}

type ServiceState string

const (
	ServiceStateUnknown ServiceState = "unknown"
	ServiceStateRunning ServiceState = "running"
	ServiceStateStopped ServiceState = "stopped"
	ServiceStateFailed  ServiceState = "failed"
)

type ServiceEvidence struct {
	Source   string `json:"source,omitempty"`
	RawState string `json:"raw_state,omitempty"`
	Detail   string `json:"detail,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

type NativeServiceResult struct {
	Name     string          `json:"name"`
	Path     string          `json:"path,omitempty"`
	Scope    string          `json:"scope,omitempty"`
	Running  bool            `json:"running"`
	Enabled  bool            `json:"enabled"`
	State    ServiceState    `json:"state"`
	Evidence ServiceEvidence `json:"evidence,omitempty"`
}

func InstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	return installNativeService(options)
}

func UninstallNativeService(options NativeServiceOptions) (NativeServiceResult, error) {
	return uninstallNativeService(options)
}

func StartNativeService(options NativeServiceOptions) error   { return startNativeService(options) }
func StopNativeService(options NativeServiceOptions) error    { return stopNativeService(options) }
func RestartNativeService(options NativeServiceOptions) error { return restartNativeService(options) }
func NativeServiceStatus(options NativeServiceOptions) (NativeServiceResult, error) {
	return nativeServiceStatus(options)
}

func NativeServiceLogs(options NativeServiceOptions, tail int) ([]byte, error) {
	return nativeServiceLogs(options, tail)
}

type HostLogOptions struct {
	Unit      string
	Since     string
	Tail      int
	Arguments []string
}

type HostLogResult struct {
	Source   string          `json:"source"`
	Raw      []byte          `json:"-"`
	Entries  []HostLogEntry  `json:"entries,omitempty"`
	Evidence ServiceEvidence `json:"evidence"`
}

// HostLogEntry is the platform-neutral subset of an operating-system log
// record consumed by scenario collectors. Platform backends populate the
// fields they can prove and preserve the original record in Raw.
type HostLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Process   string    `json:"process,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	EventID   string    `json:"event_id,omitempty"`
	Unit      string    `json:"unit,omitempty"`
	UserUnit  string    `json:"user_unit,omitempty"`
	Hostname  string    `json:"hostname,omitempty"`
	BootID    string    `json:"boot_id,omitempty"`
	Cursor    string    `json:"cursor,omitempty"`
	Priority  int       `json:"priority,omitempty"`
	PID       int       `json:"pid,omitempty"`
	Raw       string    `json:"raw,omitempty"`
}

func ReadHostLogs(options HostLogOptions) (HostLogResult, error) {
	return readHostLogs(options)
}

// ReadHostLogEntries is the context-shaped seam for callers that need typed
// records. The current native backends are bounded command invocations; the
// context is retained in the public contract so a future backend can honor
// cancellation without changing scenario callers.
func ReadHostLogEntries(ctx context.Context, options HostLogOptions) ([]HostLogEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := ReadHostLogs(options)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}
