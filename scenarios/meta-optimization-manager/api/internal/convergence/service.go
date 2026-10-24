package convergence

import (
	"context"

	"github.com/vrooli/api-core/schedule"
)

// Service is the convergence application surface.
type Service interface {
	GetConvergenceStatus(ctx context.Context) (Status, error)
	GetTemplateFitness(ctx context.Context, template string) ([]TemplateFitness, error)
	ListReferences(ctx context.Context, eligibility ReferenceEligibility) ([]ReferenceHealth, error)
	GetConvergenceTrend(ctx context.Context, template string) ([]FitnessTrendPoint, error)
}

type service struct {
	fitness FitnessScanner
	refs    ReferenceScanner
	repo    Repository
	clock   schedule.Clock
}

// Deps wires the convergence Service. Repo is optional (nil disables the trend).
type Deps struct {
	Fitness    FitnessScanner
	References ReferenceScanner
	Repo       Repository
	Clock      schedule.Clock
}

// NewService constructs the convergence Service, defaulting the production seams.
func NewService(d Deps) Service {
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	if d.Fitness == nil {
		d.Fitness = NewFitnessScanner()
	}
	if d.References == nil {
		d.References = NewReferenceScanner(d.Clock)
	}
	return &service{fitness: d.Fitness, refs: d.References, repo: d.Repo, clock: d.Clock}
}

var _ Service = (*service)(nil)

// GetConvergenceStatus computes fitness across all templates + reference health,
// persisting a dated audit record (best-effort) so the trend accumulates.
// Degrades gracefully: an unreadable scanner contributes nothing rather than
// failing the whole status.
func (s *service) GetConvergenceStatus(ctx context.Context) (Status, error) {
	fitness := s.scanFitness(ctx, "")
	refs := s.scanReferences(ctx)
	if s.repo != nil {
		now := s.clock.Now().UTC()
		if len(fitness) > 0 {
			_ = s.repo.SaveFitness(ctx, fitness, now) // best-effort trend write
		}
		if len(refs) > 0 {
			_ = s.repo.SaveReferences(ctx, refs, now)
		}
	}
	return Status{Templates: fitness, References: refs}, nil
}

// GetTemplateFitness computes the four-lens fitness for one template (or all).
// It persists a dated record too, so calling it also advances the trend.
func (s *service) GetTemplateFitness(ctx context.Context, template string) ([]TemplateFitness, error) {
	fitness := s.scanFitness(ctx, template)
	if s.repo != nil && len(fitness) > 0 {
		_ = s.repo.SaveFitness(ctx, fitness, s.clock.Now().UTC())
	}
	return fitness, nil
}

// ListReferences returns reference health, optionally filtered by eligibility.
func (s *service) ListReferences(ctx context.Context, eligibility ReferenceEligibility) ([]ReferenceHealth, error) {
	refs := s.scanReferences(ctx)
	if eligibility == EligibilityUnspecified {
		return refs, nil
	}
	out := make([]ReferenceHealth, 0, len(refs))
	for _, h := range refs {
		if h.Eligibility == eligibility {
			out = append(out, h)
		}
	}
	return out, nil
}

// GetConvergenceTrend returns dated trend points from the fitness-audit index.
func (s *service) GetConvergenceTrend(ctx context.Context, template string) ([]FitnessTrendPoint, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.Trend(ctx, template)
}

// scanFitness runs the fitness scanner, degrading to empty on error.
func (s *service) scanFitness(ctx context.Context, template string) []TemplateFitness {
	if s.fitness == nil {
		return nil
	}
	out, err := s.fitness.Scan(ctx, template)
	if err != nil {
		return nil
	}
	return out
}

// scanReferences runs the reference scanner, degrading to empty on error.
func (s *service) scanReferences(ctx context.Context) []ReferenceHealth {
	if s.refs == nil {
		return nil
	}
	out, err := s.refs.Scan(ctx)
	if err != nil {
		return nil
	}
	return out
}
