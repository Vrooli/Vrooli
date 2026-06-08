package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
)

// These tests pin the fire-and-forget status writes the review lifecycle
// depends on. A regression here would silently break the review gate:
// execution would auto-flip items to completed/failed before the user has
// seen the review round.
//
// The plan explicitly calls for these tests (plan §W1):
//   "assert fire-and-forget paths land in `in_review`, not `completed`/`failed`"

// seedBacklogSpec writes a minimal spec.json at the same path that
// loadBacklogItem / updateBacklogStatus use. Returns the spec path.
func seedBacklogSpec(t *testing.T, svc *Service, kind, name, initialStatus string) string {
	t.Helper()
	dir := svc.itemDir(kind, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	specPath := filepath.Join(dir, "spec.json")
	item := backlogItem{
		Name:   name,
		Title:  name,
		Status: initialStatus,
		Kind:   kind,
		Tags:   []string{},
	}
	body, _ := json.Marshal(item)
	if err := os.WriteFile(specPath, body, 0o644); err != nil {
		t.Fatalf("write spec.json: %v", err)
	}
	return specPath
}

// loadSpecStatus re-reads the on-disk status for assertion.
func loadSpecStatus(t *testing.T, specPath string) string {
	t.Helper()
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s: %v", specPath, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse spec.json: %v", err)
	}
	s, _ := m["status"].(string)
	return s
}

// runRefresh acquires the service mutex the same way production does.
func runRefresh(t *testing.T, svc *Service) {
	t.Helper()
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if _, _, err := svc.refreshRunningLocked(context.Background()); err != nil {
		t.Fatalf("refreshRunningLocked: %v", err)
	}
}

// TestPolling_CompletedNonEligible_WritesReviewPending covers the branch in
// polling.go where an execution type that skips finalization reports complete.
// The backlog item must land directly in review_pending (the user decides
// terminal; no review agent runs).
func TestPolling_CompletedNonEligible_WritesReviewPending(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-np": {RunID: "run-np", Status: "complete"},
		},
	}
	svc := newTestPollingService(t, inspector)

	specPath := seedBacklogSpec(t, svc, "idea", "non-eligible", "in_progress")

	// PromptTrace.Purpose takes precedence in effectiveRunType(). Setting it
	// to a value outside {"", process, fixup, followup, custom} makes
	// isFinalizationEligible return false (see finalization_types.go:156).
	rec := Record{
		ExecutionID: "exec-np",
		BacklogKind: "idea",
		BacklogName: "non-eligible",
		RunID:       "run-np",
		Status:      StatusRunning,
		PromptTrace: &PromptTrace{Purpose: "non-eligible-purpose"},
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	// Sanity: the test relies on this branch selection. If future changes to
	// isFinalizationEligible accept our Operation, the whole scenario shifts.
	if isFinalizationEligible(rec) {
		t.Fatalf("test setup invalid: record must be non-finalization-eligible")
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	if got := loadSpecStatus(t, specPath); got != backlogStatusReviewPending {
		t.Fatalf("backlog status = %q, want %q", got, backlogStatusReviewPending)
	}
}

// TestPolling_CompletedEligible_DoesNotWriteBacklog covers the branch where a
// finalization-eligible item reports complete. Backlog status MUST NOT flip
// yet — it remains in_progress until finalization's finishFinalization writes
// in_review. (Polling only flips the execution record to StatusValidating and
// queues finalization.)
func TestPolling_CompletedEligible_DoesNotWriteBacklog(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-ok": {RunID: "run-ok", Status: "complete"},
		},
	}
	svc := newTestPollingService(t, inspector)

	specPath := seedBacklogSpec(t, svc, "execute", "eligible-item", "in_progress")

	rec := Record{
		ExecutionID: "exec-ok",
		BacklogKind: "execute",
		BacklogName: "eligible-item",
		RunID:       "run-ok",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if !isFinalizationEligible(rec) {
		t.Fatalf("test setup invalid: record must be finalization-eligible")
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	// The backlog is untouched until finalization runs. finalization.go
	// writes in_review; polling must not pre-empt that.
	if got := loadSpecStatus(t, specPath); got != "in_progress" {
		t.Fatalf("backlog status = %q, want %q (untouched by polling)", got, "in_progress")
	}

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusValidating {
		t.Errorf("record status = %s, want StatusValidating (finalization pending)", loaded[0].Status)
	}
}

// TestPolling_RunFailed_WritesInReview covers the branch where the
// agent-manager run fails cleanly. The backlog item must land in in_review
// (not terminal) so the review agent can document the failure and the user
// decides the terminal state via review-decide. Terminal transitions are
// user-only (plan §W1).
func TestPolling_RunFailed_WritesInReview(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-fail": {RunID: "run-fail", Status: "failed", ErrorMsg: "boom"},
		},
	}
	svc := newTestPollingService(t, inspector)

	specPath := seedBacklogSpec(t, svc, "execute", "failed-item", "in_progress")

	rec := Record{
		ExecutionID: "exec-fail",
		BacklogKind: "execute",
		BacklogName: "failed-item",
		RunID:       "run-fail",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	if got := loadSpecStatus(t, specPath); got != backlogStatusInReview {
		t.Fatalf("backlog status = %q, want %q (review agent documents failure, user decides terminal)", got, backlogStatusInReview)
	}

	// Execution record still records the technical failure so operators can
	// see what agent-manager reported; only the user-facing backlog item
	// lingers in in_review for the review gate.
	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Errorf("execution record status = %s, want StatusFailed", loaded[0].Status)
	}
}

