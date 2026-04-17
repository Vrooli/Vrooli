package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

func (a *App) collectProcessHealthSnapshot() maintenance.HealthSnapshot {
	snapshot, err := a.processSnapshot()
	if err != nil {
		return maintenance.HealthSnapshot{
			ZombieStatus:  "unknown",
			ZombieEmoji:   "❔",
			OrphanStatus:  "unknown",
			OrphanEmoji:   "❔",
			OverallStatus: "unknown",
		}
	}
	return snapshot.HealthSnapshot()
}

func (a *App) getEnhancedProcessMetrics() map[string]interface{} {
	snapshot, err := a.processSnapshot()
	if err != nil {
		return map[string]interface{}{
			"tracked_processes": 0,
			"running_tracked":   0,
			"child_processes":   0,
			"total_processes":   0,
			"zombie_processes":  0,
			"orphan_processes":  0,
		}
	}
	return map[string]interface{}{
		"tracked_processes": snapshot.TrackedProcesses,
		"running_tracked":   snapshot.RunningTracked,
		"child_processes":   snapshot.ChildProcesses,
		"total_processes":   snapshot.TotalProcesses,
		"zombie_processes":  snapshot.ZombieProcesses,
		"orphan_processes":  snapshot.OrphanProcesses,
	}
}

func (a *App) DiscoverScenarioPorts(scenarioName string) map[string]int {
	item, err := scenario.Load(a.Root, scenarioName, scenario.SandboxEnvFromEnv())
	if err != nil {
		return map[string]int{}
	}
	records, err := process.ReadScenarioRecords(a.Home, scenarioName)
	if err != nil {
		return map[string]int{}
	}
	return scenario.RuntimePorts(item.Manifest, process.LiveRecords(records))
}

func checkForkBomb() error {
	output, err := shell.Output(shell.Spec{
		Name: "ps",
		Args: []string{"aux"},
	})
	if err != nil {
		return err
	}
	if strings.Count(string(output), "\n") > 2000 {
		return &vroolierr.Error{
			Code:       "system_overload",
			Category:   "Runtime",
			HTTPStatus: http.StatusServiceUnavailable,
			Message:    "system overload: too many processes",
		}
	}
	return nil
}

func (a *App) discoverRunningScenarios() ([]RunningScenario, error) {
	views, err := a.Scenarios.Running()
	if err != nil {
		return nil, err
	}
	result := make([]RunningScenario, 0, len(views))
	for _, item := range views {
		result = append(result, RunningScenario{
			Name:      item.Name,
			Status:    item.Status,
			Processes: item.Processes,
			StartedAt: item.StartedAt,
			Runtime:   item.Runtime,
			Ports:     item.Ports,
		})
	}
	return result, nil
}

func (a *App) loadScenarioRuntime(name string) (scenario.Scenario, process.ScenarioRuntime, scenario.RuntimeDetails, error) {
	detail, err := a.Scenarios.Detail(name)
	if err != nil {
		return scenario.Scenario{}, process.ScenarioRuntime{}, scenario.RuntimeDetails{}, err
	}
	return detail.Scenario, detail.Runtime, detail.Details, nil
}

func (a *App) PerformHealthCheck(check HealthCheckConfig, scenarioName string, ports map[string]int) error {
	switch check.Type {
	case "http":
		target := check.Target
		for varName, port := range ports {
			target = strings.ReplaceAll(target, "${"+varName+"}", strconv.Itoa(port))
			target = strings.ReplaceAll(target, "$"+varName, strconv.Itoa(port))
		}
		if _, err := url.Parse(target); err != nil {
			return &vroolierr.Error{
				Code:       "invalid_healthcheck_url",
				Category:   "Usage",
				HTTPStatus: http.StatusBadRequest,
				Message:    fmt.Sprintf("invalid URL: %s", target),
				Err:        err,
			}
		}
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 5 * time.Second
		}
		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(target)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &vroolierr.Error{
				Code:       "http_healthcheck_failed",
				Category:   "Runtime",
				HTTPStatus: http.StatusBadGateway,
				Message:    fmt.Sprintf("HTTP %d", resp.StatusCode),
			}
		}
		return nil
	case "postgres":
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 3 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if _, err := a.LookPathFn("vrooli"); err == nil {
			if output, cmdErr := a.CommandFn(ctx, "vrooli", "resource", "status", "postgres", "--json"); cmdErr == nil {
				var status struct {
					Running   bool  `json:"running"`
					Healthy   *bool `json:"healthy"`
					Installed bool  `json:"installed"`
				}
				if err := json.Unmarshal(output, &status); err == nil {
					if !status.Installed {
						return &vroolierr.Error{Code: "postgres_not_installed", Category: "Runtime", HTTPStatus: http.StatusServiceUnavailable, Message: "postgres resource not installed"}
					}
					if !status.Running {
						return &vroolierr.Error{Code: "postgres_not_running", Category: "Runtime", HTTPStatus: http.StatusServiceUnavailable, Message: "postgres resource not running"}
					}
					if status.Healthy != nil && !*status.Healthy {
						return &vroolierr.Error{Code: "postgres_unhealthy", Category: "Runtime", HTTPStatus: http.StatusServiceUnavailable, Message: "postgres resource unhealthy"}
					}
					return nil
				}
			}
		}
		address := "127.0.0.1:5432"
		if parsed, err := parsePostgresAddress(check.Target); err == nil && parsed != "" {
			address = parsed
		}
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return &vroolierr.Error{
				Code:       "postgres_healthcheck_failed",
				Category:   "Runtime",
				HTTPStatus: http.StatusServiceUnavailable,
				Message:    fmt.Sprintf("postgres health check failed for %q", address),
				Err:        err,
			}
		}
		_ = conn.Close()
		return nil
	default:
		return &vroolierr.Error{
			Code:       "unsupported_healthcheck_type",
			Category:   "Usage",
			HTTPStatus: http.StatusBadRequest,
			Message:    fmt.Sprintf("unsupported health check type: %s", check.Type),
		}
	}
}

func parsePostgresAddress(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}
	if strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://") {
		u, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := u.Hostname()
		if host == "" {
			return "", nil
		}
		port := u.Port()
		if port == "" {
			port = "5432"
		}
		return net.JoinHostPort(host, port), nil
	}
	if strings.Contains(target, ":") {
		host, port, err := net.SplitHostPort(target)
		if err == nil && host != "" && port != "" {
			return net.JoinHostPort(host, port), nil
		}
		return "", err
	}
	return "", nil
}
