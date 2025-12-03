// Package infra provides infrastructure health checks
// [REQ:INFRA-DOCKER-001]
package infra

import (
	"context"
	"encoding/json"
	"os/exec"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// DockerCheck verifies Docker daemon is responsive
type DockerCheck struct{}

// NewDockerCheck creates a Docker health check
func NewDockerCheck() *DockerCheck { return &DockerCheck{} }

func (c *DockerCheck) ID() string          { return "infra-docker" }
func (c *DockerCheck) Title() string       { return "Docker Engine" }
func (c *DockerCheck) Description() string { return "Verifies Docker daemon is running and responsive" }
func (c *DockerCheck) Importance() string {
	return "Required for running containers - most Vrooli scenarios depend on Docker"
}
func (c *DockerCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *DockerCheck) IntervalSeconds() int       { return 60 }
func (c *DockerCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.MacOS}
}

func (c *DockerCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	output, err := cmd.Output()

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
