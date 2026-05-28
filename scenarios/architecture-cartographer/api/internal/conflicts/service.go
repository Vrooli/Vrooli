package conflicts

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/suppressions"
)

// Service is the application-layer surface for conflict operations.
type Service interface {
	DetectConflicts(ctx context.Context, in DetectOrchestrationInput) ([]Conflict, error)
	UpsertConflicts(ctx context.Context, scenario string, conflicts []Conflict) ([]Conflict, error)
	ValidateConflicts(ctx context.Context, scenario string) ([]Conflict, bool, error)
	GetConflict(ctx context.Context, id string) (Conflict, error)
	AssignConflict(ctx context.Context, id, domain, note string, dryRun bool) (Conflict, bool, error)
	ResolveConflict(ctx context.Context, id, note string, force, dryRun bool) (Conflict, bool, bool, error)
	ReopenConflict(ctx context.Context, id, note string, dryRun bool) (Conflict, bool, error)
	ListConflicts(ctx context.Context, f ListConflictsFilter) (ConflictPage, error)
	ListDetectors(ctx context.Context) []DetectorDescriptor
	ListResolvers(ctx context.Context) []ResolverDescriptor
}

// DetectOrchestrationInput bundles every input DetectConflicts needs.
// Callers (the Connect handler in production, tests in unit tests)
// load the graph snapshot + derived domain map via their own seams; the
// conflicts service does not reach into other domains.
type DetectOrchestrationInput struct {
	Scenario        string
	Snapshot        graph.GraphSnapshot
	DomainMap       domains.DerivedDomainMap
	IdempotencyKey  string
	VerdictProvider VerdictProvider
	// Suppressions are the active in-repo `// arch:allow` markers for the
	// scenario; matching conflicts are reported as suppressed-with-reason.
	Suppressions []suppressions.Marker
}

// AnalyticsRecorder is the slim seam between conflict state
// transitions and the analytics event log. Production wires an
// adapter over analytics.Service; tests use a fake.
type AnalyticsRecorder interface {
	Record(ctx context.Context, scenario string, kind string, conflictID string, payload map[string]any)
}

type service struct {
	repo      Repository
	detectors *Registry
	resolvers *ResolverRegistry
	recorder  AnalyticsRecorder
}

// NewService constructs the production Service without an analytics
// recorder; state transitions are silent.
func NewService(repo Repository, detectors *Registry, resolvers *ResolverRegistry) Service {
	return &service{repo: repo, detectors: detectors, resolvers: resolvers}
}

// NewServiceWithAnalytics constructs the Service with an analytics
// recorder wired so every state transition emits an event.
func NewServiceWithAnalytics(repo Repository, detectors *Registry, resolvers *ResolverRegistry, recorder AnalyticsRecorder) Service {
	return &service{repo: repo, detectors: detectors, resolvers: resolvers, recorder: recorder}
}

var _ Service = (*service)(nil)

func (s *service) record(ctx context.Context, scenario, kind, conflictID string, payload map[string]any) {
	if s.recorder == nil {
		return
	}
	s.recorder.Record(ctx, scenario, kind, conflictID, payload)
}

// DetectConflicts runs every registered detector against the
// (snapshot, derived domain map) pair, persists the resulting
// conflicts, and emits a conflict_detected analytics event for each new row.
func (s *service) DetectConflicts(ctx context.Context, in DetectOrchestrationInput) ([]Conflict, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return nil, ErrInvalidAssignment{Domain: "", Reason: "scenario is required"}
	}
	if s.detectors == nil {
		return nil, errors.New("no detector registry registered")
	}
	conflicts, err := s.detectors.DetectAll(ctx, DetectInput{
		Scenario:        scenario,
		Snapshot:        in.Snapshot,
		DomainMap:       in.DomainMap,
		VerdictProvider: in.VerdictProvider,
	})
	if err != nil {
		return nil, err
	}
	// Mark conflicts sanctioned by active in-repo markers before persisting,
	// so the suppressed-with-reason state is durable and visible everywhere.
	conflicts = applySuppressions(conflicts, in.Suppressions, in.DomainMap)
	persisted, err := s.UpsertConflicts(ctx, scenario, conflicts)
	if err != nil {
		return nil, err
	}
	for _, c := range persisted {
		s.record(ctx, scenario, "conflict_detected", c.ID, map[string]any{
			"type":       c.Type,
			"severity":   c.Severity,
			"suppressed": c.Suppressed,
		})
	}
	return persisted, nil
}

