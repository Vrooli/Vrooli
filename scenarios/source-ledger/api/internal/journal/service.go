package journal

import (
	"context"
	"strings"

	"source-ledger/internal/inference"
	"source-ledger/internal/policy"

	"github.com/vrooli/api-core/provenance"
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
	Assign(context.Context, string, string, string) error
	MatchRule(context.Context, string, string, string, string, string) (string, string, bool, error)
}

func NewService(repo Repository, client inference.Client, facetResolvers ...FacetResolver) *Service {
	s := &Service{repo: repo, inference: client}
	if len(facetResolvers) > 0 {
		s.facets = facetResolvers[0]
	}
	return s
}

func (s *Service) Append(ctx context.Context, e Entry) (Entry, error) {
	e.Scope = string(policy.NormalizeScope(e.Scope))
	// An empty verification status means this entry has not passed a provenance
	// seam yet — a direct caller rather than the Connect handler. Derive the
	// full attribution here so both seams persist identical correlation.
	if strings.TrimSpace(e.Attribution.VerificationStatus) == "" {
		e.Attribution, e.Correlation = AttributionFrom(provenance.FromContext(ctx), e.Attribution)
	}
	// Direct callers may provide the scope on the entry rather than on the
	// request context. Carry it into policy validation before any facet lookup.
	ctx = policy.WithScope(ctx, e.Scope)
	// Import keys are content-addressed. Resolve them before classification or
	// embedding so replaying a large source file is cheap and cannot create
	// duplicate inference work.
	if e.ImportKey != "" {
		if existing, found, err := s.repo.FindByImportKey(ctx, e.ImportKey); err != nil {
			return Entry{}, err
		} else if found {
			existing.Existing = true
			return existing, nil
		}
	}
	var retry []string
	if e.FacetID != "" && s.facets != nil {
		if err := s.facets.Validate(ctx, e.FacetID); err != nil {
			return Entry{}, err
		}
	}
	// An explicit facet is a caller-owned deterministic mapping (for example,
	// a provenance adapter importing a typed JSONL file). Never replace it with
	// a classifier guess. Automatic classification is only for entries that do
	// not carry a validated facet.
	actorID := "explicit"
	if strings.TrimSpace(e.FacetID) == "" {
		facet, ruleActor, matched, err := s.ruleFacet(ctx, e)
		if err != nil {
			return Entry{}, err
		}
		if !matched {
			facet, err = s.classify(ctx, e.Body, e.Kind)
			ruleActor = "classifier"
		}
		if err != nil || strings.TrimSpace(facet) == "" {
			e.FacetID = UnclassifiedFacet
			actorID = "classifier"
			retry = append(retry, "classify")
		} else {
			e.FacetID = strings.TrimSpace(facet)
			actorID = ruleActor
		}
	}
	if s.facets != nil {
		if err := s.facets.Validate(ctx, e.FacetID); err != nil {
			e.FacetID = UnclassifiedFacet
			retry = append(retry, "classify")
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
		if err := s.facets.Assign(ctx, persisted.ID, persisted.FacetID, actorID); err != nil {
			return Entry{}, err
		}
	}
	return persisted, nil
}

func (s *Service) Get(ctx context.Context, id string) (Entry, error) { return s.repo.Get(ctx, id) }
func (s *Service) List(ctx context.Context, limit int) ([]Entry, error) {
	return s.repo.List(ctx, limit)
}

func (s *Service) ListAfter(ctx context.Context, cursor string, limit int) ([]Entry, error) {
	return s.repo.ListAfter(ctx, cursor, limit)
}

func (s *Service) ListByRun(ctx context.Context, runID string, limit int) ([]Entry, error) {
	return s.repo.ListByRun(ctx, runID, limit)
}

func (s *Service) FindByImportKey(ctx context.Context, key string) (Entry, bool, error) {
	return s.repo.FindByImportKey(ctx, key)
}

// ProcessClassificationRetries replays failed inference through an append-only
// facet assignment. It never changes an immutable entries row; successful
// retry work is acknowledged only after its assignment is persisted.
func (s *Service) ProcessClassificationRetries(ctx context.Context, limit int) (RetryResult, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	// Reconcile by state before draining the queue. An entry that was written
	// while inference was unavailable is unclassified whether or not a queue row
	// survived, and queue rows have been lost before. State is the authority.
	if _, err := s.repo.EnqueueUnclassified(ctx); err != nil {
		return RetryResult{}, err
	}
	resolved, err := s.repo.PruneResolvedClassificationRetries(ctx)
	if err != nil {
		return RetryResult{}, err
	}
	result := RetryResult{AlreadyResolved: resolved}
	items, err := s.repo.ClassificationRetries(ctx, limit)
	if err != nil {
		return result, err
	}
	for _, item := range items {
		facet, actorID, matched, err := s.ruleFacet(ctx, item.Entry)
		if err != nil {
			result.Deferred++
			continue
		}
		if !matched {
			facet, err = s.classify(ctx, item.Entry.Body, item.Entry.Kind)
			actorID = "classifier"
		}
		if err != nil || strings.TrimSpace(facet) == "" {
			result.Deferred++
			continue
		}
		facet = strings.TrimSpace(facet)
		if s.facets == nil || s.facets.Validate(ctx, facet) != nil || s.facets.Assign(ctx, item.Entry.ID, facet, actorID) != nil {
			result.Deferred++
			continue
		}
		if err := s.repo.AcknowledgeRetry(ctx, item.ID); err != nil {
			return result, err
		}
		result.Processed++
	}
	return result, nil
}

func (s *Service) classify(ctx context.Context, body, kind string) (string, error) {
	if contextual, ok := s.inference.(inference.ContextualClassifier); ok {
		return contextual.ClassifyEntry(ctx, body, kind)
	}
	return s.inference.Classify(ctx, body)
}

func (s *Service) ruleFacet(ctx context.Context, e Entry) (string, string, bool, error) {
	if s.facets == nil {
		return "", "", false, nil
	}
	facet, ruleID, matched, err := s.facets.MatchRule(ctx, e.Scope, e.Body, e.Attribution.SourceRuntime, e.Kind, e.Import.Path)
	if err != nil || !matched {
		return "", "", matched, err
	}
	return facet, "rule:" + ruleID, true, nil
}

// ProcessEmbeddingRetries fills only missing derived vectors, then removes the
// operational retry records. Journal prose and facet-text rows stay immutable.
func (s *Service) ProcessEmbeddingRetries(ctx context.Context, limit int) (RetryResult, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	resolved, err := s.repo.PruneResolvedEmbeddingRetries(ctx)
	if err != nil {
		return RetryResult{}, err
	}
	result := RetryResult{AlreadyResolved: resolved}
	items, err := s.repo.EmbeddingRetries(ctx, limit)
	if err != nil {
		return result, err
	}
	for _, item := range items {
		complete := true
		for _, facetText := range item.Entry.FacetTexts {
			vector, err := s.inference.Embed(ctx, facetText.Text, inference.EmbeddingClustering)
			if err != nil {
				complete = false
				break
			}
			if err := s.repo.StoreFacetEmbedding(ctx, facetText.ID, vector); err != nil {
				return result, err
			}
		}
		if !complete {
			result.Deferred++
			continue
		}
		if err := s.repo.AcknowledgeEmbeddingRetries(ctx, item.Entry.ID); err != nil {
			return result, err
		}
		result.Processed++
	}
	return result, nil
}