// TestPolling_Canceled_RestoresPrevious covers the cancellation branch. The
// backlog is restored to its previous pre-queue status (ready by default) so
// the user can requeue it.
func TestPolling_Canceled_RestoresPrevious(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-cx": {RunID: "run-cx", Status: "cancelled"},
		},
	}
	svc := newTestPollingService(t, inspector)

	specPath := seedBacklogSpec(t, svc, "execute", "cx-item", "in_progress")

	rec := Record{
		ExecutionID:    "exec-cx",
		BacklogKind:    "execute",
		BacklogName:    "cx-item",
		RunID:          "run-cx",
		Status:         StatusRunning,
		PreviousStatus: "ready",
		CreatedAt:      nowRFC3339(),
		UpdatedAt:      nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	if got := loadSpecStatus(t, specPath); got != "ready" {
		t.Fatalf("backlog status = %q, want %q (restored)", got, "ready")
	}
}

// TestPolling_MaxAge_WritesInReview covers the max-age timeout branch.
// When a run exceeds MaxRunAge, polling marks the execution record failed;
// the backlog item lands in in_review so the user can decide terminal via
// review-decide. The plan calls out this site specifically (polling.go:133).
func TestPolling_MaxAge_WritesInReview(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-old": {RunID: "run-old", Status: "running"},
		},
	}
	svc := newTestPollingService(t, inspector, func(cfg *ServiceConfig) {
		cfg.MaxRunAge = 1 * time.Millisecond
	})

	specPath := seedBacklogSpec(t, svc, "execute", "old-item", "in_progress")

	rec := Record{
		ExecutionID: "exec-old",
		BacklogKind: "execute",
		BacklogName: "old-item",
		RunID:       "run-old",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}
	tracker := svc.ensureRunTracker("run-old")
	tracker.FirstSeen = time.Now().Add(-1 * time.Hour)

	runRefresh(t, svc)

	if got := loadSpecStatus(t, specPath); got != backlogStatusInReview {
		t.Fatalf("backlog status = %q, want %q (user decides terminal)", got, backlogStatusInReview)
	}
	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Errorf("execution record status = %s, want StatusFailed", loaded[0].Status)
	}
}

