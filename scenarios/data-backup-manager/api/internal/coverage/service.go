package coverage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrSensitiveReviewRequired = errors.New("sensitive targets can only be registered through a verified Notification Hub review")
var ErrNoSensitiveSuggestions = errors.New("there are no current sensitive coverage suggestions to review")

// Service is the application surface the coverage handler depends on. It holds
// no durable state; every method derives from the composed seams.
type Service interface {
	// Report derives the full coverage picture: registered targets annotated
	// with planned/backed-up/verified state, plus recommended (non-sensitive) and
	// sensitive (review-only) suggestions, plus a summary scorecard.
	Report(ctx context.Context) (Report, error)

	// AcceptDefaults bulk-registers non-sensitive discovered durable targets.
	// Sensitive suggestions always require the verified Notification Hub review;
	// opts.IncludeSensitive is retained only to reject legacy bypass attempts.
	// opts.DryRun registers nothing and reports what would be registered.
	AcceptDefaults(ctx context.Context, opts AcceptOptions) (AcceptResult, error)

	// AcceptSensitive registers only the current sensitive suggestions whose
	// stable ids were explicitly approved by the verified decision callback.
	AcceptSensitive(ctx context.Context, suggestionIDs []string) (AcceptResult, error)

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
	recommended, sensitive, reacquirable := classifySuggestions(suggestions)

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
	destinationsByTarget := map[string][]string{}
	destinationBindingsKnown := false
	independentByTarget := map[string]bool{}
	independenceWarnings := map[string][]string{}
	if source, ok := s.deps.Plans.(DestinationPlanCatalog); ok {
		destinationBindingsKnown = true
		destinationsByTarget, err = source.DestinationIDsByTarget(ctx, ids)
		if err != nil {
			return Report{}, fmt.Errorf("read enabled target destination bindings: %w", err)
		}
		if topologySource, ok := s.deps.Plans.(DestinationTopologyCatalog); ok {
			independentByTarget, independenceWarnings, err = topologySource.PhysicalIndependenceByTarget(ctx, destinationsByTarget)
			if err != nil {
				return Report{}, fmt.Errorf("assess target destination topology: %w", err)
			}
		}
	}
	lastSuccess, err := s.deps.Runs.LastSuccessByTarget(ctx, ids)
	if err != nil {
		return Report{}, fmt.Errorf("read run status: %w", err)
	}
	lastVerified, err := s.deps.Restores.LastVerifiedByTarget(ctx, ids)
	if err != nil {
		return Report{}, fmt.Errorf("read verified status: %w", err)
	}
	lastSuccessByDestination := map[string]time.Time{}
	if source, ok := s.deps.Runs.(DestinationRunStatusSource); ok {
		lastSuccessByDestination, err = source.LastSuccessByTargetDestination(ctx, ids)
		if err != nil {
			return Report{}, fmt.Errorf("read target destination backup status: %w", err)
		}
	}
	lastVerifiedByDestination := map[string]time.Time{}
	if source, ok := s.deps.Restores.(DestinationVerifiedSource); ok {
		lastVerifiedByDestination, err = source.LastVerifiedByTargetDestination(ctx, ids)
		if err != nil {
			return Report{}, fmt.Errorf("read target destination restore verification: %w", err)
		}
	}

	rows := make([]RegisteredTarget, 0, len(registered))
	var ownerCandidates map[string]DiscoveredTarget
	if source, ok := s.deps.Suggestions.(CandidateSource); ok {
		candidates, candidateErr := source.ListTargetCandidates(ctx)
		if candidateErr != nil {
			return Report{}, fmt.Errorf("list owner target candidates: %w", candidateErr)
		}
		ownerCandidates = make(map[string]DiscoveredTarget, len(candidates))
		for _, candidate := range candidates {
			ownerCandidates[candidateKey(candidate.Owner, candidate.Name, candidate.Locator)] = candidate
		}
	}
	summary := Summary{
		RegisteredCount:         len(registered),
		RecommendedCount:        len(recommended),
		SensitiveCount:          len(sensitive),
		DefaultCoverageComplete: len(recommended) == 0,
		HasSensitiveUnreviewed:  len(sensitive) > 0,
	}
	for _, t := range registered {
		row := RegisteredTarget{CatalogTarget: t}
		if candidate, ok := ownerCandidates[candidateKey(t.Owner, t.Name, t.Locator)]; ok {
			row.Reacquirable = candidate.Reacquirable
			row.ReacquisitionCommand = candidate.ReacquisitionCommand
			row.ReacquisitionEvidence = candidate.ReacquisitionEvidence
			row.ReacquisitionNeedsOperator = candidate.ReacquisitionNeedsOperator
			row.ReacquisitionAuthReason = candidate.ReacquisitionAuthReason
		}
		if _, ok := planned[t.ID]; ok {
			row.Planned = true
			summary.PlannedCount++
		} else {
			summary.HasUnplannedRegisteredTargets = true
		}
		if !destinationBindingsKnown {
			if ts, ok := lastSuccess[t.ID]; ok && !ts.IsZero() {
				row.LastSuccessAt = ts
				summary.BackedUpCount++
			}
			if ts, ok := lastVerified[t.ID]; ok && !ts.IsZero() {
				row.LastVerifiedAt = ts
			}
		}
		destinationIDs := destinationsByTarget[t.ID]
		row.PhysicalIndependenceProven = independentByTarget[t.ID]
		row.IndependenceWarnings = append([]string(nil), independenceWarnings[t.ID]...)
		if destinationBindingsKnown && len(destinationIDs) == 0 {
			row.IndependenceWarnings = append(row.IndependenceWarnings, "no enabled destination binding exists")
		}
		row.AllDestinationsBackedUp = len(destinationIDs) > 0
		row.AllDestinationsVerified = len(destinationIDs) > 0
		for _, destinationID := range destinationIDs {
			status := DestinationStatus{DestinationID: destinationID}
			status.LastSuccessAt = lastSuccessByDestination[targetDestinationKey(t.ID, destinationID)]
			status.LastVerifiedAt = lastVerifiedByDestination[targetDestinationKey(t.ID, destinationID)]
			row.Destinations = append(row.Destinations, status)
			if status.LastSuccessAt.IsZero() {
				row.AllDestinationsBackedUp = false
			} else {
				summary.DestinationCopyCount++
				if row.LastSuccessAt.IsZero() || status.LastSuccessAt.Before(row.LastSuccessAt) {
					row.LastSuccessAt = status.LastSuccessAt
				}
			}
			if status.LastVerifiedAt.IsZero() {
				row.AllDestinationsVerified = false
			} else if row.LastVerifiedAt.IsZero() || status.LastVerifiedAt.Before(row.LastVerifiedAt) {
				row.LastVerifiedAt = status.LastVerifiedAt
			}
		}
		if len(destinationIDs) >= 2 {
			summary.MultiDestinationTargetCount++
		}
		if row.PhysicalIndependenceProven {
			summary.PhysicallyIndependentTargetCount++
		}
		if destinationBindingsKnown {
			if row.AllDestinationsBackedUp {
				summary.BackedUpCount++
				summary.AllDestinationBackedUpCount++
			}
			if row.AllDestinationsVerified {
				summary.VerifiedCount++
				summary.AllDestinationVerifiedCount++
			} else {
				summary.HasUnverifiedTargets = true
			}
		} else if len(destinationIDs) == 0 {
			// Preserve the narrow legacy seam's target-level behavior for unit
			// callers that do not provide destination bindings. Production always
			// implements DestinationPlanCatalog and uses pair-specific receipts.
			row.AllDestinationsBackedUp = !row.LastSuccessAt.IsZero()
			row.AllDestinationsVerified = !row.LastVerifiedAt.IsZero()
			if row.AllDestinationsBackedUp {
				summary.AllDestinationBackedUpCount++
			}
			if row.AllDestinationsVerified {
				summary.AllDestinationVerifiedCount++
				summary.VerifiedCount++
			}
		}
		if !row.AllDestinationsVerified && destinationBindingsKnown {
			summary.HasUnverifiedTargets = true
		}
		rows = append(rows, row)
	}

	return Report{
		Summary:      summary,
		Registered:   rows,
		Recommended:  recommended,
		Sensitive:    sensitive,
		Reacquirable: reacquirable,
	}, nil
}

