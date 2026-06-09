package findings

import (
	"context"
	"strings"
	"time"
)

// DefaultActor labels mutations that arrive through the manual (UI/CLI) path.
const DefaultActor = "operator"

// Service is the application-layer surface handlers depend on. It owns
// validation and default substitution; persistence and the audit trail live in
// the Repository.
type Service interface {
	Add(ctx context.Context, in NewFinding) (Finding, error)
	Get(ctx context.Context, id string) (Finding, error)
	List(ctx context.Context, f ListFilter) ([]Finding, error)
	Edit(ctx context.Context, id string, in EditInput) (Finding, error)
	Supersede(ctx context.Context, id, replacement, reason string) (Finding, error)
	Flag(ctx context.Context, id, reason string) (Finding, error)
	// ResolveDispute closes a DISPUTED finding: resolution "keep" returns it to
	// ACTIVE (clearing the dispute note), "supersede" retires it in favor of
	// replacement. Every resolution writes an audit row.
	ResolveDispute(ctx context.Context, id, resolution, replacement, reason string) (Finding, error)
	Prune(ctx context.Context, dryRun bool) ([]string, error)
	CountInWindow(ctx context.Context, from, to time.Time) (int, error)

	// Read-path helpers used by the semantic search service.
	GetMany(ctx context.Context, ids []string) (map[string]Finding, error)
	LoadIndexable(ctx context.Context) ([]Finding, error)
	SearchArchivedLike(ctx context.Context, query string, limit int) ([]Finding, error)
}

type service struct {
	repo  Repository
	actor string
}

// NewService constructs the production Service, attributing mutations to the
// manual operator.
func NewService(repo Repository) Service {
	return &service{repo: repo, actor: DefaultActor}
}

// NewServiceWithActor constructs a Service that attributes mutations to actor
// (e.g. "agent" for the L3 reconcile loop).
func NewServiceWithActor(repo Repository, actor string) Service {
	if strings.TrimSpace(actor) == "" {
		actor = DefaultActor
	}
	return &service{repo: repo, actor: actor}
}

var _ Service = (*service)(nil)

func clampConfidence(c float64) float64 {
	switch {
	case c < 0:
		return 0
	case c > 1:
		return 1
	default:
		return c
	}
}

func (s *service) Add(ctx context.Context, in NewFinding) (Finding, error) {
	in.Claim = strings.TrimSpace(in.Claim)
	if in.Claim == "" {
		return Finding{}, ErrInvalidFinding{Field: "claim", Reason: "required"}
	}
	in.Confidence = clampConfidence(in.Confidence)
	in.Source = normalizeSource(in.Source)
	return s.repo.Add(ctx, in, s.actor)
}

func (s *service) Get(ctx context.Context, id string) (Finding, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, f ListFilter) ([]Finding, error) {
	if f.Status != "" && !ValidStatus(f.Status) {
		return nil, ErrInvalidFinding{Field: "status", Reason: "unknown status " + f.Status}
	}
	return s.repo.List(ctx, f)
}

func (s *service) Edit(ctx context.Context, id string, in EditInput) (Finding, error) {
	in.Claim = strings.TrimSpace(in.Claim)
	if in.Claim == "" {
		return Finding{}, ErrInvalidFinding{Field: "claim", Reason: "required"}
	}
	in.Confidence = clampConfidence(in.Confidence)
	return s.repo.Edit(ctx, id, in, s.actor)
}

func (s *service) Supersede(ctx context.Context, id, replacement, reason string) (Finding, error) {
	if strings.TrimSpace(id) == "" {
		return Finding{}, ErrInvalidFinding{Field: "id", Reason: "required"}
	}
	return s.repo.Supersede(ctx, id, replacement, reason, s.actor)
}

func (s *service) Flag(ctx context.Context, id, reason string) (Finding, error) {
	if strings.TrimSpace(id) == "" {
		return Finding{}, ErrInvalidFinding{Field: "id", Reason: "required"}
	}
	if strings.TrimSpace(reason) == "" {
		return Finding{}, ErrInvalidFinding{Field: "reason", Reason: "required"}
	}
	return s.repo.Flag(ctx, id, reason, s.actor)
}

func (s *service) ResolveDispute(ctx context.Context, id, resolution, replacement, reason string) (Finding, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Finding{}, ErrInvalidFinding{Field: "id", Reason: "required"}
	}
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case ResolutionKeep:
		return s.repo.Resolve(ctx, id, reason, s.actor)
	case ResolutionSupersede:
		if strings.TrimSpace(replacement) == "" {
			return Finding{}, ErrInvalidFinding{Field: "replacement", Reason: "required for supersede resolution"}
		}
		return s.repo.Supersede(ctx, id, replacement, reason, s.actor)
	default:
		return Finding{}, ErrInvalidFinding{Field: "resolution", Reason: "must be keep or supersede"}
	}
}

func (s *service) Prune(ctx context.Context, dryRun bool) ([]string, error) {
	return s.repo.Prune(ctx, dryRun, s.actor)
}

func (s *service) CountInWindow(ctx context.Context, from, to time.Time) (int, error) {
	return s.repo.Count(ctx, from, to)
}

func (s *service) GetMany(ctx context.Context, ids []string) (map[string]Finding, error) {
	return s.repo.GetMany(ctx, ids)
}

func (s *service) LoadIndexable(ctx context.Context) ([]Finding, error) {
	return s.repo.LoadIndexable(ctx)
}

func (s *service) SearchArchivedLike(ctx context.Context, query string, limit int) ([]Finding, error) {
	return s.repo.SearchArchivedLike(ctx, query, limit)
}
