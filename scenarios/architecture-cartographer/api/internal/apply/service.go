package apply

import (
	"context"
	"strings"

	"architecture-cartographer/internal/conflicts"
)

// Service is the application-layer surface for apply operations. v0.1
// implements PlanApply (deterministic), ListApplyHistory (empty),
// GetBuildBaseline (empty). RunApply returns ErrApplyUnimplemented.
type Service interface {
	PlanApply(ctx context.Context, in PlanInput) (Plan, bool, error)
	RunApply(ctx context.Context, planID string, ack bool) (ApplyRun, error)
	ListApplyHistory(ctx context.Context, f ListRunsFilter) (RunPage, error)
	GetBuildBaseline(ctx context.Context, scenario string) (BuildBaseline, error)
}

// PlanInput is the explicit input DTO for PlanApply.
type PlanInput struct {
	Scenario    string
	Domain      string
	ConflictIDs []string
	DryRun      bool
}

// ConflictLister is the seam apply uses to fetch resolved conflicts.
// Production wires conflicts.Service; tests pass a fake.
type ConflictLister interface {
	ListConflicts(ctx context.Context, f conflicts.ListConflictsFilter) (conflicts.ConflictPage, error)
}

type service struct {
	repo      Repository
	conflicts ConflictLister
	recipes   *RecipeRegistry
}

func NewService(repo Repository, conflicts ConflictLister, recipes *RecipeRegistry) Service {
	return &service{repo: repo, conflicts: conflicts, recipes: recipes}
}

var _ Service = (*service)(nil)

func (s *service) PlanApply(ctx context.Context, in PlanInput) (Plan, bool, error) {
	if strings.TrimSpace(in.Scenario) == "" {
		return Plan{}, in.DryRun, ErrInvalidPlanRequest{Field: "scenario", Reason: "required"}
	}
	if strings.TrimSpace(in.Domain) == "" {
		return Plan{}, in.DryRun, ErrInvalidPlanRequest{Field: "domain", Reason: "required"}
	}

	page, err := s.conflicts.ListConflicts(ctx, conflicts.ListConflictsFilter{
		Scenario: in.Scenario,
		Statuses: []conflicts.ResolutionStatus{
			conflicts.ResolutionStatusResolved,
			conflicts.ResolutionStatusValidated,
		},
	})
	if err != nil {
		return Plan{}, in.DryRun, err
	}

	wanted := make(map[string]struct{}, len(in.ConflictIDs))
	for _, id := range in.ConflictIDs {
		wanted[id] = struct{}{}
	}

	plan := Plan{
		Scenario: in.Scenario,
		Domain:   in.Domain,
	}
	for _, c := range page.Conflicts {
		if len(wanted) > 0 {
			if _, ok := wanted[c.ID]; !ok {
				continue
			}
		}
		if c.AssignedDomain != "" && c.AssignedDomain != in.Domain {
			continue
		}
		ops := OperationsFromConflict(c)
		if len(ops) == 0 {
			continue
		}
		plan.Operations = append(plan.Operations, ops...)
		plan.ConflictIDs = append(plan.ConflictIDs, c.ID)
	}

	// Run any recipe extensions (v0.1 registry is empty).
	if s.recipes != nil {
		for _, r := range s.recipes.All() {
			if !r.Applies(ctx, plan) {
				continue
			}
			extra, err := r.Extend(ctx, plan)
			if err != nil {
				return Plan{}, in.DryRun, err
			}
			plan.Operations = append(plan.Operations, extra...)
		}
	}

	if in.DryRun {
		return plan, true, nil
	}
	persisted, err := s.repo.SavePlan(ctx, plan)
	return persisted, false, err
}

func (s *service) RunApply(_ context.Context, planID string, ack bool) (ApplyRun, error) {
	_ = planID
	_ = ack
	return ApplyRun{}, ErrApplyUnimplemented{NextPlan: "architecture-cartographer-apply-execution"}
}

func (s *service) ListApplyHistory(ctx context.Context, f ListRunsFilter) (RunPage, error) {
	return s.repo.ListRuns(ctx, f)
}

func (s *service) GetBuildBaseline(ctx context.Context, scenario string) (BuildBaseline, error) {
	return s.repo.GetBaseline(ctx, scenario)
}
