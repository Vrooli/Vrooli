package bundleruntime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// waitForReadiness waits for a service to become ready based on its readiness configuration.
// Supported readiness types: health_success, port_open, log_match, dependencies_ready.
func (s *Supervisor) waitForReadiness(ctx context.Context, svc manifest.Service) error {
	timeout := time.Duration(svc.Readiness.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch svc.Readiness.Type {
	case "health_success":
		return s.pollHealth(ctx, svc)
	case "port_open":
		port, err := s.resolvePort(svc.ID, svc.Readiness.PortName)
		if err != nil {
			return err
		}
		return waitForPort(ctx, port)
	case "log_match":
		return s.waitForLogPattern(ctx, svc)
	case "dependencies_ready":
		return s.waitForDependencies(ctx, &svc)
	default:
		return fmt.Errorf("unknown readiness type %q", svc.Readiness.Type)
	}
}

// pollHealth repeatedly checks health until success or timeout.
func (s *Supervisor) pollHealth(ctx context.Context, svc manifest.Service) error {
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
		if s.checkHealthOnce(ctx, svc, timeout) {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("health check failed after %d attempts", retries+1)
}

// checkHealthOnce performs a single health check.
// Supported health types: http, tcp, command, log_match.
func (s *Supervisor) checkHealthOnce(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	switch svc.Health.Type {
	case "http":
		return s.checkHTTPHealth(ctx, svc, timeout)
	case "tcp":
		return s.checkTCPHealth(svc, timeout)
	case "command":
		return s.checkCommandHealth(ctx, svc, timeout)
	case "log_match":
		return s.logMatches(svc, svc.Health.Path)
	default:
		return false
	}
}

// checkHTTPHealth verifies service health via HTTP endpoint.
func (s *Supervisor) checkHTTPHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	port, err := s.resolvePort(svc.ID, svc.Health.PortName)
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
func (s *Supervisor) checkTCPHealth(svc manifest.Service, timeout time.Duration) bool {
	port, err := s.resolvePort(svc.ID, svc.Health.PortName)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkCommandHealth verifies service health by running a command.
func (s *Supervisor) checkCommandHealth(ctx context.Context, svc manifest.Service, timeout time.Duration) bool {
	if len(svc.Health.Command) == 0 {
		return false
	}
	cmd := exec.CommandContext(ctx, svc.Health.Command[0], svc.Health.Command[1:]...)
	if err := cmd.Start(); err != nil {
		return false
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err == nil
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return false
	}
}

// waitForLogPattern waits for a pattern to appear in service logs.
func (s *Supervisor) waitForLogPattern(ctx context.Context, svc manifest.Service) error {
	if svc.Readiness.Pattern == "" {
		return errors.New("log_match readiness missing pattern")
	}
	re, err := regexp.Compile(svc.Readiness.Pattern)
	if err != nil {
		return fmt.Errorf("invalid readiness pattern: %w", err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if s.logMatches(svc, re.String()) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// logMatches checks if the service log contains a pattern.
func (s *Supervisor) logMatches(svc manifest.Service, pattern string) bool {
	if svc.LogDir == "" {
		return false
	}
	logPath := manifest.ResolvePath(s.appData, svc.LogDir)
	data, err := os.ReadFile(logPath)
	if err != nil {
		return false
	}
	ok, _ := regexp.Match(pattern, data)
	return ok
}

// waitForPort waits for a TCP port to become available.
func waitForPort(ctx context.Context, port int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// waitForDependencies waits for all service dependencies to be ready.
func (s *Supervisor) waitForDependencies(ctx context.Context, svc *manifest.Service) error {
	if len(svc.Dependencies) == 0 {
		return nil
	}

	deadline := time.Now().Add(2 * time.Minute)
	for {
		allReady := true
		for _, dep := range svc.Dependencies {
			status, ok := s.getStatus(dep)
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
		if time.Now().After(deadline) {
			return fmt.Errorf("dependencies not ready: %v", svc.Dependencies)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}
