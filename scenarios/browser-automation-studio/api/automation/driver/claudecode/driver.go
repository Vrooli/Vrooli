// Package claudecode provides a Driver stub for the Claude Code CLI-based
// browser automation backend. This driver will eventually manage browser
// automation through the `claude` CLI tool for AI-driven navigation.
//
// Current status: Stub implementation that returns ErrNotImplemented.
// Future: Will spawn and manage claude CLI processes for browser automation.
//
// DOC: docs/architecture/driver-interface.md#claudecode-driver
package claudecode

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// Driver implements the driver.Driver interface for Claude Code CLI.
// This is currently a stub that returns ErrNotImplemented for all operations.
// It exists to:
// 1. Establish the interface contract for AI-driven browser automation
// 2. Enable ClaudeCodeVisionNavigator to compile and be wired
// 3. Provide a clear extension point for future implementation
type Driver struct {
	log *logrus.Logger
}

// Option configures a Driver.
type Option func(*Driver)

// WithLogger sets a custom logger.
func WithLogger(log *logrus.Logger) Option {
	return func(d *Driver) {
		d.log = log
	}
}

// NewDriver creates a new Claude Code driver stub.
func NewDriver(opts ...Option) *Driver {
	d := &Driver{
		log: logrus.StandardLogger(),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// CreateSession returns ErrNotImplemented.
// Future: Will spawn a claude CLI process with browser MCP server access.
func (d *Driver) CreateSession(ctx context.Context, spec driver.SessionSpec) (driver.Session, error) {
	d.log.Warn("ClaudeCode driver CreateSession called - not yet implemented")
	return nil, driver.ErrNotImplemented
}

// CloseSession returns ErrNotImplemented.
func (d *Driver) CloseSession(ctx context.Context, sessionID string) error {
	d.log.Warn("ClaudeCode driver CloseSession called - not yet implemented")
	return driver.ErrNotImplemented
}

// Health returns nil (stub is always "healthy" in that it exists).
// Future: Will check if claude CLI is available.
func (d *Driver) Health(ctx context.Context) error {
	// Stub is always available, though it doesn't do anything useful yet
	return nil
}

// Type returns DriverTypeClaudeCode.
func (d *Driver) Type() driver.DriverType {
	return driver.DriverTypeClaudeCode
}

// IsImplemented returns false, indicating this is a stub.
// Use this to check if the driver is ready for production use.
func (d *Driver) IsImplemented() bool {
	return false
}

// Compile-time interface compliance
var _ driver.Driver = (*Driver)(nil)
