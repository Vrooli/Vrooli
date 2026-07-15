package backlog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/workshop"
)

func opsTestDeps(t *testing.T) (OpsHandlerDeps, *FileStore) {
	t.Helper()
	store := NewFileStore(t.TempDir())
	return OpsHandlerDeps{Store: store}, store
}

func opsCtx(id string, execID string, result any) opsrunner.ActionContext {
	ac := opsrunner.ActionContext{
		Target:      opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: id},
		Workflow:    agentops.WorkflowInstance{InstanceID: "wf-1"},
		Operation:   agentops.OpWorkshopRound,
		Outcome:     "continue",
		ExecutionID: execID,
	}
	if result != nil {
		raw, _ := json.Marshal(result)
		ac.Result = raw
	}
	return ac
}

func saveTestItem(t *testing.T, store *FileStore, kind BacklogKind, name string, status BacklogStatus) {
	t.Helper()
	if err := os.MkdirAll(store.ItemDir(kind, name), 0o750); err != nil {
		t.Fatal(err)
	}
	item := BacklogItem{
		Name: name, Title: name, Status: status, Kind: kind,
		Created: "2026-01-01T00:00:00Z", Updated: "2026-01-01T00:00:00Z",
	}
	if err := store.SaveItem(item); err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
}

func workshopResult() map[string]any {
	return map[string]any{
		"handoff":   map[string]any{"summary": "did work"},
		"progress":  "continue",
		"decisions": []any{map[string]any{"id": "d1", "topic": "t", "text": "x", "options": []any{map[string]any{"key": "A", "label": "l", "rationale": "r", "recommended": true}}}},
		"self_assessment": map[string]any{
			"problem_clarity": 3, "scope_defined": 2, "approach_solid": 1, "testable": 2, "risk_awareness": 0,
		},
	}
}

func TestCommitWorkshopRoundWritesRoundFileAndIsIdempotent(t *testing.T) {
	deps, store := opsTestDeps(t)
	itemDir := store.ItemDir(KindExecute, "x")
	roundPath := filepath.Join(itemDir, "workshop", "round-001.json")

	ac := opsCtx("execute/x", "exec-1", workshopResult())
	if err := deps.commitWorkshopRound(context.Background(), ac); err != nil {
		t.Fatalf("commit: %v", err)
	}
	raw, err := os.ReadFile(roundPath)
	if err != nil {
		t.Fatalf("round file: %v", err)
	}
	var round workshop.Round
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("decode round: %v", err)
	}
	if round.RoundNum != 1 || round.Readiness["problem_clarity"] != 3 || len(round.Items) != 1 {
		t.Fatalf("round content wrong: %+v", round)
	}

	// Re-firing the SAME execution is idempotent: no round-002 is created.
	if err := deps.commitWorkshopRound(context.Background(), ac); err != nil {
		t.Fatalf("commit replay: %v", err)
	}
	if _, err := os.Stat(filepath.Join(itemDir, "workshop", "round-002.json")); !os.IsNotExist(err) {
		t.Fatalf("idempotency broken: a duplicate round was written")
	}

	// A DIFFERENT execution advances to round-002.
	if err := deps.commitWorkshopRound(context.Background(), opsCtx("execute/x", "exec-2", workshopResult())); err != nil {
		t.Fatalf("commit second: %v", err)
	}
	if _, err := os.Stat(filepath.Join(itemDir, "workshop", "round-002.json")); err != nil {
		t.Fatalf("second execution did not advance the round: %v", err)
	}
}

func TestOpenReviewWritesArtifactIdempotently(t *testing.T) {
	deps, store := opsTestDeps(t)
	reviewPath := filepath.Join(store.ItemDir(KindExecute, "x"), "workshop", "review-open.json")
	ac := opsCtx("execute/x", "exec-1", workshopResult())
	for i := 0; i < 2; i++ {
		if err := deps.openReview(context.Background(), ac); err != nil {
			t.Fatalf("open-review: %v", err)
		}
	}
	if _, err := os.Stat(reviewPath); err != nil {
		t.Fatalf("review-open not written: %v", err)
	}
}

func TestSetStatusHandlers(t *testing.T) {
	cases := []struct {
		action agentops.ActionName
		status BacklogStatus
	}{
		{agentops.ActionCompleteItem, StatusCompleted},
		{agentops.ActionFailItem, StatusFailed},
		{agentops.ActionRequestRevision, StatusBacklog},
	}
	for _, tc := range cases {
		t.Run(string(tc.action), func(t *testing.T) {
			deps, store := opsTestDeps(t)
			saveTestItem(t, store, KindExecute, "x", StatusInProgress)
			h := deps.setStatus(tc.status)
			if err := h(context.Background(), opsCtx("execute/x", "exec-1", nil)); err != nil {
				t.Fatalf("handler: %v", err)
			}
			got, err := store.LoadItem(KindExecute, "x")
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.status {
				t.Fatalf("status: want %q, got %q", tc.status, got.Status)
			}
		})
	}
}

