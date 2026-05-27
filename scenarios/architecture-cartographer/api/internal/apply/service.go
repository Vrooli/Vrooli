package apply

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/suppressions"
)

// Service is the application-layer surface for apply operations. v0.1
// implements PlanApply (deterministic), ListApplyHistory (empty),
// GetBuildBaseline (empty). RunApply returns ErrApplyUnimplemented.
// WriteSuppression is the only mutating path that lands in v0.1 — it writes
// a safe, non-destructive in-repo marker, distinct from the deferred
// destructive file-moving execution.
type Service interface {
	PlanApply(ctx context.Context, in PlanInput) (Plan, bool, error)
	RunApply(ctx context.Context, planID string, ack bool) (ApplyRun, error)
	ListApplyHistory(ctx context.Context, f ListRunsFilter) (RunPage, error)
	GetBuildBaseline(ctx context.Context, scenario string) (BuildBaseline, error)
	WriteSuppression(ctx context.Context, in SuppressionInput) (SuppressionResult, error)
}

// SuppressionInput resolves which marker to write where.
type SuppressionInput struct {
	Scenario string
	// File is the scenario-relative source file to mark.
	File string
	// ID is the detector/type/subtype the marker sanctions.
	ID string
	// Reason is the mandatory rationale.
	Reason string
	// Expires is an optional condition (e.g. "until:2026-12-31").
	Expires string
	// Line is an optional 1-based insertion point; 0 appends at EOF.
	Line int
}

// SuppressionResult reports where the marker landed.
type SuppressionResult struct {
	File   string
	Line   int
	Marker string
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
	repo          Repository
	conflicts     ConflictLister
	recipes       *RecipeRegistry
	markerWriter  suppressions.Writer
	markerLocator suppressions.Locator
}

// Option customizes the apply service.
type Option func(*service)

// WithSuppressionWriter wires the safe marker-write path. Without it,
// WriteSuppression returns a typed unconfigured error.
func WithSuppressionWriter(w suppressions.Writer, loc suppressions.Locator) Option {
	return func(s *service) {
		s.markerWriter = w
		s.markerLocator = loc
	}
}

func NewService(repo Repository, conflicts ConflictLister, recipes *RecipeRegistry, opts ...Option) Service {
	s := &service{repo: repo, conflicts: conflicts, recipes: recipes}
	for _, o := range opts {
		o(s)
	}
	return s
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

// WriteSuppression writes a durable `// arch:allow` marker into the named
// source file, sanctioning the given finding id as intentional. This is the
// safe write path: it only inserts a comment, never moves or rewrites code.
func (s *service) WriteSuppression(_ context.Context, in SuppressionInput) (SuppressionResult, error) {
	if s.markerWriter == nil || s.markerLocator == nil {
		return SuppressionResult{}, ErrSuppressionUnconfigured{}
	}
	scenario := strings.TrimSpace(in.Scenario)
	file := strings.TrimSpace(in.File)
	id := strings.TrimSpace(in.ID)
	reason := strings.TrimSpace(in.Reason)
	if scenario == "" {
		return SuppressionResult{}, ErrInvalidPlanRequest{Field: "scenario", Reason: "required"}
	}
	if file == "" {
		return SuppressionResult{}, ErrInvalidPlanRequest{Field: "file", Reason: "required"}
	}
	if id == "" {
		return SuppressionResult{}, ErrInvalidPlanRequest{Field: "id", Reason: "required"}
	}
	if reason == "" {
		return SuppressionResult{}, ErrInvalidPlanRequest{Field: "reason", Reason: "required (markers must explain why)"}
	}
	// Reject path escapes; the marker must land inside the scenario tree.
	cleaned := filepath.Clean(file)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return SuppressionResult{}, ErrInvalidPlanRequest{Field: "file", Reason: "must be a scenario-relative path"}
	}
	dir, err := s.markerLocator.Locate(scenario)
	if err != nil {
		return SuppressionResult{}, err
	}
	abs := filepath.Join(dir, cleaned)
	marker := suppressions.Marker{ID: id, Reason: reason, Expires: strings.TrimSpace(in.Expires)}
	if err := s.markerWriter.WriteMarker(abs, in.Line, marker); err != nil {
		return SuppressionResult{}, fmt.Errorf("write suppression marker: %w", err)
	}
	return SuppressionResult{File: cleaned, Line: in.Line, Marker: suppressions.Format(cleaned, marker)}, nil
}