// ValidateConflicts returns the still-outstanding conflicts (anything
// not in a terminal Resolved/ForceResolved/Committed status) and a
// clean flag (true ↔ zero outstanding of severity≥error).
func (s *service) ValidateConflicts(ctx context.Context, scenario string) ([]Conflict, bool, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, false, ErrInvalidAssignment{Domain: "", Reason: "scenario is required"}
	}
	page, err := s.repo.ListConflicts(ctx, ListConflictsFilter{Scenario: scenario, PageSize: 10000})
	if err != nil {
		return nil, false, err
	}
	var outstanding []Conflict
	clean := true
	for _, c := range page.Conflicts {
		switch c.Status {
		case ResolutionStatusResolved, ResolutionStatusForceResolved, ResolutionStatusCommitted:
			continue
		}
		// A finding sanctioned by an active in-repo marker is intentional,
		// not outstanding — it never blocks cartographer-clean closure.
		if c.Suppressed {
			continue
		}
		outstanding = append(outstanding, c)
		if c.Severity == SeverityError {
			clean = false
		}
	}
	return outstanding, clean, nil
}

func (s *service) UpsertConflicts(ctx context.Context, scenario string, conflicts []Conflict) ([]Conflict, error) {
	out := make([]Conflict, 0, len(conflicts))
	for _, c := range conflicts {
		c.Scenario = scenario
		if c.Status == "" {
			c.Status = ResolutionStatusDetected
		}
		persisted, err := s.repo.UpsertConflict(ctx, c)
		if err != nil {
			return nil, err
		}
		out = append(out, persisted)
	}
	return out, nil
}

func (s *service) GetConflict(ctx context.Context, id string) (Conflict, error) {
	return s.repo.GetConflict(ctx, id)
}

func (s *service) AssignConflict(ctx context.Context, id, domain, note string, dryRun bool) (Conflict, bool, error) {
	if strings.TrimSpace(domain) == "" {
		return Conflict{}, dryRun, ErrInvalidAssignment{Domain: domain, Reason: "required"}
	}
	current, err := s.repo.GetConflict(ctx, id)
	if err != nil {
		return Conflict{}, dryRun, err
	}
	if !canTransition(current.Status, ResolutionStatusAssigned) {
		return current, dryRun, ErrInvalidTransition{From: current.Status, To: ResolutionStatusAssigned}
	}
	if dryRun {
		current.Status = ResolutionStatusAssigned
		current.AssignedDomain = domain
		current.ResolutionNote = note
		return current, true, nil
	}
	updated, err := s.repo.UpdateStatus(ctx, id, ResolutionStatusAssigned, note, domain)
	if err == nil {
		s.record(ctx, updated.Scenario, "conflict_assigned", updated.ID, map[string]any{
			"domain": domain,
			"note":   note,
		})
	}
	return updated, false, err
}

