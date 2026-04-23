package review

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

func newTestService(spawner *capturingSpawner, promptResult string) *Service {
	svc := &Service{
		rootDir:       "/tmp/test-backlog",
		agentService:  spawner,
		inspector:     spawner,
		promptClient:  &promptmanager.MockClient{Result: promptResult},
		itemDirFn:     func(_, _ string) string { return "/tmp/test-backlog/tasks/test-item" },
		loadItemTitle: func(_, _ string) (string, error) { return "Loaded Title", nil },
		activeRounds:  make(map[string]activeRound),
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

// setupItemDir creates a temporary backlog item directory with the expected
// deliverable for the given kind.
func setupItemDir(t *testing.T, kind string) string {
	t.Helper()
	dir := t.TempDir()
	deliverable := "plan.md"
	content := "# Test Plan\nDo the thing."
	if kind == "research" {
		deliverable = "conclusion.md"
		content = "# Test Conclusion\nResearch findings."
	}
	if err := os.WriteFile(filepath.Join(dir, deliverable), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func findAttachment(req *agentmanager.BacklogSpawnRequest, key string) *domainpb.ContextAttachment {
	for _, attachment := range req.ContextAttachments {
		if attachment.Key == key {
			return attachment
		}
	}
	return nil
}

func TestStartReview_ContextAttachments(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	svc.itemDirFn = func(_, _ string) string { return setupItemDir(t, "task") }

	itemDir := svc.resolveItemDir("task", "test-item")

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID:       "exec-123",
		BacklogKind:       "task",
		BacklogName:       "test-item",
		ItemTitle:         "Test Item",
		ItemDir:           itemDir,
		AffectedScenarios: []string{"scenario-a", "scenario-b"},
		ChangedPathsByScenario: map[string][]string{
			"scenario-a": {"api/main.go", "api/handler.go"},
			"scenario-b": {"ui/src/App.tsx"},
		},
		GCTResultsJSON: `{"scenario-a":{"classification":"ready"}}`,
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	if spawner.captured == nil {
		t.Fatal("SpawnBacklog was not called")
	}

	req := spawner.captured
	atts := req.ContextAttachments
	if len(atts) == 0 {
		t.Fatal("expected context attachments, got none")
	}

	// Verify expected attachment keys.
	wantKeys := map[string]bool{
		"plan-content":       false,
		"diff-summary":       false,
		"changed-paths":      false,
		"affected-scenarios": false,
		"gct-review-results": false,
	}
	for _, att := range atts {
		if _, ok := wantKeys[att.Key]; ok {
			wantKeys[att.Key] = true
		}
		if att.Type != "note" {
			t.Errorf("attachment %q has type %q, want %q", att.Key, att.Type, "note")
		}
	}
	for key, found := range wantKeys {
		if !found {
			t.Errorf("missing expected attachment with key %q", key)
		}
	}

	// user-request should NOT be present in initial review.
	for _, att := range atts {
		if att.Key == "user-request" {
			t.Error("user-request attachment should not be present in initial review")
		}
	}
}

func TestStartReview_InstructionsOnly(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "pure review instructions")
	svc.itemDirFn = func(_, _ string) string { return setupItemDir(t, "task") }

	itemDir := svc.resolveItemDir("task", "test-item")

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID:       "exec-456",
		BacklogKind:       "task",
		BacklogName:       "test-item",
		ItemTitle:         "Test Item",
		ItemDir:           itemDir,
		AffectedScenarios: []string{"my-scenario"},
		ChangedPathsByScenario: map[string][]string{
			"my-scenario": {"api/main.go"},
		},
		GCTResultsJSON: `{"my-scenario":{"classification":"needs_work","dimensions":[{"name":"codeQuality","status":"yellow"}]}}`,
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	req := spawner.captured
	// Prompt should contain instructions, NOT raw GCT JSON or plan content.
	if req.Prompt != "pure review instructions" {
		t.Errorf("Prompt should be rendered instructions only, got %q", req.Prompt)
	}
	if req.Description != "pure review instructions" {
		t.Errorf("Description should be rendered instructions only, got %q", req.Description)
	}
}

func TestStartReview_NoGCT(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "instructions")
	svc.itemDirFn = func(_, _ string) string { return setupItemDir(t, "task") }

	itemDir := svc.resolveItemDir("task", "test-item")

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID:       "exec-789",
		BacklogKind:       "task",
		BacklogName:       "test-item",
		ItemTitle:         "Test Item",
		ItemDir:           itemDir,
		AffectedScenarios: []string{"scenario-a"},
		ChangedPathsByScenario: map[string][]string{
			"scenario-a": {"api/main.go"},
		},
		GCTResultsJSON: "", // empty
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	for _, att := range spawner.captured.ContextAttachments {
		if att.Key == "gct-review-results" {
			t.Error("gct-review-results attachment should be absent when GCTResultsJSON is empty")
		}
	}
}

func TestStartReview_ResearchUsesConclusionDeliverable(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	svc.itemDirFn = func(_, _ string) string { return setupItemDir(t, "research") }

	itemDir := svc.resolveItemDir("research", "test-item")

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID:       "exec-research",
		BacklogKind:       "research",
		BacklogName:       "test-item",
		ItemTitle:         "Research Item",
		ItemDir:           itemDir,
		AffectedScenarios: []string{"scenario-a"},
	})
	if err != nil {
		t.Fatalf("startReview failed: %v", err)
	}

	attachment := findAttachment(spawner.captured, "plan-content")
	if attachment == nil {
		t.Fatal("expected deliverable attachment")
	}
	if !strings.Contains(attachment.Content, "# Test Conclusion") {
		t.Fatalf("expected research review to use conclusion.md, got %q", attachment.Content)
	}
}

