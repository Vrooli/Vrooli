package facets

import "context"

import "vrooli-memory/internal/inference"

type Service struct {
	repo       Repository
	classifier inference.Client
}

func NewService(repo Repository, classifiers ...inference.Client) *Service {
	s := &Service{repo: repo}
	if len(classifiers) > 0 {
		s.classifier = classifiers[0]
	}
	return s
}
func (s *Service) Seed(ctx context.Context) error {
	return s.repo.Seed(ctx)
}

func (s *Service) SeedExamples(ctx context.Context) error         { return s.repo.SeedExamples(ctx) }
func (s *Service) Validate(ctx context.Context, id string) error  { return s.repo.Validate(ctx, id) }
func (s *Service) List(ctx context.Context) ([]Definition, error) { return s.repo.List(ctx) }
func (s *Service) ReFacet(ctx context.Context, a Assignment) (Assignment, error) {
	return s.repo.Assign(ctx, a)
}

func (s *Service) Assign(ctx context.Context, entryID, facetID, actorID string) error {
	_, err := s.ReFacet(ctx, Assignment{EntryID: entryID, FacetID: facetID, ActorID: actorID})
	return err
}

func (s *Service) CompactionEligible(ctx context.Context, id string) (bool, error) {
	return s.repo.CompactionEligible(ctx, id)
}

func (s *Service) SetPin(ctx context.Context, id string, pinned bool) error {
	return s.repo.SetPin(ctx, id, pinned)
}

func (s *Service) MarkSuperseded(ctx context.Context, entryID, replacementID string) error {
	return s.repo.MarkSuperseded(ctx, entryID, replacementID)
}
func (s *Service) ResolveThread(ctx context.Context, entryID string) error {
	return s.repo.ResolveThread(ctx, entryID)
}
func (s *Service) ListPinProposals(ctx context.Context) ([]PinProposal, error) {
	return s.repo.ListPinProposals(ctx)
}
func (s *Service) ResolvePinProposal(ctx context.Context, id string, accept bool) error {
	return s.repo.ResolvePinProposal(ctx, id, accept)
}

func (s *Service) CreateRule(ctx context.Context, rule Rule) (Rule, error) {
	return s.repo.CreateRule(ctx, rule)
}

func (s *Service) ListRules(ctx context.Context, scope string) ([]Rule, error) {
	return s.repo.ListRules(ctx, scope)
}

func (s *Service) DryRunRule(ctx context.Context, id string) (DryRun, error) {
	return s.repo.DryRunRule(ctx, id)
}
func (s *Service) EnableRule(ctx context.Context, id string) error { return s.repo.EnableRule(ctx, id) }
func (s *Service) RevertRule(ctx context.Context, id string) (int, error) {
	return s.repo.RevertRule(ctx, id)
}

type RefacetResult struct {
	Total, Assigned, RuleAssigned, Classified, Failed int
}

// RefacetCorpus replays the current policy over every immutable entry. Rule
// decisions take precedence; remaining entries use the scenario inference
// seam. Each result is appended as a new assignment with its decision actor,
// making the operation auditable and reversible through rule revert.
func (s *Service) RefacetCorpus(ctx context.Context, scope string) (RefacetResult, error) {
	if scope == "" {
		scope = "agent-memory"
	}
	entries, err := s.repo.ListCorpus(ctx, scope)
	if err != nil {
		return RefacetResult{}, err
	}
	result := RefacetResult{Total: len(entries)}
	for _, entry := range entries {
		facetID, actorID, matched, err := s.MatchRule(ctx, scope, entry.Body, entry.SourceRuntime, entry.Kind, entry.SourcePath)
		if err != nil {
			result.Failed++
			continue
		}
		if matched {
			result.RuleAssigned++
		} else {
			if s.classifier == nil {
				result.Failed++
				continue
			}
			if contextual, ok := s.classifier.(inference.ContextualClassifier); ok {
				facetID, err = contextual.ClassifyEntry(ctx, entry.Body, entry.Kind)
			} else {
				facetID, err = s.classifier.Classify(ctx, entry.Body)
			}
			actorID = "classifier"
			if err != nil {
				result.Failed++
				continue
			}
			result.Classified++
		}
		if err := s.Validate(ctx, facetID); err != nil {
			result.Failed++
			continue
		}
		if err := s.Assign(ctx, entry.ID, facetID, actorID); err != nil {
			result.Failed++
			continue
		}
		result.Assigned++
	}
	return result, nil
}

func (s *Service) MatchRule(ctx context.Context, scope, body, sourceRuntime, kind, sourcePath string) (string, string, bool, error) {
	rule, matched, err := s.repo.MatchRule(ctx, scope, RuleInput{Body: body, SourceRuntime: sourceRuntime, Kind: kind, SourcePath: sourcePath})
	if err != nil || !matched {
		return "", "", matched, err
	}
	return rule.FacetID, rule.ID, true, nil
}
