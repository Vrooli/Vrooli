package rewrite_test

import (
	"context"
	"errors"
	"testing"

	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"
)

func newSvc(t *testing.T, fake *sidecarmocks.FakeSidecarClient) (*rewrite.Service, *rewrite.MemoryPlanStore) {
	t.Helper()
	if fake == nil {
		fake = &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady}
	}
	store := rewrite.NewMemoryPlanStore()
	svc := rewrite.NewService(store, fake, intgraph.NewPathMutex())
	return svc, store
}

func TestServicePlan_HappyPath(t *testing.T) {
	svc, store := newSvc(t, nil)
	out, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "/abs/proj",
		Operations: []rewrite.Operation{
			{FileMove: &rewrite.FileMove{FromPath: "./src/a.ts", ToPath: "src/b.ts"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if out.PlanID == "" || len(out.NormalizedOperations) != 1 {
		t.Errorf("unexpected output: %+v", out)
	}
	if _, err := store.Get("/abs/proj", out.PlanID); err != nil {
		t.Errorf("plan not persisted: %v", err)
	}
}

func TestServicePlan_RejectsRelativeScenarioPath(t *testing.T) {
	svc, _ := newSvc(t, nil)
	_, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "relative/path",
		Operations:   []rewrite.Operation{{FileMove: &rewrite.FileMove{FromPath: "a", ToPath: "b"}}},
	})
	var re rewrite.RewriteError
	if !errors.As(err, &re) || re.Kind != rewrite.RewriteErrorInvalidInput {
		t.Errorf("want RewriteErrorInvalidInput, got %v", err)
	}
}

func TestServicePlan_RejectsEmptyOps(t *testing.T) {
	svc, _ := newSvc(t, nil)
	_, err := svc.Plan(context.Background(), rewrite.PlanInput{ScenarioPath: "/abs/p", Operations: nil})
	var re rewrite.RewriteError
	if !errors.As(err, &re) || re.Kind != rewrite.RewriteErrorInvalidInput {
		t.Errorf("want RewriteErrorInvalidInput, got %v", err)
	}
}

// TestServiceApply_RejectsNonCanonicalStatus pins the D4 contract: the
// sidecar emits exactly "OPERATION_STATUS_OK"; any other spelling
// (including the empty string or a legacy "ok") is treated as a failure,
// never silently coerced to OK.
func TestServiceApply_RejectsNonCanonicalStatus(t *testing.T) {
	for _, bad := range []string{"ok", "OK", "", "OPERATION_STATUS_WEIRD"} {
		bad := bad
		fake := &sidecarmocks.FakeSidecarClient{
			StatusValue: sidecar.StatusReady,
			RewriteApplyFn: func(ctx context.Context, p string, ops []sidecar.Operation) ([]sidecar.OperationResult, error) {
				out := make([]sidecar.OperationResult, len(ops))
				for i := range ops {
					out[i] = sidecar.OperationResult{Status: bad}
				}
				return out, nil
			},
		}
		svc, _ := newSvc(t, fake)
		planOut, err := svc.Plan(context.Background(), rewrite.PlanInput{
			ScenarioPath: "/abs/proj",
			Operations:   []rewrite.Operation{{FileMove: &rewrite.FileMove{FromPath: "a.ts", ToPath: "b.ts"}}},
		})
		if err != nil {
			t.Fatalf("Plan: %v", err)
		}
		out, err := svc.Apply(context.Background(), rewrite.ApplyInput{ScenarioPath: "/abs/proj", PlanID: planOut.PlanID})
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if out.Results[0].Status != rewrite.StatusFailed {
			t.Errorf("status %q should normalize to FAILED, got %q", bad, out.Results[0].Status)
		}
	}
}

func TestServiceApply_HappyPath(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		RewriteApplyFn: func(ctx context.Context, p string, ops []sidecar.Operation) ([]sidecar.OperationResult, error) {
			out := make([]sidecar.OperationResult, len(ops))
			for i := range ops {
				out[i] = sidecar.OperationResult{Status: "OPERATION_STATUS_OK"}
			}
			return out, nil
		},
	}
	svc, _ := newSvc(t, fake)
	planOut, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "/abs/proj",
		Operations: []rewrite.Operation{
			{FileMove: &rewrite.FileMove{FromPath: "a.ts", ToPath: "b.ts"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	out, err := svc.Apply(context.Background(), rewrite.ApplyInput{
		ScenarioPath: "/abs/proj",
		PlanID:       planOut.PlanID,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.DryRun {
		t.Errorf("DryRun should be false for normal apply")
	}
	if len(out.Results) != 1 || out.Results[0].Status != rewrite.StatusOK {
		t.Errorf("unexpected results: %+v", out.Results)
	}
	if fake.RewriteApplyCalls != 1 {
		t.Errorf("expected 1 sidecar call, got %d", fake.RewriteApplyCalls)
	}
}

func TestServiceApply_PlanNotFound(t *testing.T) {
	svc, _ := newSvc(t, nil)
	_, err := svc.Apply(context.Background(), rewrite.ApplyInput{
		ScenarioPath: "/abs/proj",
		PlanID:       "no-such-plan",
	})
	var re rewrite.RewriteError
	if !errors.As(err, &re) || re.Kind != rewrite.RewriteErrorPlanNotFound {
		t.Errorf("want RewriteErrorPlanNotFound, got %v", err)
	}
}

func TestServiceApply_PlanScenarioMismatch(t *testing.T) {
	svc, _ := newSvc(t, nil)
	planOut, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "/abs/one",
		Operations:   []rewrite.Operation{{FileMove: &rewrite.FileMove{FromPath: "a", ToPath: "b"}}},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// Try to apply against a different scenario_path — must surface
	// as PlanNotFound (the (scenario_path, plan_id) composite is the
	// store's lookup key).
	_, err = svc.Apply(context.Background(), rewrite.ApplyInput{
		ScenarioPath: "/abs/two",
		PlanID:       planOut.PlanID,
	})
	var re rewrite.RewriteError
	if !errors.As(err, &re) || re.Kind != rewrite.RewriteErrorPlanNotFound {
		t.Errorf("want RewriteErrorPlanNotFound, got %v", err)
	}
}

func TestServiceApply_SidecarUnavailable(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusUnhealthy}
	svc, _ := newSvc(t, fake)
	// Even an unhealthy sidecar accepts plan(), since plan never talks
	// to the sidecar.
	planOut, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "/abs/proj",
		Operations:   []rewrite.Operation{{FileMove: &rewrite.FileMove{FromPath: "a", ToPath: "b"}}},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	_, err = svc.Apply(context.Background(), rewrite.ApplyInput{
		ScenarioPath: "/abs/proj",
		PlanID:       planOut.PlanID,
	})
	var re rewrite.RewriteError
	if !errors.As(err, &re) || re.Kind != rewrite.RewriteErrorSidecarUnavailable {
		t.Errorf("want RewriteErrorSidecarUnavailable, got %v", err)
	}
}
