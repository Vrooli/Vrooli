package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"time"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/shell"
)

type Result struct {
	Healthy bool
	Message string
}

type Config struct {
	Root       string
	Env        []string
	Runner     func(context.Context, *exec.Cmd) ([]byte, error)
	HTTPClient *http.Client
}

func RunChecks(ctx context.Context, checks []manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
	if len(checks) == 0 {
		return Result{}, nil
	}
	for _, check := range checks {
		result, err := RunCheck(ctx, check, cfg)
		if err != nil {
			return result, err
		}
		if !result.Healthy {
			return result, nil
		}
	}
	return Result{Healthy: true, Message: "healthy"}, nil
}

func RunCheck(ctx context.Context, check manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
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
			return Result{Message: fmt.Sprintf("tcp check failed for %s", check.Target)}, nil
		}
		_ = conn.Close()
		return Result{Healthy: true, Message: "healthy"}, nil
	case "http":
		client := cfg.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, check.Target, nil)
		if err != nil {
			return Result{}, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return Result{Message: fmt.Sprintf("http check failed for %s", check.Target)}, nil
		}
		defer resp.Body.Close()
		if len(check.ExpectedStatus) == 0 {
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return Result{Healthy: true, Message: "healthy"}, nil
			}
		} else if slices.Contains(check.ExpectedStatus, resp.StatusCode) {
			return Result{Healthy: true, Message: "healthy"}, nil
		}
		return Result{Message: fmt.Sprintf("http check returned %d", resp.StatusCode)}, nil
	case "command":
		if len(check.Command) == 0 {
			return Result{}, fmt.Errorf("command health check requires a command")
		}
		runner := cfg.Runner
		if runner == nil {
			runner = func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
				return cmd.CombinedOutput()
			}
		}
		cmd := shell.Command(shell.Spec{
			Name:  check.Command[0],
			Args:  check.Command[1:],
			Dir:   cfg.Root,
			Env:   cfg.Env,
			Stdin: nil,
		})
		if _, err := runner(checkCtx, cmd); err != nil {
			return Result{Message: fmt.Sprintf("command check failed for %s", strings.Join(check.Command, " "))}, nil
		}
		return Result{Healthy: true, Message: "healthy"}, nil
	default:
		return Result{}, fmt.Errorf("unsupported health check type %q", check.Type)
	}
}
