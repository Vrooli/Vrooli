// Package gpuaccess contains the dependency-light, container-scoped GPU
// probe shared by the resource lifecycle and host maintenance repair paths.
package gpuaccess

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
)

type State string

const (
	OK      State = "ok"
	Revoked State = "revoked"
	Unknown State = "unknown"
)

type ExecFunc func(context.Context, string, string) ([]byte, error)

func Verify(ctx context.Context, container, probe string) (State, string) {
	return VerifyWithExec(ctx, container, probe, DefaultExec)
}

func VerifyWithExec(ctx context.Context, container, probe string, run ExecFunc) (State, string) {
	container = strings.TrimSpace(container)
	probe = strings.TrimSpace(probe)
	if container == "" {
		return Unknown, "container name is empty"
	}
	if probe != "nvidia" {
		return Unknown, fmt.Sprintf("GPU probe %q is unsupported", probe)
	}
	output, err := run(ctx, container, probe)
	text := strings.TrimSpace(string(output))
	if err == nil {
		return OK, "container opened /dev/nvidiactl"
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "not permitted") || strings.Contains(lower, "operation not permitted") || strings.Contains(lower, "permission denied") {
		return Revoked, text
	}
	if text == "" {
		text = err.Error()
	}
	return Unknown, text
}

func DefaultExec(ctx context.Context, container, probe string) ([]byte, error) {
	if probe != "nvidia" {
		return nil, fmt.Errorf("unsupported GPU probe %q", probe)
	}
	return shell.NewCommandContext(ctx, "docker", "exec", container, "sh", "-c", "exec 3<>/dev/nvidiactl && printf ok").CombinedOutput()
}
