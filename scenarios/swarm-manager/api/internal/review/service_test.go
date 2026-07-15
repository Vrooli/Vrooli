package review

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
)

// capturingSpawner records the BacklogSpawnRequest for inspection.
type capturingSpawner struct {
	enabled  bool
	captured *agentmanager.BacklogSpawnRequest
	runState agentmanager.RunState
	stateErr error
}

func (s *capturingSpawner) IsEnabled() bool { return s.enabled }

func (s *capturingSpawner) SpawnBacklog(_ context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	s.captured = &req
	return agentmanager.RunResult{RunID: "test-run-id"}, nil
}

func (s *capturingSpawner) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return s.runState, s.stateErr
}

// fakeReviewOperationStarter records the last OperationStartRequest and returns a
// configurable OperationStartResult. It stands in for the operation runner adapter
// so review tests can assert the review flow starts the right operation without
// wiring the real declarative runner.
type fakeReviewOperationStarter struct {
	calls   int
	lastReq OperationStartRequest
	result  OperationStartResult
	err     error
}

func newFakeReviewOperationStarter() *fakeReviewOperationStarter {
	return &fakeReviewOperationStarter{
		result: OperationStartResult{RunID: "test-run-id", WorkflowID: "wf-test", ExecutionID: "exec-test"},
	}
}

func (f *fakeReviewOperationStarter) StartReviewOperation(_ context.Context, req OperationStartRequest) (OperationStartResult, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return OperationStartResult{}, f.err
	}
	return f.result, nil
}

func newTestService(spawner *capturingSpawner, promptResult string) *Service {
	svc := &Service{
		dataRoot:      "/tmp/test-backlog",
		agentService:  spawner,
		inspector:     spawner,
		promptClient:  &promptmanager.MockClient{Result: promptResult},
		itemDirFn:     func(_, _ string) string { return "/tmp/test-backlog/tasks/test-item" },
		loadItemTitle: func(_, _ string) (string, error) { return "Loaded Title", nil },
		planContentResolver: func(_ context.Context, kind, _, itemDir string) (string, error) {
			if kind == "research" {
				return "", nil
			}
			return "# Test Plan\nDo the thing from plan-manager.", nil
		},
		// Disable the abandoned-round age backstop by default so tests that use
		// fixed placeholder timestamps aren't force-failed by it. Tests that
		// exercise the backstop set roundMaxAge + clock explicitly.
		roundMaxAge:  100 * 365 * 24 * time.Hour,
		activeRounds: make(map[string]activeRound),
	}
	return svc
}

// TestRefreshGatheringRounds_InvokesOnRoundTerminal verifies that when a
// gathering round transitions to a terminal status, the OnRoundTerminal
// callback is fired with the item's kind, name, and final round state.
// This is the hook the backlog layer uses to flip items from in_review to
// review_pending.
func TestRefreshGatheringRounds_InvokesOnRoundTerminal(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		ExecutionID: "exec-callback",
		Status:      RoundStatusGathering,
		RunID:       "run-callback",
		Evidence: []EvidenceItem{
			{ID: "e1", Type: EvidenceTypeScreenshot, Title: "t", Description: "d", Verified: true},
		},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-callback",
			Status: "complete",
		},
	}
	svc := newTestService(spawner, "")

	var gotKind, gotName string
	var gotStatus RoundStatus
	svc.onRoundTerminal = func(_ context.Context, kind, name string, r Round) {
		gotKind = kind
		gotName = name
		gotStatus = r.Status
	}

	svc.trackActiveRound("run-callback", "execute", "sample-item", itemDir, 1)
	svc.RefreshGatheringRounds(context.Background())

	if gotKind != "execute" {
		t.Errorf("callback kind = %q, want %q", gotKind, "execute")
	}
	if gotName != "sample-item" {
		t.Errorf("callback name = %q, want %q", gotName, "sample-item")
	}
	if gotStatus != RoundStatusComplete && gotStatus != RoundStatusFailed {
		t.Errorf("callback round status = %q, want complete or failed", gotStatus)
	}
}