func TestBuildReviewAttachments_UserRequest(t *testing.T) {
	atts := buildReviewAttachments("plan text", []string{"a.go"}, []string{"scen"}, "", "show me the logs")

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
	atts := buildReviewAttachments("plan", nil, nil, "", "")
	for _, att := range atts {
		if att.Key == "user-request" {
			t.Error("user-request should be absent when empty")
		}
	}
}

func TestBuildReviewAttachments_NonSandboxDiffSummary(t *testing.T) {
	// When there are 0 changed paths (non-sandbox execution), the diff-summary
	// should explain that changes may exist but weren't tracked.
	atts := buildReviewAttachments("plan", nil, nil, "", "")

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
	atts := buildReviewAttachments("plan", []string{"a.go", "b.go"}, []string{"scen"}, "", "")

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
	atts := buildReviewAttachments("plan", []string{"a.go"}, []string{"scen"}, `{"s":"ready"}`, "")

	priorities := make(map[string]string)
	for _, att := range atts {
		priorities[att.Key] = att.Priority
	}

	highKeys := []string{"plan-content", "diff-summary", "changed-paths", "gct-review-results"}
	for _, key := range highKeys {
		if priorities[key] != "high" {
			t.Errorf("attachment %q priority = %q, want %q", key, priorities[key], "high")
		}
	}
	if priorities["affected-scenarios"] != "medium" {
		t.Errorf("affected-scenarios priority = %q, want %q", priorities["affected-scenarios"], "medium")
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

func TestStartReview_TracksActiveRound(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "instructions")
	svc.itemDirFn = func(_, _ string) string { return setupItemDir(t, "task") }

	itemDir := svc.resolveItemDir("task", "test-item")

	err := svc.startReview(context.Background(), startReviewParams{
		ExecutionID:       "exec-track",
		BacklogKind:       "task",
		BacklogName:       "test-item",
		ItemTitle:         "Test",
		ItemDir:           itemDir,
		AffectedScenarios: []string{"s"},
		ChangedPathsByScenario: map[string][]string{
			"s": {"a.go"},
		},
	})
	if err != nil {
		t.Fatalf("startReview: %v", err)
	}

	svc.mu.Lock()
	count := len(svc.activeRounds)
	_, tracked := svc.activeRounds["test-run-id"]
	svc.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 active round, got %d", count)
	}
	if !tracked {
		t.Error("expected run-id 'test-run-id' to be tracked")
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

func TestTriggerReviewAgent_RebuildsContextFromExecution(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "review instructions here")
	itemDir := setupItemDir(t, "research")
	svc.itemDirFn = func(_, _ string) string { return itemDir }
	svc.loadExecutionContext = func(_ context.Context, executionID string) (*ExecutionContext, error) {
		if executionID != "exec-rebuild" {
			t.Fatalf("unexpected execution id: %s", executionID)
		}
		return &ExecutionContext{
			BacklogKind:       "research",
			BacklogName:       "test-item",
			ItemTitle:         "Research Item",
			AffectedScenarios: []string{"scenario-b", "scenario-a"},
			ChangedPathsByScenario: map[string][]string{
				"scenario-a": {"api/main.go"},
				"scenario-b": {"ui/src/App.tsx"},
			},
			GCTResultsJSON: `{"scenario-a":{"classification":"ready"}}`,
		}, nil
	}

	if err := svc.TriggerReviewAgent(context.Background(), "exec-rebuild"); err != nil {
		t.Fatalf("TriggerReviewAgent failed: %v", err)
	}

	if spawner.captured == nil {
		t.Fatal("expected SpawnBacklog to be called")
	}
	if spawner.captured.Kind != "research" || spawner.captured.Name != "test-item" {
		t.Fatalf("expected research/test-item, got %s/%s", spawner.captured.Kind, spawner.captured.Name)
	}
	if spawner.captured.Title != "Review: Research Item" {
		t.Fatalf("unexpected title: %s", spawner.captured.Title)
	}

	paths := findAttachment(spawner.captured, "changed-paths")
	if paths == nil || !strings.Contains(paths.Content, "api/main.go") || !strings.Contains(paths.Content, "ui/src/App.tsx") {
		t.Fatalf("expected changed-paths attachment, got %#v", paths)
	}
	scenarios := findAttachment(spawner.captured, "affected-scenarios")
	if scenarios == nil || scenarios.Content != "scenario-a\nscenario-b" {
		t.Fatalf("expected sorted affected scenarios, got %#v", scenarios)
	}
	gct := findAttachment(spawner.captured, "gct-review-results")
	if gct == nil || !strings.Contains(gct.Content, "\"classification\":\"ready\"") {
		t.Fatalf("expected gct review attachment, got %#v", gct)
	}
}
