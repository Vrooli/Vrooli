package execution

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"
)

type fakeGoalRunCreator struct {
	requests []agentmanager.GoalRunRequest
}

func (f *fakeGoalRunCreator) CreateGoalRun(_ context.Context, req agentmanager.GoalRunRequest) (agentmanager.GoalRunResult, error) {
	f.requests = append(f.requests, req)
	return agentmanager.GoalRunResult{RunID: "run-1", TaskID: "task-1"}, nil
}

func TestLaunchGoalRunCreatesOneRunAndPersistsFields(t *testing.T) {
	root := t.TempDir()
	fake := &fakeGoalRunCreator{}
	svc := NewService(ServiceConfig{
		DataRoot:           root,
		StorePath:          filepath.Join(root, "executions.json"),
		RepoRoot:           filepath.Join(root, "scenarios", "swarm-manager"),
		PlanRenderer:       testPlanRenderer(),
		TransitionRegistry: testTransitionRegistry(t),
		GoalRunCreator:     fake,
	})
	record := Record{ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "goal-item", ExecutionMode: transitions.ExecutionModeGoal, Status: StatusPending, PlanManagerExecutionID: "pm-exec-1", OperatorNote: "keep it unattended", ExecutionPreferences: &ExecutionPreferences{PreferredRunner: "opencode", Model: "m", Effort: "low"}}
	item := backlogItem{Name: "goal-item", Kind: "execute", ExecutionMode: transitions.ExecutionModeGoal, AcceptanceAllow: []string{"scenarios/swarm-manager/**"}, ScopePolicy: "fixed"}
	records := []Record{record}

	got, err := svc.launchGoalRun(context.Background(), records, 0, record, item, "goal-plan")
	if err != nil {
		t.Fatalf("launchGoalRun: %v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("expected exactly one goal run, got %d", len(fake.requests))
	}
	req := fake.requests[0]
	if !strings.Contains(req.Prompt, "Operator note: keep it unattended") {
		t.Fatalf("operator note missing from the composed message: %q", req.Prompt)
	}
	if !strings.Contains(req.Prompt, "Boundary: scenarios/swarm-manager/**") {
		t.Fatalf("acceptance allow missing from the boundary slot: %q", req.Prompt)
	}
	if req.Until == "" || req.IdempotencyKey != "goal/exec-1/1/1" || req.Tag != "swarm-execution-exec-1" {
		t.Fatalf("goal run request incomplete: %+v", req)
	}
	// Agent Manager requires a non-empty, workspace-root task scope; the goal
	// message boundary carries the editable globs.
	wantRoot := filepath.Dir(filepath.Join(root, "scenarios", "swarm-manager"))
	if req.ScopePath != wantRoot || req.ProjectRoot != wantRoot {
		t.Fatalf("goal run scope = (%q,%q), want workspace root %q", req.ScopePath, req.ProjectRoot, wantRoot)
	}
	if req.PreferredRunner != "opencode" || req.Model != "m" || req.Effort != "low" {
		t.Fatalf("execution preferences not forwarded: runner=%q model=%q effort=%q", req.PreferredRunner, req.Model, req.Effort)
	}
	if got.RunID != "run-1" || got.GoalMessageDigest == "" || got.Status != StatusStarting {
		t.Fatalf("record fields not persisted: %+v", got)
	}
}
