package facets

import "context"

type Service struct{ repo Repository }

func NewService(repo Repository) *Service                         { return &Service{repo: repo} }
func (s *Service) Seed(ctx context.Context) error                 { return s.repo.Seed(ctx) }
func (s *Service) Validate(ctx context.Context, id string) error  { return s.repo.Validate(ctx, id) }
func (s *Service) List(ctx context.Context) ([]Definition, error) { return s.repo.List(ctx) }
func (s *Service) ReFacet(ctx context.Context, a Assignment) (Assignment, error) {
	return s.repo.Assign(ctx, a)
}

func (s *Service) Assign(ctx context.Context, entryID, facetID string) error {
	_, err := s.ReFacet(ctx, Assignment{EntryID: entryID, FacetID: facetID})
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
