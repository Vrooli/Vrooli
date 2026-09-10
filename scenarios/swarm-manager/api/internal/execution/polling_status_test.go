package execution

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
func TestCommitExecutionRound_CompletedNonEligible_WritesReviewPending(t *testing.T) {
	svc := newTestPollingService(t)

	specPath := seedBacklogSpec(t, svc, "idea", "non-eligible", "in_progress")

	// PromptTrace.Purpose takes precedence in effectiveRunType(). Setting it
	// to a value outside {"", process, fixup, followup, custom} makes
	// isFinalizationEligible return false (see finalization_types.go:156).
	rec := Record{
		ExecutionID:   "exec-np",
		BacklogKind:   "idea",
		BacklogName:   "non-eligible",
		RunID:         "run-np",
		OpExecutionID: "op-np",
		Status:        StatusRunning,
		PromptTrace:   &PromptTrace{Purpose: "non-eligible-purpose"},
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	// Sanity: the test relies on this branch selection. If future changes to
	// isFinalizationEligible accept our Operation, the whole scenario shifts.
	if isFinalizationEligible(rec) {
		t.Fatalf("test setup invalid: record must be non-finalization-eligible")
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	commitExecutionRoundForTest(t, svc, "op-np", "completed")

	if got := loadSpecStatus(t, specPath); got != backlogStatusReviewPending {
		t.Fatalf("backlog status = %q, want %q", got, backlogStatusReviewPending)
	}
}

// TestPolling_CompletedEligible_DoesNotWriteBacklog covers the branch where a
// finalization-eligible item reports complete. Backlog status MUST NOT flip
// yet — it remains in_progress until finalization's finishFinalization writes
// in_review. (Polling only flips the execution record to StatusValidating and
// queues finalization.)
func TestCommitExecutionRound_CompletedEligible_DoesNotWriteBacklog(t *testing.T) {
	svc := newTestPollingService(t)

	specPath := seedBacklogSpec(t, svc, "execute", "eligible-item", "in_progress")

	rec := Record{
		ExecutionID:   "exec-ok",
		BacklogKind:   "execute",
		BacklogName:   "eligible-item",
		RunID:         "run-ok",
		OpExecutionID: "op-ok",
		Status:        StatusRunning,
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if !isFinalizationEligible(rec) {
		t.Fatalf("test setup invalid: record must be finalization-eligible")
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	commitExecutionRoundForTest(t, svc, "op-ok", "completed")

	// The backlog is untouched until finalization runs. finalization.go
	// writes in_review; the completion bridge must not pre-empt that.
	if got := loadSpecStatus(t, specPath); got != "in_progress" {
		t.Fatalf("backlog status = %q, want %q (untouched by polling)", got, "in_progress")
	}

	loaded, _ := svc.store.Load()
	if loaded[0].Status != StatusValidating {
		t.Errorf("record status = %s, want StatusValidating (finalization pending)", loaded[0].Status)
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
			svc := newTestPollingService(t)
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
	svc := newTestPollingService(t)
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

func TestFinalization_SelfChecksRemainUnassessedWithoutCodingFixup(t *testing.T) {
	svc := newTestPollingService(t)
	svc.selfScenarioName = "swarm-manager"
	svc.policyProvider = &stubPolicyProvider{}
	specPath := seedBacklogSpec(t, svc, "execute", "self-change", "in_progress")
	item := backlogItem{Name: "self-change", Kind: "execute", Status: "in_progress", AcceptanceAllow: []string{"scenarios/swarm-manager/**"}}
	body, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.store.Save([]Record{{ExecutionID: "exec-self", BacklogKind: "execute", BacklogName: "self-change", Status: StatusValidating}}); err != nil {
		t.Fatal(err)
	}
	// No lifecycle or health transport is configured: self-checks must be
	// explicitly deferred, without attempting to restart the serving process.
	if err := svc.processFinalization(context.Background(), "exec-self"); err != nil {
		t.Fatal(err)
	}
	records, err := svc.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	rec := records[0]
	if rec.Status != StatusCompleted || rec.Finalization.AggregateClassification != FinalizationAggregateNotAssessable {
		t.Fatalf("deferred self-checks must finish for review without a coding fixup or ready verdict: %+v", rec)
	}
	if len(rec.Finalization.Scenarios) != 1 {
		t.Fatalf("self must remain in the evidence inventory: %+v", rec.Finalization)
	}
	state := rec.Finalization.Scenarios[0]
	if state.Restart.Status != FinalizationStatusSkipped || state.Restart.Attempts != 0 || state.Restart.LastError != "" || state.Health.Status != FinalizationStatusSkipped || state.Review.Status != FinalizationStatusSkipped || state.Review.Result != nil || state.Review.SkipReason == "" {
		t.Fatalf("deferred checks must be explicit without fabricated attempts, errors or results: %+v", state)
	}
	if got := loadSpecStatus(t, specPath); got != backlogStatusReviewPending {
		t.Fatalf("deferred checks require an operator review decision, got %q", got)
	}
	classification, summary, actionable := summarizeFinalization(*rec.Finalization, true)
	if actionable || classification != FinalizationAggregateNotAssessable || summary == "" {
		t.Fatalf("self deferral alone must not request automated coding: %s %q %v", classification, summary, actionable)
	}
	// A self deferral must not hide a genuine regression or another scenario's
	// failed restart. Both remain actionable in the same aggregate.
	for _, regression := range []bool{false, true} {
		fin := *rec.Finalization
		fin.Scenarios = append([]ScenarioFinalization(nil), fin.Scenarios...)
		if regression {
			fin.Scenarios[0].BaselineDiff = regressionDiff("swarm-manager")
		} else {
			fin.Scenarios = append(fin.Scenarios, ScenarioFinalization{ScenarioName: "other", Restart: RestartResult{Status: FinalizationStatusFailed}})
		}
		classification, _, actionable := summarizeFinalization(fin, true)
		if !actionable || classification != FinalizationAggregateNeedsWork {
			t.Fatalf("real failure must remain actionable alongside self deferral: regression=%v classification=%s actionable=%v", regression, classification, actionable)
		}
	}
}