func targetDestinationKey(targetID, destinationID string) string {
	return targetID + "\x00" + destinationID
}

func candidateKey(owner, name, locator string) string {
	if owner != "" && name != "" {
		return owner + "\x00" + name
	}
	return "\x00" + locator
}

func (s *service) AcceptDefaults(ctx context.Context, opts AcceptOptions) (AcceptResult, error) {
	if opts.IncludeSensitive {
		return AcceptResult{}, ErrSensitiveReviewRequired
	}
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("list target suggestions: %w", err)
	}

	result := AcceptResult{DryRun: opts.DryRun}
	for _, sug := range suggestions {
		if sug.Reacquirable {
			continue
		}
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

func (s *service) AcceptSensitive(ctx context.Context, suggestionIDs []string) (AcceptResult, error) {
	wanted := make(map[string]struct{}, len(suggestionIDs))
	for _, id := range suggestionIDs {
		if id == "" {
			return AcceptResult{}, fmt.Errorf("sensitive suggestion id is required")
		}
		if _, duplicate := wanted[id]; duplicate {
			return AcceptResult{}, fmt.Errorf("sensitive suggestion %q is repeated", id)
		}
		wanted[id] = struct{}{}
	}
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("list target suggestions: %w", err)
	}
	current := make(map[string]Suggestion, len(suggestions))
	for _, suggestion := range suggestions {
		if suggestion.Sensitive {
			current[suggestion.ID] = suggestion
		}
	}
	for id := range wanted {
		if _, ok := current[id]; !ok {
			return AcceptResult{}, fmt.Errorf("sensitive suggestion %q is no longer current", id)
		}
	}
	result := AcceptResult{}
	for _, suggestion := range suggestions {
		if _, ok := wanted[suggestion.ID]; !ok || !suggestion.Sensitive {
			continue
		}
		target, registerErr := s.deps.Targets.Register(ctx, RegisterInput{
			Owner: suggestion.Owner, Name: suggestion.Name, SourceKind: suggestion.SourceKind,
			Locator: suggestion.Locator, Critical: suggestion.Critical,
		})
		if registerErr != nil {
			result.Failed = append(result.Failed, AcceptFailure{SuggestionID: suggestion.ID, Owner: suggestion.Owner, Name: suggestion.Name, Message: registerErr.Error()})
			continue
		}
		result.Accepted = append(result.Accepted, AcceptedTarget{
			TargetID: target.ID, SuggestionID: suggestion.ID, Owner: suggestion.Owner, Name: suggestion.Name,
			SourceKind: suggestion.SourceKind, Locator: suggestion.Locator, Sensitive: true, Critical: suggestion.Critical,
		})
	}
	return result, nil
}

func (s *service) UnregisteredDefaultTargets(ctx context.Context) ([]Suggestion, error) {
	suggestions, err := s.deps.Suggestions.ListTargetSuggestions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list target suggestions: %w", err)
	}
	recommended, _, _ := classifySuggestions(suggestions)
	return recommended, nil
}

// splitSensitive partitions suggestions into non-sensitive (recommended
// defaults) and sensitive (review-only), preserving discovery's order.
func classifySuggestions(in []Suggestion) (recommended, sensitive, reacquirable []Suggestion) {
	for _, s := range in {
		if s.Sensitive {
			sensitive = append(sensitive, s)
		} else if s.Reacquirable {
			reacquirable = append(reacquirable, s)
		} else {
			recommended = append(recommended, s)
		}
	}
	return recommended, sensitive, reacquirable
}
