package resources

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

type HealthResult struct {
	Healthy bool
	Message string
}

func (c *Controller) runResourceHealthChecks(ctx context.Context, manifest ResourceManifest) (HealthResult, error) {
	if len(manifest.HealthChecks) == 0 {
		return HealthResult{}, nil
	}
	for _, check := range manifest.HealthChecks {
		result, err := c.runResourceHealthCheck(ctx, check)
		if err != nil {
			return result, err
		}
		if !result.Healthy {
			return result, nil
		}
	}
	return HealthResult{Healthy: true, Message: "healthy"}, nil
}

func (c *Controller) runResourceHealthCheck(ctx context.Context, check ResourceHealthCheck) (HealthResult, error) {
	timeout := 5 * time.Second
	if check.TimeoutSeconds > 0 {
		timeout = time.Duration(check.TimeoutSeconds) * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch check.Type {
	case "tcp":
		conn, err := (&net.Dialer{}).DialContext(checkCtx, "tcp", check.Target)
		if err != nil {
			return HealthResult{Message: fmt.Sprintf("tcp check failed for %s", check.Target)}, nil
		}
		_ = conn.Close()
		return HealthResult{Healthy: true, Message: "healthy"}, nil
	case "http":
		req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, check.Target, nil)
		if err != nil {
			return HealthResult{}, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return HealthResult{Message: fmt.Sprintf("http check failed for %s", check.Target)}, nil
		}
		defer resp.Body.Close()
		if len(check.ExpectedStatus) == 0 {
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return HealthResult{Healthy: true, Message: "healthy"}, nil
			}
		} else if slices.Contains(check.ExpectedStatus, resp.StatusCode) {
			return HealthResult{Healthy: true, Message: "healthy"}, nil
		}
		return HealthResult{Message: fmt.Sprintf("http check returned %d", resp.StatusCode)}, nil
	case "command":
		cmd := shell.Command(shell.Spec{
			Name:   check.Command[0],
			Args:   check.Command[1:],
			Dir:    c.Root,
			Env:    resourceEnv(c.Root, c.Home),
			Stdin:  nil,
			Stdout: io.Discard,
			Stderr: io.Discard,
		})
		result := runCommandResource(checkCtx, cmd)
		if result.err != nil {
			return HealthResult{Message: fmt.Sprintf("command check failed for %s", strings.Join(check.Command, " "))}, nil
		}
		return HealthResult{Healthy: true, Message: "healthy"}, nil
	default:
		return HealthResult{}, fmt.Errorf("unsupported health check type %q", check.Type)
	}
}
