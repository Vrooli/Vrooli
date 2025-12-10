package containment

import (
	"context"
	"os"
	"os/exec"
)

// FallbackProvider runs commands without OS-level containment.
// This is the last resort when no sandbox is available.
// It provides NO security isolation - agents can access the full filesystem.
type FallbackProvider struct {
	// EmitWarnings controls whether to log warnings about missing containment.
	EmitWarnings bool
}

// NewFallbackProvider creates a new fallback containment provider.
func NewFallbackProvider() *FallbackProvider {
	return &FallbackProvider{
		EmitWarnings: true,
	}
}

// Type returns ContainmentTypeNone.
func (p *FallbackProvider) Type() ContainmentType {
	return ContainmentTypeNone
}

// IsAvailable always returns true - the fallback is always available.
func (p *FallbackProvider) IsAvailable(ctx context.Context) bool {
	return true
}

// PrepareCommand creates a standard exec.Cmd without any containment.
// The command runs with the same privileges as the parent process.
func (p *FallbackProvider) PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error) {
	if len(config.Command) == 0 {
		return nil, ErrNoCommand
	}

	cmd := exec.CommandContext(ctx, config.Command[0], config.Command[1:]...)

	// Set working directory
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range config.Environment {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	// Note: MaxMemoryMB, MaxCPUPercent, and NetworkAccess are ignored
	// in the fallback provider - there's no way to enforce them without
	// OS-level containment.

	return cmd, nil
}

// Info returns metadata about the fallback provider.
func (p *FallbackProvider) Info() ProviderInfo {
	return ProviderInfo{
		Type:          ContainmentTypeNone,
		Name:          "No Containment (Fallback)",
		Description:   "Runs commands directly without any sandboxing. Agents have full access to the filesystem.",
		SecurityLevel: 0,
		Requirements:  []string{"None - always available"},
	}
}