// TestFinalization_Finish_WritesInReview covers the two fire-and-forget sites
// in finalization.go (lines 230, 236) that set the backlog status after
// finalization completes. Both the clean-success and actionable-failure
// branches must land the item in in_review so the user can run review-decide.
func TestFinalization_Finish_WritesInReview(t *testing.T) {
	cases := []struct {
		name      string
		scenarios []ScenarioFinalization
	}{
		{
			name: "clean success lands in_review",
			scenarios: []ScenarioFinalization{
				{
					ScenarioName: "scenario-a",
					Restart:      RestartResult{Status: FinalizationStatusCompleted},
					Health:       HealthCheckResult{Status: FinalizationStatusCompleted},
					Review: ScenarioReviewStep{
						Status: FinalizationStatusCompleted,
						Result: &ReviewResult{Classification: FinalizationAggregateReady, Summary: "ok"},
					},
				},
			},
		},
		{
			name: "actionable failure also lands in_review",
			scenarios: []ScenarioFinalization{
				{
					ScenarioName: "scenario-b",
					Restart:      RestartResult{Status: FinalizationStatusFailed, LastError: "boom"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestPollingService(t, &mockInspector{})
			specPath := seedBacklogSpec(t, svc, "execute", "fin-item", "in_progress")

			rec := Record{
				ExecutionID: "exec-fin",
				BacklogKind: "execute",
				BacklogName: "fin-item",
				RunID:       "run-fin",
				Status:      StatusValidating,
				CreatedAt:   nowRFC3339(),
				UpdatedAt:   nowRFC3339(),
				Finalization: &Finalization{
					Eligible:          true,
					Status:            FinalizationStatusRunning,
					Phase:             FinalizationPhaseReviewing,
					ScopeSource:       FinalizationScopeSandboxDiff,
					AffectedScenarios: []string{"scenario-a"},
					Scenarios:         tc.scenarios,
					Warnings:          []FinalizationWarning{},
					StartedAt:         nowRFC3339(),
				},
			}
			if err := svc.store.Save([]Record{rec}); err != nil {
				t.Fatal(err)
			}

			// reviewStarted=true models a review round actually spawning, so
			// the item lands in in_review where the agent gathers evidence.
			if err := svc.finishFinalization("exec-fin", true, ""); err != nil {
				t.Fatalf("finishFinalization: %v", err)
			}

			if got := loadSpecStatus(t, specPath); got != backlogStatusInReview {
				t.Fatalf("backlog status = %q, want %q (review agent runs, user decides)", got, backlogStatusInReview)
			}
		})
	}
}

// TestFinalization_Finish_NoReviewAgent_WritesReviewPending verifies the
// orphaned-in_review source fix: when no review round was started (agent
// disabled or spawn failure), finishFinalization routes the item straight to
// review_pending instead of stranding it in in_review with no round to ever
// advance it.
func TestFinalization_Finish_NoReviewAgent_WritesReviewPending(t *testing.T) {
	svc := newTestPollingService(t, &mockInspector{})
	specPath := seedBacklogSpec(t, svc, "execute", "fin-noagent", "in_progress")

	rec := Record{
		ExecutionID: "exec-noagent",
		BacklogKind: "execute",
		BacklogName: "fin-noagent",
		RunID:       "run-noagent",
		Status:      StatusValidating,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
		Finalization: &Finalization{
			Eligible:          true,
			Status:            FinalizationStatusRunning,
			Phase:             FinalizationPhaseReviewing,
			ScopeSource:       FinalizationScopeSandboxDiff,
			AffectedScenarios: []string{"scenario-a"},
			Scenarios: []ScenarioFinalization{
				{
					ScenarioName: "scenario-a",
					Restart:      RestartResult{Status: FinalizationStatusCompleted},
					Health:       HealthCheckResult{Status: FinalizationStatusCompleted},
					Review: ScenarioReviewStep{
						Status: FinalizationStatusCompleted,
						Result: &ReviewResult{Classification: FinalizationAggregateReady, Summary: "ok"},
					},
				},
			},
			Warnings:  []FinalizationWarning{},
			StartedAt: nowRFC3339(),
		},
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	if err := svc.finishFinalization("exec-noagent", false, "review agent disabled in settings"); err != nil {
		t.Fatalf("finishFinalization: %v", err)
	}

	if got := loadSpecStatus(t, specPath); got != backlogStatusReviewPending {
		t.Fatalf("backlog status = %q, want %q (no review agent → human-decidable)", got, backlogStatusReviewPending)
	}
}

// TestPolling_ConsecutiveErrors_WritesInReview covers the
// lost-contact-with-agent-manager branch. After maxConsecutiveErrors
// inspector failures, the execution record is marked failed; the backlog
// item lands in in_review so the user can decide terminal via review-decide.
func TestPolling_ConsecutiveErrors_WritesInReview(t *testing.T) {
	errInspector := &mockInspector{err: agentmanager.ErrNotAvailable}
	maxErrors := 3
	svc := newTestPollingService(t, errInspector, func(cfg *ServiceConfig) {
		cfg.MaxConsecutiveErrors = maxErrors
	})

	specPath := seedBacklogSpec(t, svc, "execute", "lost-item", "in_progress")

	rec := Record{
		ExecutionID: "exec-lost",
		BacklogKind: "execute",
		BacklogName: "lost-item",
		RunID:       "run-lost",
		Status:      StatusRunning,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxErrors; i++ {
		runRefresh(t, svc)
	}

	if got := loadSpecStatus(t, specPath); got != backlogStatusInReview {
		t.Fatalf("backlog status after %d errors = %q, want %q (user decides terminal)", maxErrors, got, backlogStatusInReview)
	}
	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusFailed {
		t.Errorf("execution record status = %s, want StatusFailed", loaded[0].Status)
	}
}
