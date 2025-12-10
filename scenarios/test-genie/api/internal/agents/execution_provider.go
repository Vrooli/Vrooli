// Package agents provides agent lifecycle management.
// This file defines the execution provider abstraction, which localizes
// CLI/runtime-specific code for agent execution.
//
// CHANGE AXIS: Agent Execution CLI/Provider
// When switching from one agent CLI (e.g., resource-opencode) to another
// (e.g., claude, opencode, custom wrapper), changes are localized here.
//
// Extension points:
//   - Add new ExecutionProvider implementations for different CLIs
//   - Modify command argument construction in one place
//   - Configure provider selection via ProviderSelector
package agents

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ExecutionProvider abstracts the CLI/tool used to execute agent prompts.
// This is the extension point for supporting different agent execution backends.
//
// VOLATILE: This interface is expected to evolve as we add new execution backends.
// Implementations should be self-contained - all CLI-specific logic stays in the provider.
type ExecutionProvider interface {
	// Name returns a human-readable name for this provider.
	Name() string

	// IsAvailable checks if this execution provider can be used.
	// This typically involves checking if the CLI binary is installed.
	IsAvailable(ctx context.Context) AvailabilityResult

	// BuildCommand constructs the exec.Cmd for running an agent prompt.
	// All CLI-specific argument construction happens here.
	BuildCommand(ctx context.Context, params ExecutionParams) (*exec.Cmd, error)

	// ExtractSessionID parses the session ID from command output, if applicable.
	// Returns empty string if no session ID is found.
	ExtractSessionID(output string) string
}

// AvailabilityResult describes whether an execution provider is available.
type AvailabilityResult struct {
	Available  bool
	BinaryPath string
	Reason     string
}

// ExecutionParams contains all parameters needed to execute an agent prompt.
// This is the stable interface between the handler and the provider.
type ExecutionParams struct {
	// Core execution
	Prompt         string
	Model          string
	Provider       string // e.g., "openrouter", "anthropic"
	WorkingDir     string
	TimeoutSeconds int
	MaxTurns       int

	// Tool configuration
	AllowedTools []string

	// Environment
	Environment map[string]string

	// Metadata for tracking
	AgentID  string
	Scenario string
}

// --- Resource-OpenCode Provider ---

// ResourceOpenCodeProvider implements ExecutionProvider using the resource-opencode CLI.
// STABLE CORE: This is the current production provider.
type ResourceOpenCodeProvider struct {
	// binaryPath caches the path to the binary after availability check
	binaryPath string
}

// NewResourceOpenCodeProvider creates a new resource-opencode execution provider.
func NewResourceOpenCodeProvider() *ResourceOpenCodeProvider {
	return &ResourceOpenCodeProvider{}
}

// Name returns the provider name.
func (p *ResourceOpenCodeProvider) Name() string {
	return "resource-opencode"
}

// IsAvailable checks if resource-opencode is installed and available.
func (p *ResourceOpenCodeProvider) IsAvailable(ctx context.Context) AvailabilityResult {
	path, err := exec.LookPath("resource-opencode")
	if err != nil {
		return AvailabilityResult{
			Available: false,
			Reason:    "resource-opencode is not installed or not on PATH",
		}
	}
	p.binaryPath = path
	return AvailabilityResult{
		Available:  true,
		BinaryPath: path,
		Reason:     "resource-opencode found and ready",
	}
}

// BuildCommand constructs the resource-opencode command for agent execution.
// This is where all CLI-specific argument construction is localized.
func (p *ResourceOpenCodeProvider) BuildCommand(ctx context.Context, params ExecutionParams) (*exec.Cmd, error) {
	if params.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if params.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Ensure we have the binary path
	if p.binaryPath == "" {
		result := p.IsAvailable(ctx)
		if !result.Available {
			return nil, fmt.Errorf("resource-opencode not available: %s", result.Reason)
		}
	}

	// Build arguments - this is the CLI-specific part
	args := []string{
		"agents", "run",
		"--prompt", params.Prompt,
		"--model", params.Model,
	}

	// Provider (defaults to openrouter)
	provider := params.Provider
	if provider == "" {
		provider = "openrouter"
	}
	args = append(args, "--provider", provider)

	// Allowed tools
	if len(params.AllowedTools) > 0 {
		args = append(args, "--allowed-tools", strings.Join(params.AllowedTools, ","))
	}

	// Max turns
	if params.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", params.MaxTurns))
	}

	// Timeout
	if params.TimeoutSeconds > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", params.TimeoutSeconds))
	}

	// Directory scoping
	if params.WorkingDir != "" {
		args = append(args, "--directory", params.WorkingDir)
	}

	cmd := exec.CommandContext(ctx, p.binaryPath, args...)

	// Set working directory
	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	// Set environment variables
	if len(params.Environment) > 0 {
		cmd.Env = make([]string, 0, len(params.Environment))
		for k, v := range params.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd, nil
}

