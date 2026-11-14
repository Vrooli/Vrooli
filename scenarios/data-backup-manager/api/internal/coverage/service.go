package coverage

import (
	"context"
	"fmt"
)

// Service is the application surface the coverage handler depends on. It holds
// no durable state; every method derives from the composed seams.
type Service interface {
	// Report derives the full coverage picture: registered targets annotated
	// with planned/backed-up/verified state, plus recommended (non-sensitive) and
	// sensitive (review-only) suggestions, plus a summary scorecard.
	Report(ctx context.Context) (Report, error)

	// AcceptDefaults bulk-registers discovered durable targets. Sensitive
	// suggestions are skipped unless opts.IncludeSensitive. opts.DryRun registers
	// nothing and reports what would be registered. Idempotent: discovery already
	// excludes registered targets, so re-running accepts nothing new.
	AcceptDefaults(ctx context.Context, opts AcceptOptions) (AcceptResult, error)

	// UnregisteredDefaultTargets returns the non-sensitive recommended targets
	// not yet registered. The plans coverage guard consults this seam to block
	// plans created with incomplete default coverage.
	UnregisteredDefaultTargets(ctx context.Context) ([]Suggestion, error)
}

// Deps wires the seams the service composes.
type Deps struct {
	Suggestions SuggestionSource
	Targets     TargetCatalog
	Plans       PlanCatalog
	Runs        RunStatusSource
	Restores    VerifiedSource
}

type service struct {
	deps Deps
}

// NewService constructs the production Service.
func NewService(d Deps) Service { return &service{deps: d} }

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Report(ctx context.Context) (Report, error) {
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("list target suggestions: %w", err)
	}
	recommended, sensitive := splitSensitive(suggestions)

	registered, err := s.deps.Targets.List(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("read target catalog: %w", err)
	}

	ids := make([]string, 0, len(registered))
	for _, t := range registered {
		ids = append(ids, t.ID)
	}

	planned, err := s.deps.Plans.PlannedTargetIDs(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("read plan bindings: %w", err)
	}
	lastSuccess, err := s.deps.Runs.LastSuccessByTarget(ctx, ids)
	if err != nil {
		return Report{}, fmt.Errorf("read run status: %w", err)
	}
	lastVerified, err := s.deps.Restores.LastVerifiedByTarget(ctx, ids)
	if err != nil {
		return Report{}, fmt.Errorf("read verified status: %w", err)
	}

	rows := make([]RegisteredTarget, 0, len(registered))
	summary := Summary{
		RegisteredCount:         len(registered),
		RecommendedCount:        len(recommended),
		SensitiveCount:          len(sensitive),
		DefaultCoverageComplete: len(recommended) == 0,
		HasSensitiveUnreviewed:  len(sensitive) > 0,
	}
	for _, t := range registered {
		row := RegisteredTarget{CatalogTarget: t}
		if _, ok := planned[t.ID]; ok {
			row.Planned = true
			summary.PlannedCount++
		} else {
			summary.HasUnplannedRegisteredTargets = true
		}
		if ts, ok := lastSuccess[t.ID]; ok && !ts.IsZero() {
			row.LastSuccessAt = ts
			summary.BackedUpCount++
		}
		if ts, ok := lastVerified[t.ID]; ok && !ts.IsZero() {
			row.LastVerifiedAt = ts
			summary.VerifiedCount++
		} else {
			summary.HasUnverifiedTargets = true
		}
		rows = append(rows, row)
	}

	return Report{
		Summary:     summary,
		Registered:  rows,
		Recommended: recommended,
		Sensitive:   sensitive,
	}, nil
}

func (s *service) AcceptDefaults(ctx context.Context, opts AcceptOptions) (AcceptResult, error) {
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("list target suggestions: %w", err)
	}

	result := AcceptResult{DryRun: opts.DryRun}
	for _, sug := range suggestions {
		if sug.Sensitive && !opts.IncludeSensitive {
			result.SkippedSensitive = append(result.SkippedSensitive, sug)
			continue
		}
		if opts.DryRun {
			result.Accepted = append(result.Accepted, AcceptedTarget{
				SuggestionID: sug.ID,
				Owner:        sug.Owner,
				Name:         sug.Name,
				SourceKind:   sug.SourceKind,
				Locator:      sug.Locator,
				Sensitive:    sug.Sensitive,
				Critical:     sug.Critical,
			})
			continue
		}
		// Register preserves owner/name/source_kind/locator from the suggestion
		// exactly (no content read). Registration is an idempotent upsert, so a
		// concurrent or repeat accept produces no duplicate.
		t, rerr := s.deps.Targets.Register(ctx, RegisterInput{
			Owner:      sug.Owner,
			Name:       sug.Name,
			SourceKind: sug.SourceKind,
			Locator:    sug.Locator,
			Critical:   sug.Critical,
		})
		if rerr != nil {
			result.Failed = append(result.Failed, AcceptFailure{
				SuggestionID: sug.ID,
				Owner:        sug.Owner,
				Name:         sug.Name,
				Message:      rerr.Error(),
			})
			continue
		}
		result.Accepted = append(result.Accepted, AcceptedTarget{
			TargetID:     t.ID,
			SuggestionID: sug.ID,
			Owner:        sug.Owner,
			Name:         sug.Name,
			SourceKind:   sug.SourceKind,
			Locator:      sug.Locator,
			Sensitive:    sug.Sensitive,
			Critical:     sug.Critical,
		})
	}
	return result, nil
}

func (s *service) UnregisteredDefaultTargets(ctx context.Context) ([]Suggestion, error) {
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list target suggestions: %w", err)
	}
	recommended, _ := splitSensitive(suggestions)
	return recommended, nil
}

// splitSensitive partitions suggestions into non-sensitive (recommended
// defaults) and sensitive (review-only), preserving discovery's order.
func splitSensitive(in []Suggestion) (recommended, sensitive []Suggestion) {
	for _, s := range in {
		if s.Sensitive {
			sensitive = append(sensitive, s)
		} else {
			recommended = append(recommended, s)
		}
	}
	return recommended, sensitive
}
