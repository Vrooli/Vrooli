package livedesktop

import (
	"context"

	"vrooli-emulator-api/procmetrics"
)

// PlatformBackend abstracts all platform-specific desktop session operations.
//
// Local backends (LinuxBackend, future QemuWindowsBackend, AndroidEmulatorBackend)
// run displays and apps directly on the host machine.
//
// Remote backends (future RemoteNodeBackend) proxy all calls over the network
// to a Vrooli instance on another machine. This is required for platforms that
// cannot be emulated locally — notably macOS (Apple EULA restricts virtualization
// to Apple hardware) and iOS (Simulator requires macOS/Xcode). A Mac Mini or
// similar device runs a Vrooli instance exposing the same session API; the
// RemoteNodeBackend translates PlatformBackend calls into HTTP/WebSocket RPCs.
//
// Design constraints for remote compatibility:
//   - Recordings/screenshots return URLs, not byte blobs (async-friendly)
//   - The caller is stateless; the backend (local or remote) owns session state
//   - The WebSocket proxy in proxy.go is already generic and works unchanged
type PlatformBackend interface {
	// CreateDisplay starts a virtual display and returns a handle.
	CreateDisplay(width, height int) (PlatformDisplay, error)

	// StartRemoteAccess exposes the display via VNC, RDP, scrcpy, etc.
	StartRemoteAccess(display PlatformDisplay) (RemoteAccessInfo, RemoteAccessHandle, error)

	// StopRemoteAccess tears down the remote access session.
	StopRemoteAccess(handle RemoteAccessHandle)

	// LaunchApp starts an application on the display with the given options.
	LaunchApp(ctx context.Context, display PlatformDisplay, appPath string, opts LaunchOptions) (PlatformProcess, error)

	// KillApp terminates a running application.
	KillApp(proc PlatformProcess)

	// CaptureScreenshot captures the display and writes to outputPath.
	// For local backends, outputPath is a filesystem path.
	// For remote backends, it may be a URL-safe identifier.
	CaptureScreenshot(ctx context.Context, display PlatformDisplay, outputPath string) error

	// ReadClipboard reads the display clipboard contents.
	ReadClipboard(ctx context.Context, display PlatformDisplay) (string, error)

	// WriteClipboard sets the display clipboard contents.
	WriteClipboard(ctx context.Context, display PlatformDisplay, content string) error

	// ResizeDisplay changes the display resolution.
	ResizeDisplay(ctx context.Context, display PlatformDisplay, width, height int) error

	// NewMonitorFactory returns a factory for process monitoring on this platform.
	NewMonitorFactory() procmetrics.MonitorFactory

	// PlatformID returns a stable identifier for this backend type.
	// Examples: "linux-xvfb", "windows-qemu", "android-emulator", "remote-macos", "remote-ios"
	PlatformID() string
}

// PlatformDisplay is an opaque handle for a virtual display.
// The backend owns the display lifecycle; callers use this interface only.
type PlatformDisplay interface {
	DisplayID() string
	Width() int
	Height() int
	IsRunning() bool
	Stop()
}

// RemoteAccessInfo contains connection details returned by StartRemoteAccess.
type RemoteAccessInfo struct {
	Protocol string // "vnc", "rdp", "scrcpy"
	Port     int    // Primary protocol port
	WSPort   int    // WebSocket proxy port (for browser access)
}

// RemoteAccessHandle is an opaque handle for stopping remote access.
// Each backend defines its own concrete type.
type RemoteAccessHandle interface{}

// PlatformProcess is an opaque handle for a launched application.
type PlatformProcess interface {
	PID() int
	IsRunning() bool
}

// LaunchOptions collects platform-agnostic launch parameters.
type LaunchOptions struct {
	EnvVars       map[string]string
	DarkMode      bool
	Locale        string
	NetworkMode   string // "normal", "offline", "slow"
	BandwidthKbps int
}
