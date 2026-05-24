package rewrite_test

import (
	"context"
	"testing"

	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"
)

// TestServiceApply_DryRunSkipsSidecar pins OT-P0-006 / §8.6: a
// dry-run apply must NEVER call the sidecar — not even to check
// readiness. Every operation gets a synthetic OPERATION_STATUS_OK in
// the same order as the plan's normalized operations.
func TestServiceApply_DryRunSkipsSidecar(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		// Deliberately unhealthy: a buggy implementation that checks
		// sidecar status before honoring DryRun would fail here.
		StatusValue: sidecar.StatusUnhealthy,
		RewriteApplyFn: func(ctx context.Context, p string, ops []sidecar.Operation) ([]sidecar.OperationResult, error) {
			t.Fatalf("dry-run must not call sidecar.RewriteApply")
			return nil, nil
		},
	}
	store := rewrite.NewMemoryPlanStore()
	svc := rewrite.NewService(store, fake, intgraph.NewPathMutex())

	ops := []rewrite.Operation{
		{FileMove: &rewrite.FileMove{FromPath: "a.ts", ToPath: "b.ts"}},
		{ImportRewrite: &rewrite.ImportRewrite{OldPath: "./old", NewPath: "./new"}},
	}
	planOut, err := svc.Plan(context.Background(), rewrite.PlanInput{
		ScenarioPath: "/abs/proj",
		Operations:   ops,
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	out, err := svc.Apply(context.Background(), rewrite.ApplyInput{
		ScenarioPath: "/abs/proj",
		PlanID:       planOut.PlanID,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !out.DryRun {
		t.Errorf("DryRun should be true")
	}
	if len(out.Results) != len(planOut.NormalizedOperations) {
		t.Fatalf("expected %d results, got %d", len(planOut.NormalizedOperations), len(out.Results))
	}
	for i, r := range out.Results {
		if r.Status != rewrite.StatusOK {
			t.Errorf("dry-run result %d: status=%q want OK", i, r.Status)
		}
		if r.Message != "" {
			t.Errorf("dry-run result %d: message must be empty, got %q", i, r.Message)
		}
	}
	if fake.RewriteApplyCalls != 0 {
		t.Errorf("sidecar.RewriteApply must not be called on dry-run; got %d calls", fake.RewriteApplyCalls)
	}
}