// setupItemDir creates a temporary backlog item directory. Research items keep
// conclusion.md as a local deliverable; non-research plan content is resolved
// through planContentResolver.
func setupItemDir(t *testing.T, kind string) string {
	t.Helper()
	dir := t.TempDir()
	if kind == "research" {
		if err := os.WriteFile(filepath.Join(dir, "conclusion.md"), []byte("# Test Conclusion\nResearch findings."), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestStartReview_StartsOperationAndWritesRound verifies the rerouted startReview
// starts the review-round operation through the injected OperationStarter and
// writes a gathering round carrying the run association (RunID + OpExecutionID)
// the completion handler correlates back to. It no longer builds prompts/context
// attachments or spawns an agent directly — that is the operation runner's job now.
func TestStartReview_StartsOperationAndWritesRound(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	itemDir := setupItemDir(t, "task")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	starter := newFakeReviewOperationStarter()
	svc.SetOperationStarter(starter)

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID: "exec-123",
		BacklogKind: "task",
		BacklogName: "test-item",
		ItemTitle:   "Test Item",
		ItemDir:     itemDir,
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	if starter.calls != 1 {
		t.Fatalf("expected exactly 1 operation start, got %d", starter.calls)
	}
	if starter.lastReq.Operation != "review-round" {
		t.Errorf("Operation = %q, want review-round", starter.lastReq.Operation)
	}
	if starter.lastReq.TargetKind != "backlog-item" {
		t.Errorf("TargetKind = %q, want backlog-item", starter.lastReq.TargetKind)
	}
	if starter.lastReq.TargetID != "task/test-item" {
		t.Errorf("TargetID = %q, want task/test-item", starter.lastReq.TargetID)
	}

	// A gathering round must be written carrying the run association so the
	// commit-review-round handler can correlate the operation result back to it.
	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("expected round 1 on disk: err=%v round=%v", err, round)
	}
	if round.Status != RoundStatusGathering {
		t.Errorf("round status = %q, want gathering (runner finalizes on completion)", round.Status)
	}
	if round.RunID != "test-run-id" {
		t.Errorf("round RunID = %q, want test-run-id", round.RunID)
	}
	if round.OpWorkflowID != "wf-test" {
		t.Errorf("round OpWorkflowID = %q, want wf-test", round.OpWorkflowID)
	}
	if round.OpExecutionID != "exec-test" {
		t.Errorf("round OpExecutionID = %q, want exec-test", round.OpExecutionID)
	}
	if !round.RunnerOwned() {
		t.Error("round should be runner-owned (OpExecutionID set)")
	}
}

// TestStartReview_RequiresOperationStarter pins the degraded-mode guard: without
// an injected operation runner, startReview refuses rather than silently spawning.
func TestStartReview_RequiresOperationStarter(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "instructions")
	itemDir := setupItemDir(t, "task")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID: "exec-x",
		BacklogKind: "task",
		BacklogName: "test-item",
		ItemDir:     itemDir,
	})
	if err == nil || !strings.Contains(err.Error(), "review operation runner not available") {
		t.Fatalf("expected operation-runner-unavailable error, got %v", err)
	}
}

func TestBuildReviewAttachments_UserRequest(t *testing.T) {
	atts := buildReviewAttachments("plan text", []string{"a.go"}, []string{"scen"}, "", "", "show me the logs")

	var found bool
	for _, att := range atts {
		if att.Key == "user-request" {
			found = true
			if att.Content != "show me the logs" {
				t.Errorf("user-request content = %q, want %q", att.Content, "show me the logs")
			}
			if att.Priority != "high" {
				t.Errorf("user-request priority = %q, want %q", att.Priority, "high")
			}
		}
	}
	if !found {
		t.Error("expected user-request attachment")
	}
}

func TestBuildReviewAttachments_NoUserRequest(t *testing.T) {
	atts := buildReviewAttachments("plan", nil, nil, "", "", "")
	for _, att := range atts {
		if att.Key == "user-request" {
			t.Error("user-request should be absent when empty")
		}
	}
}

