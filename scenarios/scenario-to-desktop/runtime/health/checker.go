// Package health provides health and readiness checking for bundle services.
package health

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/ports"
)

// Status represents the current state of a service.
type Status struct {
	Ready    bool   `json:"ready"`
	Message  string `json:"message,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
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
	StatusGetter func(id string) (Status, bool)
}

// MonitorConfig holds the dependencies for Monitor.
type MonitorConfig struct {
	Manifest     *manifest.Manifest
	Ports        ports.Allocator
	Dialer       infra.NetworkDialer
	CmdRunner    infra.CommandRunner
	FS           infra.FileSystem
	Clock        infra.Clock
	AppData      string
	StatusGetter func(id string) (Status, bool)
}

// NewMonitor creates a new Monitor with the given dependencies.
func NewMonitor(cfg MonitorConfig) *Monitor {
	return &Monitor{
		manifest:     cfg.Manifest,
		ports:        cfg.Ports,
		dialer:       cfg.Dialer,
		cmd:          cfg.CmdRunner,
		fs:           cfg.FS,
		clock:        cfg.Clock,
		appData:      cfg.AppData,
		StatusGetter: cfg.StatusGetter,
	}
}

// findService looks up a service by ID.
func (m *Monitor) findService(id string) *manifest.Service {
	for i := range m.manifest.Services {
		if m.manifest.Services[i].ID == id {
			return &m.manifest.Services[i]
		}
	}
	return nil
}

// WaitForReadiness waits for a service to become ready based on its readiness configuration.
func (m *Monitor) WaitForReadiness(ctx context.Context, serviceID string) error {
	svc := m.findService(serviceID)
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
		return m.pollHealth(ctx, *svc)
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

// CheckOnce performs a single health check for a service.
func (m *Monitor) CheckOnce(ctx context.Context, serviceID string) bool {
	svc := m.findService(serviceID)
	if svc == nil {
		return false
	}

	timeout := time.Duration(svc.Health.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	return m.checkHealthOnce(ctx, *svc, timeout)
}

// pollHealth repeatedly checks health until success or timeout.
func (m *Monitor) pollHealth(ctx context.Context, svc manifest.Service) error {
	interval := time.Duration(svc.Health.IntervalMs) * time.Millisecond
	if interval == 0 {
		interval = 500 * time.Millisecond
	}
	timeout := time.Duration(svc.Health.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	retries := svc.Health.Retries
	if retries == 0 {
		retries = 3
	}

	for attempt := 0; attempt <= retries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if m.checkHealthOnce(ctx, svc, timeout) {
			return nil
		}
		m.clock.Sleep(interval)
	}
	return fmt.Errorf("health check failed after %d attempts", retries+1)
}

// checkHealthOnce performs a single health check.
func (m *Monitor) checkHealthOnce(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	switch svc.Health.Type {
	case "http":
		return m.checkHTTPHealth(ctx, svc, timeout)
	case "tcp":
		return m.checkTCPHealth(svc, timeout)
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
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

// checkTCPHealth verifies service health via TCP connection.
func (m *Monitor) checkTCPHealth(svc manifest.Service, timeout time.Duration) bool {
	port, err := m.ports.Resolve(svc.ID, svc.Health.PortName)
	if err != nil {
		return false
	}
	conn, err := m.dialer.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := m.dialer.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		m.clock.Sleep(250 * time.Millisecond)
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
