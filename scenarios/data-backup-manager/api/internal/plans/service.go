package plans

import (
	"context"
	"strings"
)

// defaultListLimit caps List when the caller passes 0.
const defaultListLimit = 100

// Service is the application surface the plans handlers depend on. It owns
// validation and CRUD delegation to the repository.
type Service interface {
	// Create validates and persists a new plan. ErrInvalidPlan on validation
	// failure.
	Create(ctx context.Context, in CreateInput) (Plan, error)

	// Get returns a plan by id. ErrPlanNotFound propagates verbatim.
	Get(ctx context.Context, id string) (Plan, error)

	// List returns plans (all fields including membership). limit <= 0 uses the
	// default.
	List(ctx context.Context, limit int) ([]Plan, error)

	// Update replaces the plan's fields and membership lists. Returns the
	// updated plan. ErrPlanNotFound if the id doesn't exist.
	Update(ctx context.Context, in UpdateInput) (Plan, error)

	// Delete removes a plan by id. Returns whether a row was removed.
	Delete(ctx context.Context, id string) (bool, error)

	// SchedulablePlans returns all enabled plans for the scheduler. This
	// satisfies the scheduler.PlanSource seam without the scheduler importing
	// this package concretely.
	SchedulablePlans(ctx context.Context) ([]SchedulablePlan, error)
}

// SchedulablePlan is a narrow projection of Plan used by the scheduler seam.
type SchedulablePlan struct {
	ID       string
	Schedule string
	Enabled  bool
}

type service struct {
	repo Repository
}

// NewService constructs the production Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Plan, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Plan{}, ErrInvalidPlan{Field: "name", Reason: "required"}
	}
	if len(in.TargetIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "target_ids", Reason: "at least one required"}
	}
	if len(in.DestinationIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "destination_ids", Reason: "at least one required"}
	}

	p := Plan{
		Name:           name,
		TargetIDs:      in.TargetIDs,
		DestinationIDs: in.DestinationIDs,
		Schedule:       in.Schedule,
		KeepLatest:     in.KeepLatest,
		Enabled:        true, // default true on create
	}
	// Honour explicit false only if caller set it; the proto bool defaults to
	// false but the domain contract says "default true on create". The caller
	// must set the field explicitly for false to mean "disabled". For v1 we
	// accept whatever the caller provides and default to true when the boolean
	// was not explicitly set (proto can't distinguish, so we always default).
	// The simplest safe approach: always start enabled=true, then flip if the
	// CreateInput explicitly carries false. Since proto bool zero-value is false
	// and we cannot distinguish "not set" from "set to false" in a bool field,
	// we follow the proto comment "Optional; defaults to true when unset on
	// create" by defaulting to true unconditionally on the service layer.
	_ = in.Enabled // v1: ignore caller's enabled, always create as enabled=true

	return s.repo.Create(ctx, p)
}

func (s *service) Get(ctx context.Context, id string) (Plan, error) {
	if strings.TrimSpace(id) == "" {
		return Plan{}, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, limit int) ([]Plan, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	return s.repo.List(ctx, limit)
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Plan, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Plan{}, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Plan{}, ErrInvalidPlan{Field: "name", Reason: "required"}
	}
	if len(in.TargetIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "target_ids", Reason: "at least one required"}
	}
	if len(in.DestinationIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "destination_ids", Reason: "at least one required"}
	}

	p := Plan{
		ID:             in.ID,
		Name:           name,
		TargetIDs:      in.TargetIDs,
		DestinationIDs: in.DestinationIDs,
		Schedule:       in.Schedule,
		KeepLatest:     in.KeepLatest,
		Enabled:        in.Enabled,
	}
	return s.repo.Update(ctx, p)
}

func (s *service) Delete(ctx context.Context, id string) (bool, error) {
	if strings.TrimSpace(id) == "" {
		return false, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) SchedulablePlans(ctx context.Context) ([]SchedulablePlan, error) {
	plans, err := s.repo.List(ctx, defaultListLimit)
	if err != nil {
		return nil, err
	}
	out := make([]SchedulablePlan, 0, len(plans))
	for _, p := range plans {
		out = append(out, SchedulablePlan{
			ID:       p.ID,
			Schedule: p.Schedule,
			Enabled:  p.Enabled,
		})
	}
	return out, nil
}
