package orchestrator

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// DefaultScenarioStarter implements ScenarioStarter using the vrooli CLI.
type DefaultScenarioStarter struct {
	// StartTimeout is the maximum time to wait for scenario startup.
	StartTimeout time.Duration
	// PollInterval is how often to check if the UI port is ready.
	PollInterval time.Duration
}

// NewDefaultScenarioStarter creates a new DefaultScenarioStarter with sensible defaults.
func NewDefaultScenarioStarter() *DefaultScenarioStarter {
	return &DefaultScenarioStarter{
		StartTimeout: 120 * time.Second, // 2 minutes to start
		PollInterval: 2 * time.Second,
	}
}

// Start starts the scenario using `vrooli scenario start` and waits for the UI port.
func (s *DefaultScenarioStarter) Start(ctx context.Context, scenarioName string) (int, error) {
	// Create a timeout context for the entire start operation
	startCtx, cancel := context.WithTimeout(ctx, s.StartTimeout)
	defer cancel()

	// First, check if scenario is already running
	port, err := s.getUIPort(startCtx, scenarioName)
	if err == nil && port > 0 {
		// Already running, return the port
		return port, nil
	}

	// Start the scenario
	cmd := exec.CommandContext(startCtx, "vrooli", "scenario", "start", scenarioName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to start scenario: %w\nOutput: %s", err, string(output))
	}

	// Wait for the UI port to become available
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-startCtx.Done():
			return 0, fmt.Errorf("timeout waiting for scenario UI to start: %w", startCtx.Err())
		case <-ticker.C:
			port, err := s.getUIPort(startCtx, scenarioName)
			if err == nil && port > 0 {
				// Verify the port is actually listening
				if s.isPortListening(startCtx, port) {
					return port, nil
				}
			}
		}
	}
}

// Stop stops a scenario using `vrooli scenario stop`.
func (s *DefaultScenarioStarter) Stop(ctx context.Context, scenarioName string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "stop", scenarioName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop scenario: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// getUIPort retrieves the UI port for a scenario using `vrooli scenario port`.
func (s *DefaultScenarioStarter) getUIPort(ctx context.Context, scenarioName string) (int, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, "UI_PORT")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	portStr := strings.TrimSpace(string(output))
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", portStr)
	}
	return port, nil
}

// isPortListening checks if a port is accepting connections using native Go TCP dial.
func (s *DefaultScenarioStarter) isPortListening(ctx context.Context, port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// MockScenarioStarter is a test double for ScenarioStarter.
type MockScenarioStarter struct {
	StartFunc func(ctx context.Context, scenarioName string) (int, error)
	StopFunc  func(ctx context.Context, scenarioName string) error
	StartedScenarios []string
	StoppedScenarios []string
}

// Start calls the mock function or returns a default response.
func (m *MockScenarioStarter) Start(ctx context.Context, scenarioName string) (int, error) {
	m.StartedScenarios = append(m.StartedScenarios, scenarioName)
	if m.StartFunc != nil {
		return m.StartFunc(ctx, scenarioName)
	}
	return 8080, nil
}

// Stop calls the mock function or returns nil.
func (m *MockScenarioStarter) Stop(ctx context.Context, scenarioName string) error {
	m.StoppedScenarios = append(m.StoppedScenarios, scenarioName)
	if m.StopFunc != nil {
		return m.StopFunc(ctx, scenarioName)
	}
	return nil
}