func TestBuildReviewAttachments_NonSandboxDiffSummary(t *testing.T) {
	// When there are 0 changed paths (non-sandbox execution), the diff-summary
	// should explain that changes may exist but weren't tracked.
	atts := buildReviewAttachments("plan", nil, nil, "", "", "")

	for _, att := range atts {
		if att.Key == "diff-summary" {
			if att.Content == "Changed 0 files across 0 scenarios" {
				t.Error("diff-summary should include non-sandbox explanation when 0 changed paths")
			}
			if !strings.Contains(att.Content, "without sandbox mode") {
				t.Error("diff-summary should mention non-sandbox mode")
			}
			if !strings.Contains(att.Content, "examining the codebase directly") {
				t.Error("diff-summary should instruct agent to examine codebase directly")
			}
			return
		}
	}
	t.Error("expected diff-summary attachment")
}

func TestBuildReviewAttachments_SandboxDiffSummary(t *testing.T) {
	// When there ARE changed paths, the diff-summary should NOT include the non-sandbox note.
	atts := buildReviewAttachments("plan", []string{"a.go", "b.go"}, []string{"scen"}, "", "", "")

	for _, att := range atts {
		if att.Key == "diff-summary" {
			if strings.Contains(att.Content, "without sandbox mode") {
				t.Error("diff-summary should NOT mention non-sandbox mode when changes are present")
			}
			if !strings.Contains(att.Content, "Changed 2 files across 1 scenarios") {
				t.Errorf("unexpected diff-summary content: %q", att.Content)
			}
			return
		}
	}
	t.Error("expected diff-summary attachment")
}

func TestBuildReviewAttachments_Priorities(t *testing.T) {
	atts := buildReviewAttachments("plan", []string{"a.go"}, []string{"scen"}, `{"s":"ready"}`, `{"scen":{"verdict":"regression"}}`, "")

	priorities := make(map[string]string)
	for _, att := range atts {
		priorities[att.Key] = att.Priority
	}

	highKeys := []string{"plan-content", "diff-summary", "changed-paths", "gct-review-results", "baseline-diff-results"}
	for _, key := range highKeys {
		if priorities[key] != "high" {
			t.Errorf("attachment %q priority = %q, want %q", key, priorities[key], "high")
		}
	}
	if priorities["affected-scenarios"] != "medium" {
		t.Errorf("affected-scenarios priority = %q, want %q", priorities["affected-scenarios"], "medium")
	}
}

func TestBuildReviewAttachments_BaselineDiffPresentIffData(t *testing.T) {
	// Present when baseline diff JSON is supplied.
	withDiff := buildReviewAttachments("plan", []string{"a.go"}, []string{"scen"}, "", `{"scen":{"verdict":"regression"}}`, "")
	var found bool
	for _, att := range withDiff {
		if att.Key == "baseline-diff-results" {
			found = true
			if att.Format != "json" {
				t.Errorf("baseline-diff-results format = %q, want json", att.Format)
			}
		}
	}
	if !found {
		t.Error("expected baseline-diff-results attachment when diff data is present")
	}

	// Absent when baseline diff JSON is empty.
	withoutDiff := buildReviewAttachments("plan", []string{"a.go"}, []string{"scen"}, "", "", "")
	for _, att := range withoutDiff {
		if att.Key == "baseline-diff-results" {
			t.Error("baseline-diff-results should be absent when no diff data")
		}
	}
}

// --- Polling / RefreshGatheringRounds tests ---

