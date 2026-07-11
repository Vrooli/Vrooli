package permissionpolicy

import (
	"context"
	"errors"
	"sort"
	"time"

	"agent-manager/internal/domain"
)

var ErrNoActiveCatalog = errors.New("no active permission policy catalog")

// AuditStore retains only Agent Manager's reconcile evidence. Implementations
// must not retain native resource configuration documents or their contents.
// seam: AuditStore keeps permission reconcile outcomes independently of native files.
type AuditStore interface {
	RecordReconcile(context.Context, ReconcileResult) error
	LastReconcile(context.Context) (*ReconcileResult, error)
}

// Service owns the global permission-policy workflow. Native translation and
// writes remain behind Projector, while this service owns the deterministic
// aggregate result and audit evidence.
type Service struct {
	state     *State
	planner   *AggregatePlanner
	projector Projector
	audit     AuditStore
	now       func() time.Time
}

func NewService(state *State, projector Projector, audit AuditStore) *Service {
	return newService(state, projector, audit, time.Now)
}

func newService(state *State, projector Projector, audit AuditStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		state:     state,
		planner:   NewAggregatePlanner(projector),
		projector: projector,
		audit:     audit,
		now:       now,
	}
}

func (s *Service) Plan(ctx context.Context) (AggregatePlan, error) {
	if s == nil || s.state == nil {
		return AggregatePlan{}, ErrNoActiveCatalog
	}
	revision := s.state.Active()
	if revision == nil {
		return AggregatePlan{}, ErrNoActiveCatalog
	}
	return s.planner.Plan(ctx, revision)
}

// Reconcile applies the active declared catalog in runner/scope order. A
// result is always returned after projection began, including partial failure,
// so callers cannot mistake an incomplete multi-resource operation for a
// global success.
func (s *Service) Reconcile(ctx context.Context, explicitlyAuthorized bool) (ReconcileResult, error) {
	// INVARIANT: permissionReconcileRequiresExplicitAuthorization
	if !explicitlyAuthorized {
		return ReconcileResult{}, ErrAuthorizationRequired
	}
	if s == nil || s.state == nil || s.projector == nil {
		return ReconcileResult{}, ErrNoActiveCatalog
	}
	revision := s.state.Active()
	if revision == nil || revision.Catalog() == nil {
		return ReconcileResult{}, ErrNoActiveCatalog
	}

	startedAt := s.now().UTC()
	result := ReconcileResult{
		CatalogDigest:            revision.Digest(),
		StartedAt:                startedAt,
		ExplicitlyAuthorized:     true,
		HardEnforcementSatisfied: true,
		Resources:                []ResourcePlan{},
	}
	catalog := revision.Catalog()
	runners := append([]domain.RunnerType(nil), defaultRunners...)
	sort.Slice(runners, func(i, j int) bool { return runners[i] < runners[j] })
	enforcingScopes := make(map[string]bool, len(catalog.Scopes()))

	for _, scope := range catalog.Scopes() {
		document, err := catalog.ResourceDocument(scope)
		if err != nil {
			return ReconcileResult{}, err
		}
		for _, runner := range runners {
			entry := ResourcePlan{
				Runner:              runner,
				Scope:               scope,
				Changes:             []string{},
				NativePaths:         []string{},
				UnsupportedMatchers: []Matcher{},
			}
			projection, projectionErr := s.projector.Reconcile(ctx, ProjectionRequest{Runner: runner, Document: document}, true)
			if projectionErr != nil {
				entry.Error = projectionErr.Error()
				if errors.Is(projectionErr, ErrResourceUnavailable) {
					entry.Status = "unavailable"
				} else {
					entry.Status = "failed"
					entry.Installed = true
				}
				result.Resources = append(result.Resources, entry)
				continue
			}
			entry.Installed = true
			entry.Status = "reconciled"
			entry.DesiredDigest = projection.DesiredDigest
			entry.DesiredFingerprint = projection.DesiredFingerprint
			entry.LiveFingerprint = projection.LiveFingerprint
			entry.Drift = projection.Drift
			entry.Changes = append([]string(nil), projection.Changes...)
			entry.NativePaths = append([]string(nil), projection.NativePaths...)
			entry.Enforcement = projection.Enforcement
			if projection.Enforcement.Permissions == "native" || projection.Enforcement.Permissions == "hook_backed" {
				enforcingScopes[scope] = true
			}
			result.Resources = append(result.Resources, entry)
		}
	}
	for _, rule := range catalog.Rules {
		if rule.RequiresHardEnforcement && !enforcingScopes[rule.TargetScope] {
			result.HardEnforcementSatisfied = false
			result.MissingHardEnforcementRuleIDs = append(result.MissingHardEnforcementRuleIDs, rule.ID)
		}
	}
	sort.Strings(result.MissingHardEnforcementRuleIDs)
	result.FinishedAt = s.now().UTC()
	// INVARIANT: permissionReconcileNeverClaimsPartialSuccess
	result.Success = result.HardEnforcementSatisfied && allResourcesReconciled(result.Resources)

	if s.audit != nil {
		if err := s.audit.RecordReconcile(ctx, result); err != nil {
			return *result.Clone(), err
		}
	}
	return *result.Clone(), nil
}

func (s *Service) LastReconcile(ctx context.Context) (*ReconcileResult, error) {
	if s == nil || s.audit == nil {
		return nil, nil
	}
	result, err := s.audit.LastReconcile(ctx)
	if err != nil || result == nil {
		return result, err
	}
	return result.Clone(), nil
}

func allResourcesReconciled(resources []ResourcePlan) bool {
	for _, resource := range resources {
		if resource.Status != "reconciled" {
			return false
		}
	}
	return true
}

// ReconcileResult is the durable multi-resource mutation evidence. It stores
// resource-reported metadata, never a copy of any native configuration file.
type ReconcileResult struct {
	CatalogDigest                 string         `json:"catalogDigest"`
	StartedAt                     time.Time      `json:"startedAt"`
	FinishedAt                    time.Time      `json:"finishedAt"`
	ExplicitlyAuthorized          bool           `json:"explicitlyAuthorized"`
	Success                       bool           `json:"success"`
	HardEnforcementSatisfied      bool           `json:"hardEnforcementSatisfied"`
	MissingHardEnforcementRuleIDs []string       `json:"missingHardEnforcementRuleIds,omitempty"`
	Resources                     []ResourcePlan `json:"resources"`
}

func (r ReconcileResult) Clone() *ReconcileResult {
	clone := r
	clone.MissingHardEnforcementRuleIDs = append([]string(nil), r.MissingHardEnforcementRuleIDs...)
	clone.Resources = make([]ResourcePlan, len(r.Resources))
	for index, resource := range r.Resources {
		clone.Resources[index] = cloneResourcePlan(resource)
	}
	return &clone
}

func cloneResourcePlan(plan ResourcePlan) ResourcePlan {
	clone := plan
	clone.Changes = append([]string(nil), plan.Changes...)
	clone.NativePaths = append([]string(nil), plan.NativePaths...)
	clone.UnsupportedMatchers = append([]Matcher(nil), plan.UnsupportedMatchers...)
	clone.Enforcement.Caveats = append([]string(nil), plan.Enforcement.Caveats...)
	return clone
}
