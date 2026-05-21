package apply_test

import (
	"context"
	"errors"
	"testing"

	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/apply/mocks"
	"architecture-cartographer/internal/conflicts"
)

func TestService_RunApply_ReturnsUnimplemented(t *testing.T) {
	svc := apply.NewService(&mocks.FakeRepository{}, &mocks.FakeConflictLister{}, apply.NewRecipeRegistry())
	_, err := svc.RunApply(context.Background(), "plan-1", true)
	var typed apply.ErrApplyUnimplemented
	if !errors.As(err, &typed) {
		t.Fatalf("want ErrApplyUnimplemented, got %v", err)
	}
}

func TestService_PlanApply_RequiresScenario(t *testing.T) {
	svc := apply.NewService(&mocks.FakeRepository{}, &mocks.FakeConflictLister{}, apply.NewRecipeRegistry())
	_, _, err := svc.PlanApply(context.Background(), apply.PlanInput{Domain: "graph"})
	var typed apply.ErrInvalidPlanRequest
	if !errors.As(err, &typed) || typed.Field != "scenario" {
		t.Fatalf("want ErrInvalidPlanRequest{scenario}, got %v", err)
	}
}

func TestService_PlanApply_GeneratesDeterministicOperations(t *testing.T) {
	lister := &mocks.FakeConflictLister{
		Conflicts: []conflicts.Conflict{
			{
				ID:             "c-1",
				Scenario:       "demo",
				Status:         conflicts.ResolutionStatusResolved,
				AssignedDomain: "graph",
				SuggestedFixes: []conflicts.Fix{
					{ID: "f-1", Kind: conflicts.FixKindMoveFile, Resolver: "mislocated_file"},
				},
			},
			{
				ID:             "c-2",
				Scenario:       "demo",
				Status:         conflicts.ResolutionStatusResolved,
				AssignedDomain: "manifest", // different domain — should be excluded
				SuggestedFixes: []conflicts.Fix{
					{ID: "f-2", Kind: conflicts.FixKindMoveFile, Resolver: "mislocated_file"},
				},
			},
		},
	}
	repo := &mocks.FakeRepository{}
	svc := apply.NewService(repo, lister, apply.NewRecipeRegistry())

	plan, dry, err := svc.PlanApply(context.Background(), apply.PlanInput{
		Scenario: "demo", Domain: "graph",
	})
	if err != nil {
		t.Fatalf("PlanApply: %v", err)
	}
	if dry {
		t.Fatal("dry-run should be false")
	}
	if len(plan.Operations) != 1 {
		t.Fatalf("want 1 op, got %d (%+v)", len(plan.Operations), plan.Operations)
	}
	if plan.Operations[0].Kind != apply.OperationKindMoveFile {
		t.Fatalf("unexpected kind: %s", plan.Operations[0].Kind)
	}
}

func TestService_PlanApply_DryRunDoesNotPersist(t *testing.T) {
	repo := &mocks.FakeRepository{}
	svc := apply.NewService(repo, &mocks.FakeConflictLister{}, apply.NewRecipeRegistry())
	plan, dry, err := svc.PlanApply(context.Background(), apply.PlanInput{
		Scenario: "demo", Domain: "graph", DryRun: true,
	})
	if err != nil {
		t.Fatalf("PlanApply: %v", err)
	}
	if !dry {
		t.Fatal("expected dry=true")
	}
	if plan.ID != "" {
		t.Fatal("dry-run plan should not get persisted id")
	}
	if repo.SaveCalls.Load() != 0 {
		t.Fatal("dry-run should not call SavePlan")
	}
}