func writeRound(t *testing.T, itemDir string, round Round) {
	t.Helper()
	reviewDir := filepath.Join(itemDir, "review")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(round, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewDir, RoundFilename(round.RoundNum)), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshGatheringRounds_CompletesRound(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		ExecutionID: "exec-1",
		Status:      RoundStatusGathering,
		RunID:       "run-abc",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-abc",
			Status: "complete",
		},
	}
	svc := newTestService(spawner, "")
	svc.trackActiveRound("run-abc", "execute", "sample-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	// Verify round status was updated on disk.
	round, err := LoadRound(itemDir, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round.Status != RoundStatusFailed {
		t.Errorf("round status = %q, want %q", round.Status, RoundStatusFailed)
	}
	if !strings.Contains(round.FailureReason, "agent_assessment") {
		t.Fatalf("expected missing assessment failure reason, got %q", round.FailureReason)
	}

	// Verify it was removed from active tracking.
	svc.mu.Lock()
	remaining := len(svc.activeRounds)
	svc.mu.Unlock()
	if remaining != 0 {
		t.Errorf("expected 0 active rounds after completion, got %d", remaining)
	}
}

func TestRefreshGatheringRounds_FailedRound(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		ExecutionID: "exec-2",
		Status:      RoundStatusGathering,
		RunID:       "run-fail",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-fail",
			Status: "failed",
		},
	}
	svc := newTestService(spawner, "")
	svc.trackActiveRound("run-fail", "execute", "sample-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	round, err := LoadRound(itemDir, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round.Status != RoundStatusFailed {
		t.Errorf("round status = %q, want %q", round.Status, RoundStatusFailed)
	}
}

func TestRefreshGatheringRounds_StillRunning(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		ExecutionID: "exec-3",
		Status:      RoundStatusGathering,
		RunID:       "run-running",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-running",
			Status: "running",
		},
	}
	svc := newTestService(spawner, "")
	svc.trackActiveRound("run-running", "execute", "sample-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	// Round should still be gathering.
	round, err := LoadRound(itemDir, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if round.Status != RoundStatusGathering {
		t.Errorf("round status = %q, want %q (should remain gathering)", round.Status, RoundStatusGathering)
	}

	// Should still be tracked.
	svc.mu.Lock()
	remaining := len(svc.activeRounds)
	svc.mu.Unlock()
	if remaining != 1 {
		t.Errorf("expected 1 active round while still running, got %d", remaining)
	}
}

func TestRefreshGatheringRounds_AlreadyComplete(t *testing.T) {
	itemDir := t.TempDir()
	// Round was already marked complete (e.g. agent wrote the file itself).
	writeRound(t, itemDir, Round{
		RoundNum:        1,
		GeneratedAt:     "2026-04-02T00:00:00Z",
		ExecutionID:     "exec-4",
		Status:          RoundStatusComplete,
		RunID:           "run-done",
		AgentAssessment: "Looks good.",
		Classification:  "ready",
		Evidence:        []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-done",
			Status: "complete",
		},
	}
	svc := newTestService(spawner, "")
	svc.trackActiveRound("run-done", "execute", "sample-item", itemDir, 1)

	svc.RefreshGatheringRounds(context.Background())

	// Should be removed from tracking without re-saving.
	svc.mu.Lock()
	remaining := len(svc.activeRounds)
	svc.mu.Unlock()
	if remaining != 0 {
		t.Errorf("expected 0 active rounds, got %d", remaining)
	}
}

func TestMapRunStatusToRoundStatus(t *testing.T) {
	tests := []struct {
		input string
		want  RoundStatus
	}{
		{"complete", RoundStatusComplete},
		{"Complete", RoundStatusComplete},
		{"failed", RoundStatusFailed},
		{"cancelled", RoundStatusFailed},
		{"running", ""},
		{"pending", ""},
		{"starting", ""},
	}
	for _, tt := range tests {
		got := mapRunStatusToRoundStatus(tt.input)
		if got != tt.want {
			t.Errorf("mapRunStatusToRoundStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestStartReview_DoesNotTrackRunnerOwnedRound verifies the rerouted startReview
// does NOT enter the runner-owned round into the legacy polling map. Runner-owned
// rounds are finalized by the operation runner's completion bridge, so tracking
// them would race the poller against the commit handler.
func TestStartReview_DoesNotTrackRunnerOwnedRound(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "instructions")
	itemDir := setupItemDir(t, "task")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	svc.SetOperationStarter(newFakeReviewOperationStarter())

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID: "exec-track",
		BacklogKind: "task",
		BacklogName: "test-item",
		ItemTitle:   "Test",
		ItemDir:     itemDir,
	})
	if err != nil {
		t.Fatalf("startReview: %v", err)
	}

	svc.mu.Lock()
	count := len(svc.activeRounds)
	svc.mu.Unlock()
	if count != 0 {
		t.Errorf("runner-owned rounds must not be tracked for legacy polling, got %d tracked", count)
	}
}

func TestListRounds_InlineRefresh(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:        1,
		GeneratedAt:     "2026-04-02T00:00:00Z",
		ExecutionID:     "exec-list",
		Status:          RoundStatusGathering,
		RunID:           "run-inline",
		AgentAssessment: "Looks good.",
		Classification:  "ready",
		Evidence:        []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-inline",
			Status: "complete",
		},
	}
	svc := newTestService(spawner, "")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	rounds, err := svc.ListRounds("task", "test")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != RoundStatusComplete {
		t.Errorf("round status = %q, want %q (inline refresh should update)", rounds[0].Status, RoundStatusComplete)
	}

	// Verify persisted to disk too.
	diskRound, _ := LoadRound(itemDir, 1)
	if diskRound.Status != RoundStatusComplete {
		t.Errorf("disk round status = %q, want %q", diskRound.Status, RoundStatusComplete)
	}
}

