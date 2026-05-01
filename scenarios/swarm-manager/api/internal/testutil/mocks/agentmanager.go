package mocks

import (
	"context"
	"sync"

	"swarm-manager/internal/agentmanager"
)

type AgentSpawner struct {
	mu       sync.Mutex
	Enabled  bool
	Result   agentmanager.RunResult
	SpawnErr error
	Requests []agentmanager.BacklogSpawnRequest
}

func NewAgentSpawner() *AgentSpawner {
	return &AgentSpawner{
		Enabled: true,
		Result:  agentmanager.RunResult{TaskID: "task-test", RunID: "run-test"},
	}
}

func (s *AgentSpawner) IsEnabled() bool {
	return s.Enabled
}

func (s *AgentSpawner) SpawnBacklog(_ context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Requests = append(s.Requests, req)
	if s.SpawnErr != nil {
		return agentmanager.RunResult{}, s.SpawnErr
	}
	return s.Result, nil
}

func (s *AgentSpawner) SpawnCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Requests)
}
