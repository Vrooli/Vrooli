package rewrite

import (
	"context"
	"testing"

	intgraph "go-code-graph/internal/graph"
)

func newSvc(t *testing.T, exec RewriteExecutor) *Service {
	t.Helper()
	return NewService(NewMemoryStore(), exec, intgraph.NewPathMutex())
}

type recordingExec struct {
	calls []Operation
	err   error
}

func (r *recordingExec) Execute(_ context.Context, _ string, op Operation) error {
	r.calls = append(r.calls, op)
	return r.err
}

func TestServicePlanIsDeterministic(t *testing.T) {
	t.Parallel()
	svc1 := newSvc(t, &recordingExec{})
	svc2 := newSvc(t, &recordingExec{})
	in := PlanInput{
		ScenarioPath: "/tmp/x",
		Operations: []Operation{
			FileMove{From: "a.go", To: "b.go"},
			ImportRewrite{Old: "foo", New: "bar"},
		},
	}
	p1, err := svc1.Plan(context.Background(), in)
	if err != nil {
		t.Fatalf("plan1: %v", err)
	}
	p2, err := svc2.Plan(context.Background(), in)
	if err != nil {
		t.Fatalf("plan2: %v", err)
	}
	if p1.ID != p2.ID {
		t.Fatalf("plan id should be deterministic; got %q vs %q", p1.ID, p2.ID)
	}
	if p1.ID == "" {
		t.Fatal("plan id must be non-empty")
	}
}

func TestServicePlanRejectsEmpty(t *testing.T) {
	t.Parallel()
	svc := newSvc(t, &recordingExec{})
	_, err := svc.Plan(context.Background(), PlanInput{ScenarioPath: "/tmp/x"})
	if err == nil {
		t.Fatal("expected error for empty ops")
	}
	rerr, ok := err.(RewriteError)
	if !ok || rerr.Kind != RewriteErrorNoOperations {
		t.Fatalf("want NoOperations, got %v", err)
	}
}

func TestServiceApplyUnknownPlanIsPlanNotFound(t *testing.T) {
	t.Parallel()
	svc := newSvc(t, &recordingExec{})
	_, err := svc.Apply(context.Background(), ApplyInput{
		ScenarioPath: "/tmp/x",
		PlanID:       "nope",
		Apply:        true,
	})
	if err == nil {
		t.Fatal("expected error for unknown plan")
	}
	rerr, ok := err.(RewriteError)
	if !ok || rerr.Kind != RewriteErrorPlanNotFound {
		t.Fatalf("want PlanNotFound, got %v", err)
	}
}

func TestServiceApplyApplyFalseIsApplyNotSet(t *testing.T) {
	t.Parallel()
	svc := newSvc(t, &recordingExec{})
	_, err := svc.Apply(context.Background(), ApplyInput{
		ScenarioPath: "/tmp/x",
		PlanID:       "x",
		Apply:        false,
	})
	if err == nil {
		t.Fatal("expected error for apply=false")
	}
	rerr, ok := err.(RewriteError)
	if !ok || rerr.Kind != RewriteErrorApplyNotSet {
		t.Fatalf("want ApplyNotSet, got %v", err)
	}
}

func TestServiceApplyDryRunSkipsExecutor(t *testing.T) {
	t.Parallel()
	exec := &recordingExec{}
	svc := newSvc(t, exec)
	plan, err := svc.Plan(context.Background(), PlanInput{
		ScenarioPath: "/tmp/x",
		Operations:   []Operation{FileMove{From: "a.go", To: "b.go"}},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	res, err := svc.Apply(context.Background(), ApplyInput{
		ScenarioPath: "/tmp/x",
		PlanID:       plan.ID,
		Apply:        true,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.DryRun {
		t.Fatal("expected DryRun=true")
	}
	if len(res.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res.Results))
	}
	if len(exec.calls) != 0 {
		t.Fatalf("executor must not be called in dry-run; got %d", len(exec.calls))
	}
}

func TestServiceApplyHappyPath(t *testing.T) {
	t.Parallel()
	exec := &recordingExec{}
	svc := newSvc(t, exec)
	plan, err := svc.Plan(context.Background(), PlanInput{
		ScenarioPath: "/tmp/x",
		Operations:   []Operation{FileMove{From: "a.go", To: "b.go"}},
	})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	res, err := svc.Apply(context.Background(), ApplyInput{
		ScenarioPath: "/tmp/x",
		PlanID:       plan.ID,
		Apply:        true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.DryRun {
		t.Fatal("expected DryRun=false")
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor calls: want 1, got %d", len(exec.calls))
	}
	if len(res.Results) != 1 || res.Results[0].Status != OperationStatusOK {
		t.Fatalf("unexpected results: %+v", res.Results)
	}
}
