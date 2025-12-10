package containment

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// --- Seams for testability ---

// CommandLookup is a seam for looking up command paths.
// This enables testing Docker availability without requiring Docker installation.
type CommandLookup interface {
	LookPath(file string) (string, error)
}

// CommandRunner is a seam for running commands to check availability.
// This enables testing without actually invoking Docker.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

// OSCommandLookup uses the real exec.LookPath.
type OSCommandLookup struct{}

// LookPath delegates to exec.LookPath.
func (l *OSCommandLookup) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// OSCommandRunner runs commands using exec.CommandContext.
type OSCommandRunner struct{}

// Run executes a command and returns any error.
func (r *OSCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

// --- End seams ---

// DockerProvider implements containment using Docker containers.
// Each agent execution runs in an isolated container with bind mounts
// for allowed paths.
type DockerProvider struct {
	// config holds all tunable levers for Docker containment.
	config Config

	// ExtraDockerArgs are additional arguments to pass to docker run.
	ExtraDockerArgs []string

	// Seams for testability (nil uses OS defaults)
	commandLookup CommandLookup
	commandRunner CommandRunner
}

// DockerProviderOption configures a DockerProvider during construction.
type DockerProviderOption func(*DockerProvider)

// WithCommandLookup sets a custom command lookup for testing.
func WithCommandLookup(lookup CommandLookup) DockerProviderOption {
	return func(p *DockerProvider) {
		p.commandLookup = lookup
	}
}

// WithCommandRunner sets a custom command runner for testing.
func WithCommandRunner(runner CommandRunner) DockerProviderOption {
	return func(p *DockerProvider) {
		p.commandRunner = runner
	}
}

// WithContainmentConfig sets a custom containment configuration.
func WithContainmentConfig(cfg Config) DockerProviderOption {
	return func(p *DockerProvider) {
		p.config = cfg
	}
}

// NewDockerProvider creates a Docker containment provider with secure defaults.
func NewDockerProvider(opts ...DockerProviderOption) *DockerProvider {
	p := &DockerProvider{
		config:        LoadConfigFromEnv(),
		commandLookup: &OSCommandLookup{},
		commandRunner: &OSCommandRunner{},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Type returns ContainmentTypeDocker.
func (p *DockerProvider) Type() ContainmentType {
	return ContainmentTypeDocker
}

// DockerAvailabilityCheck describes the result of checking Docker availability.
type DockerAvailabilityCheck struct {
	Available        bool
	Phase            string // "binary", "daemon", or "complete"
	BinaryFound      bool
	DaemonResponsive bool
	FailureReason    string
}

// CheckDockerAvailability performs a two-phase availability check.
// This is the central decision point for determining if Docker containment can be used.
//
// Decision phases:
//   - Phase 1 (Binary): Check if `docker` binary exists in PATH
//   - If missing: Docker is not installed, cannot use Docker containment
//   - Phase 2 (Daemon): Check if Docker daemon is running via `docker info`
//   - If fails: Docker is installed but daemon is not running
//
// Both phases must pass for Docker containment to be available.
// This two-phase approach provides clear diagnostics about why Docker is unavailable.
func (p *DockerProvider) CheckDockerAvailability(ctx context.Context) DockerAvailabilityCheck {
	checkCtx, cancel := context.WithTimeout(ctx, p.config.AvailabilityTimeout())
	defer cancel()

	result := DockerAvailabilityCheck{
		Phase: "binary",
	}

	// Phase 1: Check if docker binary exists
	_, err := p.commandLookup.LookPath("docker")
	if err != nil {
		result.Available = false
		result.BinaryFound = false
		result.FailureReason = "docker binary not found in PATH; Docker may not be installed"
		return result
	}
	result.BinaryFound = true
	result.Phase = "daemon"

	// Phase 2: Check if daemon is running
	err = p.commandRunner.Run(checkCtx, "docker", "info")
	if err != nil {
		result.Available = false
		result.DaemonResponsive = false
		result.FailureReason = "docker daemon not responding; Docker may not be running"
		return result
	}
	result.DaemonResponsive = true
	result.Phase = "complete"

	// Both phases passed
	result.Available = true
	return result
}

// IsAvailable checks if Docker is installed and the daemon is running.
// For detailed availability information, use CheckDockerAvailability instead.
func (p *DockerProvider) IsAvailable(ctx context.Context) bool {
	return p.CheckDockerAvailability(ctx).Available
}

// PrepareCommand creates a docker run command that sandboxes the given command.
func (p *DockerProvider) PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error) {
	if len(config.Command) == 0 {
		return nil, ErrNoCommand
	}

	// Build docker run arguments
	args := []string{"run", "--rm"}

	// Security options from config
	if p.config.NoNewPrivileges {
		args = append(args, "--security-opt=no-new-privileges:true")
	}

	// Drop capabilities based on config
	if p.config.DropAllCapabilities {
		args = append(args, "--cap-drop=ALL")
	}

	// Read-only root filesystem (from config)
	if p.config.ReadOnlyRootFS {
		args = append(args, "--read-only")
	}

	// Working directory
	if config.WorkingDir != "" {
		// Mount working directory as writable
		args = append(args, "-v", config.WorkingDir+":"+config.WorkingDir)
		args = append(args, "-w", config.WorkingDir)
	}

	// Mount allowed paths
	for _, path := range config.AllowedPaths {
		// If path is relative, make it absolute relative to working dir
		fullPath := path
		if !strings.HasPrefix(path, "/") && config.WorkingDir != "" {
			fullPath = config.WorkingDir + "/" + path
		}
		args = append(args, "-v", fullPath+":"+fullPath)
	}

	// Mount read-only paths
	for _, path := range config.ReadOnlyPaths {
		fullPath := path
		if !strings.HasPrefix(path, "/") && config.WorkingDir != "" {
			fullPath = config.WorkingDir + "/" + path
		}
		args = append(args, "-v", fullPath+":"+fullPath+":ro")
	}

	// Environment variables
	for key, value := range config.Environment {
		args = append(args, "-e", key+"="+value)
	}

	// Resource limits
	if config.MaxMemoryMB > 0 {
		args = append(args, "--memory="+fmt.Sprintf("%dm", config.MaxMemoryMB))
		// Also set memory-swap to same value to prevent swap usage
		args = append(args, "--memory-swap="+fmt.Sprintf("%dm", config.MaxMemoryMB))
	}

	if config.MaxCPUPercent > 0 {
		// Docker expects CPU quota in terms of 100000 = 1 CPU
		// So 50% = 50000, 200% = 200000 (2 CPUs)
		cpuQuota := config.MaxCPUPercent * 1000
		args = append(args, "--cpu-quota="+fmt.Sprintf("%d", cpuQuota))
	}

	// Network access
	if !config.NetworkAccess {
		args = append(args, "--network=none")
	}

	// Add any extra docker args
	args = append(args, p.ExtraDockerArgs...)

	// Image name from config
	args = append(args, p.config.DockerImage)

	// The actual command to run
	args = append(args, config.Command...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	return cmd, nil
}

// Info returns metadata about the Docker containment provider.
func (p *DockerProvider) Info() ProviderInfo {
	return ProviderInfo{
		Type: ContainmentTypeDocker,
		Name: "Docker Container Isolation",
		Description: "Runs agents in isolated Docker containers with bind mounts. " +
			"Provides strong process and filesystem isolation with resource limits.",
		SecurityLevel: 7,
		Requirements: []string{
			"Docker installed (docker binary available)",
			"Docker daemon running",
			"User has permission to run docker commands",
		},
	}
}

// GetConfig returns the current containment configuration.
func (p *DockerProvider) GetConfig() Config {
	return p.config
}

// WithExtraArgs returns a new DockerProvider with additional docker arguments.
func (p *DockerProvider) WithExtraArgs(args ...string) *DockerProvider {
	newProvider := *p
	newProvider.ExtraDockerArgs = append(newProvider.ExtraDockerArgs, args...)
	return &newProvider
}
