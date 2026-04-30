package agentactivity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
)

type stubAgentService struct {
	enabled              bool
	spawnResult          agentmanager.RunResult
	spawnErr             error
	initiativeSpawnReq   agentmanager.InitiativeSpawnRequest
	initiativeSpawnCalls int
	initiativeSpawnRes   agentmanager.RunResult
	initiativeSpawnErr   error
	runStates            map[string]agentmanager.RunState
	stopErr              error
	continueErr          error
	continueRuns         []string
	stopRuns             []string
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

func (s *stubAgentService) SpawnInitiative(_ context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	s.initiativeSpawnCalls++
	s.initiativeSpawnReq = req
	if s.initiativeSpawnErr != nil {
		return agentmanager.RunResult{}, s.initiativeSpawnErr
	}
	return s.initiativeSpawnRes, nil
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

// ---------------------------------------------------------------------------
// Per-backlog-item guard tests
// ---------------------------------------------------------------------------

func TestSpawnBacklog_RejectsWhenItemAlreadyActive(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:     true,
		spawnResult: agentmanager.RunResult{TaskID: "t1", RunID: "r1"},
	}
	svc := newTestService(t, raw)

	// First spawn succeeds.
	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeWorkshop,
	})
	if _, err := svc.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	}); err != nil {
		t.Fatalf("first spawn should succeed: %v", err)
	}

	// Second spawn for the same item should fail.
	ctx2 := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeFinalize,
	})
	_, err := svc.SpawnBacklog(ctx2, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	})
	if !errors.Is(err, ErrBacklogItemBusy) {
		t.Fatalf("expected ErrBacklogItemBusy, got %v", err)
	}
}

func TestSpawnBacklog_AllowsWhenDifferentItem(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:     true,
		spawnResult: agentmanager.RunResult{TaskID: "t1", RunID: "r1"},
	}
	svc := newTestService(t, raw)

	// Spawn for item-a.
	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeWorkshop,
	})
	if _, err := svc.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	}); err != nil {
		t.Fatalf("spawn item-a should succeed: %v", err)
	}

	// Spawn for item-b should succeed (different item).
	ctx2 := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-b",
		Purpose: PurposeWorkshop,
	})
	if _, err := svc.SpawnBacklog(ctx2, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-b",
	}); err != nil {
		t.Fatalf("spawn item-b should succeed: %v", err)
	}
}

func TestSpawnBacklog_AllowsAfterPriorCompletes(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:     true,
		spawnResult: agentmanager.RunResult{TaskID: "t1", RunID: "r1"},
		runStates: map[string]agentmanager.RunState{
			"r1": {Status: "complete", TaskID: "t1"},
		},
	}
	svc := newTestService(t, raw)

	// First spawn.
	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeWorkshop,
	})
	if _, err := svc.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	}); err != nil {
		t.Fatalf("first spawn should succeed: %v", err)
	}

	// Second spawn — the refresh should detect that r1 completed,
	// clearing the way for a new spawn.
	raw.spawnResult = agentmanager.RunResult{TaskID: "t2", RunID: "r2"}
	ctx2 := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeFinalize,
	})
	if _, err := svc.SpawnBacklog(ctx2, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	}); err != nil {
		t.Fatalf("second spawn should succeed after prior completed: %v", err)
	}
}

func TestSpawnBacklog_AllowsStalePending(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:     true,
		spawnResult: agentmanager.RunResult{TaskID: "t1", RunID: "r1"},
	}
	svc := newTestService(t, raw)

	// Pre-seed a stale pending record (no RunID, old timestamp).
	staleRecord := Record{
		ActivityID:      "stale-1",
		OwnerType:       OwnerBacklog,
		OwnerKind:       "idea",
		OwnerName:       "item-a",
		Purpose:         PurposeWorkshop,
		InteractionType: InteractionSpawn,
		Status:          StatusPending,
		RequestedAt:     "2020-01-01T00:00:00Z", // well past the 5min TTL
		UpdatedAt:       "2020-01-01T00:00:00Z",
	}
	if err := svc.store.Save([]Record{staleRecord}); err != nil {
		t.Fatal(err)
	}

	// Spawn should succeed — stale pending records are auto-failed.
	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "item-a",
		Purpose: PurposeWorkshop,
	})
	if _, err := svc.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind: "idea", Name: "item-a",
	}); err != nil {
		t.Fatalf("spawn should succeed with stale pending record: %v", err)
	}

	// The stale record should now be failed.
	records, _ := svc.store.Load()
	for _, rec := range records {
		if rec.ActivityID == "stale-1" && rec.Status != StatusFailed {
			t.Errorf("expected stale record to be auto-failed, got %q", rec.Status)
		}
	}
}

