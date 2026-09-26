package plans

import (
	"context"
	"fmt"
	"strings"
	"time"
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

// SchedulablePlan is a narrow projection of Plan used by the scheduler seam and
// the next-scheduled-fire rollup (TargetIDs lets the rollup attribute a plan's
// next fire to each of its targets).
type SchedulablePlan struct {
	ID             string
	Schedule       string
	DrillSchedule  string
	Enabled        bool
	TargetIDs      []string
	DestinationIDs []string
}

type service struct {
	repo         Repository
	guard        CoverageGuard
	critical     CriticalTargetPolicy
	destinations CriticalDestinationPolicy
}

// NewService constructs the production Service. guard may be nil to disable the
// default-coverage check (the scheduler/run services that never create plans
// pass nil); the API-serving service is wired with a discovery-backed guard.
func NewService(repo Repository, guard CoverageGuard) Service {
	return &service{repo: repo, guard: guard}
}

// NewServiceWithTargetPolicy constructs the API-serving plans service with
// critical-tier validation enabled. NewService remains available for narrow
// tests and legacy composition, but a production composition that exposes
// critical tiers must provide this policy or critical plan creation fails
// closed.
func NewServiceWithTargetPolicy(repo Repository, guard CoverageGuard, critical CriticalTargetPolicy) Service {
	return &service{repo: repo, guard: guard, critical: critical}
}

// NewServiceWithPolicies constructs the API service with both critical target
// classification and destination independence validation.
func NewServiceWithPolicies(repo Repository, guard CoverageGuard, critical CriticalTargetPolicy, destinations CriticalDestinationPolicy) Service {
	return &service{repo: repo, guard: guard, critical: critical, destinations: destinations}
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
	if err := s.checkCoverage(ctx, in.AllowIncompleteCoverage); err != nil {
		return Plan{}, err
	}
	tier := in.ProtectionTier
	if tier == "" {
		tier = TierFullPrimary
	}
	if !validTier(tier) {
		return Plan{}, ErrInvalidPlan{Field: "protection_tier", Reason: "must be full_primary, critical_primary, or critical_secondary"}
	}
	if err := s.validateTier(ctx, tier, in.TargetIDs); err != nil {
		return Plan{}, err
	}
	if s.destinations != nil {
		if err := s.destinations.Validate(ctx, tier, in.TargetIDs, in.DestinationIDs); err != nil {
			return Plan{}, err
		}
	}
	if err := validateDrillSchedule(in.RecoveryDrillSchedule); err != nil {
		return Plan{}, err
	}

	p := Plan{
		Name:                  name,
		TargetIDs:             in.TargetIDs,
		DestinationIDs:        in.DestinationIDs,
		Schedule:              in.Schedule,
		KeepLatest:            in.KeepLatest,
		Enabled:               true, // default true on create
		ProtectionTier:        tier,
		RecoveryDrillSchedule: in.RecoveryDrillSchedule,
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

	saved, err := s.repo.Create(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	return s.decorateTopology(ctx, saved), nil
}

func (s *service) Get(ctx context.Context, id string) (Plan, error) {
	if strings.TrimSpace(id) == "" {
		return Plan{}, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	return s.decorateTopology(ctx, p), nil
}

func (s *service) List(ctx context.Context, limit int) ([]Plan, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	list, err := s.repo.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i] = s.decorateTopology(ctx, list[i])
	}
	return list, nil
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
	if err := s.checkCoverage(ctx, in.AllowIncompleteCoverage); err != nil {
		return Plan{}, err
	}
	tier := in.ProtectionTier
	if tier == "" {
		tier = TierFullPrimary
	}
	if !validTier(tier) {
		return Plan{}, ErrInvalidPlan{Field: "protection_tier", Reason: "must be full_primary, critical_primary, or critical_secondary"}
	}
	if err := s.validateTier(ctx, tier, in.TargetIDs); err != nil {
		return Plan{}, err
	}
	if s.destinations != nil {
		if err := s.destinations.Validate(ctx, tier, in.TargetIDs, in.DestinationIDs); err != nil {
			return Plan{}, err
		}
	}
	if err := validateDrillSchedule(in.RecoveryDrillSchedule); err != nil {
		return Plan{}, err
	}

	p := Plan{
		ID:                    in.ID,
		Name:                  name,
		TargetIDs:             in.TargetIDs,
		DestinationIDs:        in.DestinationIDs,
		Schedule:              in.Schedule,
		KeepLatest:            in.KeepLatest,
		Enabled:               in.Enabled,
		ProtectionTier:        tier,
		RecoveryDrillSchedule: in.RecoveryDrillSchedule,
	}
	saved, err := s.repo.Update(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	return s.decorateTopology(ctx, saved), nil
}

func (s *service) decorateTopology(ctx context.Context, p Plan) Plan {
	reporter, ok := s.destinations.(CriticalDestinationReporter)
	if !ok {
		return p
	}
	report, err := reporter.Assess(ctx, p.ProtectionTier, p.TargetIDs, p.DestinationIDs)
	if err != nil {
		p.DestinationsPhysicallyIndependent = false
		p.SharedRiskWarnings = []string{"destination topology assessment unavailable; physical independence is not proven"}
		return p
	}
	p.DestinationsPhysicallyIndependent = report.PhysicallyIndependent
	p.SharedRiskWarnings = append([]string(nil), report.Warnings...)
	return p
}

func validTier(tier ProtectionTier) bool {
	return tier == TierFullPrimary || tier == TierCriticalPrimary || tier == TierCriticalSecondary
}

func (s *service) validateTier(ctx context.Context, tier ProtectionTier, targetIDs []string) error {
	if tier == TierFullPrimary {
		return nil
	}
	if s.critical == nil {
		return ErrInvalidPlan{Field: "protection_tier", Reason: "critical-tier validation is unavailable; refuse to create an unvalidated critical plan"}
	}
	for _, id := range targetIDs {
		critical, err := s.critical.IsCritical(ctx, strings.TrimSpace(id))
		if err != nil {
			return ErrInvalidPlan{Field: "target_ids", Reason: fmt.Sprintf("cannot validate critical target %q: %v", id, err)}
		}
		if !critical {
			return ErrInvalidPlan{Field: "target_ids", Reason: fmt.Sprintf("target %q is not approved for critical protection", id)}
		}
	}
	return nil
}

// checkCoverage enforces the default-coverage guard: unless the caller opts out
// (allowIncomplete) or no guard is wired, a plan cannot be created/updated while
// non-sensitive recommended targets remain unregistered. Sensitive suggestions
// never block (they require deliberate opt-in via coverage accept-defaults
// --include-sensitive); the guard returns non-sensitive recommendations only.
func (s *service) checkCoverage(ctx context.Context, allowIncomplete bool) error {
	if allowIncomplete || s.guard == nil {
		return nil
	}
	missing, err := s.guard.UnregisteredDefaultTargets(ctx)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return ErrIncompleteCoverage{Missing: missing}
	}
	return nil
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
			ID:             p.ID,
			Schedule:       p.Schedule,
			DrillSchedule:  p.RecoveryDrillSchedule,
			Enabled:        p.Enabled,
			TargetIDs:      p.TargetIDs,
			DestinationIDs: p.DestinationIDs,
		})
	}
	return out, nil
}

func validateDrillSchedule(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return ErrInvalidPlan{Field: "recovery_drill_schedule", Reason: "must be empty or a positive Go duration (for example 168h)"}
	}
	return nil
}
