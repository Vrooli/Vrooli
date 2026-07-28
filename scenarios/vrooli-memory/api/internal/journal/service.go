package journal

import (
	"context"
	"strings"

	"vrooli-memory/internal/inference"
)

type Service struct {
	repo      Repository
	inference inference.Client
	facets    FacetResolver
}

// FacetResolver is the journal's narrow policy seam. The facets domain owns
// the seeded vocabulary and assignment history; journal only consumes it.
type FacetResolver interface {
	Validate(context.Context, string) error
	Assign(context.Context, string, string) error
}

func NewService(repo Repository, client inference.Client, facetResolvers ...FacetResolver) *Service {
	s := &Service{repo: repo, inference: client}
	if len(facetResolvers) > 0 {
		s.facets = facetResolvers[0]
	}
	return s
}

func (s *Service) Append(ctx context.Context, e Entry) (Entry, error) {
	var retry []string
	if e.FacetID != "" && s.facets != nil {
		if err := s.facets.Validate(ctx, e.FacetID); err != nil {
			return Entry{}, err
		}
	}
	facet, err := s.inference.Classify(ctx, e.Body)
	if err != nil || strings.TrimSpace(facet) == "" {
		e.FacetID = UnclassifiedFacet
		retry = append(retry, "classify")
	} else {
		e.FacetID = strings.TrimSpace(facet)
		if s.facets != nil {
			if err := s.facets.Validate(ctx, e.FacetID); err != nil {
				e.FacetID = UnclassifiedFacet
				retry = append(retry, "classify")
			}
		}
	}
	e.FacetTexts = []FacetText{{Kind: "topic", Text: e.Body}, {Kind: "rule", Text: "Implication: " + e.Body}, {Kind: "entities", Text: "Entities: " + e.Body}}
	for i := range e.FacetTexts {
		v, err := s.inference.Embed(ctx, e.FacetTexts[i].Text, inference.EmbeddingClustering)
		if err != nil {
			retry = append(retry, "embed")
			continue
		}
		e.FacetTexts[i].Vector = v
	}
	persisted, err := s.repo.Append(ctx, e, retry)
	if err != nil {
		return Entry{}, err
	}
	if s.facets != nil && !persisted.Existing && persisted.FacetID != UnclassifiedFacet {
		if err := s.facets.Assign(ctx, persisted.ID, persisted.FacetID); err != nil {
			return Entry{}, err
		}
	}
	return persisted, nil
}

func (s *Service) Get(ctx context.Context, id string) (Entry, error) { return s.repo.Get(ctx, id) }
func (s *Service) List(ctx context.Context, limit int) ([]Entry, error) {
	return s.repo.List(ctx, limit)
}

func (s *Service) FindByImportKey(ctx context.Context, key string) (Entry, bool, error) {
	return s.repo.FindByImportKey(ctx, key)
}
