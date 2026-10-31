//go:build !linux

package livedesktop

import (
	"context"
	"fmt"
	"log/slog"

	"scenario-to-desktop-api/procmetrics"
)

// unavailableBackend is intentional: local live-desktop sessions require the
// Linux X11 toolchain. Other targets compile cleanly and report the limitation
// explicitly instead of pretending a local display backend exists.
type unavailableBackend struct{}

func NewBackend(_ *slog.Logger) PlatformBackend { return unavailableBackend{} }
func (unavailableBackend) unavailable() error {
	return fmt.Errorf("live desktop is unavailable on this platform; use a Linux runner")
}

func (b unavailableBackend) CreateDisplay(int, int) (PlatformDisplay, error) {
	return nil, b.unavailable()
}

func (b unavailableBackend) StartRemoteAccess(PlatformDisplay) (RemoteAccessInfo, RemoteAccessHandle, error) {
	return RemoteAccessInfo{}, nil, b.unavailable()
}
func (unavailableBackend) StopRemoteAccess(RemoteAccessHandle) {}
func (b unavailableBackend) LaunchApp(context.Context, PlatformDisplay, string, LaunchOptions) (PlatformProcess, error) {
	return nil, b.unavailable()
}
func (unavailableBackend) KillApp(PlatformProcess) {}
func (b unavailableBackend) CaptureScreenshot(context.Context, PlatformDisplay, string) error {
	return b.unavailable()
}

func (b unavailableBackend) ReadClipboard(context.Context, PlatformDisplay) (string, error) {
	return "", b.unavailable()
}

func (b unavailableBackend) WriteClipboard(context.Context, PlatformDisplay, string) error {
	return b.unavailable()
}

func (b unavailableBackend) ResizeDisplay(context.Context, PlatformDisplay, int, int) error {
	return b.unavailable()
}
func (unavailableBackend) NewMonitorFactory() procmetrics.MonitorFactory { return nil }
func (unavailableBackend) PlatformID() string                            { return "unavailable" }
