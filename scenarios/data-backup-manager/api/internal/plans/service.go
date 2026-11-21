package plans

import (
	"context"
	"fmt"
	"sort"
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
	catalog      TargetCatalog
}

// TargetCatalog lists every registered target id. A plan with a target selector
// reads its membership from it on every read.
type TargetCatalog interface {
	RegisteredTargetIDs(ctx context.Context) ([]string, error)
}

// WithTargetCatalog lets svc resolve selector plans. The composition root wires
// it once; a service without a catalog returns a selector plan with no targets,
// which Create and Update refuse to save.
func WithTargetCatalog(svc Service, catalog TargetCatalog) Service {
	if s, ok := svc.(*service); ok {
		s.catalog = catalog
	}
	return svc
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
	// Recovery evidence is part of every enabled protection plan by default.
	// Callers can explicitly choose a different positive cadence; an empty
	// create value gets the weekly owner policy.
	if strings.TrimSpace(in.RecoveryDrillSchedule) == "" {
		in.RecoveryDrillSchedule = DefaultRecoveryDrillSchedule
	}
	if in.KeepHourly+in.KeepDaily+in.KeepWeekly+in.KeepMonthly+in.KeepAnnual == 0 {
		if in.KeepLatest == 0 {
			in.KeepLatest = 2
		}
		in.KeepHourly, in.KeepDaily, in.KeepWeekly, in.KeepMonthly, in.KeepAnnual = 24, 7, 4, 6, 2
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Plan{}, ErrInvalidPlan{Field: "name", Reason: "required"}
	}
	selector, excluded, err := normalizeSelector(in.TargetSelector, in.ExcludedTargetIDs)
	if err != nil {
		return Plan{}, err
	}
	if selector != SelectNone {
		in.TargetIDs = nil
	}
	if selector == SelectNone && len(in.TargetIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "target_ids", Reason: "at least one required (or a target selector)"}
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
	if selector != SelectNone && tier != TierFullPrimary {
		return Plan{}, ErrInvalidPlan{Field: "protection_tier", Reason: "a target selector is only valid for full_primary; critical plans name their approved targets"}
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
		Name:           name,
		TargetIDs:      in.TargetIDs,
		DestinationIDs: in.DestinationIDs,
		Schedule:       in.Schedule,
		KeepLatest:     in.KeepLatest,
		KeepHourly:     in.KeepHourly, KeepDaily: in.KeepDaily, KeepWeekly: in.KeepWeekly, KeepMonthly: in.KeepMonthly, KeepAnnual: in.KeepAnnual,
		TargetSelector: selector, ExcludedTargetIDs: excluded,
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
	return s.expose(ctx, saved)
}

const DefaultRecoveryDrillSchedule = "168h"

const ManualRecoveryDrillSchedule = "manual"

func (s *service) Get(ctx context.Context, id string) (Plan, error) {
	if strings.TrimSpace(id) == "" {
		return Plan{}, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	return s.expose(ctx, p)
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
		if list[i], err = s.expose(ctx, list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Plan, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Plan{}, ErrInvalidPlan{Field: "id", Reason: "required"}
	}
	if strings.TrimSpace(in.RecoveryDrillSchedule) == "" {
		in.RecoveryDrillSchedule = DefaultRecoveryDrillSchedule
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Plan{}, ErrInvalidPlan{Field: "name", Reason: "required"}
	}
	selector, excluded, err := normalizeSelector(in.TargetSelector, in.ExcludedTargetIDs)
	if err != nil {
		return Plan{}, err
	}
	if selector != SelectNone {
		in.TargetIDs = nil
	}
	if selector == SelectNone && len(in.TargetIDs) == 0 {
		return Plan{}, ErrInvalidPlan{Field: "target_ids", Reason: "at least one required (or a target selector)"}
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
	if selector != SelectNone && tier != TierFullPrimary {
		return Plan{}, ErrInvalidPlan{Field: "protection_tier", Reason: "a target selector is only valid for full_primary; critical plans name their approved targets"}
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
		ID:             in.ID,
		Name:           name,
		TargetIDs:      in.TargetIDs,
		DestinationIDs: in.DestinationIDs,
		Schedule:       in.Schedule,
		KeepLatest:     in.KeepLatest,
		KeepHourly:     in.KeepHourly, KeepDaily: in.KeepDaily, KeepWeekly: in.KeepWeekly, KeepMonthly: in.KeepMonthly, KeepAnnual: in.KeepAnnual,
		TargetSelector: selector, ExcludedTargetIDs: excluded,
		Enabled:               in.Enabled,
		ProtectionTier:        tier,
		RecoveryDrillSchedule: in.RecoveryDrillSchedule,
	}
	saved, err := s.repo.Update(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	return s.expose(ctx, saved)
}

// normalizeSelector validates a plan's target selector and cleans its exclusions.
func normalizeSelector(selector TargetSelector, excluded []string) (TargetSelector, []string, error) {
	switch selector {
	case SelectNone, SelectRegistered:
	default:
		return "", nil, ErrInvalidPlan{Field: "target_selector", Reason: fmt.Sprintf("unknown selector %q (use \"registered\" or leave it empty)", selector)}
	}
	seen := map[string]bool{}
	var clean []string
	for _, id := range excluded {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			clean = append(clean, id)
		}
	}
	if selector == SelectNone && len(clean) > 0 {
		return "", nil, ErrInvalidPlan{Field: "excluded_target_ids", Reason: "only valid with a target selector"}
	}
	sort.Strings(clean)
	return selector, clean, nil
}

// expose is every read of a plan: it resolves a selector into the plan's
// targets, then assesses the destination topology.
func (s *service) expose(ctx context.Context, p Plan) (Plan, error) {
	p, err := s.resolve(ctx, p)
	if err != nil {
		return Plan{}, err
	}
	return s.decorateTopology(ctx, p), nil
}

// resolve turns a selector into the targets it names right now: the catalog's
// registered targets minus the plan's exclusions. An explicit plan is unchanged.
func (s *service) resolve(ctx context.Context, p Plan) (Plan, error) {
	if p.TargetSelector != SelectRegistered {
		return p, nil
	}
	p.TargetIDs = nil
	if s.catalog == nil {
		return p, nil
	}
	ids, err := s.catalog.RegisteredTargetIDs(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("resolve targets of plan %q: %w", p.Name, err)
	}
	skip := make(map[string]bool, len(p.ExcludedTargetIDs))
	for _, id := range p.ExcludedTargetIDs {
		skip[id] = true
	}
	for _, id := range ids {
		if !skip[id] {
			p.TargetIDs = append(p.TargetIDs, id)
		}
	}
	sort.Strings(p.TargetIDs)
	return p, nil
}

func (s *service) decorateTopology(ctx context.Context, p Plan) Plan {
	reporter, ok := s.destinations.(CriticalDestinationReporter)
	if !ok {
		return p
	}
	report, err := reporter.Assess(ctx, p.ProtectionTier, p.TargetIDs, p.DestinationIDs)
	if err != nil {
		p.DestinationsPhysicallyIndependent = false
		p.SharedRiskWarnings = []string{"destination placement could not be assessed; off-host placement is not proven (this says nothing about drive-loss protection)"}
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
// never block the plan guard; they require the separate verified Notification
// Hub review before DBM registers them.
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
		if p, err = s.resolve(ctx, p); err != nil {
			return nil, err
		}
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
	if raw == ManualRecoveryDrillSchedule {
		return nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return ErrInvalidPlan{Field: "recovery_drill_schedule", Reason: "must be empty or a positive Go duration (for example 168h)"}
	}
	return nil
}
