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

// ProviderSelector is a seam for selecting containment providers.
// This interface enables dependency injection and testing of containment selection logic.
type ProviderSelector interface {
	// SelectProvider returns the best available containment provider.
	SelectProvider(ctx context.Context) Provider

	// GetStatus returns the current containment status.
	GetStatus(ctx context.Context) Status

	// ListProviders returns information about all registered providers.
	ListProviders(ctx context.Context) []ProviderInfo
}

// Manager selects and manages containment providers.
// Manager implements ProviderSelector.
type Manager struct {
	providers    []Provider
	fallback     Provider
	preferDocker bool
}

// Ensure Manager implements ProviderSelector
var _ ProviderSelector = (*Manager)(nil)

// NewManager creates a containment manager with the given providers.
// The first available provider will be used, with fallback as the last resort.
func NewManager(providers []Provider, fallback Provider) *Manager {
	return &Manager{
		providers:    providers,
		fallback:     fallback,
		preferDocker: true,
	}
}

// DefaultManager creates a containment manager with Docker and Fallback providers.
// This is the standard configuration for production use.
func DefaultManager() *Manager {
	return NewManager(
		[]Provider{NewDockerProvider()}, // Uses OS defaults for command lookup/runner
		NewFallbackProvider(),
	)
}

// ProviderSelectionDecision describes why a particular provider was chosen.
type ProviderSelectionDecision struct {
	SelectedProvider Provider
	SelectedType     ContainmentType
	Reason           string
	CheckedProviders []ProviderCheckResult
	UsedFallback     bool
}

// ProviderCheckResult describes the availability check for a single provider.
type ProviderCheckResult struct {
	Type      ContainmentType
	Available bool
	Reason    string
}

// DecideProvider evaluates providers and selects the best available one.
// This is the central decision point for containment strategy selection.
//
// Decision criteria (in order of preference):
//  1. Check each registered provider in order (typically Docker first)
//  2. Select the first available provider
//  3. If no registered providers are available, use the fallback
//
// Selection rationale:
//   - Docker is preferred because it provides strong isolation (security level 7/10)
//   - Bubblewrap is acceptable for Linux-only deployments (security level 5/10)
//   - Fallback (no containment) is last resort with security warnings (level 0/10)
//
// The fallback is always available but provides NO security isolation.
// Production deployments should ensure at least one real provider is available.
func (m *Manager) DecideProvider(ctx context.Context) ProviderSelectionDecision {
	decision := ProviderSelectionDecision{
		CheckedProviders: make([]ProviderCheckResult, 0, len(m.providers)),
	}

	// Check each provider in preference order
	for _, p := range m.providers {
		checkResult := ProviderCheckResult{
			Type:      p.Type(),
			Available: p.IsAvailable(ctx),
		}

		if checkResult.Available {
			checkResult.Reason = "provider is available and ready"
			decision.CheckedProviders = append(decision.CheckedProviders, checkResult)

			// First available provider wins
			decision.SelectedProvider = p
			decision.SelectedType = p.Type()
			decision.Reason = "selected first available provider in preference order"
			decision.UsedFallback = false
			return decision
		}

		checkResult.Reason = "provider is not available"
		decision.CheckedProviders = append(decision.CheckedProviders, checkResult)
	}

	// No registered providers available - use fallback
	decision.SelectedProvider = m.fallback
	decision.SelectedType = m.fallback.Type()
	decision.Reason = "no preferred providers available; using fallback (WARNING: no security isolation)"
	decision.UsedFallback = true
	return decision
}

// SelectProvider returns the best available containment provider.
// For detailed selection information, use DecideProvider instead.
func (m *Manager) SelectProvider(ctx context.Context) Provider {
	return m.DecideProvider(ctx).SelectedProvider
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

// SecurityLevelThresholds defines the thresholds for security level warnings.
// These are the decision points for what's considered acceptable security.
const (
	// SecurityLevelNone indicates no security isolation (0).
	SecurityLevelNone = 0

	// SecurityLevelMinimumAcceptable is the minimum level that doesn't trigger warnings.
	// Below this level, we warn users to consider stronger isolation.
	SecurityLevelMinimumAcceptable = 5

	// SecurityLevelMaximum is the maximum possible security level (hardware isolation).
	SecurityLevelMaximum = 10
)

// SecurityAssessment describes the security posture and any concerns.
type SecurityAssessment struct {
	Level    int
	Adequate bool
	Warnings []string
}

// AssessSecurityLevel evaluates a containment provider's security level and returns
// appropriate warnings. This is the central decision point for security adequacy.
//
// Decision criteria:
//   - Level 0 (no containment): Always warns - agents have full system access
//   - Level 1-4 (weak isolation): Warns - some isolation but not production-ready
//   - Level 5+ (adequate isolation): No warnings - acceptable for production
func AssessSecurityLevel(providerType ContainmentType, securityLevel int) SecurityAssessment {
	assessment := SecurityAssessment{
		Level:    securityLevel,
		Warnings: make([]string, 0),
	}

	switch {
	case providerType == ContainmentTypeNone || securityLevel == SecurityLevelNone:
		// No containment - always a concern
		assessment.Adequate = false
		assessment.Warnings = append(assessment.Warnings,
			"No containment available. Agents will run without OS-level isolation. "+
				"Consider installing Docker for improved security.")

	case securityLevel < SecurityLevelMinimumAcceptable:
		// Weak isolation - warn but allow
		assessment.Adequate = false
		assessment.Warnings = append(assessment.Warnings,
			fmt.Sprintf("Using %s containment (security level %d/%d). "+
				"Consider Docker for stronger isolation.",
				providerType, securityLevel, SecurityLevelMaximum))

	default:
		// Adequate isolation
		assessment.Adequate = true
	}

	return assessment
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

	// Use the centralized security assessment decision
	assessment := AssessSecurityLevel(status.ActiveProvider, status.SecurityLevel)
	status.Warnings = assessment.Warnings

	return status
}
