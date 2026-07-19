package agentactivity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
)

type stubAgentService struct {
	enabled       bool
	runStates     map[string]agentmanager.RunState
	runStateCalls int
	stopErr       error
	continueErr   error
	continueRuns  []string
	stopRuns      []string
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

func (s *stubAgentService) GetRunState(_ context.Context, runID string) (agentmanager.RunState, error) {
	s.runStateCalls++
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

func (s *stubAgentService) ApproveRun(_ context.Context, _ string, _ string, _ string) error {
	return nil
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

// stubLanePolicy implements LanePolicy for tests.
type stubLanePolicy struct {
	limits map[Lane]int
}

func (s *stubLanePolicy) LimitFor(lane Lane) int {
	if s == nil {
		return 0
	}
	return s.limits[lane]
}

func newTestServiceWithLanePolicy(t *testing.T, raw *stubAgentService, policy LanePolicy) *Service {
	t.Helper()
	return NewService(ServiceConfig{
		StorePath:    filepath.Join(t.TempDir(), "agent-activities.json"),
		AgentService: raw,
		LanePolicy:   policy,
	})
}

func TestListSnapshotDoesNotRefreshRunState(t *testing.T) {
	t.Parallel()

	agent := &stubAgentService{
		enabled: true,
		runStates: map[string]agentmanager.RunState{
			"run-1": {Status: "completed", FinishedAt: "2026-05-14T00:00:00Z"},
		},
	}
	svc := newTestService(t, agent)
	if err := svc.store.Save([]Record{{
		ActivityID: "act-1",
		OwnerType:  OwnerBacklog,
		OwnerKind:  "execute",
		OwnerName:  "slow-graph",
		Status:     StatusRunning,
		RunID:      "run-1",
	}}); err != nil {
		t.Fatalf("save activity: %v", err)
	}

	records, err := svc.ListSnapshot(context.Background(), ListFilters{ActiveOnly: true})
	if err != nil {
		t.Fatalf("ListSnapshot: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListSnapshot returned %d records, want 1", len(records))
	}
	if records[0].Status != StatusRunning {
		t.Fatalf("snapshot status = %q, want persisted running", records[0].Status)
	}
	if agent.runStateCalls != 0 {
		t.Fatalf("ListSnapshot called GetRunState %d times, want 0", agent.runStateCalls)
	}
}

func TestServiceContinueRunCreatesContinuationActivity(t *testing.T) {
	t.Parallel()

	raw := &stubAgentService{enabled: true}
	service := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType:   OwnerSession,
		OwnerName:   "sess-a",
		Purpose:     PurposeSwarmOperations,
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
	if record.ExecutionID != "" {
		t.Fatalf("session continuation must not carry an execution link, got %q", record.ExecutionID)
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
// Raw Run continuations are reserved for human-led sessions.
// ---------------------------------------------------------------------------

func TestContinueRun_RejectsInitiativeOwner(t *testing.T) {
	t.Parallel()
	raw := &stubAgentService{enabled: true}
	svc := newTestService(t, raw)

	ctx := WithSpec(context.Background(), Spec{
		OwnerType: OwnerInitiative,
		OwnerName: "init-x",
		Purpose:   PurposeFeedbackContinue,
		Metadata:  map[string]string{"round_number": "1"},
	})
	if err := svc.ContinueRun(ctx, "run-1", "more please"); err == nil {
		t.Fatal("ContinueRun accepted an initiative-owned programmatic continuation")
	}

	records, _ := svc.store.Load()
	if len(records) != 0 {
		t.Fatalf("rejected programmatic continuation created activity records: %#v", records)
	}
}

func TestSpec_RejectsUnknownOwnerType(t *testing.T) {
	t.Parallel()
	_, err := Spec{OwnerType: OwnerType("zoot"), OwnerName: "x", Purpose: PurposeFeedback}.normalized()
	if err == nil {
		t.Fatal("expected error for unknown owner type")
	}
}

func TestSpec_AcceptsRegistryAuthoredInitiativePurpose(t *testing.T) {
	t.Parallel()
	purpose := Purpose("new_mode_execute_phase")
	if _, err := (Spec{OwnerType: OwnerInitiative, OwnerName: "init-a", Purpose: purpose}).normalized(); err != nil {
		t.Fatalf("purpose %q rejected: %v", purpose, err)
	}
}

func TestSpec_RejectsUnknownPurposeForNonInitiativeOwner(t *testing.T) {
	t.Parallel()
	purpose := Purpose("new_mode_execute_phase")
	if _, err := (Spec{OwnerType: OwnerBacklog, OwnerKind: "execute", OwnerName: "task-a", Purpose: purpose}).normalized(); err == nil {
		t.Fatalf("purpose %q accepted for backlog owner", purpose)
	}
}

func TestSpec_RejectsMalformedPurpose(t *testing.T) {
	t.Parallel()
	for _, purpose := range []Purpose{"", "has-dash", "has space"} {
		if _, err := (Spec{OwnerType: OwnerInitiative, OwnerName: "init-a", Purpose: purpose}).normalized(); err == nil {
			t.Fatalf("purpose %q accepted", purpose)
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

func TestLaneActiveCounts_ReadFromService(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &stubAgentService{enabled: true})

	records := []Record{
		{ActivityID: "a", OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "x", Purpose: PurposeProcess, Status: StatusRunning, RunID: "r1"},
		{ActivityID: "b", OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "y", Purpose: PurposeWorkshop, Status: StatusRunning, RunID: "r2"},
		{ActivityID: "c", OwnerType: OwnerBacklog, OwnerKind: "idea", OwnerName: "z", Purpose: PurposeReview, Status: StatusComplete, RunID: "r3"}, // inactive
	}
	if err := svc.store.Save(records); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := svc.LaneActiveCounts()
	if err != nil {
		t.Fatalf("LaneActiveCounts: %v", err)
	}
	if got[LaneExecute] != 1 {
		t.Errorf("Execute = %d, want 1", got[LaneExecute])
	}
	if got[LaneInvestigate] != 1 {
		t.Errorf("Investigate = %d, want 1", got[LaneInvestigate])
	}
	if got[LaneReview] != 0 {
		t.Errorf("Review = %d, want 0 (record c is complete)", got[LaneReview])
	}
}
