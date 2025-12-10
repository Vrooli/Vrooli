// Package containment provides OS-level sandboxing for agent execution.
// It supports multiple containment strategies via a dependency injection pattern,
// allowing the system to use the best available option (Docker, bubblewrap, etc.)
// or fall back gracefully when no sandbox is available.
package containment

import (
	"context"
	"fmt"
	"os/exec"
)

// ContainmentType identifies which containment strategy is in use.
type ContainmentType string

const (
	// ContainmentTypeNone indicates no OS-level containment (unsafe fallback).
	ContainmentTypeNone ContainmentType = "none"

	// ContainmentTypeDocker indicates Docker-based containment.
	ContainmentTypeDocker ContainmentType = "docker"

	// ContainmentTypeBubblewrap indicates bubblewrap-based containment (Linux only).
	ContainmentTypeBubblewrap ContainmentType = "bubblewrap"
)

// ExecutionConfig contains the parameters for a contained execution.
type ExecutionConfig struct {
	// WorkingDir is the directory where the command should run.
	WorkingDir string

	// AllowedPaths is a list of paths (relative to WorkingDir) that the process
	// can access. If empty, only WorkingDir is accessible.
	AllowedPaths []string

	// ReadOnlyPaths is a list of paths that should be mounted read-only.
	// Useful for providing access to tools without allowing modification.
	ReadOnlyPaths []string

	// Environment variables to pass to the contained process.
	Environment map[string]string

	// MaxMemoryMB limits the memory available to the process (0 = unlimited).
	MaxMemoryMB int

	// MaxCPUPercent limits CPU usage (0 = unlimited, 100 = one core).
	MaxCPUPercent int

	// NetworkAccess controls whether the process can access the network.
	NetworkAccess bool

	// Command is the command to execute (first element) with arguments.
	Command []string

	// Timeout for the command execution (0 = no timeout).
	TimeoutSeconds int
}

// ExecutionResult contains the outcome of a contained execution.
type ExecutionResult struct {
	// ExitCode is the exit code of the contained process.
	ExitCode int

	// Stdout contains the standard output.
	Stdout string

	// Stderr contains the standard error.
	Stderr string

	// ContainmentUsed indicates which containment type was actually used.
	ContainmentUsed ContainmentType

	// ContainmentWarnings contains any warnings from the containment layer.
	// For example, "Docker not available, running without containment".
	ContainmentWarnings []string
}

// Provider is the interface for containment implementations.
// Each provider encapsulates a specific sandboxing technology.
type Provider interface {
	// Type returns the containment type this provider implements.
	Type() ContainmentType

	// IsAvailable checks if this containment method is available on the system.
	// This should be a quick check (e.g., verifying Docker is installed).
	IsAvailable(ctx context.Context) bool

	// PrepareCommand wraps the given command with containment.
	// Returns an exec.Cmd ready to run in a sandbox.
	// The caller is responsible for calling cmd.Start() and cmd.Wait().
	PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error)

	// Info returns human-readable information about this containment provider.
	Info() ProviderInfo
}

// ProviderInfo contains metadata about a containment provider.
type ProviderInfo struct {
	// Type identifies the containment method.
	Type ContainmentType

	// Name is a human-readable name.
	Name string

	// Description explains what this provider does.
	Description string

	// SecurityLevel indicates how secure this containment is (0-10).
	// 0 = no security, 10 = hardware-level isolation.
	SecurityLevel int

	// Requirements lists what's needed for this provider to work.
	Requirements []string
}

// Manager selects and manages containment providers.
type Manager struct {
	providers    []Provider
	fallback     Provider
	preferDocker bool
}

// NewManager creates a containment manager with the given providers.
// The first available provider will be used, with fallback as the last resort.
func NewManager(providers []Provider, fallback Provider) *Manager {
	return &Manager{
		providers:    providers,
		fallback:     fallback,
		preferDocker: true,
	}
}

// SelectProvider returns the best available containment provider.
// It checks each provider in order and returns the first available one.
// If none are available, it returns the fallback provider.
func (m *Manager) SelectProvider(ctx context.Context) Provider {
	for _, p := range m.providers {
		if p.IsAvailable(ctx) {
			return p
		}
	}
	return m.fallback
}

// ListProviders returns information about all registered providers.
func (m *Manager) ListProviders(ctx context.Context) []ProviderInfo {
	infos := make([]ProviderInfo, 0, len(m.providers)+1)
	for _, p := range m.providers {
		info := p.Info()
		// Check availability
		if !p.IsAvailable(ctx) {
			info.Name += " (unavailable)"
		}
		infos = append(infos, info)
	}
	if m.fallback != nil {
		infos = append(infos, m.fallback.Info())
	}
	return infos
}

// Status returns a summary of containment availability.
type Status struct {
	// ActiveProvider is the type of containment that will be used.
	ActiveProvider ContainmentType

	// AvailableProviders lists all providers that are available.
	AvailableProviders []ContainmentType

	// SecurityLevel is the security level of the active provider.
	SecurityLevel int

	// Warnings contains any security warnings (e.g., "no containment available").
	Warnings []string
}

// GetStatus returns the current containment status.
func (m *Manager) GetStatus(ctx context.Context) Status {
	status := Status{
		ActiveProvider:     ContainmentTypeNone,
		AvailableProviders: make([]ContainmentType, 0),
		Warnings:           make([]string, 0),
	}

	for _, p := range m.providers {
		if p.IsAvailable(ctx) {
			status.AvailableProviders = append(status.AvailableProviders, p.Type())
		}
	}

	active := m.SelectProvider(ctx)
	status.ActiveProvider = active.Type()
	status.SecurityLevel = active.Info().SecurityLevel

	if status.ActiveProvider == ContainmentTypeNone {
		status.Warnings = append(status.Warnings,
			"No containment available. Agents will run without OS-level isolation. "+
				"Consider installing Docker for improved security.")
	} else if status.SecurityLevel < 5 {
		status.Warnings = append(status.Warnings,
			fmt.Sprintf("Using %s containment (security level %d/10). "+
				"Consider Docker for stronger isolation.",
				status.ActiveProvider, status.SecurityLevel))
	}

	return status
}
