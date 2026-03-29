package agentactivity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
)

type stubAgentService struct {
	enabled      bool
	spawnResult  agentmanager.RunResult
	spawnErr     error
	runStates    map[string]agentmanager.RunState
	stopErr      error
	continueErr  error
	continueRuns []string
	stopRuns     []string
}

func (s *stubAgentService) IsEnabled() bool {
	return s.enabled
}

func (s *stubAgentService) IsAvailable(_ context.Context) bool {
	return s.enabled
}

func (s *stubAgentService) ResolveURL(_ context.Context) (string, error) {
	if !s.enabled {
		return "", agentmanager.ErrNotAvailable
	}
	return "http://agent-manager.local", nil
}

func (s *stubAgentService) GetProfileID() string {
	return "swarm-manager"
}

func (s *stubAgentService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	if s.spawnErr != nil {
		return agentmanager.RunResult{}, s.spawnErr
	}
	return s.spawnResult, nil
}

func (s *stubAgentService) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	state, ok := s.runStates[runID]
	if !ok {
		return agentmanager.RunState{}, errors.New("run state not found")
	}
	return state, nil
}

func (s *stubAgentService) StopRun(_ context.Context, runID string) error {
	s.stopRuns = append(s.stopRuns, runID)
	return s.stopErr
}

func (s *stubAgentService) ContinueRun(_ context.Context, runID string, _ string) error {
	s.continueRuns = append(s.continueRuns, runID)
	return s.continueErr
}

func newTestService(t *testing.T, raw *stubAgentService) *Service {
	t.Helper()
	return NewService(ServiceConfig{
		StorePath:    filepath.Join(t.TempDir(), "agent-activities.json"),
		AgentService: raw,
	})
}

func TestServiceSpawnBacklogCreatesTrackedActivity(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{
		enabled: true,
		spawnResult: agentmanager.RunResult{
			TaskID: "task-1",
			RunID:  "run-1",
		},
	}
	service := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog,
		OwnerKind: "execute",
		OwnerName: "task-a",
		Purpose:   PurposeProcess,
	})

	result, err := service.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind: "execute",
		Name: "task-a",
	})
	if err != nil {
		t.Fatalf("SpawnBacklog returned error: %v", err)
	}
	if result.RunID != "run-1" || result.TaskID != "task-1" {
		t.Fatalf("unexpected run result: %+v", result)
	}

	records, err := service.store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if record.OwnerType != OwnerBacklog {
		t.Fatalf("expected backlog owner, got %q", record.OwnerType)
	}
	if record.OwnerKind != "execute" || record.OwnerName != "task-a" {
		t.Fatalf("unexpected owner: %+v", record)
	}
	if record.InteractionType != InteractionSpawn {
		t.Fatalf("expected spawn interaction, got %q", record.InteractionType)
	}
	if record.Status != StatusStarting {
		t.Fatalf("expected starting status, got %q", record.Status)
	}
	if record.RunID != "run-1" || record.TaskID != "task-1" {
		t.Fatalf("unexpected identifiers: %+v", record)
	}
	if record.RequestedBy != "swarm-manager" {
		t.Fatalf("expected default requested_by, got %q", record.RequestedBy)
	}
	if record.RequestedAt == "" || record.StartedAt == "" || record.UpdatedAt == "" {
		t.Fatalf("expected timestamps to be populated: %+v", record)
	}
}

func TestServiceContinueRunCreatesContinuationActivity(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{enabled: true}
	service := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType:   OwnerBacklog,
		OwnerKind:   "execute",
		OwnerName:   "task-a",
		ExecutionID: "exec-1",
		Purpose:     PurposeFollowUp,
		RequestedBy: "tester",
	})

	if err := service.ContinueRun(ctx, "run-continue", "please continue"); err != nil {
		t.Fatalf("ContinueRun returned error: %v", err)
	}
	if len(raw.continueRuns) != 1 || raw.continueRuns[0] != "run-continue" {
		t.Fatalf("unexpected continuation calls: %v", raw.continueRuns)
	}

	records, err := service.store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	record := records[0]
	if record.InteractionType != InteractionContinue {
		t.Fatalf("expected continue interaction, got %q", record.InteractionType)
	}
	if record.Status != StatusRunning {
		t.Fatalf("expected running status, got %q", record.Status)
	}
	if record.RunID != "run-continue" {
		t.Fatalf("expected continued run id, got %q", record.RunID)
	}
	if record.ExecutionID != "exec-1" {
		t.Fatalf("expected execution link, got %q", record.ExecutionID)
	}
	if record.RequestedBy != "tester" {
		t.Fatalf("expected requested_by to be preserved, got %q", record.RequestedBy)
	}
	if record.StartedAt == "" {
		t.Fatalf("expected started_at to be populated: %+v", record)
	}
}

func TestServiceListRefreshesActiveRunState(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{
		enabled: true,
		runStates: map[string]agentmanager.RunState{
			"run-1": {
				RunID:      "run-1",
				TaskID:     "task-1",
				Status:     "complete",
				StartedAt:  "2026-03-28T12:00:00Z",
				FinishedAt: "2026-03-28T12:05:00Z",
			},
		},
	}
	service := newTestService(t, raw)
	if err := service.store.Save([]Record{
		{
			ActivityID:      "act-1",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "execute",
			OwnerName:       "task-a",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-1",
			Status:          StatusRunning,
			RequestedAt:     "2026-03-28T11:59:00Z",
			UpdatedAt:       "2026-03-28T11:59:00Z",
		},
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	records, err := service.List(context.Background(), ListFilters{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != StatusComplete {
		t.Fatalf("expected refreshed status complete, got %q", records[0].Status)
	}
	if records[0].TaskID != "task-1" {
		t.Fatalf("expected refreshed task id, got %q", records[0].TaskID)
	}
	if records[0].FinishedAt != "2026-03-28T12:05:00Z" {
		t.Fatalf("expected finished_at to be refreshed, got %q", records[0].FinishedAt)
	}
}

func TestServiceStopRunCancelsActiveActivities(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{enabled: true}
	service := newTestService(t, raw)
	if err := service.store.Save([]Record{
		{
			ActivityID:      "act-active",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "execute",
			OwnerName:       "task-a",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-1",
			Status:          StatusRunning,
			RequestedAt:     "2026-03-28T11:59:00Z",
			UpdatedAt:       "2026-03-28T11:59:00Z",
		},
		{
			ActivityID:      "act-complete",
			OwnerType:       OwnerBacklog,
			OwnerKind:       "execute",
			OwnerName:       "task-a",
			Purpose:         PurposeProcess,
			InteractionType: InteractionSpawn,
			RunID:           "run-1",
			Status:          StatusComplete,
			RequestedAt:     "2026-03-28T11:59:00Z",
			UpdatedAt:       "2026-03-28T12:10:00Z",
			FinishedAt:      "2026-03-28T12:10:00Z",
		},
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := service.StopRun(context.Background(), "run-1"); err != nil {
		t.Fatalf("StopRun returned error: %v", err)
	}
	if len(raw.stopRuns) != 1 || raw.stopRuns[0] != "run-1" {
		t.Fatalf("unexpected stop calls: %v", raw.stopRuns)
	}

	records, err := service.store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if records[0].Status != StatusCancelled {
		t.Fatalf("expected active record to be cancelled, got %q", records[0].Status)
	}
	if records[0].FinishedAt == "" {
		t.Fatalf("expected cancelled record to get finished_at: %+v", records[0])
	}
	if records[1].Status != StatusComplete {
		t.Fatalf("expected completed record to remain unchanged, got %q", records[1].Status)
	}
}
