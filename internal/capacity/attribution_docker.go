package capacity

import (
	"context"
	"os/exec"
	"strings"
)

// dockerCLINameSource resolves a container ID to its name via `docker inspect`.
// This is the live attribution seam; unit tests inject a fake DockerNameSource
// instead so they never shell out.
type dockerCLINameSource struct{}

func (dockerCLINameSource) ContainerName(ctx context.Context, containerID string) (string, bool) {
	id := strings.TrimSpace(containerID)
	if id == "" {
		return "", false
	}
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Name}}", id)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", false
	}
	return name, true
}
