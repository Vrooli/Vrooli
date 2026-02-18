package mocks

import (
	"context"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"system-monitor-api/internal/agentmanager"
)

// AgentExecutor is a configurable test double for services.AgentExecutor.
type AgentExecutor struct {
	enabled          bool
	available        bool
	profileID        string
	executeResult    *agentmanager.ExecuteResult
	executeErr       error
	resolveURL       string
	resolveURLErr    error
	availableRunners []agentmanager.RunnerInfo
	activeRuns       []*domainpb.Run
}

func NewAgentExecutor() *AgentExecutor {
	return &AgentExecutor{
		enabled:       true,
		available:     true,
		executeResult: &agentmanager.ExecuteResult{Success: true, Output: "ok"},
	}
}

func (m *AgentExecutor) WithEnabled(enabled bool) *AgentExecutor {
	m.enabled = enabled
	return m
}

func (m *AgentExecutor) WithAvailable(available bool) *AgentExecutor {
	m.available = available
	return m
}

func (m *AgentExecutor) WithExecuteResult(result *agentmanager.ExecuteResult) *AgentExecutor {
	m.executeResult = result
	return m
}

func (m *AgentExecutor) WithExecuteError(err error) *AgentExecutor {
	m.executeErr = err
	return m
}

func (m *AgentExecutor) WithProfileID(profileID string) *AgentExecutor {
	m.profileID = profileID
	return m
}

func (m *AgentExecutor) WithResolveURL(url string, err error) *AgentExecutor {
	m.resolveURL = url
	m.resolveURLErr = err
	return m
}

func (m *AgentExecutor) IsEnabled() bool { return m.enabled }

func (m *AgentExecutor) IsAvailable(_ context.Context) bool { return m.available }

func (m *AgentExecutor) Initialize(_ context.Context, _ *agentmanager.ProfileConfig) error {
	return nil
}

func (m *AgentExecutor) Execute(_ context.Context, _ agentmanager.ExecuteRequest) (*agentmanager.ExecuteResult, error) {
	return m.executeResult, m.executeErr
}

func (m *AgentExecutor) GetProfile(_ context.Context) (*domainpb.AgentProfile, error) {
	return nil, nil
}

func (m *AgentExecutor) GetProfileID() string { return m.profileID }

func (m *AgentExecutor) UpdateProfile(_ context.Context, _ *agentmanager.ProfileConfig) (*domainpb.AgentProfile, error) {
	return nil, nil
}

func (m *AgentExecutor) GetAvailableRunners(_ context.Context) ([]agentmanager.RunnerInfo, error) {
	return m.availableRunners, nil
}

func (m *AgentExecutor) GetRunByTag(_ context.Context, _ string) (*domainpb.Run, error) {
	return nil, nil
}

func (m *AgentExecutor) ListActiveRuns(_ context.Context) ([]*domainpb.Run, error) {
	return m.activeRuns, nil
}

func (m *AgentExecutor) StopRun(_ context.Context, _ string) error { return nil }

func (m *AgentExecutor) ResolveURL(_ context.Context) (string, error) {
	return m.resolveURL, m.resolveURLErr
}
