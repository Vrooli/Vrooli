package conflicts

import (
	"context"
	"errors"
	"strings"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/suppressions"

	"github.com/google/uuid"
)

// assignStableIDs canonicalizes the identity contract on the detector
// outputs before they reach the repository: ID == StableID (deterministic
// content hash) so ON CONFLICT(id) dedupes across runs; InstanceID is a
// fresh UUID per run for log correlation. Multiple detector emissions
// that hash to the same stable_id within one run collapse to the first,
// preserving evidence order.
func assignStableIDs(in []Conflict) []Conflict {
	seen := make(map[string]struct{}, len(in))
	out := make([]Conflict, 0, len(in))
	for _, c := range in {
		if c.StableID == "" {
			c.StableID = StableID(c)
		}
		c.ID = c.StableID
		if c.InstanceID == "" {
			c.InstanceID = uuid.NewString()
		}
		if _, dup := seen[c.StableID]; dup {
			continue
		}
		seen[c.StableID] = struct{}{}
		out = append(out, c)
	}
	return out
}

// Service is the application-layer surface for conflict operations.
type Service interface {
	DetectConflicts(ctx context.Context, in DetectOrchestrationInput) ([]Conflict, error)
	UpsertConflicts(ctx context.Context, scenario string, conflicts []Conflict) ([]Conflict, error)
	ValidateConflicts(ctx context.Context, scenario string) ([]Conflict, bool, error)
	GetConflict(ctx context.Context, id string) (Conflict, error)
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
	ClaimProvider   ClaimProvider
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
	repo          Repository
	detectors     *Registry
	resolvers     *ResolverRegistry
	recorder      AnalyticsRecorder
	claimProvider ClaimProvider
}

type ServiceOption func(*service)

func WithClaimProvider(provider ClaimProvider) ServiceOption {
	return func(s *service) {
		s.claimProvider = provider
	}
}

// NewService constructs the production Service without an analytics
// recorder; state transitions are silent.
func NewService(repo Repository, detectors *Registry, resolvers *ResolverRegistry, opts ...ServiceOption) Service {
	s := &service{repo: repo, detectors: detectors, resolvers: resolvers}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewServiceWithAnalytics constructs the Service with an analytics
// recorder wired so every state transition emits an event.
func NewServiceWithAnalytics(repo Repository, detectors *Registry, resolvers *ResolverRegistry, recorder AnalyticsRecorder, opts ...ServiceOption) Service {
	s := &service{repo: repo, detectors: detectors, resolvers: resolvers, recorder: recorder}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
		return nil, ErrInvalidInput{Reason: "scenario is required"}
	}
	if s.detectors == nil {
		return nil, errors.New("no detector registry registered")
	}
	claimProvider := in.ClaimProvider
	if claimProvider == nil {
		claimProvider = s.claimProvider
	}
	conflicts, err := s.detectors.DetectAll(ctx, DetectInput{
		Scenario:        scenario,
		Snapshot:        in.Snapshot,
		DomainMap:       in.DomainMap,
		VerdictProvider: in.VerdictProvider,
		ClaimProvider:   claimProvider,
	})
	if err != nil {
		return nil, err
	}
	// Mark conflicts sanctioned by active in-repo markers before persisting,
	// so the suppressed-with-reason state is durable and visible everywhere.
	conflicts = applySuppressions(conflicts, in.Suppressions, in.DomainMap)
	// Assign deterministic stable_id (and a per-run instance_id) so two
	// runs that detect the same underlying drift collapse onto the same
	// row, and so external callers can dedupe across runs.
	conflicts = assignStableIDs(conflicts)
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

// ValidateConflicts returns the currently-detected conflicts and a clean
// flag (true ↔ zero outstanding of severity≥error, suppressed excluded).
// Detection-only: there is no lifecycle to net out, so every persisted,
// non-suppressed conflict is "outstanding." Walking findings toward zero
// over time is the campaign domain's job.
func (s *service) ValidateConflicts(ctx context.Context, scenario string) ([]Conflict, bool, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, false, ErrInvalidInput{Reason: "scenario is required"}
	}
	page, err := s.repo.ListConflicts(ctx, ListConflictsFilter{Scenario: scenario, PageSize: 10000})
	if err != nil {
		return nil, false, err
	}
	var outstanding []Conflict
	clean := true
	for _, c := range page.Conflicts {
		// A finding sanctioned by an active in-repo marker is intentional,
		// not outstanding — it never blocks cartographer-clean closure.
		if c.Suppressed {
			continue
		}
		outstanding = append(outstanding, c)
		if IsDeterministicGateFinding(c) {
			clean = false
		}
	}
	return outstanding, clean, nil
}

// IsDeterministicGateFinding is the single gate predicate for conflict
// findings: only deterministic ERROR/BLOCKER findings can fail an audit.
func IsDeterministicGateFinding(c Conflict) bool {
	return c.FindingClass == FindingClassDeterministic &&
		(c.Severity == SeverityError || c.Severity == SeverityBlocker)
}

func (s *service) UpsertConflicts(ctx context.Context, scenario string, conflicts []Conflict) ([]Conflict, error) {
	out := make([]Conflict, 0, len(conflicts))
	for _, c := range conflicts {
		c.Scenario = scenario
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