func TestSpawnBacklog_SkipsGuardForNonBacklog(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:     true,
		spawnResult: agentmanager.RunResult{TaskID: "t1", RunID: "r1"},
	}
	svc := newTestService(t, raw)

	// Pre-seed an active capture record — should not block a new capture spawn
	// because the per-item guard only applies to OwnerBacklog.
	activeRecord := Record{
		ActivityID:      "capture-1",
		OwnerType:       OwnerCapture,
		OwnerKind:       "",
		OwnerName:       "some-capture",
		Purpose:         PurposeClassify,
		InteractionType: InteractionSpawn,
		Status:          StatusRunning,
		RunID:           "r0",
		RequestedAt:     nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	}
	if err := svc.store.Save([]Record{activeRecord}); err != nil {
		t.Fatal(err)
	}

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerCapture, OwnerName: "some-capture",
		Purpose: PurposeClassify,
	})
	if _, err := svc.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Name: "some-capture",
	}); err != nil {
		t.Fatalf("capture spawn should not be blocked by per-item guard: %v", err)
	}
}

func TestHasActiveAgent_ReturnsTrueWhenActive(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{enabled: true}
	svc := newTestService(t, raw)

	activeRecord := Record{
		ActivityID:      "active-1",
		OwnerType:       OwnerBacklog,
		OwnerKind:       "idea",
		OwnerName:       "item-a",
		Purpose:         PurposeWorkshop,
		InteractionType: InteractionSpawn,
		Status:          StatusRunning,
		RunID:           "r1",
		RequestedAt:     nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	}
	if err := svc.store.Save([]Record{activeRecord}); err != nil {
		t.Fatal(err)
	}

	// Mock the run state to stay running.
	raw.runStates = map[string]agentmanager.RunState{
		"r1": {Status: "running", TaskID: "t1"},
	}

	if !svc.HasActiveAgent(context.Background(), "idea", "item-a") {
		t.Error("expected HasActiveAgent to return true for active item")
	}
}

// ---------------------------------------------------------------------------
// Initiative-owned spawn flow
// ---------------------------------------------------------------------------

func TestSpawnInitiative_TracksRecordWithInitiativeOwner(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:            true,
		initiativeSpawnRes: agentmanager.RunResult{TaskID: "init-task-1", RunID: "init-run-1"},
	}
	svc := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerInitiative,
		OwnerName: "command-center-foundation",
		Purpose:   PurposeFeedback,
		Metadata: map[string]string{
			"round_number": "3",
			"round_slug":   "ui-rewrite",
			"entrypoint":   "initiative.feedback",
		},
	})
	res, err := svc.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{
		Name:        "command-center-foundation",
		Purpose:     "feedback",
		RoundNumber: 3,
		RoundSlug:   "ui-rewrite",
	})
	if err != nil {
		t.Fatalf("SpawnInitiative: %v", err)
	}
	if res.RunID != "init-run-1" {
		t.Fatalf("unexpected run id: %q", res.RunID)
	}
	if raw.initiativeSpawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", raw.initiativeSpawnCalls)
	}

	records, _ := svc.store.Load()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec.OwnerType != OwnerInitiative {
		t.Fatalf("expected OwnerInitiative, got %q", rec.OwnerType)
	}
	if rec.OwnerName != "command-center-foundation" {
		t.Fatalf("unexpected owner name: %q", rec.OwnerName)
	}
	if rec.Purpose != PurposeFeedback {
		t.Fatalf("expected PurposeFeedback, got %q", rec.Purpose)
	}
	if rec.Status != StatusStarting {
		t.Fatalf("expected starting status, got %q", rec.Status)
	}
	if rec.RunID != "init-run-1" {
		t.Fatalf("unexpected run id on record: %q", rec.RunID)
	}
	if rec.Metadata["round_number"] != "3" || rec.Metadata["round_slug"] != "ui-rewrite" || rec.Metadata["entrypoint"] != "initiative.feedback" {
		t.Fatalf("metadata not preserved: %+v", rec.Metadata)
	}
}

