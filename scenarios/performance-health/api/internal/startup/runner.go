package startup

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/api-core/metrics"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// StatusClient is the narrow slice of vrooli-cli-go the runner needs.
type StatusClient interface {
	ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error)
}

// CLIRunner is the real Runner: it restarts the target scenario and polls
// `vrooli scenario status` until it reports healthy, timing the overall
// time-to-healthy plus per-surface (api/ui) port reachability, and wrapping the
// whole operation in a metrics collector for the resource envelope.
//
// This is the migrated home of structure-health's former perf CLIRunner (axis ②
// of the three-axis performance model).
type CLIRunner struct {
	Status StatusClient
	// Restart restarts the scenario; defaults to `vrooli scenario restart <name>`.
	Restart func(ctx context.Context, scenario string) error
	// Env is the host CaptureEnvironment for richer metrics (optional).
	Env *commonv1.CaptureEnvironment
	// PollInterval between status polls (defaults to 500ms).
	PollInterval time.Duration
}

var _ Runner = (*CLIRunner)(nil)

// Measure restarts the scenario and records its startup performance.
func (r *CLIRunner) Measure(ctx context.Context, scenario string, timeout time.Duration) (Measurement, error) {
	if strings.TrimSpace(scenario) == "" {
		return Measurement{}, fmt.Errorf("startup: scenario is required")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	poll := r.PollInterval
	if poll <= 0 {
		poll = 500 * time.Millisecond
	}
	restart := r.Restart
	if restart == nil {
		restart = execRestart
	}

	collector := metrics.Start(metrics.WithEnvironment(r.Env))
	start := time.Now()

	m := Measurement{Scenario: scenario, CapturedAt: start}
	if err := restart(ctx, scenario); err != nil {
		m.Metrics = collector.Stop()
		m.Note = fmt.Sprintf("restart failed: %v", err)
		return m, fmt.Errorf("startup: restart %q: %w", scenario, err)
	}

	deadline := start.Add(timeout)
	surfaceFirstReachable := map[string]time.Duration{}
	ports := map[string]int32{}

	for {
		item := r.statusItem(ctx, scenario)
		if item != nil {
			for name, port := range item.GetPorts() {
				ports[name] = port
			}
			for name, port := range ports {
				if _, seen := surfaceFirstReachable[name]; seen {
					continue
				}
				if portReachable(port) {
					surfaceFirstReachable[name] = time.Since(start)
				}
			}
			if isHealthy(item) {
				m.Healthy = true
				m.TimeToHealthyMs = time.Since(start).Milliseconds()
				break
			}
		}
		if time.Now().After(deadline) {
			m.TimeToHealthyMs = time.Since(start).Milliseconds()
			m.Note = fmt.Sprintf("did not reach healthy within %s", timeout)
			break
		}
		select {
		case <-ctx.Done():
			m.Metrics = collector.Stop()
			m.Note = "context cancelled"
			return m, ctx.Err()
		case <-time.After(poll):
		}
	}

	m.SurfaceTimings = surfaceTimings(surfaceFirstReachable)
	m.Metrics = collector.Stop()
	return m, nil
}

func (r *CLIRunner) statusItem(ctx context.Context, scenario string) *cliv1.ScenarioStatusItem {
	if r.Status == nil {
		return nil
	}
	resp, err := r.Status.ScenarioStatus(ctx, scenario)
	if err != nil || resp == nil {
		return nil
	}
	return resp.GetScenario()
}

// isHealthy reports whether the status item indicates a healthy, running scenario.
func isHealthy(item *cliv1.ScenarioStatusItem) bool {
	if item == nil {
		return false
	}
	if !strings.EqualFold(item.GetStatus(), "running") {
		return false
	}
	if strings.TrimSpace(item.GetHealthError()) != "" {
		return false
	}
	if hv := item.GetHealthStatus(); hv != nil {
		if s := strings.TrimSpace(hv.GetStringValue()); s != "" {
			return strings.EqualFold(s, "healthy")
		}
	}
	// No explicit health value: a running scenario with no health error is
	// treated as healthy.
	return true
}

func portReachable(port int32) bool {
	if port <= 0 {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func surfaceTimings(first map[string]time.Duration) []SurfaceTiming {
	if len(first) == 0 {
		return nil
	}
	out := make([]SurfaceTiming, 0, len(first))
	for name, d := range first {
		out = append(out, SurfaceTiming{Surface: name, TimeToHealthyMs: d.Milliseconds(), Healthy: true})
	}
	return out
}

// execRestart runs `vrooli scenario restart <scenario>` to completion.
func execRestart(ctx context.Context, scenario string) error {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "restart", scenario)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
