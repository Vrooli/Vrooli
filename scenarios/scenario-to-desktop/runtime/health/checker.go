// Package health provides health and readiness checking for bundle services.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/deps"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/ports"
)

// Status represents the current state of a service.
type Status struct {
	Ready     bool      `json:"ready"`
	Skipped   bool      `json:"skipped,omitempty"`
	Message   string    `json:"message,omitempty"`
	ExitCode  *int      `json:"exit_code,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	ReadyAt   time.Time `json:"ready_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Checker abstracts health checking for testing.
type Checker interface {
	// WaitForReadiness waits for a service to become ready.
	WaitForReadiness(ctx context.Context, serviceID string) error
	// CheckOnce performs a single health check.
	CheckOnce(ctx context.Context, serviceID string) bool
	// WaitForDependencies waits for all service dependencies to be ready.
	WaitForDependencies(ctx context.Context, svc *manifest.Service) error
}

// Monitor implements Checker for service health monitoring.
type Monitor struct {
	manifest *manifest.Manifest
	ports    ports.Allocator
	dialer   infra.NetworkDialer
	cmd      infra.CommandRunner
	fs       infra.FileSystem
	clock    infra.Clock
	appData  string

	// StatusGetter provides access to service status from the orchestrator.
	StatusGetter     func(id string) (Status, bool)
	OnReadinessProbe func(serviceID string, attempt int, success bool, at time.Time)
}

// MonitorConfig holds the dependencies for Monitor.
type MonitorConfig struct {
	Manifest         *manifest.Manifest
	Ports            ports.Allocator
	Dialer           infra.NetworkDialer
	CmdRunner        infra.CommandRunner
	FS               infra.FileSystem
	Clock            infra.Clock
	AppData          string
	StatusGetter     func(id string) (Status, bool)
	OnReadinessProbe func(serviceID string, attempt int, success bool, at time.Time)
}

// NewMonitor creates a new Monitor with the given dependencies.
func NewMonitor(cfg MonitorConfig) *Monitor {
	return &Monitor{
		manifest:         cfg.Manifest,
		ports:            cfg.Ports,
		dialer:           cfg.Dialer,
		cmd:              cfg.CmdRunner,
		fs:               cfg.FS,
		clock:            cfg.Clock,
		appData:          cfg.AppData,
		StatusGetter:     cfg.StatusGetter,
		OnReadinessProbe: cfg.OnReadinessProbe,
	}
}

// WaitForReadiness waits for a service to become ready based on its readiness configuration.
func (m *Monitor) WaitForReadiness(ctx context.Context, serviceID string) error {
	svc := deps.FindService(m.manifest.Services, serviceID)
	if svc == nil {
		return fmt.Errorf("service %s not found", serviceID)
	}

	timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch svc.Readiness.Type {
	case "health_success":
		return m.pollReadinessHealth(ctx, *svc)
	case "port_open":
		port, err := m.ports.Resolve(svc.ID, svc.Readiness.PortName)
		if err != nil {
			return err
		}
		return m.waitForPort(ctx, port)
	case "log_match":
		return m.waitForLogPattern(ctx, *svc)
	case "dependencies_ready":
		return m.WaitForDependencies(ctx, svc)
	default:
		return fmt.Errorf("unknown readiness type %q", svc.Readiness.Type)
	}
}

// pollReadinessHealth uses the startup deadline, independently of the operator
// monitoring interval and retry count. WaitForReadiness owns that deadline.
func (m *Monitor) pollReadinessHealth(ctx context.Context, svc manifest.Service) error {
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		ok := m.checkHealthOnce(ctx, svc, 2*time.Second)
		if m.OnReadinessProbe != nil {
			m.OnReadinessProbe(svc.ID, attempt, ok, m.clock.Now())
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.clock.After(100 * time.Millisecond):
		}
	}
}

