package orchestrator

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"test-genie/internal/orchestrator/targetruntime"
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
func (s *DefaultScenarioStarter) Start(ctx context.Context, scenarioName string) (*ScenarioStartResult, error) {
	manager := targetruntime.New(scenarioName, "")
	manager.StartTimeout = s.StartTimeout
	manager.PollInterval = s.PollInterval
	lease, err := manager.EnsureRunning(ctx, targetruntime.Needs{UI: true}, nil)
	if err != nil {
		return nil, err
	}
	port, err := portFromURL(lease.URLs.UI)
	if err != nil {
		return nil, err
	}
	return &ScenarioStartResult{Started: lease.Started, UIPort: port}, nil
}

func portFromURL(raw string) (int, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	port := parsed.Port()
	if port == "" {
		return 0, fmt.Errorf("runtime URL %q does not include a port", raw)
	}
	return strconv.Atoi(port)
}

// Stop stops a scenario using `vrooli scenario stop`.
func (s *DefaultScenarioStarter) Stop(ctx context.Context, scenarioName string) error {
	return targetruntime.New(scenarioName, "").Cleanup(ctx, targetruntime.Lease{Started: true}, nil)
}

// MockScenarioStarter is a test double for ScenarioStarter.
type MockScenarioStarter struct {
	StartFunc        func(ctx context.Context, scenarioName string) (*ScenarioStartResult, error)
	StopFunc         func(ctx context.Context, scenarioName string) error
	StartedScenarios []string
	StoppedScenarios []string
}

// Start calls the mock function or returns a default response.
func (m *MockScenarioStarter) Start(ctx context.Context, scenarioName string) (*ScenarioStartResult, error) {
	m.StartedScenarios = append(m.StartedScenarios, scenarioName)
	if m.StartFunc != nil {
		return m.StartFunc(ctx, scenarioName)
	}
	return &ScenarioStartResult{
		Started: true,
		UIPort:  8080,
	}, nil
}

// Stop calls the mock function or returns nil.
func (m *MockScenarioStarter) Stop(ctx context.Context, scenarioName string) error {
	m.StoppedScenarios = append(m.StoppedScenarios, scenarioName)
	if m.StopFunc != nil {
		return m.StopFunc(ctx, scenarioName)
	}
	return nil
}