func (s *service) ResolveConflict(ctx context.Context, id, note string, force, dryRun bool) (Conflict, bool, bool, error) {
	current, err := s.repo.GetConflict(ctx, id)
	if err != nil {
		return Conflict{}, dryRun, false, err
	}
	targetStatus := ResolutionStatusResolved
	if force {
		targetStatus = ResolutionStatusForceResolved
	}
	if !canTransition(current.Status, targetStatus) {
		return current, dryRun, false, ErrInvalidTransition{From: current.Status, To: targetStatus}
	}
	// Determine whether the resolution is apply-deferred. Inspect the
	// suggested fixes' resolvers; v0.1 marks every fix that requires
	// apply as deferred so the CLI can communicate the gap.
	applyDeferred := false
	for _, fix := range current.SuggestedFixes {
		if fix.Resolver == "" {
			continue
		}
		resolver := s.resolvers.Lookup(fix.Resolver)
		if resolver == nil {
			continue
		}
		if resolver.RequiresApply() {
			applyDeferred = true
			break
		}
	}
	if dryRun {
		current.Status = targetStatus
		current.ResolutionNote = note
		return current, true, applyDeferred, nil
	}
	updated, err := s.repo.UpdateStatus(ctx, id, targetStatus, note, "")
	if err != nil {
		return Conflict{}, false, applyDeferred, err
	}
	// If a non-deferred resolver exists, invoke it. v0.1 typically
	// returns ErrResolverDeferred for all fixes.
	for _, fix := range updated.SuggestedFixes {
		resolver := s.resolvers.Lookup(fix.Resolver)
		if resolver == nil || resolver.RequiresApply() {
			continue
		}
		if rerr := resolver.Resolve(ctx, updated, fix); rerr != nil {
			var deferred ErrResolverDeferred
			if errors.As(rerr, &deferred) {
				applyDeferred = true
				continue
			}
			return updated, false, applyDeferred, rerr
		}
	}
	kind := "conflict_resolved"
	if force {
		kind = "conflict_force_resolved"
	}
	s.record(ctx, updated.Scenario, kind, updated.ID, map[string]any{
		"force":          force,
		"apply_deferred": applyDeferred,
		"note":           note,
	})
	return updated, false, applyDeferred, nil
}

func (s *service) ReopenConflict(ctx context.Context, id, note string, dryRun bool) (Conflict, bool, error) {
	current, err := s.repo.GetConflict(ctx, id)
	if err != nil {
		return Conflict{}, dryRun, err
	}
	if !canTransition(current.Status, ResolutionStatusDetected) {
		return current, dryRun, ErrInvalidTransition{From: current.Status, To: ResolutionStatusDetected}
	}
	if dryRun {
		current.Status = ResolutionStatusDetected
		current.ResolutionNote = note
		return current, true, nil
	}
	updated, err := s.repo.UpdateStatus(ctx, id, ResolutionStatusDetected, note, "")
	if err == nil {
		s.record(ctx, updated.Scenario, "conflict_reopened", updated.ID, map[string]any{"note": note})
	}
	return updated, false, err
}

func (s *service) ListConflicts(ctx context.Context, f ListConflictsFilter) (ConflictPage, error) {
	return s.repo.ListConflicts(ctx, f)
}

func (s *service) ListDetectors(_ context.Context) []DetectorDescriptor {
	if s.detectors == nil {
		return nil
	}
	return s.detectors.Describe()
}

func (s *service) ListResolvers(_ context.Context) []ResolverDescriptor {
	if s.resolvers == nil {
		return nil
	}
	return s.resolvers.Describe()
}

// canTransition encodes the lifecycle state machine documented in
// docs/concepts/FLOWS.md. Phase 6 backs this with a flow contract via
// flow-verifier; Phase 2 carries the table inline so the service is
// exercisable in tests.
func canTransition(from, to ResolutionStatus) bool {
	allowed := map[ResolutionStatus]map[ResolutionStatus]struct{}{
		ResolutionStatusDetected: {
			ResolutionStatusAssigned:      {},
			ResolutionStatusSplit:         {},
			ResolutionStatusResolved:      {},
			ResolutionStatusForceResolved: {},
		},
		ResolutionStatusAssigned: {
			ResolutionStatusResolved:      {},
			ResolutionStatusSplit:         {},
			ResolutionStatusForceResolved: {},
			ResolutionStatusDetected:      {},
		},
		ResolutionStatusSplit: {
			ResolutionStatusResolved:      {},
			ResolutionStatusForceResolved: {},
		},
		ResolutionStatusResolved: {
			ResolutionStatusValidated: {},
			ResolutionStatusDetected:  {}, // reopen
		},
		ResolutionStatusValidated: {
			ResolutionStatusCommitted: {},
			ResolutionStatusDetected:  {}, // reopen
		},
		ResolutionStatusCommitted: {},
		ResolutionStatusForceResolved: {
			ResolutionStatusDetected: {}, // reopen
		},
	}
	if from == "" {
		from = ResolutionStatusDetected
	}
	t, ok := allowed[from]
	if !ok {
		return false
	}
	_, ok = t[to]
	return ok
}
