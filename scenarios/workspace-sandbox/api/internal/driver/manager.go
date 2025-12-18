// Package driver provides sandbox driver interfaces and implementations.
package driver

import (
	"context"
	"fmt"
	"log"
	"sync"

	"workspace-sandbox/internal/types"
)

// Manager provides thread-safe access to drivers with hot-swap capability.
// It implements the Driver interface by proxying all calls to the current driver,
// allowing it to be used anywhere a Driver is expected.
//
// When Switch is called, in-flight operations continue with their original driver
// reference (captured at the start of the operation), while new operations use
// the newly selected driver.
type Manager struct {
	mu     sync.RWMutex
	driver Driver
	config Config
}

// Ensure Manager implements Driver interface at compile time.
var _ Driver = (*Manager)(nil)

// NewManager creates a new driver manager with the given initial driver.
func NewManager(initial Driver, cfg Config) *Manager {
	return &Manager{
		driver: initial,
		config: cfg,
	}
}

// Current returns the current active driver.
// This is safe to call concurrently.
func (m *Manager) Current() Driver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driver
}

// Switch atomically switches to a new driver based on the option ID.
// The new driver is validated to be available before switching.
// In-flight operations using the old driver will complete normally;
// only new operations will use the new driver.
func (m *Manager) Switch(ctx context.Context, optionID DriverOptionID) error {
	// Create the new driver based on option ID
	var newDriver Driver
	switch optionID {
	case DriverOptionFuseOverlayfs:
		newDriver = NewFuseOverlayfsDriver(m.config)
	case DriverOptionOverlayfsUserNS, DriverOptionOverlayfsRoot:
		newDriver = NewOverlayfsDriver(m.config)
	case DriverOptionCopy:
		newDriver = NewCopyDriver(m.config)
	default:
		return fmt.Errorf("unknown driver option: %s", optionID)
	}

	// Verify the new driver is available
	available, err := newDriver.IsAvailable(ctx)
	if err != nil {
		return fmt.Errorf("failed to check driver availability: %w", err)
	}
	if !available {
		return fmt.Errorf("driver %s is not available on this system", optionID)
	}

	// Atomically swap the driver
	m.mu.Lock()
	oldDriver := m.driver
	m.driver = newDriver
	m.mu.Unlock()

	log.Printf("driver: switched from %s to %s", oldDriver.Type(), newDriver.Type())

	// Save the preference for future restarts
	if err := SaveDriverPreference(m.config.BaseDir, string(optionID)); err != nil {
		// Log but don't fail - the switch succeeded
		log.Printf("driver: warning: failed to save preference: %v", err)
	}

	return nil
}

// Config returns the driver configuration.
func (m *Manager) Config() Config {
	return m.config
}

// --- Driver Interface Implementation ---
// All Driver methods are proxied to the current driver.

// Type returns the driver type.
func (m *Manager) Type() DriverType {
	return m.Current().Type()
}

// Version returns the driver version.
func (m *Manager) Version() string {
	return m.Current().Version()
}

// IsAvailable checks if this driver can be used on the current system.
func (m *Manager) IsAvailable(ctx context.Context) (bool, error) {
	return m.Current().IsAvailable(ctx)
}

// Mount creates the overlay mount for a sandbox.
func (m *Manager) Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error) {
	return m.Current().Mount(ctx, s)
}

// Unmount removes the overlay mount.
func (m *Manager) Unmount(ctx context.Context, s *types.Sandbox) error {
	return m.Current().Unmount(ctx, s)
}

// GetChangedFiles returns the list of files changed in the upper layer.
func (m *Manager) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	return m.Current().GetChangedFiles(ctx, s)
}

// Cleanup removes all sandbox artifacts (dirs, mounts).
func (m *Manager) Cleanup(ctx context.Context, s *types.Sandbox) error {
	return m.Current().Cleanup(ctx, s)
}

// IsMounted verifies whether the sandbox overlay is currently mounted.
func (m *Manager) IsMounted(ctx context.Context, s *types.Sandbox) (bool, error) {
	return m.Current().IsMounted(ctx, s)
}

// VerifyMountIntegrity checks that the mount is healthy and accessible.
func (m *Manager) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return m.Current().VerifyMountIntegrity(ctx, s)
}

// RemoveFromUpper removes a file from the upper (writable) layer.
func (m *Manager) RemoveFromUpper(ctx context.Context, s *types.Sandbox, filePath string) error {
	return m.Current().RemoveFromUpper(ctx, s, filePath)
}

// Exec executes a command inside the sandbox with process isolation.
func (m *Manager) Exec(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (*ExecResult, error) {
	return m.Current().Exec(ctx, s, cfg, cmd, args...)
}

// StartProcess starts a long-running process in the sandbox.
func (m *Manager) StartProcess(ctx context.Context, s *types.Sandbox, cfg BwrapConfig, cmd string, args ...string) (int, error) {
	return m.Current().StartProcess(ctx, s, cfg, cmd, args...)
}