// ExtractSessionID parses the session ID from resource-opencode output.
func (p *ResourceOpenCodeProvider) ExtractSessionID(output string) string {
	// Pattern: "Created OpenCode session: <session-id>"
	const prefix = "Created OpenCode session:"
	idx := strings.Index(output, prefix)
	if idx == -1 {
		return ""
	}

	// Extract the session ID after the prefix
	remaining := output[idx+len(prefix):]
	// Split on whitespace to get the session ID
	fields := strings.Fields(remaining)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// --- Provider Selection ---

// ExecutionProviderSelector manages execution provider selection.
// This is the extension point for adding new execution backends.
type ExecutionProviderSelector struct {
	providers []ExecutionProvider
	fallback  ExecutionProvider
}

// NewExecutionProviderSelector creates a selector with the given providers.
// Providers are checked in order; the first available one is used.
func NewExecutionProviderSelector(providers []ExecutionProvider, fallback ExecutionProvider) *ExecutionProviderSelector {
	return &ExecutionProviderSelector{
		providers: providers,
		fallback:  fallback,
	}
}

// DefaultExecutionProviderSelector creates a selector with the default provider chain.
func DefaultExecutionProviderSelector() *ExecutionProviderSelector {
	return NewExecutionProviderSelector(
		[]ExecutionProvider{
			NewResourceOpenCodeProvider(),
		},
		nil, // No fallback - must have resource-opencode
	)
}

// ProviderSelectionResult describes which provider was selected and why.
type ProviderSelectionResult struct {
	Provider     ExecutionProvider
	ProviderName string
	Available    bool
	Reason       string
	CheckedCount int
	UsedFallback bool
}

// SelectProvider returns the first available execution provider.
// This is the central decision point for execution provider selection.
//
// Decision criteria:
//   - Check each provider in order
//   - Return the first one that is available
//   - If none available and fallback exists, use fallback
//   - If no fallback, return nil with error reason
func (s *ExecutionProviderSelector) SelectProvider(ctx context.Context) ProviderSelectionResult {
	result := ProviderSelectionResult{
		CheckedCount: 0,
	}

	for _, provider := range s.providers {
		result.CheckedCount++
		availability := provider.IsAvailable(ctx)

		if availability.Available {
			result.Provider = provider
			result.ProviderName = provider.Name()
			result.Available = true
			result.Reason = "selected first available provider"
			return result
		}
	}

	// No providers available - try fallback
	if s.fallback != nil {
		availability := s.fallback.IsAvailable(ctx)
		result.Provider = s.fallback
		result.ProviderName = s.fallback.Name()
		result.Available = availability.Available
		result.UsedFallback = true
		result.Reason = "using fallback provider"
		return result
	}

	// No providers at all
	result.Available = false
	result.Reason = "no execution providers available; install resource-opencode or configure an alternative"
	return result
}

// ListProviders returns information about all configured providers.
func (s *ExecutionProviderSelector) ListProviders(ctx context.Context) []ProviderInfo {
	infos := make([]ProviderInfo, 0, len(s.providers)+1)

	for _, provider := range s.providers {
		availability := provider.IsAvailable(ctx)
		info := ProviderInfo{
			Name:      provider.Name(),
			Available: availability.Available,
			Reason:    availability.Reason,
		}
		infos = append(infos, info)
	}

	if s.fallback != nil {
		availability := s.fallback.IsAvailable(ctx)
		infos = append(infos, ProviderInfo{
			Name:       s.fallback.Name() + " (fallback)",
			Available:  availability.Available,
			Reason:     availability.Reason,
			IsFallback: true,
		})
	}

	return infos
}

// ProviderInfo describes an execution provider's status.
type ProviderInfo struct {
	Name       string
	Available  bool
	Reason     string
	IsFallback bool
}