func TestListRounds_ExposesCurrentRunStatusForNeedsReview(t *testing.T) {
	itemDir := t.TempDir()
	writeRound(t, itemDir, Round{
		RoundNum:    1,
		GeneratedAt: "2026-04-02T00:00:00Z",
		ExecutionID: "exec-awaiting-review",
		Status:      RoundStatusGathering,
		RunID:       "run-awaiting-review",
		Evidence:    []EvidenceItem{},
	})

	spawner := &capturingSpawner{
		enabled: true,
		runState: agentmanager.RunState{
			RunID:  "run-awaiting-review",
			Status: "needs_review",
		},
	}
	svc := newTestService(spawner, "")
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	rounds, err := svc.ListRounds("task", "test")
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("expected 1 round, got %d", len(rounds))
	}
	if rounds[0].Status != RoundStatusGathering {
		t.Fatalf("round status = %q, want %q", rounds[0].Status, RoundStatusGathering)
	}
	if rounds[0].CurrentRunStatus != "needs_review" {
		t.Fatalf("current run status = %q, want needs_review", rounds[0].CurrentRunStatus)
	}
}

// TestTriggerReviewAgent_ReroutesToOperation verifies the manual trigger rebuilds
// the backlog identity from the execution context and reroutes it to the
// review-round operation (correct target ref), writing a runner-owned gathering
// round. The old prompt/attachment building is gone — the operation runner owns
// context assembly now.
func TestTriggerReviewAgent_ReroutesToOperation(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	itemDir := setupItemDir(t, "research")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	starter := newFakeReviewOperationStarter()
	svc.SetOperationStarter(starter)
	svc.loadExecutionContext = func(_ context.Context, executionID string) (*ExecutionContext, error) {
		if executionID != "exec-rebuild" {
			t.Fatalf("unexpected execution id: %s", executionID)
		}
		return &ExecutionContext{
			BacklogKind:       "research",
			BacklogName:       "test-item",
			ItemTitle:         "Research Item",
			AffectedScenarios: []string{"scenario-b", "scenario-a"},
		}, nil
	}

	if err := svc.TriggerReviewAgent(context.Background(), "exec-rebuild"); err != nil {
		t.Fatalf("TriggerReviewAgent failed: %v", err)
	}

	if starter.calls != 1 {
		t.Fatalf("expected exactly 1 operation start, got %d", starter.calls)
	}
	if starter.lastReq.Operation != "review-round" {
		t.Errorf("Operation = %q, want review-round", starter.lastReq.Operation)
	}
	if starter.lastReq.TargetKind != "backlog-item" {
		t.Errorf("TargetKind = %q, want backlog-item", starter.lastReq.TargetKind)
	}
	if starter.lastReq.TargetID != "research/test-item" {
		t.Errorf("TargetID = %q, want research/test-item", starter.lastReq.TargetID)
	}

	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("expected round 1 on disk: err=%v round=%v", err, round)
	}
	if round.RunID != "test-run-id" || round.OpExecutionID != "exec-test" {
		t.Errorf("round missing run association: RunID=%q OpExecutionID=%q", round.RunID, round.OpExecutionID)
	}
}