// CheckOnce performs a single health check for a service.
func (m *Monitor) CheckOnce(ctx context.Context, serviceID string) bool {
	svc := deps.FindService(m.manifest.Services, serviceID)
	if svc == nil {
		return false
	}

	timeout := time.Duration(svc.Health.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	return m.checkHealthOnce(ctx, *svc, timeout)
}

// checkHealthOnce performs a single health check.
func (m *Monitor) checkHealthOnce(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	switch svc.Health.Type {
	case "http":
		return m.checkHTTPHealth(ctx, svc, timeout)
	case "tcp":
		port, err := m.ports.Resolve(svc.ID, svc.Health.PortName)
		return err == nil && m.checkTCPPort(ctx, port, timeout)
	case "command":
		return m.checkCommandHealth(ctx, svc, timeout)
	case "log_match":
		return m.logMatches(svc, svc.Health.Path)
	default:
		return false
	}
}

// checkHTTPHealth verifies service health via HTTP endpoint.
func (m *Monitor) checkHTTPHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	port, err := m.ports.Resolve(svc.ID, svc.Health.PortName)
	if err != nil {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, svc.Health.Path)
	client := &http.Client{Timeout: timeout}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	if svc.Health.Path != "/health" {
		return true
	}

	return parseHealthBody(resp.Body)
}

// parseHealthBody checks if the JSON health response body indicates a healthy state.
func parseHealthBody(body io.Reader) bool {
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return false
	}
	var payload struct {
		Status    string `json:"status"`
		Readiness *bool  `json:"readiness"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	if payload.Status == "" && payload.Readiness == nil {
		return false
	}
	if payload.Status == "unhealthy" {
		return false
	}
	if payload.Readiness != nil && !*payload.Readiness {
		return false
	}
	if payload.Status != "" && payload.Status != "healthy" && payload.Status != "degraded" && payload.Status != "ok" {
		return false
	}
	return true
}

// checkTCPPort shares deadline-bound dialing for health and port readiness.
func (m *Monitor) checkTCPPort(ctx context.Context, port int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := m.dialer.DialContext(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// checkCommandHealth verifies service health by running a command.
func (m *Monitor) checkCommandHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	if len(svc.Health.Command) == 0 {
		return false
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := m.cmd.Run(cmdCtx, svc.Health.Command[0], svc.Health.Command[1:])
	return err == nil
}

// waitForLogPattern waits for a pattern to appear in service logs.
func (m *Monitor) waitForLogPattern(ctx context.Context, svc manifest.Service) error {
	if svc.Readiness.Pattern == "" {
		return fmt.Errorf("log_match readiness missing pattern")
	}
	re, err := regexp.Compile(svc.Readiness.Pattern)
	if err != nil {
		return fmt.Errorf("invalid readiness pattern: %w", err)
	}

	ticker := m.clock.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if m.logMatches(svc, re.String()) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C():
		}
	}
}

// logMatches checks if the service log contains a pattern.
func (m *Monitor) logMatches(svc manifest.Service, pattern string) bool {
	if svc.LogDir == "" {
		return false
	}
	logPath := manifest.ResolvePath(m.appData, svc.LogDir)
	data, err := m.fs.ReadFile(logPath)
	if err != nil {
		return false
	}
	ok, _ := regexp.Match(pattern, data)
	return ok
}

// waitForPort waits for a TCP port to become available.
func (m *Monitor) waitForPort(ctx context.Context, port int) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if m.checkTCPPort(ctx, port, 500*time.Millisecond) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.clock.After(250 * time.Millisecond):
		}
	}
}

// WaitForDependencies waits for all service dependencies to be ready.
func (m *Monitor) WaitForDependencies(ctx context.Context, svc *manifest.Service) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}

	if m.StatusGetter == nil {
		return fmt.Errorf("status getter not configured")
	}

	deadline := m.clock.Now().Add(2 * time.Minute)
	for {
		allReady := true
		for _, dep := range svc.Dependencies {
			status, ok := m.StatusGetter(dep)
			if !ok {
				return fmt.Errorf("dependency %s missing", dep)
			}
			if !status.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return nil
		}
		if m.clock.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready: %v", svc.Dependencies)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.clock.After(250 * time.Millisecond):
		}
	}
}

// Ensure Monitor implements Checker.
var _ Checker = (*Monitor)(nil)
