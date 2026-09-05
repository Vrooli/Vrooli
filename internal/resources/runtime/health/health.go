package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/shell"
)

type Result struct {
	// Healthy is true only when every readiness check passed and no liveness
	// check failed.
	Healthy bool
	Message string
	// Serving is true when every readiness check passed, whether or not a
	// liveness check failed. A resource whose liveness check fails still
	// answers requests, so a consumer must be able to tell it from a resource
	// that is down.
	Serving bool
	// LivenessFailed names the first liveness check that failed. Empty means
	// none did.
	LivenessFailed string
}

type Config struct {
	Root       string
	Env        []string
	Runner     func(context.Context, *exec.Cmd) ([]byte, error)
	HTTPClient *http.Client
}

// RunChecks runs both declared check kinds and combines them.
//
// A failing readiness check means the resource cannot answer: healthy and
// serving are both false. A failing liveness check means the resource answers
// but is not meeting its contract — running below its declared accelerator
// backend is the canonical case — so healthy is false while serving stays true.
// Collapsing the two would either hide a degradation or turn a working resource
// into an outage.
func RunChecks(ctx context.Context, checks []manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
	if len(checks) == 0 {
		return Result{}, nil
	}
	var liveness []manifestpkg.ResourceHealthCheck
	for _, check := range checks {
		if isLivenessCheck(check) {
			liveness = append(liveness, check)
			continue
		}
		result, err := RunCheck(ctx, check, cfg)
		if err != nil {
			return result, err
		}
		if !result.Healthy {
			return result, nil
		}
	}

	combined := Result{Healthy: true, Serving: true, Message: "healthy"}
	for _, check := range liveness {
		result, err := RunCheck(ctx, check, cfg)
		if err != nil {
			// A liveness check that could not run is not a failing liveness
			// check. Report the resource as serving and say what could not be
			// determined rather than inventing a verdict.
			combined.Message = fmt.Sprintf("liveness check %q could not run: %v", checkName(check), err)
			return combined, nil
		}
		if result.Healthy {
			continue
		}
		combined.Healthy = false
		combined.LivenessFailed = checkName(check)
		combined.Message = fmt.Sprintf("degraded: liveness check %q failed", combined.LivenessFailed)
		if strings.TrimSpace(result.Message) != "" {
			combined.Message += ": " + result.Message
		}
		return combined, nil
	}
	return combined, nil
}

func isLivenessCheck(check manifestpkg.ResourceHealthCheck) bool {
	return strings.EqualFold(strings.TrimSpace(check.Kind), "liveness")
}

// checkName gives a liveness check a stable name for the operator message.
func checkName(check manifestpkg.ResourceHealthCheck) string {
	if len(check.Command) > 0 {
		return strings.Join(check.Command, " ")
	}
	if target := strings.TrimSpace(check.Target); target != "" {
		return target
	}
	return check.Type
}

func RunCheck(ctx context.Context, check manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
	timeout := tuning.ServiceHealthTimeout()
	if check.TimeoutSeconds > 0 {
		timeout = time.Duration(check.TimeoutSeconds) * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, tuning.ResourceHealthCheckTimeout(timeout))
	defer cancel()

	switch check.Type {
	case "tcp":
		target := renderTarget(check.Target, cfg.Env)
		conn, err := (&net.Dialer{}).DialContext(checkCtx, "tcp", target)
		if err != nil {
			return Result{Message: fmt.Sprintf("tcp check failed for %s", target)}, nil
		}
		_ = conn.Close()
		return Result{Healthy: true, Message: "healthy"}, nil
	case "http":
		target := renderTarget(check.Target, cfg.Env)
		client := cfg.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, target, nil)
		if err != nil {
			return Result{}, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return Result{Message: fmt.Sprintf("http check failed for %s", target)}, nil
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

func renderTarget(target string, env []string) string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	return os.Expand(target, func(key string) string {
		if value, ok := values[key]; ok {
			return value
		}
		return "${" + key + "}"
	})
}