func TestEscalateNeedsAttentionWritesMarkerAndStatus(t *testing.T) {
	deps, store := opsTestDeps(t)
	saveTestItem(t, store, KindExecute, "x", StatusInProgress)
	ac := opsCtx("execute/x", "exec-1", workshopResult())
	ac.Outcome = "blocked"
	if err := deps.escalateNeedsAttention(context.Background(), ac); err != nil {
		t.Fatalf("escalate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.ItemDir(KindExecute, "x"), "needs-attention.json")); err != nil {
		t.Fatalf("needs-attention marker not written: %v", err)
	}
	got, _ := store.LoadItem(KindExecute, "x")
	if got.Status != StatusNeedsFollowup {
		t.Fatalf("status: want needs_followup, got %q", got.Status)
	}
}

func TestBindPlanWritesCanonicalPlanRef(t *testing.T) {
	deps, store := opsTestDeps(t)
	saveTestItem(t, store, KindExecute, "x", StatusInProgress)
	result := map[string]any{
		"progress": "complete",
		"plan_ref": map[string]any{"provider": "plan-manager", "plan_id": "plan-123", "slug": "my-plan", "role": "execution_spec"},
	}
	ac := opsCtx("execute/x", "exec-1", result)
	ac.Operation = agentops.OpWorkshopFinalize
	ac.Outcome = "completed"
	if err := deps.bindPlan(context.Background(), ac); err != nil {
		t.Fatalf("bind-plan: %v", err)
	}
	got, err := store.LoadItem(KindExecute, "x")
	if err != nil {
		t.Fatal(err)
	}
	if got.PlanRef == nil || got.PlanRef.PlanID != "plan-123" || got.PlanRef.Provider != PlanRefProviderPlanManager {
		t.Fatalf("plan_ref not bound: %+v", got.PlanRef)
	}
}

func TestBindPlanFailsClosedWithoutPlanRef(t *testing.T) {
	deps, store := opsTestDeps(t)
	saveTestItem(t, store, KindExecute, "x", StatusInProgress)
	// A finalize completion whose result carries no canonical plan_ref must bind
	// nothing and leave the item untouched.
	result := map[string]any{
		"handoff":  map[string]any{"summary": "finished but authored no plan"},
		"progress": "complete",
	}
	ac := opsCtx("execute/x", "exec-1", result)
	ac.Operation = agentops.OpWorkshopFinalize
	ac.Outcome = "completed"
	if err := deps.bindPlan(context.Background(), ac); err == nil {
		t.Fatal("bind-plan: expected fail-closed error when result carries no plan_ref")
	}
	got, err := store.LoadItem(KindExecute, "x")
	if err != nil {
		t.Fatal(err)
	}
	if got.PlanRef != nil {
		t.Fatalf("item must be left unbound on fail-closed bind-plan, got %+v", got.PlanRef)
	}
}

func TestBindPlanRejectsInvalidPlanRefAndLeavesItemUntouched(t *testing.T) {
	deps, store := opsTestDeps(t)
	saveTestItem(t, store, KindExecute, "x", StatusInProgress)
	// A ref with the wrong provider is invalid; bind-plan must reject it without
	// touching the item.
	result := map[string]any{
		"progress": "complete",
		"plan_ref": map[string]any{"provider": "not-plan-manager", "plan_id": "p", "role": "execution_spec"},
	}
	ac := opsCtx("execute/x", "exec-1", result)
	ac.Operation = agentops.OpWorkshopFinalize
	ac.Outcome = "completed"
	if err := deps.bindPlan(context.Background(), ac); err == nil {
		t.Fatal("bind-plan: expected error for invalid plan_ref")
	}
	got, _ := store.LoadItem(KindExecute, "x")
	if got.PlanRef != nil {
		t.Fatalf("item must be left unbound when plan_ref is invalid, got %+v", got.PlanRef)
	}
}

func TestRegisterOpsHandlersCoversPolicyRoutedActions(t *testing.T) {
	deps, _ := opsTestDeps(t)
	reg := opsrunner.NewActionRegistry()
	// Must not panic: every registered name is in the closed vocabulary.
	RegisterOpsHandlers(reg, deps)
}

func TestSplitItemRefRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "noSlash", "/name", "kind/"} {
		if _, _, err := splitItemRef(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	kind, name, err := splitItemRef("execute/my-item")
	if err != nil || kind != KindExecute || name != "my-item" {
		t.Fatalf("split failed: %v %q %q", err, kind, name)
	}
}
