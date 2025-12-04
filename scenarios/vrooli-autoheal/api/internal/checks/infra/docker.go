// Package infra provides infrastructure health checks
// [REQ:INFRA-DOCKER-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package infra

import (
	"context"
	"encoding/json"
	"runtime"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DockerCheck verifies Docker daemon is responsive
type DockerCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
}

// DockerCheckOption configures a DockerCheck.
type DockerCheckOption func(*DockerCheck)

// WithDockerExecutor sets the command executor (for testing).
func WithDockerExecutor(executor checks.CommandExecutor) DockerCheckOption {
	return func(c *DockerCheck) {
		c.executor = executor
	}
}

// NewDockerCheck creates a Docker health check
// Platform capabilities are optional (for recovery actions).
func NewDockerCheck(caps ...*platform.Capabilities) *DockerCheck {
	c := &DockerCheck{
		executor: checks.DefaultExecutor,
	}
	if len(caps) > 0 {
		c.caps = caps[0]
	}
	return c
}

// NewDockerCheckWithOptions creates a Docker health check with options.
func NewDockerCheckWithOptions(caps *platform.Capabilities, opts ...DockerCheckOption) *DockerCheck {
	c := &DockerCheck{
		caps:     caps,
		executor: checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *DockerCheck) ID() string          { return "infra-docker" }
func (c *DockerCheck) Title() string       { return "Docker Engine" }
func (c *DockerCheck) Description() string { return "Verifies Docker daemon is running and responsive" }
func (c *DockerCheck) Importance() string {
	return "Required for running containers - most Vrooli scenarios depend on Docker"
}
func (c *DockerCheck) Category() checks.Category { return checks.CategoryInfrastructure }
func (c *DockerCheck) IntervalSeconds() int      { return 60 }
func (c *DockerCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.MacOS}
}

func (c *DockerCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Use injected executor
	output, err := c.executor.Output(ctx, "docker", "info", "--format", "{{json .}}")

	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Docker daemon not responsive"
		result.Details["error"] = err.Error()
		return result
	}

	// Parse docker info
	var info struct {
		ServerVersion string `json:"ServerVersion"`
		Containers    int    `json:"Containers"`
		Running       int    `json:"ContainersRunning"`
	}
	if err := json.Unmarshal(output, &info); err == nil {
		result.Details["version"] = info.ServerVersion
		result.Details["containers"] = info.Containers
		result.Details["running"] = info.Running
	}

	result.Status = checks.StatusOK
	result.Message = "Docker daemon is healthy"
	return result
}

// RecoveryActions returns available recovery actions for Docker check
// [REQ:HEAL-ACTION-001]
func (c *DockerCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	isLinux := runtime.GOOS == "linux"
	isMac := runtime.GOOS == "darwin"
	hasSystemd := c.caps != nil && c.caps.SupportsSystemd

	isResponsive := false
	if lastResult != nil && lastResult.Status == checks.StatusOK {
		isResponsive = true
	}

	actions := []checks.RecoveryAction{
		{
			ID:          "restart",
			Name:        "Restart Docker",
			Description: "Restart the Docker daemon service",
			Dangerous:   true, // Restarting Docker stops all containers
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "start",
			Name:        "Start Docker",
			Description: "Start the Docker daemon service",
			Dangerous:   false,
			Available:   (isLinux && hasSystemd) && !isResponsive,
		},
		{
			ID:          "prune",
			Name:        "Prune System",
			Description: "Remove unused Docker data (stopped containers, unused networks, dangling images)",
			Dangerous:   true, // Removes data
			Available:   isResponsive,
		},
		{
			ID:          "logs",
			Name:        "View Logs",
			Description: "View recent Docker daemon logs",
			Dangerous:   false,
			Available:   isLinux && hasSystemd,
		},
		{
			ID:          "info",
			Name:        "Docker Info",
			Description: "Get detailed Docker daemon information",
			Dangerous:   false,
			Available:   true,
		},
	}

	// macOS uses Docker Desktop, not systemd
	if isMac {
		actions = append(actions, checks.RecoveryAction{
			ID:          "open-desktop",
			Name:        "Open Docker Desktop",
			Description: "Open Docker Desktop application (macOS)",
			Dangerous:   false,
			Available:   true,
		})
	}

	return actions
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *DockerCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "restart":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "docker")
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to restart Docker daemon"
			return result
		}

		// Verify Docker is responsive after restart
		return c.verifyRecovery(ctx, result, "restart", start)

	case "start":
		output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "start", "docker")
		result.Output = string(output)

		if err != nil {
			result.Duration = time.Since(start)
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to start Docker daemon"
			return result
		}

		// Verify Docker is responsive after start
		return c.verifyRecovery(ctx, result, "start", start)

	case "prune":
		// Use docker system prune with --force to avoid interactive prompt
		output, err := c.executor.CombinedOutput(ctx, "docker", "system", "prune", "--force")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to prune Docker system"
			return result
		}

		result.Success = true
		result.Message = "Docker system pruned successfully"
		return result

	case "logs":
		output, err := c.executor.CombinedOutput(ctx, "journalctl", "-u", "docker", "-n", "100", "--no-pager")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to retrieve Docker logs"
			return result
		}

		result.Success = true
		result.Message = "Retrieved Docker daemon logs"
		return result

	case "info":
		output, err := c.executor.CombinedOutput(ctx, "docker", "info")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to get Docker info"
			return result
		}

		result.Success = true
		result.Message = "Retrieved Docker daemon information"
		return result

	case "open-desktop":
		// macOS: open Docker Desktop
		output, err := c.executor.CombinedOutput(ctx, "open", "-a", "Docker")
		result.Duration = time.Since(start)
		result.Output = string(output)

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			result.Message = "Failed to open Docker Desktop"
			return result
		}

		result.Success = true
		result.Message = "Docker Desktop is opening"
		return result

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// verifyRecovery checks that Docker is actually responsive after a start/restart action
func (c *DockerCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, actionID string, start time.Time) checks.ActionResult {
	// Wait for Docker to initialize
	time.Sleep(5 * time.Second)

	// Check Docker status
	checkResult := c.Run(ctx)
	result.Duration = time.Since(start)

	if checkResult.Status == checks.StatusOK {
		result.Success = true
		result.Message = "Docker daemon " + actionID + " successful and verified responsive"
		result.Output += "\n\n=== Verification ===\n" + checkResult.Message
	} else {
		result.Success = false
		result.Error = "Docker not responsive after " + actionID
		result.Message = "Docker daemon " + actionID + " completed but verification failed"
		result.Output += "\n\n=== Verification Failed ===\n" + checkResult.Message
	}

	return result
}

// Ensure DockerCheck implements HealableCheck
var _ checks.HealableCheck = (*DockerCheck)(nil)
