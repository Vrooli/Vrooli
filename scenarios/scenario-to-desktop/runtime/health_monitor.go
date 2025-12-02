package bundleruntime

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// HealthMonitor implements HealthChecker for service health monitoring.
type HealthMonitor struct {
	manifest      *manifest.Manifest
	portAllocator PortAllocator
	dialer        NetworkDialer
	cmdRunner     CommandRunner
	fs            FileSystem
	clock         Clock
	appData       string

	// statusGetter provides access to service status from Supervisor.
	statusGetter func(id string) (ServiceStatus, bool)
}

// HealthMonitorConfig holds the dependencies for HealthMonitor.
type HealthMonitorConfig struct {
	Manifest      *manifest.Manifest
	PortAllocator PortAllocator
	Dialer        NetworkDialer
	CmdRunner     CommandRunner
	FS            FileSystem
	Clock         Clock
	AppData       string
	StatusGetter  func(id string) (ServiceStatus, bool)
}

// NewHealthMonitor creates a new HealthMonitor with the given dependencies.
func NewHealthMonitor(cfg HealthMonitorConfig) *HealthMonitor {
	return &HealthMonitor{
		manifest:      cfg.Manifest,
		portAllocator: cfg.PortAllocator,
		dialer:        cfg.Dialer,
		cmdRunner:     cfg.CmdRunner,
		fs:            cfg.FS,
		clock:         cfg.Clock,
		appData:       cfg.AppData,
		statusGetter:  cfg.StatusGetter,
	}
}

// findService looks up a service by ID.
func (h *HealthMonitor) findService(id string) *manifest.Service {
	for i := range h.manifest.Services {
		if h.manifest.Services[i].ID == id {
			return &h.manifest.Services[i]
		}
	}
	return nil
}

// WaitForReadiness waits for a service to become ready based on its readiness configuration.
func (h *HealthMonitor) WaitForReadiness(ctx context.Context, serviceID string) error {
	svc := h.findService(serviceID)
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
		return h.pollHealth(ctx, *svc)
	case "port_open":
		port, err := h.portAllocator.Resolve(svc.ID, svc.Readiness.PortName)
		if err != nil {
			return err
		}
		return h.waitForPort(ctx, port)
	case "log_match":
		return h.waitForLogPattern(ctx, *svc)
	case "dependencies_ready":
		return h.waitForDependencies(ctx, svc)
	default:
		return fmt.Errorf("unknown readiness type %q", svc.Readiness.Type)
	}
}

// CheckOnce performs a single health check for a service.
func (h *HealthMonitor) CheckOnce(ctx context.Context, serviceID string) bool {
	svc := h.findService(serviceID)
	if svc == nil {
		return false
	}

	timeout := time.Duration(svc.Health.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	return h.checkHealthOnce(ctx, *svc, timeout)
}

// pollHealth repeatedly checks health until success or timeout.
func (h *HealthMonitor) pollHealth(ctx context.Context, svc manifest.Service) error {
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
		if h.checkHealthOnce(ctx, svc, timeout) {
			return nil
		}
		h.clock.Sleep(interval)
	}
	return fmt.Errorf("health check failed after %d attempts", retries+1)
}

// checkHealthOnce performs a single health check.
func (h *HealthMonitor) checkHealthOnce(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	switch svc.Health.Type {
	case "http":
		return h.checkHTTPHealth(ctx, svc, timeout)
	case "tcp":
		return h.checkTCPHealth(svc, timeout)
	case "command":
		return h.checkCommandHealth(ctx, svc, timeout)
	case "log_match":
		return h.logMatches(svc, svc.Health.Path)
	default:
		return false
	}
}

// checkHTTPHealth verifies service health via HTTP endpoint.
func (h *HealthMonitor) checkHTTPHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	port, err := h.portAllocator.Resolve(svc.ID, svc.Health.PortName)
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
func (h *HealthMonitor) checkTCPHealth(svc manifest.Service, timeout time.Duration) bool {
	port, err := h.portAllocator.Resolve(svc.ID, svc.Health.PortName)
	if err != nil {
		return false
	}
	conn, err := h.dialer.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkCommandHealth verifies service health by running a command.
func (h *HealthMonitor) checkCommandHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	if len(svc.Health.Command) == 0 {
		return false
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := h.cmdRunner.Run(cmdCtx, svc.Health.Command[0], svc.Health.Command[1:])
	return err == nil
}

// waitForLogPattern waits for a pattern to appear in service logs.
func (h *HealthMonitor) waitForLogPattern(ctx context.Context, svc manifest.Service) error {
	if svc.Readiness.Pattern == "" {
		return fmt.Errorf("log_match readiness missing pattern")
	}
	re, err := regexp.Compile(svc.Readiness.Pattern)
	if err != nil {
		return fmt.Errorf("invalid readiness pattern: %w", err)
	}

	ticker := h.clock.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if h.logMatches(svc, re.String()) {
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
func (h *HealthMonitor) logMatches(svc manifest.Service, pattern string) bool {
	if svc.LogDir == "" {
		return false
	}
	logPath := manifest.ResolvePath(h.appData, svc.LogDir)
	data, err := h.fs.ReadFile(logPath)
	if err != nil {
		return false
	}
	ok, _ := regexp.Match(pattern, data)
	return ok
}

// waitForPort waits for a TCP port to become available.
func (h *HealthMonitor) waitForPort(ctx context.Context, port int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := h.dialer.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		h.clock.Sleep(250 * time.Millisecond)
	}
}

// waitForDependencies waits for all service dependencies to be ready.
func (h *HealthMonitor) waitForDependencies(ctx context.Context, svc *manifest.Service) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}

	if h.statusGetter == nil {
		return fmt.Errorf("status getter not configured")
	}

	deadline := h.clock.Now().Add(2 * time.Minute)
	for {
		allReady := true
		for _, dep := range svc.Dependencies {
			status, ok := h.statusGetter(dep)
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
		if h.clock.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready: %v", svc.Dependencies)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.clock.After(250 * time.Millisecond):
		}
	}
}

// Ensure HealthMonitor implements HealthChecker.
var _ HealthChecker = (*HealthMonitor)(nil)
