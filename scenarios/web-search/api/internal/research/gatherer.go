package research

import (
	"context"

	"web-search/internal/findings"
)

// GatherHit is one semantic-search hit the gatherer consumes: a finding id and
// its relevance score. It keeps the gatherer decoupled from the concrete
// semantic-index package (main.go adapts findingindex.Hit into this shape).
type GatherHit struct {
	FindingID string
	Score     float64
}

// SemanticIndex returns finding ids semantically near a query, already bounded
// by the caller's limit. The production impl wraps the findings semantic index;
// a test fake satisfies it.
type SemanticIndex interface {
	Search(ctx context.Context, query string, limit int) ([]GatherHit, error)
}

// FindingsStore loads full findings by id so the gatherer can project the
// fields the reconcile step reasons over.
type FindingsStore interface {
	GetMany(ctx context.Context, ids []string) (map[string]findings.Finding, error)
}

// IndexGatherer is the production FindingsGatherer: it runs the semantic index
// for nearby finding ids, loads the full findings, and projects them to
// GatheredFinding preserving relevance order. A finding that the index returned
// but the store no longer has (e.g. raced supersede) is skipped.
type IndexGatherer struct {
	Index SemanticIndex
	Store FindingsStore
}

// Gather satisfies FindingsGatherer.
func (g IndexGatherer) Gather(ctx context.Context, query string, limit int) ([]GatheredFinding, error) {
	hits, err := g.Index.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		if h.FindingID != "" {
			ids = append(ids, h.FindingID)
		}
	}
	byID, err := g.Store.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]GatheredFinding, 0, len(hits))
	for _, h := range hits {
		f, ok := byID[h.FindingID]
		if !ok {
			continue
		}
		out = append(out, GatheredFinding{
			FindingID:  f.ID,
			Claim:      f.Claim,
			Confidence: f.Confidence,
			Status:     f.Status,
			Score:      h.Score,
		})
	}
	return out, nil
}
