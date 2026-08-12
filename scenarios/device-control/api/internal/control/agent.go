package control

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"device-control/strategy"
	"github.com/google/uuid"
)

func (s *Service) StartAgent(ctx context.Context, goal, deviceID, actor string, skillAvailable bool) (AgentRun, error) {
	if !skillAvailable {
		return AgentRun{}, fmt.Errorf("agent mode refused: prompt-manager device-control skill is unavailable")
	}
	if strings.TrimSpace(goal) == "" {
		return AgentRun{}, fmt.Errorf("agent goal is required")
	}
	run, err := s.Run(ctx, Flow{ID: uuid.NewString(), Name: "agent-observe", Steps: []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}, TimeoutMS: 5000}}}, deviceID, actor)
	if err != nil {
		return AgentRun{}, err
	}
	state := "completed"
	if run.Disposition != "passed" {
		state = "blocked"
	}
	agent := AgentRun{ID: uuid.NewString(), Goal: goal, DeviceID: deviceID, Actor: actor, State: state, Skill: "prompt-manager/device-control", Result: run, CreatedAt: time.Now().UTC()}
	s.mu.Lock()
	s.agents[agent.ID] = agent
	s.mu.Unlock()
	return agent, nil
}

func (s *Service) AbortAgent(id, reason string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	a.State = "aborted"
	a.Result.Disposition = "aborted"
	a.Result.Chapters = append(a.Result.Chapters, Chapter{ID: "abort", Title: "Agent abort", Disposition: "passed", Message: reason})
	s.agents[id] = a
	return a, nil
}

func (s *Service) PromoteAgent(id string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	if a.State != "completed" || a.Result.Disposition != "passed" {
		return AgentRun{}, fmt.Errorf("agent run %q is not eligible for promotion", id)
	}
	a.State = "promoted"
	s.agents[id] = a
	return a, nil
}

func (s *Service) ListAgents() []AgentRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AgentRun, 0, len(s.agents))
	for _, a := range s.agents {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
