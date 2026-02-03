package heartbeat

import (
	"context"
	"testing"

	"prompt-manager/store"
)

type stubConfigStore struct {
	config *store.HeartbeatConfig
	err    error
}

func (s *stubConfigStore) GetHeartbeatConfig(ctx context.Context, teamID, agentID string) (*store.HeartbeatConfig, error) {
	return s.config, s.err
}

type captureExecutor struct {
	calls []executionCall
}

type executionCall struct {
	teamID     string
	agentID    string
	profileKey string
}

func (c *captureExecutor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	c.calls = append(c.calls, executionCall{
		teamID:     teamID,
		agentID:    agentID,
		profileKey: profileKey,
	})
	return &ExecutionResult{
		TeamID:  teamID,
		AgentID: agentID,
		Status:  store.HeartbeatStatusRunning,
	}, nil
}

func TestSchedulerUsesConfigProfileKey(t *testing.T) {
	exec := &captureExecutor{}
	configStore := &stubConfigStore{
		config: &store.HeartbeatConfig{
			TeamID:     "team-1",
			AgentID:    "agent-1",
			Enabled:    true,
			Schedule:   "0 * * * *",
			ProfileKey: "custom-profile",
		},
	}
	scheduler := NewScheduler(exec, nil, configStore)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if len(exec.calls) != 1 {
		t.Fatalf("expected executor to be called once, got %d", len(exec.calls))
	}
	if exec.calls[0].profileKey != "custom-profile" {
		t.Fatalf("expected profileKey to be custom-profile, got %s", exec.calls[0].profileKey)
	}
}

func TestSchedulerUsesDefaultProfileWhenEmpty(t *testing.T) {
	exec := &captureExecutor{}
	configStore := &stubConfigStore{
		config: &store.HeartbeatConfig{
			TeamID:   "team-1",
			AgentID:  "agent-1",
			Enabled:  true,
			Schedule: "0 * * * *",
		},
	}
	scheduler := NewScheduler(exec, nil, configStore)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if len(exec.calls) != 1 {
		t.Fatalf("expected executor to be called once, got %d", len(exec.calls))
	}
	if exec.calls[0].profileKey != "prompt-manager-heartbeat" {
		t.Fatalf("expected default profileKey, got %s", exec.calls[0].profileKey)
	}
}

func TestSchedulerSkipsWhenConfigMissing(t *testing.T) {
	exec := &captureExecutor{}
	configStore := &stubConfigStore{config: nil}
	scheduler := NewScheduler(exec, nil, configStore)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if len(exec.calls) != 0 {
		t.Fatalf("expected executor not to be called, got %d calls", len(exec.calls))
	}
}

func TestSchedulerSkipsWhenConfigDisabled(t *testing.T) {
	exec := &captureExecutor{}
	configStore := &stubConfigStore{
		config: &store.HeartbeatConfig{
			TeamID:   "team-1",
			AgentID:  "agent-1",
			Enabled:  false,
			Schedule: "0 * * * *",
		},
	}
	scheduler := NewScheduler(exec, nil, configStore)

	scheduler.executeHeartbeat(context.Background(), "team-1", "agent-1")

	if len(exec.calls) != 0 {
		t.Fatalf("expected executor not to be called, got %d calls", len(exec.calls))
	}
}
