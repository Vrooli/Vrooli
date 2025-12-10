package containment

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	// DefaultDockerImage is the default image used for containment.
	// This should be a minimal image with common development tools.
	DefaultDockerImage = "ubuntu:22.04"

	// DockerAvailabilityCheckTimeout is how long to wait when checking if Docker is available.
	DockerAvailabilityCheckTimeout = 5 * time.Second
)

// DockerProvider implements containment using Docker containers.
// Each agent execution runs in an isolated container with bind mounts
// for allowed paths.
type DockerProvider struct {
	// Image is the Docker image to use for containment.
	Image string

	// ExtraDockerArgs are additional arguments to pass to docker run.
	ExtraDockerArgs []string

	// DropCapabilities lists Linux capabilities to drop (default: all).
	DropCapabilities []string

	// NoNewPrivileges prevents privilege escalation in the container.
	NoNewPrivileges bool

	// ReadOnlyRootFS makes the container's root filesystem read-only.
	ReadOnlyRootFS bool
}

// NewDockerProvider creates a Docker containment provider with secure defaults.
func NewDockerProvider() *DockerProvider {
	return &DockerProvider{
		Image: DefaultDockerImage,
		DropCapabilities: []string{
			"ALL", // Drop all capabilities by default
		},
		NoNewPrivileges: true,
		ReadOnlyRootFS:  false, // Can't be true if we want to write test files
	}
}

// Type returns ContainmentTypeDocker.
func (p *DockerProvider) Type() ContainmentType {
	return ContainmentTypeDocker
}

// IsAvailable checks if Docker is installed and the daemon is running.
func (p *DockerProvider) IsAvailable(ctx context.Context) bool {
	checkCtx, cancel := context.WithTimeout(ctx, DockerAvailabilityCheckTimeout)
	defer cancel()

	// First check if docker binary exists
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	// Then check if daemon is running
	cmd := exec.CommandContext(checkCtx, "docker", "info")
	err = cmd.Run()
	return err == nil
}

// PrepareCommand creates a docker run command that sandboxes the given command.
func (p *DockerProvider) PrepareCommand(ctx context.Context, config ExecutionConfig) (*exec.Cmd, error) {
	if len(config.Command) == 0 {
		return nil, ErrNoCommand
	}

	// Build docker run arguments
	args := []string{"run", "--rm"}

	// Security options
	if p.NoNewPrivileges {
		args = append(args, "--security-opt=no-new-privileges:true")
	}

	// Drop capabilities
	for _, cap := range p.DropCapabilities {
		args = append(args, "--cap-drop="+cap)
	}

	// Read-only root filesystem (optional)
	if p.ReadOnlyRootFS {
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

	// Image name
	args = append(args, p.Image)

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

// WithImage returns a new DockerProvider using the specified image.
func (p *DockerProvider) WithImage(image string) *DockerProvider {
	newProvider := *p
	newProvider.Image = image
	return &newProvider
}

// WithExtraArgs returns a new DockerProvider with additional docker arguments.
func (p *DockerProvider) WithExtraArgs(args ...string) *DockerProvider {
	newProvider := *p
	newProvider.ExtraDockerArgs = append(newProvider.ExtraDockerArgs, args...)
	return &newProvider
}