func TestSpawnInitiative_RejectsBacklogOwner(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{enabled: true}
	svc := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerBacklog,
		OwnerKind: "execute",
		OwnerName: "x",
		Purpose:   PurposeFeedback,
	})
	if _, err := svc.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{Name: "x"}); err == nil {
		t.Fatal("expected error when context spec is OwnerBacklog")
	}
	if raw.initiativeSpawnCalls != 0 {
		t.Fatalf("expected no spawn call, got %d", raw.initiativeSpawnCalls)
	}
}

func TestSpawnInitiative_FailureMarksRecordFailed(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{
		enabled:            true,
		initiativeSpawnErr: errors.New("agent-manager exploded"),
	}
	svc := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerInitiative,
		OwnerName: "init-x",
		Purpose:   PurposeFeedback,
	})
	if _, err := svc.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{Name: "init-x"}); err == nil {
		t.Fatal("expected spawn error")
	}

	records, _ := svc.store.Load()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec.Status != StatusFailed {
		t.Fatalf("expected failed status, got %q", rec.Status)
	}
	if rec.FailureReason == "" {
		t.Fatal("expected failure reason to be populated")
	}
}

func TestContinueRun_AcceptsInitiativeOwner(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{enabled: true}
	svc := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerInitiative,
		OwnerName: "init-x",
		Purpose:   PurposeFeedbackContinue,
		Metadata:  map[string]string{"round_number": "1"},
	})
	if err := svc.ContinueRun(ctx, "run-1", "more please"); err != nil {
		t.Fatalf("ContinueRun: %v", err)
	}

	records, _ := svc.store.Load()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].OwnerType != OwnerInitiative {
		t.Fatalf("expected OwnerInitiative, got %q", records[0].OwnerType)
	}
	if records[0].Purpose != PurposeFeedbackContinue {
		t.Fatalf("expected PurposeFeedbackContinue, got %q", records[0].Purpose)
	}
}

func TestSpec_RejectsUnknownOwnerType(t *testing.T) {
	t.Parallel()
	_, err := Spec{OwnerType: OwnerType("zoot"), OwnerName: "x", Purpose: PurposeFeedback}.normalized()
	if err == nil {
		t.Fatal("expected error for unknown owner type")
	}
}

func TestSpec_AcceptsOperatingModePurposes(t *testing.T) {
	t.Parallel()
	purposes := []Purpose{
		PurposeHolisticLoopInvestigate,
		PurposeHolisticLoopPlan,
		PurposeHolisticLoopExecute,
		PurposeHolisticLoopReview,
		PurposePhasedPlanPrepare,
		PurposePhasedPlanExecuteNext,
		PurposePhasedPlanClassifyProgress,
		PurposePhasedPlanReview,
	}
	for _, purpose := range purposes {
		if _, err := (Spec{OwnerType: OwnerInitiative, OwnerName: "init-a", Purpose: purpose}).normalized(); err != nil {
			t.Fatalf("purpose %q rejected: %v", purpose, err)
		}
	}
}

func TestHasActiveAgent_ReturnsFalseAfterComplete(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{enabled: true}
	svc := newTestService(t, raw)

	record := Record{
		ActivityID:      "done-1",
		OwnerType:       OwnerBacklog,
		OwnerKind:       "idea",
		OwnerName:       "item-a",
		Purpose:         PurposeWorkshop,
		InteractionType: InteractionSpawn,
		Status:          StatusRunning, // stored as running but agent-manager says complete
		RunID:           "r1",
		RequestedAt:     nowRFC3339(),
		UpdatedAt:       nowRFC3339(),
	}
	if err := svc.store.Save([]Record{record}); err != nil {
		t.Fatal(err)
	}

	// Agent-manager reports this run as complete.
	raw.runStates = map[string]agentmanager.RunState{
		"r1": {Status: "complete", TaskID: "t1"},
	}

	if svc.HasActiveAgent(context.Background(), "idea", "item-a") {
		t.Error("expected HasActiveAgent to return false after refresh shows complete")
	}
}
